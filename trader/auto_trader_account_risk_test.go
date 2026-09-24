package trader

import (
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// newBreakerTrader builds the smallest AutoTrader the breaker needs: a strategy
// engine carrying the risk control under test. No exchange, no store, no AI.
func newBreakerTrader(t *testing.T, risk store.RiskControlConfig) *AutoTrader {
	t.Helper()
	config := &store.StrategyConfig{RiskControl: risk}
	return &AutoTrader{
		id:             "test",
		name:           "test",
		strategyEngine: kernel.NewStrategyEngine(config),
		config:         AutoTraderConfig{StrategyConfig: config},
		initialBalance: 10000,
		// The account risk window is seeded by the first measurement, exactly as
		// the constructor leaves it.
	}
}

func TestAccountCircuitBreakerTripsOnDailyLoss(t *testing.T) {
	at := newBreakerTrader(t, store.RiskControlConfig{
		MaxDailyLossPct:    10,
		StopTradingMinutes: 60,
	})

	// The first cycle opens the day's window at the live equity.
	at.evaluateAccountRisk(10000)

	if verdict := at.evaluateAccountRisk(9600); verdict.Reason != "" {
		t.Fatalf("a 4%% loss must not trip a 10%% limit, got %q", verdict.Reason)
	}
	if _, _, paused := at.riskPause(); paused {
		t.Fatal("trader must not be paused below the limit")
	}

	verdict := at.evaluateAccountRisk(9000)
	if verdict.Reason == "" {
		t.Fatal("a 10% loss must trip a 10% limit")
	}
	if !strings.Contains(verdict.Reason, "daily loss") {
		t.Fatalf("reason should name the daily-loss rule, got %q", verdict.Reason)
	}
	remaining, reason, paused := at.riskPause()
	if !paused {
		t.Fatal("breaching the daily loss limit must pause trading")
	}
	if remaining <= 0 || remaining > 61*time.Minute {
		t.Fatalf("pause window should be about an hour, got %v", remaining)
	}
	if !strings.Contains(reason, "daily loss") {
		t.Fatalf("pause must carry the trigger reason, got %q", reason)
	}
}

func TestAccountCircuitBreakerTripsOnDrawdownFromPeak(t *testing.T) {
	at := newBreakerTrader(t, store.RiskControlConfig{
		MaxDrawdownPct:     20,
		StopTradingMinutes: 30,
	})

	at.evaluateAccountRisk(10000)
	// A peak the account then falls 25% from.
	at.evaluateAccountRisk(12000)
	if verdict := at.evaluateAccountRisk(11000); verdict.Reason != "" {
		t.Fatalf("a 8.3%% drawdown must not trip a 20%% limit, got %q", verdict.Reason)
	}

	verdict := at.evaluateAccountRisk(9000)
	if verdict.Reason == "" || !strings.Contains(verdict.Reason, "drawdown") {
		t.Fatalf("a 25%% drawdown must trip the 20%% limit, got %q", verdict.Reason)
	}
	if _, _, paused := at.riskPause(); !paused {
		t.Fatal("breaching the drawdown limit must pause trading")
	}
}

func TestAccountCircuitBreakerDisabledMeansNothingIsClaimed(t *testing.T) {
	at := newBreakerTrader(t, store.RiskControlConfig{})

	// A strategy with no limits must never pause, however far equity falls.
	for _, equity := range []float64{10000, 5000, 100} {
		if verdict := at.evaluateAccountRisk(equity); verdict.Reason != "" {
			t.Fatalf("unconfigured limits must not trip at %.0f equity, got %q", equity, verdict.Reason)
		}
	}
	if _, _, paused := at.riskPause(); paused {
		t.Fatal("unconfigured limits must not pause trading")
	}
}

func TestAccountCircuitBreakerClearsPauseWhenLimitsRemoved(t *testing.T) {
	at := newBreakerTrader(t, store.RiskControlConfig{MaxDailyLossPct: 10, StopTradingMinutes: 60})
	at.evaluateAccountRisk(10000)
	at.evaluateAccountRisk(8000)
	if _, _, paused := at.riskPause(); !paused {
		t.Fatal("expected a pause before the config changes")
	}

	// The user edits the strategy and switches the breaker off. The pause must
	// not outlive the rule that created it.
	at.strategyEngine = kernel.NewStrategyEngine(&store.StrategyConfig{
		RiskControl: store.RiskControlConfig{},
	})
	if verdict := at.evaluateAccountRisk(8000); verdict.Reason != "" {
		t.Fatalf("removing the limits must not re-trip, got %q", verdict.Reason)
	}
	if _, _, paused := at.riskPause(); paused {
		t.Fatal("removing the limits must clear the active pause")
	}
}

func TestAccountCircuitBreakerRearmsAfterPauseExpires(t *testing.T) {
	at := newBreakerTrader(t, store.RiskControlConfig{
		MaxDrawdownPct:     20,
		StopTradingMinutes: 1,
	})

	at.evaluateAccountRisk(10000)
	at.evaluateAccountRisk(7500) // 25% drawdown: trips
	if _, _, paused := at.riskPause(); !paused {
		t.Fatal("expected the drawdown rule to pause trading")
	}

	// Let the pause lapse, then measure again at the recovered equity. The peak
	// must be re-armed at the current equity, otherwise the same historical peak
	// would re-trip the breaker forever.
	at.accountRiskMu.Lock()
	at.stopUntil = time.Now().Add(-time.Second)
	at.accountRiskMu.Unlock()

	if verdict := at.evaluateAccountRisk(7500); verdict.Reason != "" {
		t.Fatalf("an expired pause must re-arm instead of re-tripping, got %q", verdict.Reason)
	}
	if _, _, paused := at.riskPause(); paused {
		t.Fatal("the pause must be over once it expires and the baseline is re-armed")
	}
}

func TestAccountCircuitBreakerDailyBaseLineIsMeasuredNotAssumed(t *testing.T) {
	at := newBreakerTrader(t, store.RiskControlConfig{MaxDailyLossPct: 10, StopTradingMinutes: 60})

	// First cycle establishes the window at the real equity, not at the value
	// the process was configured with.
	at.evaluateAccountRisk(5000)
	at.evaluateAccountRisk(4800) // -4% from the real baseline
	if verdict := at.evaluateAccountRisk(4800); verdict.Reason != "" {
		t.Fatalf("-4%% from the measured baseline must not trip a 10%% limit, got %q", verdict.Reason)
	}
	if verdict := at.evaluateAccountRisk(4400); verdict.Reason == "" {
		t.Fatal("-12% from the measured baseline must trip a 10% limit")
	}
}

func TestAccountCircuitBreakerTreatsSignedLimitsAsMagnitude(t *testing.T) {
	// The prompt has always written the daily limit as "-10". Both spellings
	// describe the same limit.
	negative := store.RiskControlConfig{MaxDailyLossPct: -10, MaxDrawdownPct: -15}
	daily, drawdown, pause, enforced := negative.AccountCircuitBreaker()
	if !enforced || daily != 10 || drawdown != 15 {
		t.Fatalf("signed limits must read as magnitude, got %v/%v enforced=%v", daily, drawdown, enforced)
	}
	if pause != time.Duration(store.DefaultStopTradingMinutes)*time.Minute {
		t.Fatalf("unset pause must fall back to the default, got %v", pause)
	}
}

func TestAccountRiskReportExposesWhatTheBreakerActsOn(t *testing.T) {
	at := newBreakerTrader(t, store.RiskControlConfig{MaxDailyLossPct: 10, StopTradingMinutes: 60})
	at.evaluateAccountRisk(9500)
	at.evaluateAccountRisk(9950)

	report := at.accountRiskReport()
	if report["daily_pnl"] != 450.0 && report["daily_pnl"] != 450 {
		t.Fatalf("daily_pnl must be the measured change, got %v", report["daily_pnl"])
	}
	if report["enforced"] != true {
		t.Fatal("report must say the limits are enforced")
	}
	if report["limit_daily_loss"] != 10.0 {
		t.Fatalf("report must carry the configured limit, got %v", report["limit_daily_loss"])
	}
	if stopped, _ := report["stopped"].(bool); stopped {
		t.Fatal("a 5% loss inside a 10% limit must not report as stopped")
	}
}
