package marketdata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"nofx/logger"
)

// Direction bias labels. They are the same three words the retired Vergex board
// used, so existing prompts and operator habits still read correctly.
const (
	BiasBullish = "bullish"
	BiasBearish = "bearish"
	BiasNeutral = "neutral"
)

// DefaultDirectionHistoryPath is where the direction timeline is persisted.
// It sits next to the SQLite database so a single volume carries all state.
const DefaultDirectionHistoryPath = "data/direction_history.json"

// DirectionHistoryPathEnv overrides the persistence path.
const DirectionHistoryPathEnv = "MARKET_DIRECTION_HISTORY_PATH"

const (
	// directionHistoryMaxChanges bounds the change log. The timeline is prompt
	// context, not an audit trail: keeping it small keeps the file small and the
	// prompt cheap.
	directionHistoryMaxChanges = 200
	// directionHistoryMaxStates bounds how many instruments are tracked.
	directionHistoryMaxStates = 400
)

// DirectionComponent is one auditable input behind a bias. Keeping the
// components (rather than only the verdict) is what makes a change explainable:
// the reason for a flip is whichever component's vote flipped.
type DirectionComponent struct {
	// Name is the stable component id ("momentum", "premium", "flow").
	Name string `json:"name"`
	// Vote is BiasBullish, BiasBearish or BiasNeutral.
	Vote string `json:"vote"`
	// Detail is the human-readable magnitude, e.g. "-2.10σ" or "-0.052%".
	Detail string `json:"detail"`
}

// DirectionState is one instrument's most recent directional read.
type DirectionState struct {
	Symbol     string               `json:"symbol"`
	Bias       string               `json:"bias"`
	Score      float64              `json:"score"`
	Bullish    int                  `json:"bullish"`
	Bearish    int                  `json:"bearish"`
	Neutral    int                  `json:"neutral"`
	Components []DirectionComponent `json:"components,omitempty"`
	UpdatedAt  time.Time            `json:"updated_at"`
}

// DirectionChange is a recorded bias transition plus the evidence that caused it.
type DirectionChange struct {
	Symbol    string    `json:"symbol"`
	FromBias  string    `json:"from_bias"`
	ToBias    string    `json:"to_bias"`
	FromScore float64   `json:"from_score"`
	ToScore   float64   `json:"to_score"`
	Reason    string    `json:"reason"`
	ChangedAt time.Time `json:"changed_at"`
}

// directionHistoryFile is the on-disk shape.
type directionHistoryFile struct {
	States  []DirectionState  `json:"states"`
	Changes []DirectionChange `json:"changes"`
}

// DirectionTracker keeps the latest state per instrument and a bounded timeline
// of bias transitions.
//
// It survives restarts on purpose: the value of a direction timeline is that it
// still says "flipped two hours ago" after a redeploy, and a process that
// restarts frequently would otherwise always report an empty history.
//
// Every failure mode degrades to in-memory operation. A missing directory, a
// read-only volume or a corrupt file must never stop a trading cycle, so
// persistence errors are swallowed after a single warning.
type DirectionTracker struct {
	mu      sync.Mutex
	path    string
	states  map[string]DirectionState
	changes []DirectionChange

	warnOnce sync.Once
}

// NewDirectionTracker builds a tracker, loading any existing history. A path of
// "" disables persistence and keeps everything in memory.
func NewDirectionTracker(path string) *DirectionTracker {
	tracker := &DirectionTracker{
		path:   path,
		states: make(map[string]DirectionState),
	}
	tracker.load()
	return tracker
}

// Path reports the backing file, for logs and diagnostics.
func (t *DirectionTracker) Path() string {
	if t == nil {
		return ""
	}
	return t.path
}

// Observe records the latest reads and returns the transitions they produced.
//
// The first observation of an instrument is not a change: an empty prerequisite
// state would otherwise report every instrument as having "flipped" on the first
// cycle after a fresh install.
func (t *DirectionTracker) Observe(now time.Time, next []DirectionState) []DirectionChange {
	if t == nil || len(next) == 0 {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	var produced []DirectionChange
	for _, state := range next {
		state.Symbol = strings.TrimSpace(state.Symbol)
		if state.Symbol == "" {
			continue
		}
		if state.UpdatedAt.IsZero() {
			state.UpdatedAt = now
		}

		previous, seen := t.states[state.Symbol]
		t.states[state.Symbol] = state

		if !seen {
			continue
		}
		if previous.Bias == state.Bias {
			continue
		}
		produced = append(produced, DirectionChange{
			Symbol:    state.Symbol,
			FromBias:  previous.Bias,
			ToBias:    state.Bias,
			FromScore: previous.Score,
			ToScore:   state.Score,
			Reason:    DescribeTransition(previous, state),
			ChangedAt: now,
		})
	}

	if len(produced) > 0 {
		t.changes = append(t.changes, produced...)
		t.trimLocked()
	}
	t.saveLocked()
	return produced
}

// Recent returns the most recent changes, newest first, restricted to the given
// window. A non-positive window means "no time limit".
func (t *DirectionTracker) Recent(limit int, within time.Duration) []DirectionChange {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := time.Time{}
	if within > 0 {
		cutoff = time.Now().Add(-within)
	}

	out := make([]DirectionChange, 0, limit)
	for i := len(t.changes) - 1; i >= 0; i-- {
		change := t.changes[i]
		if !cutoff.IsZero() && change.ChangedAt.Before(cutoff) {
			continue
		}
		out = append(out, change)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

// States returns a copy of the current state per instrument, symbol-ordered.
func (t *DirectionTracker) States() []DirectionState {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	out := make([]DirectionState, 0, len(t.states))
	for _, state := range t.states {
		out = append(out, state)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Symbol < out[j].Symbol })
	return out
}

// trimLocked bounds the in-memory log. Callers must hold the mutex.
func (t *DirectionTracker) trimLocked() {
	if len(t.changes) > directionHistoryMaxChanges {
		t.changes = t.changes[len(t.changes)-directionHistoryMaxChanges:]
	}
	if len(t.states) > directionHistoryMaxStates {
		// Drop the least recently updated instruments so the tracked set cannot
		// grow without bound as the candidate universe rotates.
		type entry struct {
			symbol string
			at     time.Time
		}
		entries := make([]entry, 0, len(t.states))
		for symbol, state := range t.states {
			entries = append(entries, entry{symbol: symbol, at: state.UpdatedAt})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].at.Before(entries[j].at) })
		for _, e := range entries[:len(entries)-directionHistoryMaxStates] {
			delete(t.states, e.symbol)
		}
	}
}

// load reads the backing file. Any failure leaves the tracker empty rather than
// propagating: history is an enhancement, never a prerequisite.
func (t *DirectionTracker) load() {
	if t.path == "" {
		return
	}
	raw, err := os.ReadFile(t.path)
	if err != nil {
		return
	}

	var file directionHistoryFile
	if err := json.Unmarshal(raw, &file); err != nil {
		// A corrupt file is more likely to be a truncated write than a trap.
		// Start clean; the next save overwrites it.
		t.warn("direction history at %s is unreadable (%v); starting a fresh timeline", t.path, err)
		return
	}

	for _, state := range file.States {
		if symbol := strings.TrimSpace(state.Symbol); symbol != "" {
			t.states[symbol] = state
		}
	}
	t.changes = file.Changes
	t.trimLocked()
}

// saveLocked writes the backing file atomically. Callers must hold the mutex.
func (t *DirectionTracker) saveLocked() {
	if t.path == "" {
		return
	}

	payload, err := json.Marshal(directionHistoryFile{
		States:  t.statesSliceLocked(),
		Changes: t.changes,
	})
	if err != nil {
		t.warn("direction history could not be encoded: %v", err)
		return
	}

	dir := filepath.Dir(t.path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.warn("direction history directory %s is not writable: %v", dir, err)
			return
		}
	}

	// Write to a sibling then rename, so a crash mid-write cannot leave a
	// half-encoded file for the next start to trip over.
	tmp := t.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		t.warn("direction history could not be written: %v", err)
		return
	}
	if err := os.Rename(tmp, t.path); err != nil {
		t.warn("direction history could not be committed: %v", err)
	}
}

// statesSliceLocked flattens the state map. Callers must hold the mutex.
func (t *DirectionTracker) statesSliceLocked() []DirectionState {
	out := make([]DirectionState, 0, len(t.states))
	for _, state := range t.states {
		out = append(out, state)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Symbol < out[j].Symbol })
	return out
}

// warn emits a persistence warning once, so a read-only volume does not spam the
// log every trading cycle.
func (t *DirectionTracker) warn(format string, args ...any) {
	t.warnOnce.Do(func() {
		logger.Warnf(format, args...)
	})
}

// DescribeTransition explains a bias flip using the component votes, which is
// the whole point of carrying them: "momentum bullish→bearish" is actionable,
// "score changed" is not.
func DescribeTransition(previous, next DirectionState) string {
	if len(previous.Components) == 0 {
		return "first observation with component detail"
	}

	previousVotes := make(map[string]string, len(previous.Components))
	for _, component := range previous.Components {
		previousVotes[component.Name] = component.Vote
	}

	var flips []string
	for _, component := range next.Components {
		was, known := previousVotes[component.Name]
		delete(previousVotes, component.Name)
		switch {
		case !known:
			flips = append(flips, fmt.Sprintf("%s became available (%s)", component.Name, component.Vote))
		case was != component.Vote:
			flips = append(flips, fmt.Sprintf("%s %s→%s", component.Name, was, component.Vote))
		}
	}
	for name := range previousVotes {
		flips = append(flips, name+" became unavailable")
	}

	if len(flips) == 0 {
		// The verdict moved without any single input flipping, which can only
		// happen when the aggregate landed on the other side of the threshold.
		return fmt.Sprintf("no component flipped; score moved %.2f→%.2f", previous.Score, next.Score)
	}
	sort.Strings(flips)
	return strings.Join(flips, "; ")
}

// ScoreBias converts component votes into a verdict and the vote counts.
//
// Ties resolve to neutral on purpose: a two-component split is genuinely
// undecided, and forcing a direction there would manufacture conviction the
// inputs do not support.
func ScoreBias(votes []string) (bias string, bullish, bearish, neutral int) {
	for _, vote := range votes {
		switch vote {
		case BiasBullish:
			bullish++
		case BiasBearish:
			bearish++
		default:
			neutral++
		}
	}
	switch {
	case bullish > bearish:
		return BiasBullish, bullish, bearish, neutral
	case bearish > bullish:
		return BiasBearish, bullish, bearish, neutral
	default:
		return BiasNeutral, bullish, bearish, neutral
	}
}

// ResolveDirectionHistoryPath returns the configured history path, falling back
// to DefaultDirectionHistoryPath.
func ResolveDirectionHistoryPath() string {
	if custom := strings.TrimSpace(os.Getenv(DirectionHistoryPathEnv)); custom != "" {
		return custom
	}
	return DefaultDirectionHistoryPath
}
