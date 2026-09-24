package trader

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nofx/kernel"
	"nofx/logger"
	"nofx/marketdata"
	"nofx/marketdata/providers"
)

// Signal book persistence. The file sits next to the SQLite database so one
// volume carries all state, and can be redirected for deployments that keep
// runtime state elsewhere.
const (
	signalBookDirEnv      = "SIGNAL_BOOK_DIR"
	defaultSignalBookDir  = "data"
	signalBookFilePattern = "signal_book_%s.json"
)

// Signal-managed exits.
//
// Under the default exit_mode every trade ends the way the model set it up: a
// stop, a target, or its own closing decision. This file adds the third owner —
// the direction signal — for strategies that configure it.
//
// The mechanics are deliberately boring: remember the bias a position was opened
// on, compare it every cycle, and close when the read has turned against it or
// faded below a floor. What matters is what is NOT done here. Nothing overrides
// the model's own closes, nothing cancels the stop-loss, and no read is invented
// when the signal has no opinion: missing evidence is not the same as evidence
// of nothing, so a position whose instrument has never been observed is left
// alone rather than closed.

// signalRead is one direction observation: a verdict plus its strength.
type signalRead struct {
	Bias  string
	Score float64
}

// signalEntry is the read a position was opened against.
//
// Persisted to disk (one small JSON file per trader) because the entry thesis
// is not recoverable from the exchange: positions survive a restart, the reason
// they were opened does not. A book that lived only in memory meant that after
// every restart the loop had to guess, and the guess closed positions.
type signalEntry struct {
	Symbol string     `json:"symbol"`
	Side   string     `json:"side"`
	Read   signalRead `json:"read"`
	SeenAt time.Time  `json:"seen_at"`
}

// signalBook remembers what each open position was opened on.
//
// Entries are written through to disk so a restart does not erase the thesis
// behind an open position. When an entry genuinely is missing — a position that
// predates this file, or a lost/corrupt book — the position is evaluated on the
// direction its own side implies, and only the flip rule may act on it: the
// strength half of the rule needs a recorded entry score to compare against, and
// inventing one would be a fabricated reason to close.
type signalBook struct {
	mu      sync.Mutex
	entries map[string]signalEntry
	path    string
}

func newSignalBook(path string) *signalBook {
	book := &signalBook{entries: make(map[string]signalEntry), path: path}
	book.loadFromDisk()
	return book
}

// SignalBookPath returns the file a trader persists its entry theses to. One
// file per trader so two traders sharing a host cannot overwrite each other.
func SignalBookPath(traderID string) string {
	dir := strings.TrimSpace(os.Getenv(signalBookDirEnv))
	if dir == "" {
		dir = defaultSignalBookDir
	}
	return filepath.Join(dir, fmt.Sprintf(signalBookFilePattern, sanitizeFileToken(traderID)))
}

func sanitizeFileToken(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "default"
	}
	var sb strings.Builder
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			sb.WriteRune(r)
		default:
			sb.WriteRune('_')
		}
	}
	return sb.String()
}

// loadFromDisk restores a previous run's entries. A missing file is normal (first
// run); a corrupt one is reported and treated as empty rather than silently
// producing an empty book that looks like "nothing was ever recorded".
func (b *signalBook) loadFromDisk() {
	if b == nil || b.path == "" {
		return
	}
	raw, err := os.ReadFile(b.path)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warnf("⚠️ Could not read the signal book %s: %v — signal-managed exits will rely on flips only", b.path, err)
		}
		return
	}

	var entries map[string]signalEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		logger.Warnf("⚠️ Signal book %s is not valid JSON (%v) — starting empty", b.path, err)
		return
	}
	if len(entries) == 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.entries == nil {
		b.entries = make(map[string]signalEntry, len(entries))
	}
	for key, entry := range entries {
		if entry.Symbol == "" || entry.Side == "" {
			continue
		}
		b.entries[key] = entry
	}
	logger.Infof("📡 Restored %d signal book entr(ies) from %s", len(b.entries), b.path)
}

// persist writes the book atomically (temp file + rename) so a crash mid-write
// cannot leave a half-written book behind. Failures are logged, never fatal:
// losing the book degrades the exit rule, it does not stop trading.
func (b *signalBook) persist() {
	if b == nil || b.path == "" {
		return
	}
	b.mu.Lock()
	payload, err := json.MarshalIndent(b.entries, "", "  ")
	b.mu.Unlock()
	if err != nil {
		logger.Warnf("⚠️ Could not serialise the signal book: %v", err)
		return
	}

	if dir := filepath.Dir(b.path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			logger.Warnf("⚠️ Could not create the signal book directory %s: %v", dir, err)
			return
		}
	}
	tmp := b.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		logger.Warnf("⚠️ Could not write the signal book: %v", err)
		return
	}
	if err := os.Rename(tmp, b.path); err != nil {
		logger.Warnf("⚠️ Could not replace the signal book: %v", err)
		_ = os.Remove(tmp)
	}
}

func signalKey(symbol, side string) string {
	return normalizeUniverseSymbol(symbol) + "|" + strings.ToLower(strings.TrimSpace(side))
}

// Record stores the read a position was opened against.
func (b *signalBook) Record(symbol, side string, read signalRead) {
	if b == nil {
		return
	}
	b.mu.Lock()
	if b.entries == nil {
		b.entries = make(map[string]signalEntry)
	}
	b.entries[signalKey(symbol, side)] = signalEntry{
		Symbol: normalizeUniverseSymbol(symbol),
		Side:   strings.ToLower(strings.TrimSpace(side)),
		Read:   read,
		SeenAt: time.Now(),
	}
	b.mu.Unlock()
	b.persist()
}

// Read returns the recorded entry read, if any.
func (b *signalBook) Read(symbol, side string) (signalEntry, bool) {
	if b == nil {
		return signalEntry{}, false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	entry, ok := b.entries[signalKey(symbol, side)]
	return entry, ok
}

// Prune drops positions that are no longer open so the book cannot grow for the
// lifetime of the process. It only rewrites the file when something was actually
// removed, because Prune runs every managed cycle.
func (b *signalBook) Prune(active map[string]bool) {
	if b == nil {
		return
	}
	b.mu.Lock()
	removed := false
	for key := range b.entries {
		if !active[key] {
			delete(b.entries, key)
			removed = true
		}
	}
	b.mu.Unlock()

	if removed {
		b.persist()
	}
}

// Signal exit reasons. They are written into the decision reasoning and the
// cycle log so an operator can tell which rule ended the trade.
const (
	signalReasonFlip  = "signal_flip"
	signalReasonDecay = "signal_decay"
)

// oppositeBias reports whether two verdicts point at each other.
func oppositeBias(a, b string) bool {
	switch {
	case a == marketdata.BiasBullish && b == marketdata.BiasBearish:
		return true
	case a == marketdata.BiasBearish && b == marketdata.BiasBullish:
		return true
	default:
		return false
	}
}

// impliedRead is the direction a position's side expects: a long wants a
// bullish read, a short a bearish one. It stands in for the recorded entry when
// the book has no record, so a flip against the position's own side still ends
// it.
func impliedRead(side string) signalRead {
	if strings.EqualFold(strings.TrimSpace(side), "short") {
		return signalRead{Bias: marketdata.BiasBearish}
	}
	return signalRead{Bias: marketdata.BiasBullish}
}

// evaluateSignalExit decides whether one position should be closed.
//
// Two rules, both comparing now against the position's own thesis:
//
//   - flip: the read now points the other way. Neutral does not count — the
//     signal losing its opinion is the decay case, not a reversal.
//   - decay: absolute strength has fallen below the floor, i.e. whatever edge
//     justified the trade has faded to noise.
//
// entryKnown says whether the thesis behind the position was actually recorded.
// When it was not — a position opened before the book was persisted, or a lost
// book — only the flip rule runs. Comparing today's strength against a made-up
// entry would close positions for a reason nobody can audit, and on a restart
// that is every position at once.
//
// It returns "" for everything else, notably for instruments with no read at
// all: an unobserved symbol carries no evidence either way.
func evaluateSignalExit(side string, entry, current signalRead, entryKnown bool, floor float64) string {
	if current.Bias == "" {
		return ""
	}
	if entry.Bias == "" {
		entry = impliedRead(side)
	}
	if entry.Bias == marketdata.BiasNeutral {
		return ""
	}

	if oppositeBias(entry.Bias, current.Bias) {
		return signalReasonFlip
	}
	if !entryKnown {
		return ""
	}
	if floor > 0 && math.Abs(current.Score) < floor {
		return signalReasonDecay
	}
	return ""
}

// signalExitDecision renders a verdict as the decision the executor expects.
func signalExitDecision(symbol, side, reason string, entry, current signalRead) kernel.Decision {
	closeAction := "close_long"
	if strings.EqualFold(strings.TrimSpace(side), "short") {
		closeAction = "close_short"
	}
	return kernel.Decision{
		Symbol: symbol,
		Action: closeAction,
		Reasoning: fmt.Sprintf("Signal-managed exit (%s): opened on %s %+.2f, now %s %+.2f",
			reason, entry.Bias, entry.Score, current.Bias, current.Score),
	}
}

// directionalStateProvider is the production lookup, swapped in tests. It reads
// the same shared timeline the terminal panel renders, so the number that ends a
// trade is the number shown to the operator.
func (at *AutoTrader) directionalStateProvider(symbol string) (marketdata.DirectionState, bool) {
	if at == nil {
		return marketdata.DirectionState{}, false
	}
	if at.directionalState != nil {
		return at.directionalState(symbol)
	}
	return providers.DirectionalStateFor(symbol)
}

type signalExitConfig struct {
	managed bool
	floor   float64
}

// signalExitDecisions produces platform-driven closes for open positions.
//
// It runs before the model's own decisions are executed and only ever adds
// closes: entries are still the AI's call. Symbols the AI already decided on are
// skipped so the same position is not worked twice in one cycle.
func (at *AutoTrader) signalExitDecisions(ctx *kernel.Context, aiDecisions []kernel.Decision) []kernel.Decision {
	cfg := at.signalExitPolicy()
	if !cfg.managed || ctx == nil || len(ctx.Positions) == 0 {
		return nil
	}

	already := make(map[string]bool, len(aiDecisions))
	for _, decision := range aiDecisions {
		already[decision.Symbol] = true
	}

	active := make(map[string]bool, len(ctx.Positions))
	var exits []kernel.Decision
	for _, position := range ctx.Positions {
		side := strings.ToLower(strings.TrimSpace(position.Side))
		if side != "long" && side != "short" {
			continue
		}
		key := signalKey(position.Symbol, side)
		active[key] = true

		if already[position.Symbol] {
			continue
		}

		current, ok := at.directionalRead(position.Symbol)
		if !ok {
			at.logInfof("   • %s: no direction read yet, signal exit skipped", position.Symbol)
			continue
		}

		entry := impliedRead(side)
		entryKnown := false
		if recorded, found := at.signalBook.Read(position.Symbol, side); found {
			entry = recorded.Read
			entryKnown = true
		}

		reason := evaluateSignalExit(side, entry, current, entryKnown, cfg.floor)
		if reason == "" {
			continue
		}

		at.logWarnf("📡 Signal exit %s %s (%s): %s %+.2f → %s %+.2f",
			position.Symbol, side, reason, entry.Bias, entry.Score, current.Bias, current.Score)
		exits = append(exits, signalExitDecision(position.Symbol, side, reason, entry, current))
	}

	at.signalBook.Prune(active)
	return exits
}

// recordSignalEntries captures the read each freshly opened position starts on,
// so later cycles can tell whether the thesis still holds. A position opened
// while its instrument has never been observed falls back to the side's implied
// direction, which is the honest default rather than a fabricated read.
func (at *AutoTrader) recordSignalEntries(decisions []kernel.Decision) {
	if at == nil || at.signalBook == nil {
		return
	}
	for _, decision := range decisions {
		action := strings.ToLower(strings.TrimSpace(decision.Action))
		side := "long"
		switch action {
		case "open_long":
		case "open_short":
			side = "short"
		default:
			continue
		}

		read, ok := at.directionalRead(decision.Symbol)
		if !ok {
			at.logInfof("   • %s: opened without a direction read; signal exit will only act on a flip against the %s direction",
				decision.Symbol, side)
			continue
		}
		at.signalBook.Record(decision.Symbol, side, read)
		at.logInfof("   • %s: recorded entry signal %s %+.2f", decision.Symbol, read.Bias, read.Score)
	}
}

// signalExitPolicy reads the configured exit ownership. Everything unknown or
// missing resolves to "not managed": relaxing an exit rule because a field was
// left blank would be a silent risk-policy change.
func (at *AutoTrader) signalExitPolicy() signalExitConfig {
	if at == nil || at.strategyEngine == nil {
		return signalExitConfig{}
	}
	config := at.strategyEngine.GetConfig()
	if config == nil {
		return signalExitConfig{}
	}
	return signalExitConfig{
		managed: config.RiskControl.SignalManagedExit(),
		floor:   config.RiskControl.EffectiveSignalScoreFloor(),
	}
}

// directionalRead looks up the latest direction state for a symbol. Extracted so
// signal exits can be tested without a live insight pipeline.
func (at *AutoTrader) directionalRead(symbol string) (signalRead, bool) {
	state, ok := at.directionalStateProvider(symbol)
	if !ok {
		return signalRead{}, false
	}
	return signalRead{Bias: state.Bias, Score: state.Score}, true
}
