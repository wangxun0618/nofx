package providers

import (
	"context"
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/marketdata"
	"nofx/provider/hyperliquid"
	"nofx/store"
)

// --- Pure scoring helpers -------------------------------------------------

func TestVoteBySigmaAbstainsNearZero(t *testing.T) {
	cases := []struct {
		z    float64
		want string
	}{
		{2.5, marketdata.BiasBullish},
		{directionVoteSigma, marketdata.BiasBullish},
		{0.2, marketdata.BiasNeutral},
		{0, marketdata.BiasNeutral},
		{-0.2, marketdata.BiasNeutral},
		{-directionVoteSigma, marketdata.BiasBearish},
		{-2.5, marketdata.BiasBearish},
	}
	for _, tc := range cases {
		if got := voteBySigma(tc.z); got != tc.want {
			t.Fatalf("voteBySigma(%.2f) = %s, want %s", tc.z, got, tc.want)
		}
	}
}

func TestStandardizeAbstainsOnFlatUniverse(t *testing.T) {
	// A cross-section with no dispersion carries no relative information, so the
	// correct answer is an abstention rather than a division by zero.
	if got := standardize(5, 5, 0); got != 0 {
		t.Fatalf("standardize with zero dispersion = %v, want 0", got)
	}
	if got := standardize(7, 5, 2); got != 1 {
		t.Fatalf("standardize = %v, want 1", got)
	}
}

func TestMeanStd(t *testing.T) {
	coins := []hyperliquid.CoinInfo{{Change24hPct: 0}, {Change24hPct: 2}, {Change24hPct: 4}}
	mean, std := meanStd(coins, func(c hyperliquid.CoinInfo) float64 { return c.Change24hPct })

	if math.Abs(mean-2) > 1e-9 {
		t.Fatalf("mean = %v, want 2", mean)
	}
	// Population standard deviation of {0,2,4} is sqrt(8/3) ≈ 1.633.
	if math.Abs(std-1.632993) > 1e-5 {
		t.Fatalf("std = %v, want ~1.633", std)
	}

	if mean, std := meanStd(nil, func(hyperliquid.CoinInfo) float64 { return 0 }); mean != 0 || std != 0 {
		t.Fatalf("empty board should yield zeroes, got %v/%v", mean, std)
	}
}

func TestClamp(t *testing.T) {
	if got := clamp(7, -3, 3); got != 3 {
		t.Fatalf("clamp high = %v", got)
	}
	if got := clamp(-7, -3, 3); got != -3 {
		t.Fatalf("clamp low = %v", got)
	}
	if got := clamp(1.5, -3, 3); got != 1.5 {
		t.Fatalf("clamp pass-through = %v", got)
	}
}

func TestHumanizeAge(t *testing.T) {
	cases := []struct {
		age  time.Duration
		want string
	}{
		{10 * time.Second, "just now"},
		{3 * time.Minute, "3m"},
		{5 * time.Hour, "5h"},
		{50 * time.Hour, "2d"},
	}
	for _, tc := range cases {
		if got := humanizeAge(tc.age); got != tc.want {
			t.Fatalf("humanizeAge(%s) = %q, want %q", tc.age, got, tc.want)
		}
	}
}

// --- Verdict construction -------------------------------------------------

func TestDirectionRowIsAuditable(t *testing.T) {
	// Momentum far above its peers and a positive premium should both vote
	// bullish, and the row has to carry the evidence for each.
	coin := hyperliquid.CoinInfo{Symbol: "BTC", Change24hPct: 6.0, Premium: 0.0008}

	row := directionRow(coin, 2.0, 1.0, 0.0, 0.0002, map[string]float64{"BTC": 500_000})

	if row.Bias != marketdata.BiasBullish {
		t.Fatalf("bias = %s, want bullish", row.Bias)
	}
	if len(row.Components) != 3 {
		t.Fatalf("components = %+v, want momentum, premium and flow", row.Components)
	}

	byName := map[string]marketdata.DirectionComponent{}
	for _, component := range row.Components {
		byName[component.Name] = component
	}
	for name, wantVote := range map[string]string{
		"momentum": marketdata.BiasBullish,
		"premium":  marketdata.BiasBullish,
		"flow":     marketdata.BiasBullish,
	} {
		if got := byName[name].Vote; got != wantVote {
			t.Fatalf("%s voted %s, want %s", name, got, wantVote)
		}
		if strings.TrimSpace(byName[name].Detail) == "" {
			t.Fatalf("%s has no evidence detail", name)
		}
	}
	// Momentum z = (6-2)/1 = 4, clipped into the average with the others.
	if row.Score <= 0 || row.Score > directionScoreClip {
		t.Fatalf("score = %v, want a positive value within the clip", row.Score)
	}
	if row.Bullish != 3 || row.Bearish != 0 {
		t.Fatalf("vote counts = %d/%d/%d", row.Bullish, row.Bearish, row.Neutral)
	}
}

func TestDirectionRowOmitsFlowWhenUnavailable(t *testing.T) {
	// Order flow is optional by design, so its absence must reduce the component
	// count rather than silently voting neutral.
	coin := hyperliquid.CoinInfo{Symbol: "ETH", Change24hPct: 3.0, Premium: 0.0005}

	row := directionRow(coin, 2.0, 1.0, 0.0, 0.0002, nil)

	if len(row.Components) != 2 {
		t.Fatalf("components = %+v, want momentum and premium only", row.Components)
	}
	for _, component := range row.Components {
		if component.Name == "flow" {
			t.Fatal("flow must not appear when no order-flow data was fetched")
		}
	}
	if row.Bias != marketdata.BiasBullish {
		t.Fatalf("bias = %s, want bullish", row.Bias)
	}
}

func TestDirectionRowFlowFloorSuppressesNoise(t *testing.T) {
	// A trivial tape imbalance should not cast a vote; otherwise every instrument
	// would score a direction on noise.
	coin := hyperliquid.CoinInfo{Symbol: "BTC", Change24hPct: 0, Premium: 0}
	row := directionRow(coin, 0, 0, 0, 0, map[string]float64{"BTC": directionFlowFloor - 1})

	var flow marketdata.DirectionComponent
	for _, component := range row.Components {
		if component.Name == "flow" {
			flow = component
		}
	}
	if flow.Vote != marketdata.BiasNeutral {
		t.Fatalf("sub-threshold flow voted %s, want neutral", flow.Vote)
	}
}

func TestDirectionRowNeutralOnZeroDispersion(t *testing.T) {
	// With a flat universe and a sub-threshold premium, every component abstains
	// and the verdict must be neutral rather than an arbitrary side.
	coin := hyperliquid.CoinInfo{Symbol: "BTC", Change24hPct: 1.0, Premium: 0}
	row := directionRow(coin, 1.0, 0, 0, 0, nil)

	if row.Bias != marketdata.BiasNeutral {
		t.Fatalf("bias = %s, want neutral", row.Bias)
	}
	if row.Score != 0 {
		t.Fatalf("score = %v, want 0", row.Score)
	}
}

func TestDirectionPayloadListsAvailableComponents(t *testing.T) {
	if got := directionComponentNames(nil); strings.Join(got, ",") != "momentum,premium" {
		t.Fatalf("without flow: %v", got)
	}
	if got := directionComponentNames(map[string]float64{"BTC": 1}); strings.Join(got, ",") != "momentum,premium,flow" {
		t.Fatalf("with flow: %v", got)
	}
}

// --- Provider wiring ------------------------------------------------------

func TestDirectionalSignalProviderIsFreeAndOnByDefault(t *testing.T) {
	provider := NewDirectionalSignalProvider()

	if !provider.Enabled(store.IndicatorConfig{}) {
		t.Fatal("the direction source runs on free public data and should be on by default")
	}
	if provider.RequiresAPIKey() {
		t.Fatal("the direction source needs no credential")
	}
	if provider.Name() != "directional_signal" {
		t.Fatalf("name = %q", provider.Name())
	}
	if _, ok := any(provider).(marketdata.ServiceBacked); ok {
		t.Fatal("the direction source has a free fallback and must not require the sidecar")
	}
}

func TestDirectionalSignalFlowComponentRequiresOptIn(t *testing.T) {
	provider := NewDirectionalSignalProvider()

	if got := provider.flowClientFor(store.IndicatorConfig{}); got != nil {
		t.Fatal("order flow must not be fetched when the sidecar is disabled")
	}
	if got := provider.flowClientFor(store.IndicatorConfig{EnableHyperData: true}); got == nil {
		t.Fatal("order flow should be fetched once the sidecar is enabled")
	}

	// The client is cached and rebuilt when the endpoint changes, so a strategy
	// edit does not require a restart.
	first := provider.flowClientFor(store.IndicatorConfig{EnableHyperData: true, HyperDataBaseURL: "http://a:1"})
	second := provider.flowClientFor(store.IndicatorConfig{EnableHyperData: true, HyperDataBaseURL: "http://b:2"})
	if first == second {
		t.Fatal("a changed base URL should rebuild the client")
	}
	if second.BaseURL() != "http://b:2" {
		t.Fatalf("base URL = %q", second.BaseURL())
	}
}

func TestDirectionalSignalRecordsTimeline(t *testing.T) {
	// The timeline is the one capability of the retired board with no substitute
	// in free data, so it is worth asserting end to end.
	tracker := marketdata.NewDirectionTracker(filepath.Join(t.TempDir(), "history.json"))
	provider := &DirectionalSignalProvider{tracker: tracker}

	provider.record([]DirectionRow{{
		Symbol:  "BTC",
		Bias:    marketdata.BiasBullish,
		Score:   1.4,
		Bullish: 2,
		Components: []marketdata.DirectionComponent{
			{Name: "momentum", Vote: marketdata.BiasBullish, Detail: "+1.9σ"},
			{Name: "premium", Vote: marketdata.BiasBullish, Detail: "+0.080%"},
		},
	}})

	// First observation is not a change.
	if got := len(provider.record([]DirectionRow{{
		Symbol:  "BTC",
		Bias:    marketdata.BiasBullish,
		Score:   1.4,
		Bullish: 2,
		Components: []marketdata.DirectionComponent{
			{Name: "momentum", Vote: marketdata.BiasBullish, Detail: "+1.9σ"},
			{Name: "premium", Vote: marketdata.BiasBullish, Detail: "+0.080%"},
		},
	}})); got != 0 {
		t.Fatalf("an unchanged bias produced %d changes, want none", got)
	}

	changes := provider.record([]DirectionRow{{
		Symbol:  "BTC",
		Bias:    marketdata.BiasBearish,
		Score:   -1.2,
		Bearish: 2,
		Components: []marketdata.DirectionComponent{
			{Name: "momentum", Vote: marketdata.BiasBearish, Detail: "-1.4σ"},
			{Name: "premium", Vote: marketdata.BiasBullish, Detail: "+0.010%"},
		},
	}})

	if len(changes) != 1 {
		t.Fatalf("flip produced %d changes, want 1", len(changes))
	}
	if changes[0].FromBias != marketdata.BiasBullish || changes[0].ToBias != marketdata.BiasBearish {
		t.Fatalf("change = %+v", changes[0])
	}
	if changes[0].Reason != "momentum bullish→bearish" {
		t.Fatalf("reason = %q", changes[0].Reason)
	}
}

func TestDirectionalSignalFlowFetchFailsSoftWithoutSidecar(t *testing.T) {
	// A wedged sidecar must cost the flow component, not the whole block.
	provider := NewDirectionalSignalProvider()

	got := provider.fetchFlow(context.Background(),
		store.IndicatorConfig{EnableHyperData: true, HyperDataBaseURL: "http://127.0.0.1:1"},
		marketdata.Request{Symbols: []string{"BTC"}})

	if got != nil {
		t.Fatalf("unreachable sidecar should yield no flow, got %v", got)
	}
}

func TestRankDirectionRowsFiltersAndOrdersByStrength(t *testing.T) {
	rows := []DirectionRow{
		{Symbol: "A", Bias: marketdata.BiasBullish, Score: 0.4},
		{Symbol: "B", Bias: marketdata.BiasBearish, Score: -2.0},
		{Symbol: "C", Bias: marketdata.BiasBullish, Score: 2.5},
		{Symbol: "D", Bias: marketdata.BiasNeutral, Score: 0},
	}

	bullish := rankDirectionRows(rows, 10, func(r DirectionRow) bool { return r.Bias == marketdata.BiasBullish })
	if len(bullish) != 2 || bullish[0].Symbol != "C" || bullish[1].Symbol != "A" {
		t.Fatalf("bullish ranking = %+v, want C then A", bullish)
	}

	if got := rankDirectionRows(rows, 1, func(r DirectionRow) bool { return r.Bias == marketdata.BiasBearish }); len(got) != 1 {
		t.Fatalf("limit not applied: %+v", got)
	}

	if got := rankDirectionRows(rows, 10, func(r DirectionRow) bool { return r.Bias == marketdata.BiasBearish }); got[0].Symbol != "B" {
		t.Fatalf("bearish ranking = %+v", got)
	}
}

func TestDescribeComponentsRendersEvidence(t *testing.T) {
	got := describeComponents([]marketdata.DirectionComponent{
		{Name: "momentum", Vote: marketdata.BiasBullish, Detail: "+1.90σ"},
		{Name: "premium", Vote: marketdata.BiasBearish, Detail: "-0.050%"},
	})
	want := "momentum +1.90σ (bullish); premium -0.050% (bearish)"
	if got != want {
		t.Fatalf("describeComponents = %q, want %q", got, want)
	}

	if got := describeComponents(nil); got != "-" {
		t.Fatalf("empty components = %q", got)
	}
}
