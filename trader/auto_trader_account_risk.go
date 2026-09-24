package trader

import (
	"fmt"
	"time"

	"nofx/store"
)

// Account-level circuit breaker.
//
// Per-position protection already lived in this package: every order ships with
// protective prices (auto_trader_protection.go) and the profit-giveback monitor
// runs every minute (auto_trader_risk.go). What was missing was the rule above
// the position: nothing measured the whole book. MaxDailyLoss and MaxDrawdown
// were declared on the config with no reader, stopUntil was only ever checked,
// dailyPnL was only ever reset, and the daily_pnl reported to the UI was
// therefore always zero.
//
// This file closes that gap with mark-to-market equity, taken once per cycle
// from the same balance read the trading context already performs:
//
//	daily loss = (day-opening equity − current equity) / day-opening equity
//	drawdown   = (peak equity − current equity) / peak equity
//
// Breaching either pauses trading for the configured duration. It deliberately
// does NOT close positions: exchange-side stops keep working while paused, and
// the profit-giveback monitor keeps running, so a pause cannot strand a
// position with no protection. Closing would also turn a pause into a
// market-order cascade at the worst possible moment.

// accountRiskPolicy is the configured breaker in the form the loop uses.
type accountRiskPolicy struct {
	dailyLossPct float64
	drawdownPct  float64
	pause        time.Duration
	enforced     bool
}

// accountRiskVerdict is one cycle's measurement.
type accountRiskVerdict struct {
	// Reason is non-empty only when a limit was breached this cycle.
	Reason    string
	DailyLoss float64
	Drawdown  float64
	PauseLeft time.Duration
}

// accountRiskState is the measurement history. It is process-local on purpose:
// it is derived from live equity, so rebuilding it after a restart from a stale
// file would be less accurate than starting from the current balance.
type accountRiskState struct {
	dayStartEquity float64
	dayStartAt     time.Time
	equityPeak     float64
	breakerReason  string
	lastDailyLoss  float64
	lastDrawdown   float64
}

// accountRiskPolicy reads the configured limits. The strategy's RiskControl is
// the primary source (it is hot-reloaded with the rest of the strategy); the
// AutoTraderConfig fields remain as an explicit programmatic fallback for
// embedders that set them directly.
func (at *AutoTrader) accountRiskPolicy() accountRiskPolicy {
	policy := accountRiskPolicy{}
	if at == nil {
		return policy
	}

	if at.strategyEngine != nil {
		if config := at.strategyEngine.GetConfig(); config != nil {
			policy.dailyLossPct, policy.drawdownPct, policy.pause, policy.enforced =
				config.RiskControl.AccountCircuitBreaker()
		}
	}
	if policy.enforced {
		return policy
	}

	// Both legacy fields are documented as percentages and are read as their
	// magnitude, so -10 and 10 mean the same limit.
	policy.dailyLossPct = absPercentValue(at.config.MaxDailyLoss)
	policy.drawdownPct = absPercentValue(at.config.MaxDrawdown)
	policy.enforced = policy.dailyLossPct > 0 || policy.drawdownPct > 0
	policy.pause = at.config.StopTradingTime
	if policy.pause <= 0 {
		policy.pause = time.Duration(store.DefaultStopTradingMinutes) * time.Minute
	}
	return policy
}

func absPercentValue(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// evaluateAccountRisk measures equity and arms the pause when a limit is
// breached. Call it every cycle after equity is known and before the model is
// asked to trade. An empty Reason means trading may continue.
func (at *AutoTrader) evaluateAccountRisk(equity float64) accountRiskVerdict {
	if at == nil || equity <= 0 {
		return accountRiskVerdict{}
	}
	policy := at.accountRiskPolicy()

	at.accountRiskMu.Lock()
	defer at.accountRiskMu.Unlock()

	now := time.Now()

	// A pause whose trigger has been switched off must not outlive its config.
	if !policy.enforced {
		if !at.stopUntil.IsZero() {
			at.logInfof("▶️ Account risk limits are no longer configured; clearing the active pause")
		}
		at.stopUntil = time.Time{}
		at.accountRisk.breakerReason = ""
		at.accountRisk.dayStartEquity = equity
		at.accountRisk.equityPeak = equity
		at.accountRisk.dayStartAt = now
		at.dailyPnL = 0
		return accountRiskVerdict{}
	}

	// Daily window. The measured window is what the limit is written against,
	// so the baseline is the equity at the start of it, not the equity the
	// process happened to boot at.
	if at.accountRisk.dayStartAt.IsZero() {
		at.accountRisk.dayStartAt = now
	}
	if now.Sub(at.accountRisk.dayStartAt) >= 24*time.Hour {
		at.logInfof("📅 New trading day: account risk window reset at %.2f USDT equity", equity)
		at.accountRisk.dayStartAt = now
		at.accountRisk.dayStartEquity = equity
		at.accountRisk.breakerReason = ""
		at.stopUntil = time.Time{}
	}
	if at.accountRisk.dayStartEquity <= 0 {
		at.accountRisk.dayStartEquity = equity
	}

	// A pause that has run its course releases the drawdown baseline before the
	// new measurement is taken. Without this re-arm the historical peak would
	// keep describing a level the account has already recovered to, and the
	// first cycle after every pause would immediately trip the same rule again.
	if !at.stopUntil.IsZero() && now.After(at.stopUntil) {
		at.stopUntil = time.Time{}
		at.accountRisk.breakerReason = ""
		at.accountRisk.equityPeak = equity
		at.logInfof("▶️ Risk pause expired; trading resumes with the drawdown baseline re-armed at %.2f USDT", equity)
	}

	if at.accountRisk.equityPeak <= 0 || equity > at.accountRisk.equityPeak {
		at.accountRisk.equityPeak = equity
	}

	verdict := accountRiskVerdict{}
	if policy.dailyLossPct > 0 && at.accountRisk.dayStartEquity > 0 {
		verdict.DailyLoss = (at.accountRisk.dayStartEquity - equity) / at.accountRisk.dayStartEquity * 100
	}
	if policy.drawdownPct > 0 && at.accountRisk.equityPeak > 0 {
		verdict.Drawdown = (at.accountRisk.equityPeak - equity) / at.accountRisk.equityPeak * 100
	}

	switch {
	case policy.dailyLossPct > 0 && verdict.DailyLoss >= policy.dailyLossPct:
		verdict.Reason = fmt.Sprintf("daily loss %.2f%% reached the configured %.2f%% limit (day opened at %.2f USDT)",
			verdict.DailyLoss, policy.dailyLossPct, at.accountRisk.dayStartEquity)
	case policy.drawdownPct > 0 && verdict.Drawdown >= policy.drawdownPct:
		verdict.Reason = fmt.Sprintf("drawdown %.2f%% from the %.2f USDT equity peak reached the configured %.2f%% limit",
			verdict.Drawdown, at.accountRisk.equityPeak, policy.drawdownPct)
	}

	// Mark-to-market daily P&L, which is the number the limit is written
	// against. It replaced a counter that was only ever reset to zero.
	at.dailyPnL = equity - at.accountRisk.dayStartEquity
	at.accountRisk.lastDailyLoss = verdict.DailyLoss
	at.accountRisk.lastDrawdown = verdict.Drawdown

	if verdict.Reason == "" {
		return verdict
	}

	pause := policy.pause
	if pause <= 0 {
		pause = time.Duration(store.DefaultStopTradingMinutes) * time.Minute
	}
	if until := now.Add(pause); until.After(at.stopUntil) {
		at.stopUntil = until
	}
	verdict.PauseLeft = time.Until(at.stopUntil)
	if at.accountRisk.breakerReason != verdict.Reason {
		at.logWarnf("🛑 Account-level circuit breaker: %s — pausing for %v", verdict.Reason, pause)
	}
	at.accountRisk.breakerReason = verdict.Reason
	return verdict
}

// riskPause reports an active pause: how long is left, why, and whether one is
// in force at all.
func (at *AutoTrader) riskPause() (remaining time.Duration, reason string, paused bool) {
	if at == nil {
		return 0, "", false
	}
	at.accountRiskMu.RLock()
	defer at.accountRiskMu.RUnlock()

	if at.stopUntil.IsZero() || !time.Now().Before(at.stopUntil) {
		return 0, "", false
	}
	reason = at.accountRisk.breakerReason
	if reason == "" {
		reason = "account-level risk control"
	}
	return time.Until(at.stopUntil), reason, true
}

// accountRiskReport is the read-only view handed to the API payload, so the
// dashboard can show the same numbers the breaker acts on.
func (at *AutoTrader) accountRiskReport() map[string]interface{} {
	if at == nil {
		return nil
	}
	policy := at.accountRiskPolicy()

	at.accountRiskMu.RLock()
	defer at.accountRiskMu.RUnlock()

	report := map[string]interface{}{
		"daily_pnl":        at.dailyPnL,
		"stopped":          !at.stopUntil.IsZero() && time.Now().Before(at.stopUntil),
		"stop_until":       at.stopUntil.Format(time.RFC3339),
		"enforced":         policy.enforced,
		"day_start_equity": at.accountRisk.dayStartEquity,
		"equity_peak":      at.accountRisk.equityPeak,
		"daily_loss_pct":   at.accountRisk.lastDailyLoss,
		"drawdown_pct":     at.accountRisk.lastDrawdown,
		"limit_daily_loss": policy.dailyLossPct,
		"limit_drawdown":   policy.drawdownPct,
		"pause_minutes":    int(policy.pause.Minutes()),
	}
	if at.accountRisk.breakerReason != "" {
		report["reason"] = at.accountRisk.breakerReason
	}
	return report
}
