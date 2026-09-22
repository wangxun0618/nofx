package providers

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"nofx/marketdata"
	"nofx/store"
)

// OrderflowRow is the structured form of one instrument's order flow.
type OrderflowRow struct {
	Symbol           string   `json:"symbol"`
	CumulativeCVD    float64  `json:"cumulative_cvd"`
	CVDHyperliquid   float64  `json:"cvd_hyperliquid"`
	CVDBinance       float64  `json:"cvd_binance"`
	NetVolume        float64  `json:"net_volume,omitempty"`
	NetVolumeWindow  string   `json:"net_volume_window,omitempty"`
	TradesPerSecond  float64  `json:"trades_per_second"`
	AggregateSignal  string   `json:"aggregate_signal,omitempty"`
	Venues           []string `json:"venues,omitempty"`
	MetricsAvailable bool     `json:"metrics_available"`
}

// OrderflowPayload is the structured order-flow board.
type OrderflowPayload struct {
	Rows           []OrderflowRow                `json:"rows"`
	LongShort      map[string]HyperDataLongShort `json:"long_short,omitempty"`
	Basis          map[string]HyperDataBasis     `json:"basis,omitempty"`
	VenueStatus    []string                      `json:"venue_status,omitempty"`
	Unavailable    []string                      `json:"unavailable_symbols,omitempty"`
	Version        string                        `json:"hyperdata_version,omitempty"`
	VersionWarning string                        `json:"version_warning,omitempty"`
}

// HyperDataOrderflowProvider surfaces aggressive buy/sell pressure and leverage
// sentiment from a local HyperData Terminal instance.
//
// This is the closest free substitute for the retired Vergex direction board:
// a cumulative volume delta answers "who is crossing the spread right now",
// which is the one directional input the Hyperliquid aggregate feed cannot
// express. It is still an observation, not a verdict — the block says so.
type HyperDataOrderflowProvider struct {
	mu     sync.Mutex
	client *HyperDataClient
	config string
}

// NewHyperDataOrderflowProvider builds the order-flow source.
func NewHyperDataOrderflowProvider() *HyperDataOrderflowProvider {
	return &HyperDataOrderflowProvider{}
}

// Name implements marketdata.Provider.
func (p *HyperDataOrderflowProvider) Name() string { return "hyperdata_orderflow" }

// Description implements marketdata.Provider.
func (p *HyperDataOrderflowProvider) Description() string {
	return "Aggressive buy/sell pressure (CVD), order-flow imbalance and leverage sentiment from a local HyperData Terminal instance."
}

// RequiresAPIKey reports false: the sidecar needs no credential unless it was
// started with its own API key.
func (p *HyperDataOrderflowProvider) RequiresAPIKey() bool { return false }

// RequiredService implements marketdata.ServiceBacked.
func (p *HyperDataOrderflowProvider) RequiredService() string {
	return "HyperData Terminal (python run_api.py)"
}

// Enabled implements marketdata.Provider. Off unless the user has opted in,
// because it needs a separate process to be running.
func (p *HyperDataOrderflowProvider) Enabled(cfg store.IndicatorConfig) bool {
	return cfg.EnableHyperData
}

// clientFor returns the shared HTTP client, rebuilding it when the strategy
// points at a different sidecar. Caching it preserves the health payload that
// every provider in the cycle reads.
func (p *HyperDataOrderflowProvider) clientFor(cfg store.IndicatorConfig) *HyperDataClient {
	return hyperDataClientFor(&p.mu, &p.client, &p.config, cfg)
}

// Fetch implements marketdata.Provider.
func (p *HyperDataOrderflowProvider) Fetch(ctx context.Context, req marketdata.Request, cfg store.IndicatorConfig) (*marketdata.Insight, error) {
	client := p.clientFor(cfg)

	// Health first: it is what tells us whether a venue is contributing at all.
	// Without it the block would present single-venue flow as two-venue flow.
	health, healthErr := client.HealthCached(ctx)
	if healthErr != nil {
		// No sidecar: fail fast and soft. This is the common case when the
		// feature is enabled but the process was never started.
		return nil, fmt.Errorf("hyperdata health: %w", healthErr)
	}
	if err := hyperDataHealthError(health); err != nil {
		return nil, err
	}

	targets := hyperDataTargets(req, maxOrderflowSymbols)

	rows := make([]OrderflowRow, 0, len(targets))
	unavailable := make([]string, 0)
	for _, symbol := range targets {
		flow, err := client.Orderflow(ctx, symbol)
		if err != nil {
			// A symbol the instance does not track is reported as an error by
			// upstream. Record it rather than aborting the whole block.
			unavailable = append(unavailable, symbol)
			continue
		}
		if flow == nil {
			unavailable = append(unavailable, symbol)
			continue
		}
		rows = append(rows, orderflowRow(*flow))
	}

	// The ratio and basis endpoints are market-wide, so a failure here must not
	// discard the per-symbol flow already collected.
	longShort, lsErr := client.LongShortRatio(ctx)
	if lsErr != nil {
		longShort = nil
	}
	basis, basisErr := client.Basis(ctx)
	if basisErr != nil {
		basis = nil
	}

	if len(rows) == 0 && len(longShort) == 0 && len(basis) == 0 {
		return nil, nil
	}

	venues := venueStatuses(health)
	warning := VersionWarning(health)

	var sb strings.Builder
	sb.WriteString("## Aggressive Order Flow & Leverage Sentiment (HyperData, multi-venue)\n\n")
	sb.WriteString("Volume delta measures which side is crossing the spread. This is an observation of current pressure, not a direction call, and it is not the same thing as open interest or funding.\n\n")

	if len(venues) > 0 {
		sb.WriteString("Order-flow venue status: " + strings.Join(venues, ", ") + ".\n\n")
		sb.WriteString("When only one venue is contributing the delta is single-venue evidence and is correspondingly weaker. Some regions cannot receive Binance Futures streams at all; the instance reports that instead of hiding it.\n\n")
	}

	if warning != "" {
		sb.WriteString("⚠ " + warning + "\n\n")
	}

	if len(rows) > 0 {
		sb.WriteString("### Cumulative volume delta\n\n")
		sb.WriteString("| Instrument | CVD (net) | Hyperliquid | Binance | Window net | Trades/s | Signal |\n")
		sb.WriteString("| :--- | ---: | ---: | ---: | ---: | ---: | :--- |\n")
		for _, row := range rows {
			window := "-"
			if row.NetVolumeWindow != "" {
				window = fmt.Sprintf("%s %s", formatUsd(row.NetVolume), row.NetVolumeWindow)
			}
			signal := row.AggregateSignal
			if signal == "" {
				signal = "-"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %.1f | %s |\n",
				row.Symbol, formatUsd(row.CumulativeCVD),
				formatUsd(row.CVDHyperliquid), formatUsd(row.CVDBinance),
				window, row.TradesPerSecond, signal))
		}
		sb.WriteString("\n")
	}

	if len(unavailable) > 0 {
		sb.WriteString("Not tracked by this instance (no order-flow data): " + strings.Join(unavailable, ", ") + ".\n\n")
	}

	if len(longShort) > 0 {
		sb.WriteString("### Leverage sentiment (account long/short ratio)\n\n")
		sb.WriteString("| Instrument | Long | Short | Ratio |\n")
		sb.WriteString("| :--- | ---: | ---: | ---: |\n")
		for _, symbol := range sortedKeys(longShort) {
			entry := longShort[symbol]
			sb.WriteString(fmt.Sprintf("| %s | %.1f%% | %.1f%% | %.2f |\n",
				symbol, entry.LongRatio, entry.ShortRatio, entry.LongShortRatio))
		}
		sb.WriteString("\n")
	}

	if len(basis) > 0 {
		sb.WriteString("### Perpetual basis (perp vs spot)\n\n")
		sb.WriteString("A positive basis means the perpetual trades above spot, i.e. leveraged demand is paying a premium.\n\n")
		sb.WriteString("| Instrument | Spot | Perp | Basis |\n")
		sb.WriteString("| :--- | ---: | ---: | ---: |\n")
		for _, symbol := range sortedKeys(basis) {
			entry := basis[symbol]
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %+.3f%% |\n",
				symbol, formatUsd(entry.SpotPrice), formatUsd(entry.PerpPrice), entry.BasisPct))
		}
		sb.WriteString("\n")
	}

	payload := OrderflowPayload{
		Rows:           rows,
		LongShort:      longShort,
		Basis:          basis,
		VenueStatus:    venues,
		Unavailable:    unavailable,
		Version:        health.Version,
		VersionWarning: warning,
	}

	return &marketdata.Insight{
		Provider: p.Name(),
		Title:    "Order flow & leverage sentiment",
		Markdown: sb.String(),
		Payload:  payload,
	}, nil
}

const (
	// maxOrderflowSymbols caps per-symbol upstream calls: each one is a separate
	// request inside the provider timeout.
	maxOrderflowSymbols = 4
)

// orderflowRow flattens one upstream payload into a table row. The window is
// chosen as the first non-empty entry in orderflowWindowPreference, so the
// displayed horizon stays stable across cycles even if the instance reports
// several.
func orderflowRow(flow HyperDataOrderflow) OrderflowRow {
	row := OrderflowRow{
		Symbol:          strings.ToUpper(flow.Symbol),
		CumulativeCVD:   flow.CumulativeCVD,
		CVDHyperliquid:  flow.CumulativeCVDByVenue["hyperliquid"],
		CVDBinance:      flow.CumulativeCVDByVenue["binance"],
		TradesPerSecond: flow.TradesPerSecond,
		AggregateSignal: strings.TrimSpace(flow.AggregateSignal.String()),
		Venues:          append([]string(nil), flow.VenuesContributing...),
	}

	for _, window := range orderflowWindowPreference {
		frame, ok := flow.Timeframes[window]
		if !ok {
			continue
		}
		if frame.BuyVolume == 0 && frame.SellVolume == 0 && frame.NetVolume == 0 {
			continue
		}
		row.NetVolume = frame.NetVolume
		row.NetVolumeWindow = window
		row.MetricsAvailable = true
		break
	}

	if row.Symbol == "" {
		row.Symbol = "?"
	}
	return row
}

// orderflowWindowPreference is the display order for reported order-flow
// windows, shortest first.
var orderflowWindowPreference = []string{"5m", "15m", "1h", "30m", "1m", "4h", "1d"}

// hyperDataTargets picks the instruments worth querying: open positions first
// (they carry live risk), then the candidate pool, de-duplicated and capped.
func hyperDataTargets(req marketdata.Request, limit int) []string {
	ordered := make([]string, 0, len(req.Positions)+len(req.Symbols))
	ordered = append(ordered, req.Positions...)
	ordered = append(ordered, req.Symbols...)

	seen := make(map[string]bool, len(ordered))
	targets := make([]string, 0, limit)
	for _, symbol := range ordered {
		name := normalizeHyperDataSymbol(symbol)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		targets = append(targets, name)
		if len(targets) >= limit {
			break
		}
	}
	return targets
}

// normalizeHyperDataSymbol reduces any platform symbol to the plain base coin
// the sidecar keys its feeds by, dropping the xyz equities prefix.
func normalizeHyperDataSymbol(symbol string) string {
	trimmed := strings.TrimSpace(symbol)
	if trimmed == "" {
		return ""
	}
	if idx := strings.LastIndex(trimmed, ":"); idx >= 0 {
		trimmed = trimmed[idx+1:]
	}
	return strings.ToUpper(strings.TrimSpace(trimmed))
}

// hyperDataHealthError converts a sidecar health payload into an error when the
// instance is not in a state worth querying.
func hyperDataHealthError(health *HyperDataHealth) error {
	if health == nil {
		return fmt.Errorf("hyperdata health: empty payload")
	}
	switch strings.ToLower(strings.TrimSpace(health.Status)) {
	case "", "ok", "warn", "initializing":
		// "warn" means one feed is degraded, not that the instance is unusable;
		// the block reports which venue is quiet.
		return nil
	default:
		return fmt.Errorf("hyperdata sidecar status %q (failed components: %s)",
			health.Status, strings.Join(health.FailedComponents, ", "))
	}
}

// sortedKeys returns the map keys in a stable order so the prompt does not churn
// between cycles. Go randomises map iteration, which would otherwise make the
// same input render differently every time.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
