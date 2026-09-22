package marketdata

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func componentVotes(components ...DirectionComponent) []DirectionComponent {
	return components
}

func TestScoreBiasResolvesTiesToNeutral(t *testing.T) {
	cases := []struct {
		name     string
		votes    []string
		want     string
		bulls    int
		bears    int
		neutrals int
	}{
		{"clear bullish", []string{BiasBullish, BiasBullish, BiasBearish}, BiasBullish, 2, 1, 0},
		{"clear bearish", []string{BiasBearish, BiasNeutral, BiasBearish}, BiasBearish, 0, 2, 1},
		{"split vote abstains", []string{BiasBullish, BiasBearish}, BiasNeutral, 1, 1, 0},
		{"all neutral", []string{BiasNeutral, BiasNeutral}, BiasNeutral, 0, 0, 2},
		{"no components", nil, BiasNeutral, 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bias, bulls, bears, neutrals := ScoreBias(tc.votes)
			if bias != tc.want || bulls != tc.bulls || bears != tc.bears || neutrals != tc.neutrals {
				t.Fatalf("got %s (%d/%d/%d), want %s (%d/%d/%d)",
					bias, bulls, bears, neutrals, tc.want, tc.bulls, tc.bears, tc.neutrals)
			}
		})
	}
}

func TestDescribeTransitionNamesTheComponentThatFlipped(t *testing.T) {
	previous := DirectionState{
		Bias:  BiasBullish,
		Score: 1.4,
		Components: componentVotes(
			DirectionComponent{Name: "momentum", Vote: BiasBullish, Detail: "+1.9σ"},
			DirectionComponent{Name: "premium", Vote: BiasBullish, Detail: "+0.050%"},
		),
	}
	next := DirectionState{
		Bias:  BiasBearish,
		Score: -1.1,
		Components: componentVotes(
			DirectionComponent{Name: "momentum", Vote: BiasBearish, Detail: "-1.4σ"},
			DirectionComponent{Name: "premium", Vote: BiasBullish, Detail: "+0.010%"},
		),
	}

	got := DescribeTransition(previous, next)
	if got != "momentum bullish→bearish" {
		t.Fatalf("reason = %q, want the flipped component named", got)
	}
}

func TestDescribeTransitionReportsAvailabilityChanges(t *testing.T) {
	previous := DirectionState{
		Bias:       BiasBullish,
		Score:      1.0,
		Components: componentVotes(DirectionComponent{Name: "momentum", Vote: BiasBullish}),
	}
	next := DirectionState{
		Bias:  BiasBearish,
		Score: -1.0,
		Components: componentVotes(
			DirectionComponent{Name: "momentum", Vote: BiasBearish},
			DirectionComponent{Name: "flow", Vote: BiasBullish},
		),
	}

	got := DescribeTransition(previous, next)
	want := "flow became available (bullish); momentum bullish→bearish"
	if got != want {
		t.Fatalf("reason = %q, want %q", got, want)
	}
}

func TestDescribeTransitionHandlesNoComponentFlip(t *testing.T) {
	previous := DirectionState{
		Bias:       BiasBullish,
		Score:      0.6,
		Components: componentVotes(DirectionComponent{Name: "momentum", Vote: BiasBullish}),
	}
	next := DirectionState{
		Bias:       BiasBearish,
		Score:      -0.6,
		Components: componentVotes(DirectionComponent{Name: "momentum", Vote: BiasBullish}),
	}

	got := DescribeTransition(previous, next)
	if got == "" || got == "momentum bullish→bearish" {
		t.Fatalf("an unexplained flip must say so rather than invent a cause: %q", got)
	}
}

func TestDescribeTransitionFirstObservation(t *testing.T) {
	got := DescribeTransition(DirectionState{}, DirectionState{Bias: BiasBullish})
	if got != "first observation with component detail" {
		t.Fatalf("reason = %q", got)
	}
}

func TestDirectionTrackerFirstObservationIsNotAChange(t *testing.T) {
	// Otherwise every instrument reports a flip on the first cycle after a fresh
	// install, which is exactly the noise the timeline exists to avoid.
	tracker := NewDirectionTracker("")
	now := time.Now()

	if got := tracker.Observe(now, []DirectionState{{Symbol: "BTC", Bias: BiasBullish}}); len(got) != 0 {
		t.Fatalf("first observation produced %d changes, want none", len(got))
	}
	if got := tracker.Observe(now.Add(time.Minute), []DirectionState{{Symbol: "BTC", Bias: BiasBullish}}); len(got) != 0 {
		t.Fatalf("an unchanged bias produced %d changes, want none", len(got))
	}

	changes := tracker.Observe(now.Add(2*time.Minute), []DirectionState{{Symbol: "BTC", Bias: BiasBearish}})
	if len(changes) != 1 {
		t.Fatalf("a flip produced %d changes, want 1", len(changes))
	}
	if changes[0].FromBias != BiasBullish || changes[0].ToBias != BiasBearish {
		t.Fatalf("change = %+v", changes[0])
	}
}

func TestDirectionTrackerPersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "direction_history.json")
	now := time.Now()

	first := NewDirectionTracker(path)
	first.Observe(now, []DirectionState{{
		Symbol: "BTC",
		Bias:   BiasBullish,
		Score:  1.25,
		Components: componentVotes(
			DirectionComponent{Name: "momentum", Vote: BiasBullish, Detail: "+1.8σ"},
		),
	}})
	first.Observe(now.Add(time.Minute), []DirectionState{{
		Symbol: "BTC",
		Bias:   BiasBearish,
		Score:  -1.1,
		Components: componentVotes(
			DirectionComponent{Name: "momentum", Vote: BiasBearish, Detail: "-1.5σ"},
		),
	}})

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("history should have been written: %v", err)
	}

	// A restart must not report the existing instrument as a fresh flip, and must
	// still hold the transition recorded before it went down.
	restarted := NewDirectionTracker(path)
	if got := restarted.Observe(now.Add(2*time.Minute), []DirectionState{{Symbol: "BTC", Bias: BiasBearish}}); len(got) != 0 {
		t.Fatalf("reloaded state produced %d changes, want none", len(got))
	}

	recent := restarted.Recent(10, 24*time.Hour)
	if len(recent) != 1 {
		t.Fatalf("reloaded %d changes, want the one recorded before restart", len(recent))
	}
	if recent[0].Reason != "momentum bullish→bearish" {
		t.Fatalf("reloaded reason = %q", recent[0].Reason)
	}
}

func TestDirectionTrackerSurvivesCorruptFile(t *testing.T) {
	// A truncated write is the likely cause, so the safe answer is to start over
	// rather than to refuse to run.
	path := filepath.Join(t.TempDir(), "direction_history.json")
	if err := os.WriteFile(path, []byte(`{"states": [{"symbol":`), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	tracker := NewDirectionTracker(path)
	if got := tracker.States(); len(got) != 0 {
		t.Fatalf("corrupt history should load empty, got %+v", got)
	}
	if got := tracker.Observe(time.Now(), []DirectionState{{Symbol: "BTC", Bias: BiasBullish}}); len(got) != 0 {
		t.Fatal("a tracker that recovered should still work")
	}
}

func TestDirectionTrackerWorksWithoutPersistence(t *testing.T) {
	// path == "" is the in-memory mode used by tests and by callers that have no
	// writable volume.
	tracker := NewDirectionTracker("")
	now := time.Now()

	tracker.Observe(now, []DirectionState{{Symbol: "ETH", Bias: BiasNeutral}})
	changes := tracker.Observe(now.Add(time.Minute), []DirectionState{{Symbol: "ETH", Bias: BiasBullish}})

	if len(changes) != 1 {
		t.Fatalf("in-memory tracking produced %d changes, want 1", len(changes))
	}
	if len(tracker.Recent(10, 0)) != 1 {
		t.Fatal("Recent should return the in-memory change")
	}
}

func TestDirectionTrackerRecentIsNewestFirstAndWindowed(t *testing.T) {
	tracker := NewDirectionTracker("")
	now := time.Now()

	tracker.Observe(now.Add(-3*time.Hour), []DirectionState{{Symbol: "BTC", Bias: BiasBullish}})
	tracker.Observe(now.Add(-3*time.Hour+time.Minute), []DirectionState{{Symbol: "BTC", Bias: BiasNeutral}})
	tracker.Observe(now.Add(-time.Minute), []DirectionState{{Symbol: "BTC", Bias: BiasBearish}})

	recent := tracker.Recent(10, time.Hour)
	if len(recent) != 1 {
		t.Fatalf("window should exclude the 3h-old change, got %d", len(recent))
	}
	if recent[0].ToBias != BiasBearish {
		t.Fatalf("newest change = %+v", recent[0])
	}

	all := tracker.Recent(10, 0)
	if len(all) != 2 {
		t.Fatalf("unwindowed Recent = %d, want 2", len(all))
	}
	if all[0].ToBias != BiasBearish || all[1].ToBias != BiasNeutral {
		t.Fatalf("ordering should be newest first: %+v", all)
	}
}

func TestDirectionTrackerBoundsTheTimeline(t *testing.T) {
	tracker := NewDirectionTracker("")
	now := time.Now()

	bias := BiasBullish
	for i := 0; i < directionHistoryMaxChanges+50; i++ {
		if bias == BiasBullish {
			bias = BiasBearish
		} else {
			bias = BiasBullish
		}
		tracker.Observe(now.Add(time.Duration(i)*time.Second), []DirectionState{{Symbol: "BTC", Bias: bias}})
	}

	if got := len(tracker.Recent(0, 0)); got > directionHistoryMaxChanges {
		t.Fatalf("timeline grew to %d, want at most %d", got, directionHistoryMaxChanges)
	}
}

func TestDirectionTrackerRecentLimit(t *testing.T) {
	tracker := NewDirectionTracker("")
	now := time.Now()

	tracker.Observe(now, []DirectionState{{Symbol: "BTC", Bias: BiasBullish}})
	tracker.Observe(now.Add(time.Second), []DirectionState{{Symbol: "BTC", Bias: BiasBearish}})
	tracker.Observe(now.Add(2*time.Second), []DirectionState{{Symbol: "BTC", Bias: BiasBullish}})

	if got := len(tracker.Recent(2, 0)); got != 2 {
		t.Fatalf("Recent(2) = %d, want 2", got)
	}
}

func TestDirectionTrackerStatesAreSorted(t *testing.T) {
	tracker := NewDirectionTracker("")
	now := time.Now()

	tracker.Observe(now, []DirectionState{
		{Symbol: "SOL", Bias: BiasNeutral},
		{Symbol: "BTC", Bias: BiasBullish},
		{Symbol: "ETH", Bias: BiasBearish},
	})

	states := tracker.States()
	// A randomised order would make the same input render differently each cycle.
	want := []string{"BTC", "ETH", "SOL"}
	for i, symbol := range want {
		if states[i].Symbol != symbol {
			t.Fatalf("states = %+v, want %v", states, want)
		}
	}
}

func TestResolveDirectionHistoryPathHonoursEnv(t *testing.T) {
	t.Setenv(DirectionHistoryPathEnv, "/tmp/custom-direction.json")
	if got := ResolveDirectionHistoryPath(); got != "/tmp/custom-direction.json" {
		t.Fatalf("path = %q", got)
	}

	t.Setenv(DirectionHistoryPathEnv, "")
	if got := ResolveDirectionHistoryPath(); got != DefaultDirectionHistoryPath {
		t.Fatalf("path = %q, want the default", got)
	}
}
