package marketdata

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"nofx/store"
)

// fakeProvider is a scriptable Provider. The registry's whole job is to run many
// of these correctly, so the tests drive it entirely through fakes.
type fakeProvider struct {
	name       string
	requires   bool
	enabled    bool
	delay      time.Duration
	insight    *Insight
	err        error
	fetched    int32
	lastReq    Request
	lastConfig store.IndicatorConfig
}

func (f *fakeProvider) Name() string                           { return f.name }
func (f *fakeProvider) Description() string                    { return "fake " + f.name }
func (f *fakeProvider) RequiresAPIKey() bool                   { return f.requires }
func (f *fakeProvider) Enabled(cfg store.IndicatorConfig) bool { return f.enabled }

func (f *fakeProvider) Fetch(ctx context.Context, req Request, cfg store.IndicatorConfig) (*Insight, error) {
	f.fetched++
	f.lastReq = req
	f.lastConfig = cfg

	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.insight, nil
}

func enabledConfig() store.IndicatorConfig {
	return store.IndicatorConfig{EnableMarketInsights: true, MarketInsightLimit: 10}
}

// --- Insight ---------------------------------------------------------------

func TestInsightPromptBlockFallsBackToEnglish(t *testing.T) {
	cases := []struct {
		name     string
		insight  *Insight
		language string
		want     string
	}{
		{"english request", &Insight{Markdown: "en"}, "en", "en"},
		{"chinese request with translation", &Insight{Markdown: "en", MarkdownZh: "zh"}, "zh", "zh"},
		{"chinese request without translation", &Insight{Markdown: "en"}, "zh", "en"},
		{"nil insight", nil, "zh", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.insight.PromptBlock(tc.language); got != tc.want {
				t.Fatalf("PromptBlock(%q) = %q, want %q", tc.language, got, tc.want)
			}
		})
	}
}

func TestRequestNormalizeLimit(t *testing.T) {
	if got := (Request{}).NormalizeLimit(10); got != 10 {
		t.Fatalf("unset limit = %d, want fallback 10", got)
	}
	if got := (Request{Limit: 3}).NormalizeLimit(10); got != 3 {
		t.Fatalf("explicit limit = %d, want 3", got)
	}
}

// --- Selection -------------------------------------------------------------

func TestSourceSettingEmptyListSelectsEverything(t *testing.T) {
	// An empty allow-list must mean "all", otherwise a newly registered source
	// would never run until every saved strategy was migrated.
	cfg := store.IndicatorConfig{}
	for _, name := range []string{"anything", "hyperliquid_flow", ""} {
		if !SourceSetting(cfg, name) {
			t.Fatalf("SourceSetting(empty list, %q) = false, want true", name)
		}
	}
}

func TestSourceSettingNonEmptyListIsAllowList(t *testing.T) {
	cfg := store.IndicatorConfig{MarketInsightSources: []string{"a", "b"}}
	if !SourceSetting(cfg, "a") || !SourceSetting(cfg, "b") {
		t.Fatal("listed sources should be selected")
	}
	if SourceSetting(cfg, "c") {
		t.Fatal("unlisted source should be rejected")
	}
}

func TestRegistryActiveHonoursMasterSwitchAndAllowList(t *testing.T) {
	a := &fakeProvider{name: "a", enabled: true}
	b := &fakeProvider{name: "b", enabled: true}
	c := &fakeProvider{name: "c", enabled: false}
	r := NewRegistry(a, b, c)

	if got := r.Active(store.IndicatorConfig{}); got != nil {
		t.Fatalf("master switch off should select nothing, got %d", len(got))
	}

	active := r.Active(enabledConfig())
	if len(active) != 2 || active[0].Name() != "a" || active[1].Name() != "b" {
		t.Fatalf("expected [a b] in registration order, got %v", names(active))
	}

	cfg := enabledConfig()
	cfg.MarketInsightSources = []string{"b"}
	if active := r.Active(cfg); len(active) != 1 || active[0].Name() != "b" {
		t.Fatalf("allow-list should keep only b, got %v", names(active))
	}
}

func TestRegistryIgnoresDuplicateNames(t *testing.T) {
	r := NewRegistry(&fakeProvider{name: "dup", enabled: true})
	r.Register(&fakeProvider{name: "dup", enabled: true})

	// A duplicate would otherwise emit its prompt block twice.
	if got := len(r.All()); got != 1 {
		t.Fatalf("registry holds %d providers, want 1", got)
	}
}

func TestRegistryAllIgnoresConfiguration(t *testing.T) {
	r := NewRegistry(&fakeProvider{name: "a", enabled: false}, &fakeProvider{name: "b", enabled: true})
	if got := len(r.All()); got != 2 {
		t.Fatalf("All() = %d, want every registered provider (2)", got)
	}
}

// --- Collection ------------------------------------------------------------

func TestCollectPreservesRegistrationOrder(t *testing.T) {
	// "slow" is registered first but finishes last. The prompt must still see it
	// first, or the AI's context would reshuffle between cycles.
	slow := &fakeProvider{name: "slow", enabled: true, delay: 120 * time.Millisecond,
		insight: &Insight{Markdown: "slow block"}}
	fast := &fakeProvider{name: "fast", enabled: true,
		insight: &Insight{Markdown: "fast block"}}

	r := NewRegistry(slow, fast)
	got := r.Collect(context.Background(), Request{}, enabledConfig())

	if len(got) != 2 {
		t.Fatalf("collected %d insights, want 2", len(got))
	}
	if got[0].Provider != "slow" || got[1].Provider != "fast" {
		t.Fatalf("order = %s, %s; want slow, fast", got[0].Provider, got[1].Provider)
	}
}

func TestCollectSkipsFailuresWithoutAborting(t *testing.T) {
	// A dead upstream must never block trading, so the healthy sources still
	// have to come back — with the gap named after them.
	broken := &fakeProvider{name: "broken", enabled: true, err: errors.New("upstream 500")}
	healthy := &fakeProvider{name: "healthy", enabled: true,
		insight: &Insight{Markdown: "fine"}}

	r := NewRegistry(broken, healthy)
	got := r.Collect(context.Background(), Request{}, enabledConfig())

	if delivered := dataProvidersOf(got); delivered != "healthy" {
		t.Fatalf("expected only the healthy insight to deliver, got %v", providersOf(got))
	}
	if coverage := coverageBlock(t, got); !strings.Contains(coverage.Markdown, "broken") {
		t.Fatalf("the failed source must be named in the coverage block:\n%s", coverage.Markdown)
	}
}

func TestCollectSkipsEmptyResults(t *testing.T) {
	cases := map[string]*Insight{
		"nil insight":     nil,
		"blank markdown":  {Markdown: "   \n\t "},
		"whitespace only": {Markdown: "\n"},
	}
	for name, insight := range cases {
		t.Run(name, func(t *testing.T) {
			p := &fakeProvider{name: "p", enabled: true, insight: insight}
			got := NewRegistry(p).Collect(context.Background(), Request{}, enabledConfig())

			if delivered := dataProvidersOf(got); delivered != "" {
				t.Fatalf("an empty result must not become a data block, got %v", delivered)
			}
			// It must still be reported: "no data" and "source down" both leave
			// the same hole in the prompt, and the model has to know about it.
			if coverage := coverageBlock(t, got); !strings.Contains(coverage.Markdown, "p") {
				t.Fatalf("the empty source must be named:\n%s", coverage.Markdown)
			}
		})
	}
}

func TestCollectSkipsDisabledAndUnselectedSources(t *testing.T) {
	off := &fakeProvider{name: "off", enabled: false, insight: &Insight{Markdown: "x"}}
	on := &fakeProvider{name: "on", enabled: true, insight: &Insight{Markdown: "x"}}
	other := &fakeProvider{name: "other", enabled: true, insight: &Insight{Markdown: "x"}}

	r := NewRegistry(off, on, other)
	cfg := enabledConfig()
	cfg.MarketInsightSources = []string{"on"}

	got := r.Collect(context.Background(), Request{}, cfg)
	if len(got) != 1 || got[0].Provider != "on" {
		t.Fatalf("expected only %q, got %v", "on", providersOf(got))
	}
	// Skipped sources must not even be contacted.
	if off.fetched != 0 || other.fetched != 0 {
		t.Fatalf("skipped providers were fetched: off=%d other=%d", off.fetched, other.fetched)
	}
}

func TestCollectFillsDefaultLimitAndMetadata(t *testing.T) {
	// Providers are allowed to leave Provider/FetchedAt blank; the registry
	// stamps them so every consumer sees a complete insight.
	p := &fakeProvider{name: "p", enabled: true, insight: &Insight{Markdown: "x"}}
	r := NewRegistry(p)

	got := r.Collect(context.Background(), Request{}, enabledConfig())
	if len(got) != 1 {
		t.Fatalf("collected %d insights, want 1", len(got))
	}
	if got[0].Provider != "p" {
		t.Fatalf("Provider = %q, want %q", got[0].Provider, "p")
	}
	if got[0].FetchedAt.IsZero() {
		t.Fatal("FetchedAt should be stamped when the provider leaves it zero")
	}
	if p.lastReq.Limit != defaultRowLimit {
		t.Fatalf("Request.Limit = %d, want default %d", p.lastReq.Limit, defaultRowLimit)
	}
}

func TestCollectKeepsExplicitLimitAndProviderMetadata(t *testing.T) {
	stamp := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	p := &fakeProvider{name: "p", enabled: true, insight: &Insight{
		Provider:  "explicit",
		Markdown:  "x",
		FetchedAt: stamp,
	}}

	got := NewRegistry(p).Collect(context.Background(), Request{Limit: 4}, enabledConfig())
	if p.lastReq.Limit != 4 {
		t.Fatalf("Request.Limit = %d, want 4", p.lastReq.Limit)
	}
	// A provider that names itself and stamps its own time wins.
	if got[0].Provider != "explicit" {
		t.Fatalf("Provider = %q, want explicit", got[0].Provider)
	}
	if !got[0].FetchedAt.Equal(stamp) {
		t.Fatalf("FetchedAt = %v, want %v", got[0].FetchedAt, stamp)
	}
}

func TestCollectPropagatesStrategyConfigToProviders(t *testing.T) {
	// Providers read credentials and per-source settings from the config, so the
	// registry must hand it through untouched.
	p := &fakeProvider{name: "p", enabled: true, insight: &Insight{Markdown: "x"}}
	cfg := enabledConfig()
	cfg.CoinankAPIKey = "secret"
	cfg.MarketInsightLimit = 7

	NewRegistry(p).Collect(context.Background(), Request{}, cfg)
	if p.lastConfig.CoinankAPIKey != "secret" || p.lastConfig.MarketInsightLimit != 7 {
		t.Fatalf("provider saw %+v", p.lastConfig)
	}
}

func TestCollectReturnsNilWhenNothingIsActive(t *testing.T) {
	r := NewRegistry(&fakeProvider{name: "a", enabled: false})
	if got := r.Collect(context.Background(), Request{}, enabledConfig()); got != nil {
		t.Fatalf("expected nil, got %v", providersOf(got))
	}
}

func TestCollectTimeoutDoesNotAbortOtherSources(t *testing.T) {
	// One source hanging must not consume the whole cycle.
	hung := &fakeProvider{name: "hung", enabled: true, delay: time.Hour}
	quick := &fakeProvider{name: "quick", enabled: true, insight: &Insight{Markdown: "x"}}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan []*Insight, 1)
	go func() { done <- NewRegistry(hung, quick).Collect(ctx, Request{}, enabledConfig()) }()

	select {
	case got := <-done:
		if delivered := dataProvidersOf(got); delivered != "quick" {
			t.Fatalf("expected only %q to deliver, got %v", "quick", providersOf(got))
		}
		if coverage := coverageBlock(t, got); !strings.Contains(coverage.Markdown, "hung") {
			t.Fatalf("the timed-out source must be named:\n%s", coverage.Markdown)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Collect did not return after the context deadline")
	}
}

func TestInsightsByProvider(t *testing.T) {
	byName := InsightsByProvider([]*Insight{
		{Provider: "a", Markdown: "x"},
		nil,
		{Provider: "b", Markdown: "y"},
	})
	if len(byName) != 2 || byName["a"] == nil || byName["b"] == nil {
		t.Fatalf("index = %v, want a and b", byName)
	}
}

// --- Coverage reporting ----------------------------------------------------

func TestCollectReportsSourcesThatDidNotContribute(t *testing.T) {
	// A partially-blind model is worse than a blind one: without this block it
	// reads "the source was down" as "the market is quiet".
	ok := &fakeProvider{name: "ok", enabled: true, insight: &Insight{Markdown: "data"}}
	broken := &fakeProvider{name: "broken", enabled: true, err: errors.New("upstream 503")}
	quiet := &fakeProvider{name: "quiet", enabled: true, insight: nil}

	got := NewRegistry(ok, broken, quiet).Collect(context.Background(), Request{}, enabledConfig())

	if len(got) != 2 {
		t.Fatalf("got %d insights (%s), want the data plus one coverage block", len(got), providersOf(got))
	}
	if got[0].Provider != "ok" {
		t.Fatalf("first insight = %q, want the delivered source leading", got[0].Provider)
	}
	coverage := got[1]
	if coverage.Provider != CoverageProvider {
		t.Fatalf("last insight = %q, want %q", coverage.Provider, CoverageProvider)
	}
	for _, want := range []string{"broken", "upstream 503", "quiet", "reported nothing", "1 of 3"} {
		if !strings.Contains(coverage.Markdown, want) {
			t.Fatalf("coverage block missing %q:\n%s", want, coverage.Markdown)
		}
	}
}

func TestCollectOmitsCoverageWhenEverySourceDelivers(t *testing.T) {
	a := &fakeProvider{name: "a", enabled: true, insight: &Insight{Markdown: "a"}}
	b := &fakeProvider{name: "b", enabled: true, insight: &Insight{Markdown: "b"}}

	got := NewRegistry(a, b).Collect(context.Background(), Request{}, enabledConfig())

	if len(got) != 2 {
		t.Fatalf("got %s, want exactly a and b", providersOf(got))
	}
}

func TestCoverageInsightBoundsUpstreamErrors(t *testing.T) {
	long := strings.Repeat("x", 900)
	got := NewRegistry(&fakeProvider{name: "noisy", enabled: true, err: errors.New(long)}).
		Collect(context.Background(), Request{}, enabledConfig())

	if len(got) != 1 || got[0].Provider != CoverageProvider {
		t.Fatalf("got %s, want a coverage block", providersOf(got))
	}
	if strings.Contains(got[0].Markdown, long) {
		t.Fatal("an unbounded upstream error must not reach the prompt verbatim")
	}
	if !strings.Contains(got[0].Markdown, "…") {
		t.Fatalf("truncation should be visible:\n%s", got[0].Markdown)
	}
}

func TestTruncateReasonFlattensNewlines(t *testing.T) {
	// A multi-line error would break the bullet list it is rendered into.
	got := truncateReason("first line\nsecond line\n\tthird")
	if strings.ContainsAny(got, "\n\t") {
		t.Fatalf("truncateReason left control characters: %q", got)
	}
}

// --- Helpers ---------------------------------------------------------------

// dataProvidersOf lists the providers that actually delivered data, excluding
// the synthetic coverage block, so a test can assert on the payload without the
// gap reporting that rides along with it.
func dataProvidersOf(insights []*Insight) string {
	out := make([]string, 0, len(insights))
	for _, i := range insights {
		if i.Provider != CoverageProvider {
			out = append(out, i.Provider)
		}
	}
	return strings.Join(out, ", ")
}

// coverageBlock returns the gap report, failing the test when it is missing.
func coverageBlock(t *testing.T, insights []*Insight) *Insight {
	t.Helper()
	for _, i := range insights {
		if i.Provider == CoverageProvider {
			return i
		}
	}
	t.Fatalf("no coverage block in %v", providersOf(insights))
	return nil
}

func names(providers []Provider) string {
	out := make([]string, 0, len(providers))
	for _, p := range providers {
		out = append(out, p.Name())
	}
	return strings.Join(out, ", ")
}

func providersOf(insights []*Insight) string {
	out := make([]string, 0, len(insights))
	for _, i := range insights {
		out = append(out, i.Provider)
	}
	return strings.Join(out, ", ")
}
