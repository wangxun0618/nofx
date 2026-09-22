// Package marketdata defines the pluggable market-intelligence layer.
//
// A Provider is one source of market context (fund flow, open-interest
// structure, liquidation clusters, ...). The strategy engine does not know any
// concrete source: it asks a Registry to collect whatever is enabled, and turns
// the returned Insights into prompt blocks plus API payloads.
//
// Adding a new data source therefore means implementing Provider and registering
// it in marketdata/providers — the engine, the trading loop and the prompt
// builder stay untouched.
package marketdata

import (
	"context"
	"time"

	"nofx/store"
)

// Insight is one rendered market-intelligence block. It is intentionally
// format-agnostic: Markdown feeds the LLM prompt, while Payload feeds the HTTP
// API and the web UI.
type Insight struct {
	// Provider is the stable identifier of the source, e.g. "hyperliquid_flow".
	Provider string `json:"provider"`
	// Title is a short human-readable heading for the UI.
	Title string `json:"title"`
	// Markdown is the English prompt block.
	Markdown string `json:"markdown"`
	// MarkdownZh is the Chinese prompt block; falls back to Markdown when empty.
	MarkdownZh string `json:"markdown_zh,omitempty"`
	// Payload is the structured form of the same data, consumed by the API.
	Payload any `json:"payload,omitempty"`
	// FetchedAt records when the underlying data was retrieved.
	FetchedAt time.Time `json:"fetched_at"`
}

// PromptBlock returns the markdown for the requested language, falling back to
// English whenever no translation is available. Providers are encouraged but not
// required to supply both languages.
func (i *Insight) PromptBlock(language string) string {
	if i == nil {
		return ""
	}
	if language == "zh" && i.MarkdownZh != "" {
		return i.MarkdownZh
	}
	return i.Markdown
}

// Request carries the per-cycle arguments a provider may need. Providers must
// tolerate every field being empty: the engine may collect insights before
// candidates are known.
type Request struct {
	// Symbols is the candidate universe plus any open position symbols.
	Symbols []string
	// Positions is the subset of Symbols that currently has an open position.
	Positions []string
	// Language is the strategy language ("en" or "zh").
	Language string
	// Limit bounds how many rows a ranking-style provider should emit.
	Limit int
}

// NormalizeLimit returns the requested row count, falling back to the supplied
// default when the request left it unset.
func (r Request) NormalizeLimit(fallback int) int {
	if r.Limit > 0 {
		return r.Limit
	}
	return fallback
}

// Provider is a pluggable market-intelligence source.
//
// Implementations must be safe for concurrent use: the registry may call Fetch
// from several goroutines, and long-running processes reuse one instance across
// trading cycles.
type Provider interface {
	// Name is the stable identifier used in configuration, logs and the API.
	Name() string
	// Description explains what the source offers, shown in the UI.
	Description() string
	// RequiresAPIKey reports whether the source needs a third-party credential.
	// The UI uses it to badge optional sources and to decide whether to prompt
	// for a key instead of enabling them silently.
	RequiresAPIKey() bool
	// Enabled reports whether this source should run under the given strategy
	// configuration. Returning false skips the fetch entirely.
	Enabled(cfg store.IndicatorConfig) bool
	// Fetch retrieves the insight. Returning (nil, nil) means "enabled but no
	// data this cycle" and is not treated as an error.
	Fetch(ctx context.Context, req Request, cfg store.IndicatorConfig) (*Insight, error)
}

// ServiceBacked is an optional capability a Provider may implement when it needs
// a separate process to be running.
//
// RequiresAPIKey cannot express this: a locally hosted sidecar needs no
// credential, yet it is still unavailable until the user starts it. The UI reads
// RequiredService to explain why such a source is off by default.
type ServiceBacked interface {
	// RequiredService names the external process in user-facing terms.
	RequiredService() string
}

// SourceSetting reports whether a provider is selected by the strategy config.
//
// The selection rule is deliberately simple so that behaviour is predictable:
//   - an empty MarketInsightSources list selects every provider that reports
//     itself Enabled — this keeps newly added sources working out of the box
//   - a non-empty list is an explicit allow-list
func SourceSetting(cfg store.IndicatorConfig, name string) bool {
	if len(cfg.MarketInsightSources) == 0 {
		return true
	}
	for _, allowed := range cfg.MarketInsightSources {
		if allowed == name {
			return true
		}
	}
	return false
}
