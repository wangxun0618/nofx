package marketdata

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"nofx/logger"
	"nofx/store"
)

const (
	// defaultFetchTimeout bounds a single provider fetch so one slow upstream
	// cannot stall an entire trading cycle.
	defaultFetchTimeout = 12 * time.Second
	// defaultRowLimit is used when the strategy left the row count unset.
	defaultRowLimit = 10
)

// Registry holds the provider set and collects their insights. It is the only
// thing the strategy engine talks to, which is what keeps concrete data sources
// swappable.
type Registry struct {
	providers []Provider
}

// NewRegistry builds a registry from the supplied providers.
func NewRegistry(providers ...Provider) *Registry {
	r := &Registry{}
	for _, p := range providers {
		r.Register(p)
	}
	return r
}

// Register appends a provider. Duplicate names are ignored so a provider
// registered twice cannot emit its block twice into the prompt.
func (r *Registry) Register(p Provider) {
	if r == nil || p == nil {
		return
	}
	for _, existing := range r.providers {
		if existing.Name() == p.Name() {
			logger.Warnf("marketdata: ignoring duplicate provider %q", p.Name())
			return
		}
	}
	r.providers = append(r.providers, p)
}

// All returns the registered providers in registration order, regardless of
// configuration. The UI uses it to list every available source.
func (r *Registry) All() []Provider {
	if r == nil {
		return nil
	}
	out := make([]Provider, len(r.providers))
	copy(out, r.providers)
	return out
}

// Active returns the providers that this configuration selects, in registration
// order, so callers can preview exactly which sources will run.
func (r *Registry) Active(cfg store.IndicatorConfig) []Provider {
	if r == nil || !cfg.EnableMarketInsights {
		return nil
	}
	var active []Provider
	for _, p := range r.providers {
		if !SourceSetting(cfg, p.Name()) || !p.Enabled(cfg) {
			continue
		}
		active = append(active, p)
	}
	return active
}

// SourceFailure records a provider that was selected but produced nothing.
//
// Providers are silently dropped from the prompt when they fail, which leaves
// the model unable to tell "nothing worth reporting" from "the source was down".
// That distinction matters: a block missing because its feed is dead deserves
// less trust in the blocks that remain, and the model has no other way to know.
type SourceFailure struct {
	Provider string `json:"provider"`
	Reason   string `json:"reason"`
}

// CoverageProvider is the synthetic provider name of the block that reports
// which sources did not contribute.
const CoverageProvider = "data_coverage"

// Collect fetches every active provider concurrently and returns the insights
// that produced data, ordered by provider registration so the generated prompt
// stays stable between cycles.
//
// Failures are logged and skipped: a broken or rate-limited source must never
// block trading, matching the rest of the market-data pipeline. What they are
// not allowed to be is invisible — see SourceFailure.
func (r *Registry) Collect(ctx context.Context, req Request, cfg store.IndicatorConfig) []*Insight {
	active := r.Active(cfg)
	if len(active) == 0 {
		return nil
	}

	if req.Limit <= 0 {
		req.Limit = defaultRowLimit
	}

	insights := make([]*Insight, len(active))
	failures := make([]SourceFailure, len(active))
	var wg sync.WaitGroup

	for i, p := range active {
		wg.Add(1)
		go func(index int, provider Provider) {
			defer wg.Done()

			fetchCtx, cancel := context.WithTimeout(ctx, defaultFetchTimeout)
			defer cancel()

			started := time.Now()
			insight, err := provider.Fetch(fetchCtx, req, cfg)
			if err != nil {
				logger.Warnf("⚠️  Market insight %q failed after %s: %v",
					provider.Name(), time.Since(started).Round(time.Millisecond), err)
				failures[index] = SourceFailure{Provider: provider.Name(), Reason: err.Error()}
				return
			}
			if insight == nil || strings.TrimSpace(insight.Markdown) == "" {
				logger.Infof("⏭️  Market insight %q returned no data", provider.Name())
				failures[index] = SourceFailure{
					Provider: provider.Name(),
					Reason:   "reachable but reported nothing this cycle",
				}
				return
			}
			if insight.Provider == "" {
				insight.Provider = provider.Name()
			}
			if insight.FetchedAt.IsZero() {
				insight.FetchedAt = time.Now()
			}
			insights[index] = insight
		}(i, p)
	}

	wg.Wait()

	out := make([]*Insight, 0, len(active)+1)
	missing := make([]SourceFailure, 0, len(failures))
	for i, ins := range insights {
		if ins != nil {
			out = append(out, ins)
			continue
		}
		if failures[i].Provider != "" {
			missing = append(missing, failures[i])
		}
	}

	// Report the gaps last, after the data that did arrive, so the model reads
	// what it has before being told what it lacks.
	if len(missing) > 0 {
		out = append(out, coverageInsight(missing, len(out)))
	}
	return out
}

// coverageInsight renders the list of sources that did not contribute.
//
// It is deliberately terse and stated as a fact rather than a warning: its job
// is to stop the model from treating a partial picture as a complete one, not to
// alarm it.
func coverageInsight(missing []SourceFailure, delivered int) *Insight {
	var sb strings.Builder
	sb.WriteString("## Data source coverage\n\n")
	sb.WriteString(fmt.Sprintf("%d of %d selected market-context sources contributed this cycle. The ones below did not, so their subjects are simply absent from the blocks above rather than confirmed quiet.\n\n",
		delivered, delivered+len(missing)))
	for _, failure := range missing {
		sb.WriteString(fmt.Sprintf("- %s — %s\n", failure.Provider, truncateReason(failure.Reason)))
	}
	sb.WriteString("\nWeight the remaining blocks accordingly; do not read a gap as a neutral reading.\n")

	reasons := make([]string, 0, len(missing))
	for _, failure := range missing {
		reasons = append(reasons, failure.Provider)
	}

	return &Insight{
		Provider:  CoverageProvider,
		Title:     "Data source coverage",
		Markdown:  sb.String(),
		Payload:   missing,
		FetchedAt: time.Now(),
	}
}

// truncateReason keeps an upstream error short and single-line, so it cannot
// break the bullet list it is rendered into.
func truncateReason(reason string) string {
	const maxReason = 240
	reason = strings.Join(strings.Fields(reason), " ")
	if len(reason) <= maxReason {
		return reason
	}
	return reason[:maxReason] + "…"
}

// InsightsByProvider indexes a collected slice for API consumers.
func InsightsByProvider(insights []*Insight) map[string]*Insight {
	byProvider := make(map[string]*Insight, len(insights))
	for _, ins := range insights {
		if ins != nil {
			byProvider[ins.Provider] = ins
		}
	}
	return byProvider
}
