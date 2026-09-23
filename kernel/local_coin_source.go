package kernel

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"nofx/logger"
	"nofx/market"
	"nofx/provider/hyperliquid"
)

// This file holds the local, free replacements for the retired nofxos
// candidate-pool endpoints (/api/ai500/list and /api/oi/{top,low}-ranking).
//
// Both upstream endpoints answer 402 today, and they were the only reason the
// "ai500" and "oi_top" / "oi_low" candidate sources existed. Rather than leaving
// three dead sources that fail silently, they are rebuilt here on top of data
// the platform already fetches for free:
//
//   - OI change ranking: open interest per instrument comes from the keyless
//     Hyperliquid asset-contexts endpoint. What the vendor added was the
//     *history* — the 1h-old OI — so all that is missing locally is a snapshot
//     log. OISnapshotStore keeps one, bounded and persisted next to the rest of
//     the runtime state.
//   - Screener: there is no free equivalent of the vendor's AI score, so the
//     replacement is not a copy. It is an explicit attention score over inputs
//     every operator can see, and it says so in its own documentation.
//
// Everything degrades the same way a missing upstream did before, except louder:
// a ranking that cannot be computed returns an error, and the caller falls back
// to the static list with a warning that also reaches the prompt.

const (
	// DefaultOISnapshotPath is where the open-interest history is persisted.
	// It sits next to the SQLite database so one volume carries all state.
	DefaultOISnapshotPath = "data/oi_snapshot.json"
	// OISnapshotPathEnv overrides the persistence path.
	OISnapshotPathEnv = "MARKET_OI_SNAPSHOT_PATH"
)

const (
	// oiTargetWindow is the lookback the retired vendor ranking used (1h), and
	// therefore the window this replacement aims to report.
	oiTargetWindow = time.Hour
	// oiMinBaselineAge is the youngest history this ranking will report against.
	// A five-minute-old baseline would produce a noisy percentage and read like
	// a real 1h number, so during the cold start nothing is reported at all.
	oiMinBaselineAge = 15 * time.Minute
	// oiSnapshotMinInterval throttles writes. A cycle is minutes long and OI
	// moves slowly; recording every cycle would only grow the file.
	oiSnapshotMinInterval = 10 * time.Minute
	// oiMaxSnapshots bounds history to roughly a day and a half of coverage.
	oiMaxSnapshots = 36
	// oiMinOpenInterestUSD skips dust instruments whose percentage change is
	// dominated by rounding.
	oiMinOpenInterestUSD = 250_000
)

// OISnapshot is one observation of open interest across the universe.
type OISnapshot struct {
	TakenAt time.Time          `json:"taken_at"`
	OI      map[string]float64 `json:"oi"` // canonical symbol -> open interest in USD
}

// oiSnapshotFile is the on-disk shape.
type oiSnapshotFile struct {
	Snapshots []OISnapshot `json:"snapshots"`
}

// OISnapshotStore keeps a bounded history of open-interest observations.
//
// It follows the same contract as the direction tracker: it survives restarts so
// a redeploy does not erase the hour of history the ranking depends on, and it
// degrades to memory on every I/O failure because history is an enhancement,
// never a prerequisite for a trading cycle.
type OISnapshotStore struct {
	mu        sync.Mutex
	path      string
	snapshots []OISnapshot
	warnOnce  sync.Once
}

// ResolveOISnapshotPath returns the configured path, honouring the env override.
func ResolveOISnapshotPath() string {
	if custom := strings.TrimSpace(os.Getenv(OISnapshotPathEnv)); custom != "" {
		return custom
	}
	return DefaultOISnapshotPath
}

var (
	oiStoreOnce   sync.Once
	oiStoreShared *OISnapshotStore
)

// defaultOIStore returns the process-wide store. It is shared so every engine
// instance contributes observations to the same timeline.
func defaultOIStore() *OISnapshotStore {
	oiStoreOnce.Do(func() {
		oiStoreShared = NewOISnapshotStore(ResolveOISnapshotPath())
	})
	return oiStoreShared
}

// NewOISnapshotStore builds a store, loading any existing history. A path of ""
// disables persistence and keeps everything in memory.
func NewOISnapshotStore(path string) *OISnapshotStore {
	store := &OISnapshotStore{path: path}
	store.load()
	return store
}

// Path reports the backing file, for logs and diagnostics.
func (s *OISnapshotStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Depth reports how many observations are held. Exposed so callers can explain
// why a ranking is unavailable ("2 snapshots so far") instead of guessing.
func (s *OISnapshotStore) Depth() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.snapshots)
}

// Record stores one observation. Writes are throttled to oiSnapshotMinInterval
// and skipped entirely when the universe was unavailable, so an outage cannot
// write a sparse snapshot that would later masquerade as OI collapsing to zero.
func (s *OISnapshotStore) Record(now time.Time, oi map[string]float64) {
	if s == nil || len(oi) == 0 {
		return
	}

	copied := make(map[string]float64, len(oi))
	for symbol, value := range oi {
		if symbol = strings.TrimSpace(symbol); symbol == "" || value <= 0 {
			continue
		}
		copied[symbol] = value
	}
	if len(copied) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if n := len(s.snapshots); n > 0 {
		if now.Sub(s.snapshots[n-1].TakenAt) < oiSnapshotMinInterval {
			return
		}
	}
	s.snapshots = append(s.snapshots, OISnapshot{TakenAt: now, OI: copied})
	s.trimLocked()
	s.saveLocked()
}

// Latest returns the most recent observation.
func (s *OISnapshotStore) Latest() (OISnapshot, bool) {
	if s == nil {
		return OISnapshot{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.snapshots) == 0 {
		return OISnapshot{}, false
	}
	return s.snapshots[len(s.snapshots)-1], true
}

// Baseline picks the observation to measure the current one against, preferring
// the newest snapshot at least `window` old. When no history reaches that far —
// a fresh install, typically — it falls back to the oldest snapshot available,
// provided it is at least oiMinBaselineAge old, and reports the window it is
// actually using so the caller can label the number honestly.
func (s *OISnapshotStore) Baseline(now time.Time, window time.Duration) (OISnapshot, time.Duration, bool) {
	if s == nil {
		return OISnapshot{}, 0, false
	}
	if window <= 0 {
		window = oiTargetWindow
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.snapshots) == 0 {
		return OISnapshot{}, 0, false
	}

	target := now.Add(-window)
	for i := len(s.snapshots) - 1; i >= 0; i-- {
		if !s.snapshots[i].TakenAt.After(target) {
			return s.snapshots[i], now.Sub(s.snapshots[i].TakenAt), true
		}
	}

	oldest := s.snapshots[0]
	if age := now.Sub(oldest.TakenAt); age >= oiMinBaselineAge {
		return oldest, age, true
	}
	return OISnapshot{}, 0, false
}

// trimLocked bounds the history. Callers must hold the mutex.
func (s *OISnapshotStore) trimLocked() {
	if len(s.snapshots) > oiMaxSnapshots {
		s.snapshots = s.snapshots[len(s.snapshots)-oiMaxSnapshots:]
	}
}

// load reads the backing file; any failure leaves the store empty.
func (s *OISnapshotStore) load() {
	if s.path == "" {
		return
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}

	var file oiSnapshotFile
	if err := json.Unmarshal(raw, &file); err != nil {
		s.warn("open-interest history at %s is unreadable (%v); starting fresh", s.path, err)
		return
	}
	for _, snapshot := range file.Snapshots {
		if len(snapshot.OI) > 0 {
			s.snapshots = append(s.snapshots, snapshot)
		}
	}
	s.trimLocked()
}

// saveLocked writes the backing file atomically. Callers must hold the mutex.
func (s *OISnapshotStore) saveLocked() {
	if s.path == "" {
		return
	}

	payload, err := json.Marshal(oiSnapshotFile{Snapshots: s.snapshots})
	if err != nil {
		s.warn("open-interest history could not be encoded: %v", err)
		return
	}

	dir := filepath.Dir(s.path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			s.warn("open-interest history directory %s is not writable: %v", dir, err)
			return
		}
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		s.warn("open-interest history could not be written: %v", err)
		return
	}
	if err := os.Rename(tmp, s.path); err != nil {
		s.warn("open-interest history could not be committed: %v", err)
	}
}

// warn emits a persistence warning once, so a read-only volume does not spam the
// log every cycle.
func (s *OISnapshotStore) warn(format string, args ...any) {
	s.warnOnce.Do(func() {
		logger.Warnf(format, args...)
	})
}

// OIRow is one instrument's measured open-interest change, in the same units the
// retired nofxos ranking reported so the prompt the model reads is unchanged.
type OIRow struct {
	Symbol            string  `json:"symbol"`
	Rank              int     `json:"rank"`
	OIDeltaPercent    float64 `json:"oi_delta_percent"`    // already x100 (5.0 = 5%)
	OIDeltaValue      float64 `json:"oi_delta_value"`      // absolute USD change
	PriceDeltaPercent float64 `json:"price_delta_percent"` // already x100
	WindowMinutes     int     `json:"window_minutes"`      // the measurement actually used
	CurrentOI         float64 `json:"current_oi"`
}

// OIChangeRanking records the current universe and ranks it by open-interest
// change against the stored history.
//
// `ascending` selects the direction: false reproduces "oi_top" (largest
// increase), true reproduces "oi_low" (largest decrease).
//
// It returns an error rather than a partial answer when no usable baseline
// exists yet: reporting a 4-minute delta as a 1-hour one would be worse than
// admitting there is no ranking this cycle.
func OIChangeRanking(ctx context.Context, limit int, ascending bool) ([]OIRow, error) {
	store := defaultOIStore()

	universe, err := fetchLocalUniverse(ctx)
	if err != nil {
		return nil, err
	}
	if len(universe) == 0 {
		return nil, fmt.Errorf("open-interest universe is empty")
	}

	now := time.Now()
	current := make(map[string]float64, len(universe))
	for symbol, coin := range universe {
		if coin.OpenInterestUsd >= oiMinOpenInterestUSD {
			current[symbol] = coin.OpenInterestUsd
		}
	}
	store.Record(now, current)

	latestSnap, ok := store.Latest()
	if !ok {
		return nil, fmt.Errorf("open-interest snapshot unavailable")
	}
	baseline, age, ok := store.Baseline(now, oiTargetWindow)
	if !ok {
		return nil, fmt.Errorf("open-interest history still warming up (%d observations, need one at least %v old)",
			store.Depth(), oiMinBaselineAge)
	}

	// The snapshot written moments ago is the same data as `latest`; prefer it
	// when it survived throttling, otherwise fall back to the live read so the
	// ranking still reflects the market that was just priced.
	nowOI := latestSnap.OI
	if len(nowOI) == 0 {
		nowOI = current
	}

	windowMinutes := int(age.Round(time.Minute) / time.Minute)
	rows := oiRowsFor(leftJoinSymbols(nowOI, baseline.OI), nowOI, baseline.OI, universe, windowMinutes)
	if len(rows) == 0 {
		return nil, fmt.Errorf("no instrument had a comparable open-interest baseline")
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if ascending {
			return rows[i].OIDeltaPercent < rows[j].OIDeltaPercent
		}
		return rows[i].OIDeltaPercent > rows[j].OIDeltaPercent
	})

	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	for i := range rows {
		rows[i].Rank = i + 1
	}
	return rows, nil
}

// oiRowsFor builds one row per comparable instrument. Extracted from
// OIChangeRanking so the arithmetic can be exercised without a live exchange.
func oiRowsFor(symbols []string, current, baseline map[string]float64, universe map[string]hyperliquid.CoinInfo, windowMinutes int) []OIRow {
	rows := make([]OIRow, 0, len(symbols))
	for _, symbol := range symbols {
		now := current[symbol]
		was := baseline[symbol]
		rows = append(rows, OIRow{
			Symbol:            symbol,
			OIDeltaPercent:    (now - was) / was * 100,
			OIDeltaValue:      now - was,
			PriceDeltaPercent: universe[symbol].Change24hPct,
			WindowMinutes:     windowMinutes,
			CurrentOI:         now,
		})
	}
	return rows
}

// leftJoinSymbols returns the symbols present in both the current reading and
// the baseline. An instrument missing from either side has nothing to compare,
// so it is left out rather than reported as a fake "+100%".
func leftJoinSymbols(current, baseline map[string]float64) []string {
	symbols := make([]string, 0, len(current))
	for symbol := range current {
		if was, ok := baseline[symbol]; ok && was > 0 {
			symbols = append(symbols, symbol)
		}
	}
	sort.Strings(symbols)
	return symbols
}

// ScreenerRow is one instrument in the local attention ranking.
type ScreenerRow struct {
	Symbol  string             `json:"symbol"`
	Score   float64            `json:"score"`
	Rank    int                `json:"rank"`
	Factors map[string]float64 `json:"factors,omitempty"`
}

// Screener ranks the universe by how much it currently deserves attention.
//
// This is deliberately not presented as the vendor's AI score. That score was
// opaque and unreproducible; this one is four cross-sectional z-scores with
// fixed weights, every one of them recomputable from the same public snapshot
// the platform already reads. "Attention" is the honest word for it: it ranks
// instruments worth looking at, not directions to trade.
func Screener(ctx context.Context, limit int) ([]ScreenerRow, error) {
	universe, err := fetchLocalUniverse(ctx)
	if err != nil {
		return nil, err
	}
	if len(universe) == 0 {
		return nil, fmt.Errorf("screener universe is empty")
	}
	return screenerRows(universe, limit), nil
}

// screenerRows ranks a fetched universe. Separated from the fetch so the scoring
// can be unit-tested against a fabricated board.
func screenerRows(universe map[string]hyperliquid.CoinInfo, limit int) []ScreenerRow {
	symbols := make([]string, 0, len(universe))
	for symbol := range universe {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)

	// log-volume keeps a 200bn outlier from flattening the rest of the field.
	extract := func(fn func(hyperliquid.CoinInfo) float64) []float64 {
		out := make([]float64, 0, len(symbols))
		for _, symbol := range symbols {
			out = append(out, fn(universe[symbol]))
		}
		return zScores(out)
	}

	trend := extract(func(c hyperliquid.CoinInfo) float64 { return math.Abs(c.Change24hPct) })
	liquidity := extract(func(c hyperliquid.CoinInfo) float64 { return log10OrZero(c.Volume24h) })
	crowding := extract(func(c hyperliquid.CoinInfo) float64 {
		if c.Volume24h <= 0 {
			return 0
		}
		return c.OpenInterestUsd / c.Volume24h
	})
	squeeze := extract(func(c hyperliquid.CoinInfo) float64 { return math.Abs(c.FundingRate) })

	rows := make([]ScreenerRow, 0, len(symbols))
	for i, symbol := range symbols {
		score := screenerWeightTrend*trend[i] +
			screenerWeightLiquidity*liquidity[i] +
			screenerWeightCrowding*crowding[i] +
			screenerWeightSqueeze*squeeze[i]
		rows = append(rows, ScreenerRow{
			Symbol: symbol,
			Score:  score,
			Factors: map[string]float64{
				"trend":     round3(trend[i]),
				"liquidity": round3(liquidity[i]),
				"crowding":  round3(crowding[i]),
				"squeeze":   round3(squeeze[i]),
			},
		})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Score == rows[j].Score {
			return rows[i].Symbol < rows[j].Symbol
		}
		return rows[i].Score > rows[j].Score
	})

	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	for i := range rows {
		rows[i].Rank = i + 1
	}
	return rows
}

// Screener weights. They sum to 1 so the score stays on the same scale as its
// inputs and a change to one weight cannot silently rescale the others.
const (
	screenerWeightTrend     = 0.35
	screenerWeightLiquidity = 0.30
	screenerWeightCrowding  = 0.20
	screenerWeightSqueeze   = 0.15
)

// fetchLocalUniverse returns canonical-symbol -> instrument for every tradable
// Hyperliquid perpetual, using the same cached asset-contexts read the market
// insight providers already rely on.
func fetchLocalUniverse(ctx context.Context) (map[string]hyperliquid.CoinInfo, error) {
	coins, err := hyperliquid.GetAssetContexts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read Hyperliquid asset contexts: %w", err)
	}
	universe := make(map[string]hyperliquid.CoinInfo, len(coins))
	for _, coin := range coins {
		if strings.TrimSpace(coin.Symbol) == "" {
			continue
		}
		universe[canonicalLocalSymbol(coin.Symbol)] = coin
	}
	return universe, nil
}

// canonicalLocalSymbol maps a Hyperliquid asset name to the symbol form used
// across the candidate pool and the prompt ("BTC" -> "BTCUSDT", "xyz:TSLA"
// unchanged in its API form).
func canonicalLocalSymbol(symbol string) string {
	symbol = strings.TrimSpace(symbol)
	if strings.HasPrefix(strings.ToLower(symbol), "xyz:") {
		base := strings.TrimSpace(symbol[len("xyz:"):])
		return hyperliquid.FormatCoinForAPI("xyz:" + base)
	}
	return market.Normalize(symbol + "USDT")
}

// zScores standardises values across the universe, returning 0 for a degenerate
// (zero-variance) field so every instrument ties instead of dividing by zero.
func zScores(values []float64) []float64 {
	out := make([]float64, len(values))
	if len(values) == 0 {
		return out
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, v := range values {
		d := v - mean
		variance += d * d
	}
	std := math.Sqrt(variance / float64(len(values)))
	if std == 0 || math.IsNaN(std) {
		return out
	}
	for i, v := range values {
		out[i] = (v - mean) / std
	}
	return out
}

func log10OrZero(value float64) float64 {
	if value <= 0 {
		return 0
	}
	return math.Log10(value)
}

func round3(value float64) float64 {
	return math.Round(value*1000) / 1000
}
