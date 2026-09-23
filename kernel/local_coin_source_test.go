package kernel

import (
	"math"
	"path/filepath"
	"testing"
	"time"

	"nofx/provider/hyperliquid"
)

// The local replacements are the only thing standing between the candidate pool
// and a dead upstream, so the tests pin the arithmetic rather than the plumbing:
// the failure the old endpoints caused was silent, and a wrong percentage would
// look exactly like a real ranking.

func TestCanonicalLocalSymbolKeepsProductForms(t *testing.T) {
	if got := canonicalLocalSymbol("BTC"); got != "BTCUSDT" {
		t.Fatalf("plain asset should gain the USDT suffix, got %q", got)
	}
	if got := canonicalLocalSymbol(" xyz:TSLA "); got != "xyz:TSLA" {
		t.Fatalf("xyz assets must keep their qualified form, got %q", got)
	}
}

func TestZScoresReturnZeroForDegenerateFields(t *testing.T) {
	flat := zScores([]float64{4, 4, 4})
	for i, v := range flat {
		if v != 0 {
			t.Fatalf("zero variance must not divide by zero; index %d got %v", i, v)
		}
	}

	scored := zScores([]float64{1, 2, 3})
	if math.Abs(scored[0]+1.2247448) > 1e-6 || math.Abs(scored[2]-1.2247448) > 1e-6 {
		t.Fatalf("expected standardised tails near ±1.2247, got %v", scored)
	}
	if math.Abs(scored[1]) > 1e-9 {
		t.Fatalf("the mean observation should score 0, got %v", scored[1])
	}
}

func TestOIJoinSkipsInstrumentsWithoutBaseline(t *testing.T) {
	current := map[string]float64{"AAAUSDT": 10, "BBBUSDT": 20, "CCCUSDT": 30}
	baseline := map[string]float64{"AAAUSDT": 5, "CCCUSDT": 0}

	got := leftJoinSymbols(current, baseline)
	if len(got) != 1 || got[0] != "AAAUSDT" {
		t.Fatalf("only instruments with a positive baseline may be ranked, got %v", got)
	}
}

func TestOIArithmeticMatchesPercentAndAbsoluteChange(t *testing.T) {
	universe := map[string]hyperliquid.CoinInfo{
		"UPUSDT":   {Change24hPct: 3.5},
		"DOWNUSDT": {Change24hPct: -2.0},
	}
	current := map[string]float64{"UPUSDT": 150, "DOWNUSDT": 50}
	baseline := map[string]float64{"UPUSDT": 100, "DOWNUSDT": 100}
	symbols := leftJoinSymbols(current, baseline)

	rows := oiRowsFor(symbols, current, baseline, universe, 60)

	if len(rows) != 2 {
		t.Fatalf("expected one row per comparable instrument, got %d", len(rows))
	}
	bySymbol := map[string]OIRow{}
	for _, row := range rows {
		bySymbol[row.Symbol] = row
	}

	up := bySymbol["UPUSDT"]
	if math.Abs(up.OIDeltaPercent-50) > 1e-9 {
		t.Fatalf("expected +50%% growth, got %+v", up)
	}
	if math.Abs(up.OIDeltaValue-50) > 1e-9 || math.Abs(up.PriceDeltaPercent-3.5) > 1e-9 {
		t.Fatalf("expected the 24h price change to travel with the row, got %+v", up)
	}
	if up.WindowMinutes != 60 {
		t.Fatalf("the row must carry the window it was measured over, got %d", up.WindowMinutes)
	}
	down := bySymbol["DOWNUSDT"]
	if math.Abs(down.OIDeltaPercent+50) > 1e-9 {
		t.Fatalf("expected -50%% decline, got %+v", down)
	}
	if math.Abs(down.OIDeltaValue+50) > 1e-9 {
		t.Fatalf("expected the absolute USD change to travel with the row, got %+v", down)
	}
}

func TestOISnapshotStoreThrottlesAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oi.json")
	now := time.Now()

	store := NewOISnapshotStore(path)
	store.Record(now, map[string]float64{"AAAUSDT": 100})
	store.Record(now.Add(time.Minute), map[string]float64{"AAAUSDT": 110})

	if depth := store.Depth(); depth != 1 {
		t.Fatalf("recording every cycle would grow the file without bound; depth=%d", depth)
	}
	if latest, ok := store.Latest(); !ok || latest.OI["AAAUSDT"] != 100 {
		t.Fatalf("the throttled write must leave the first observation intact, got %+v", latest)
	}

	restarted := NewOISnapshotStore(path)
	if restarted.Depth() != 1 {
		t.Fatalf("history must survive a restart, otherwise the ranking never warms up")
	}
}

func TestOISnapshotRecordIgnoresEmptyUniverse(t *testing.T) {
	store := NewOISnapshotStore(filepath.Join(t.TempDir(), "oi.json"))
	store.Record(time.Now(), nil)
	store.Record(time.Now(), map[string]float64{"AAAUSDT": 0, "": 5})
	if store.Depth() != 0 {
		t.Fatalf("an outage must not write a sparse snapshot that later reads as a collapse")
	}
}

func TestOISnapshotBaselineSelection(t *testing.T) {
	store := NewOISnapshotStore("")
	now := time.Now()
	base := now.Add(-90 * time.Minute)
	// Inject directly: Record is throttled on purpose, which the previous test covers.
	store.snapshots = append(store.snapshots, OISnapshot{TakenAt: base, OI: map[string]float64{"AAAUSDT": 100}})
	store.snapshots = append(store.snapshots, OISnapshot{TakenAt: now.Add(-5 * time.Minute), OI: map[string]float64{"AAAUSDT": 200}})

	snapshot, age, ok := store.Baseline(now, time.Hour)
	if !ok {
		t.Fatalf("expected the 90-minute-old snapshot to serve a 1h window")
	}
	if !snapshot.TakenAt.Equal(base) {
		t.Fatalf("expected the newest snapshot outside the window, got %v", snapshot.TakenAt)
	}
	if age < 89*time.Minute {
		t.Fatalf("expected the real window so the caller can label it, got %v", age)
	}

	// Cold start: the oldest snapshot is too young to be honest about.
	warm := NewOISnapshotStore("")
	warm.snapshots = append(warm.snapshots, OISnapshot{TakenAt: now.Add(-4 * time.Minute), OI: map[string]float64{"AAAUSDT": 100}})
	if _, _, ok := warm.Baseline(now, time.Hour); ok {
		t.Fatalf("a four-minute-old baseline must not be reported as an hour")
	}
}

func TestOISnapshotBaselineFallsBackToOldestHistory(t *testing.T) {
	store := NewOISnapshotStore("")
	now := time.Now()
	oldest := now.Add(-20 * time.Minute)
	store.snapshots = append(store.snapshots, OISnapshot{TakenAt: oldest, OI: map[string]float64{"AAAUSDT": 100}})
	store.snapshots = append(store.snapshots, OISnapshot{TakenAt: now.Add(-2*time.Minute), OI: map[string]float64{"AAAUSDT": 120}})

	snapshot, age, ok := store.Baseline(now, time.Hour)
	if !ok || !snapshot.TakenAt.Equal(oldest) {
		t.Fatalf("during warm-up the oldest usable observation should serve, got ok=%v", ok)
	}
	if age < 20*time.Minute {
		t.Fatalf("the fallback must report the shorter window it actually used, got %v", age)
	}
}

func TestScreenerRanksAndSurvivesIdenticalInputs(t *testing.T) {
	universe := map[string]hyperliquid.CoinInfo{
		"BIGUSDT":   {Change24hPct: 12, Volume24h: 5e9, OpenInterestUsd: 2e9, FundingRate: 0.0004},
		"MIDUSDT":   {Change24hPct: 4, Volume24h: 5e8, OpenInterestUsd: 3e8, FundingRate: 0.0001},
		"SMALLUSDT": {Change24hPct: -1, Volume24h: 1e6, OpenInterestUsd: 5e5, FundingRate: 0},
	}

	rows := screenerRows(universe, 0)
	if len(rows) != 3 {
		t.Fatalf("expected every instrument scored, got %d", len(rows))
	}
	if rows[0].Rank != 1 || rows[2].Rank != 3 {
		t.Fatalf("ranks must follow the sorted order")
	}
	if rows[0].Symbol != "BIGUSDT" {
		t.Fatalf("the busiest and most moved instrument should lead, got %s", rows[0].Symbol)
	}
	if rows[0].Score <= rows[1].Score || rows[1].Score <= rows[2].Score {
		t.Fatalf("scores must be strictly ordered: %v %v %v", rows[0].Score, rows[1].Score, rows[2].Score)
	}
	if len(rows[0].Factors) != 4 {
		t.Fatalf("the score must be auditable component by component, got %v", rows[0].Factors)
	}

	limited := screenerRows(universe, 2)
	if len(limited) != 2 {
		t.Fatalf("limit should truncate, got %d rows", len(limited))
	}

	flat := map[string]hyperliquid.CoinInfo{
		"AAAUSDT": {Change24hPct: 1, Volume24h: 1e6, OpenInterestUsd: 1e6, FundingRate: 0.0001},
		"BBBUSDT": {Change24hPct: 1, Volume24h: 1e6, OpenInterestUsd: 1e6, FundingRate: 0.0001},
	}
	for _, row := range screenerRows(flat, 0) {
		if math.IsNaN(row.Score) {
			t.Fatalf("an identical board must not produce NaN; got %+v", row)
		}
		if row.Score != 0 {
			t.Fatalf("an identical board should tie at zero, got %+v", row)
		}
	}
}
