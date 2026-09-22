package providers

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"nofx/marketdata"
	"nofx/store"
)

// PositioningRow is the structured form of one tracked position.
type PositioningRow struct {
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`
	SizeUSD       float64 `json:"size_usd"`
	EntryPrice    float64 `json:"entry_price"`
	MarkPrice     float64 `json:"current_price"`
	LiqPrice      float64 `json:"liq_price"`
	DistancePct   float64 `json:"distance_pct"`
	Leverage      float64 `json:"leverage"`
	UnrealizedPnL float64 `json:"unrealized_pnl"`
}

// PositioningPayload is the structured positioning board.
type PositioningPayload struct {
	Largest         []PositioningRow `json:"largest"`
	NearLiquidation []PositioningRow `json:"near_liquidation"`
	DangerThreshold float64          `json:"danger_threshold_pct,omitempty"`
	ScanAgeSeconds  *float64         `json:"scan_age_seconds,omitempty"`
	TrackedCount    int              `json:"tracked_positions"`
	Version         string           `json:"hyperdata_version,omitempty"`
}

// HyperDataPositioningProvider reports where large tracked positions sit relative
// to their liquidation price.
//
// Scope caveat, stated plainly because it decides how the model should read the
// table: upstream scans a set of addresses it has discovered, within a per-cycle
// scan budget. This is therefore a sample of tracked wallets, not a census of
// the market, and the block says so rather than implying a complete order book
// of liquidation levels.
type HyperDataPositioningProvider struct {
	mu     sync.Mutex
	client *HyperDataClient
	config string
}

// NewHyperDataPositioningProvider builds the positioning source.
func NewHyperDataPositioningProvider() *HyperDataPositioningProvider {
	return &HyperDataPositioningProvider{}
}

// Name implements marketdata.Provider.
func (p *HyperDataPositioningProvider) Name() string { return "hyperdata_positioning" }

// Description implements marketdata.Provider.
func (p *HyperDataPositioningProvider) Description() string {
	return "Largest tracked positions and the ones closest to liquidation, from a local HyperData Terminal instance."
}

// RequiresAPIKey reports false.
func (p *HyperDataPositioningProvider) RequiresAPIKey() bool { return false }

// RequiredService implements marketdata.ServiceBacked.
func (p *HyperDataPositioningProvider) RequiredService() string {
	return "HyperData Terminal (python run_api.py)"
}

// Enabled implements marketdata.Provider.
func (p *HyperDataPositioningProvider) Enabled(cfg store.IndicatorConfig) bool {
	return cfg.EnableHyperData
}

// clientFor returns the shared HTTP client for this sidecar.
func (p *HyperDataPositioningProvider) clientFor(cfg store.IndicatorConfig) *HyperDataClient {
	return hyperDataClientFor(&p.mu, &p.client, &p.config, cfg)
}

// Fetch implements marketdata.Provider.
func (p *HyperDataPositioningProvider) Fetch(ctx context.Context, _ marketdata.Request, cfg store.IndicatorConfig) (*marketdata.Insight, error) {
	client := p.clientFor(cfg)

	whales, whalesErr := client.Whales(ctx)
	danger, dangerErr := client.DangerZone(ctx)

	// Both endpoints failing means the sidecar is unhealthy; report it so the
	// registry can record the source as unavailable.
	if whalesErr != nil && dangerErr != nil {
		return nil, fmt.Errorf("hyperdata positions: %w", whalesErr)
	}

	largest := hyperDataRows(whales, maxPositioningRows)
	near := hyperDataRows(danger, maxDangerRows)

	if len(largest) == 0 && len(near) == 0 {
		return nil, nil
	}

	var scanAge *float64
	if danger != nil && danger.ScanAgeSeconds != nil {
		scanAge = danger.ScanAgeSeconds
	} else if whales != nil {
		scanAge = whales.ScanAgeSeconds
	}

	version := ""
	if health, err := client.HealthCached(ctx); err == nil && health != nil {
		version = health.Version
	}

	var sb strings.Builder
	sb.WriteString("## Tracked Positions & Liquidation Distance (HyperData)\n\n")
	sb.WriteString("Sample, not a census: these are positions held by wallets this instance has discovered and scanned within its per-cycle budget. Read it as \"examples of large positioning\", never as the market's total open interest or a complete set of liquidation levels.\n\n")

	if scanAge != nil {
		sb.WriteString(fmt.Sprintf("Position scan is %.0fs old.\n\n", *scanAge))
	}

	if len(largest) > 0 {
		sb.WriteString("### Largest tracked positions\n\n")
		sb.WriteString(positioningTable(largest, false))
		sb.WriteString("\n")
	}

	if len(near) > 0 {
		threshold := ""
		if danger != nil && danger.ThresholdPct > 0 {
			threshold = fmt.Sprintf(" (within %.1f%% of liquidation)", danger.ThresholdPct)
		}
		sb.WriteString("### Closest to liquidation" + threshold + "\n\n")
		sb.WriteString("A short distance to liquidation in one direction is fuel for a squeeze in the other: longs being wiped sell into the book, shorts being wiped buy back.\n\n")
		sb.WriteString(positioningTable(near, true))
		sb.WriteString("\n")
	} else if dangerErr != nil {
		sb.WriteString("Liquidation danger zone unavailable this cycle.\n\n")
	}

	payload := PositioningPayload{
		Largest:         largest,
		NearLiquidation: near,
		ScanAgeSeconds:  scanAge,
		TrackedCount:    len(largest) + len(near),
		Version:         version,
	}
	if danger != nil {
		payload.DangerThreshold = danger.ThresholdPct
	}

	return &marketdata.Insight{
		Provider: p.Name(),
		Title:    "Tracked positions & liquidation distance",
		Markdown: sb.String(),
		Payload:  payload,
	}, nil
}

const (
	// maxPositioningRows bounds the largest-position table.
	maxPositioningRows = 8
	// maxDangerRows bounds the near-liquidation table.
	maxDangerRows = 6
)

// positioningTable renders a position table. Distance and PnL are signed the way
// a trader reads them: a position underwater shows a negative PnL, and the
// liquidation distance is always a positive percentage of the current price.
//
// The row is assembled field by field rather than from a shared prefix + suffix,
// because distance sits between Liq and Lev in the header — building it out of
// order would silently misalign every column to its right.
func positioningTable(rows []PositioningRow, withDistance bool) string {
	var sb strings.Builder
	if withDistance {
		sb.WriteString("| Instrument | Side | Size | Entry | Mark | Liq | Distance | Lev | uPnL |\n")
		sb.WriteString("| :--- | :--- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	} else {
		sb.WriteString("| Instrument | Side | Size | Entry | Mark | Liq | Lev | uPnL |\n")
		sb.WriteString("| :--- | :--- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	}

	for _, row := range rows {
		sb.WriteString("| " + row.Symbol + " | " + strings.ToUpper(row.Side) +
			" | $" + formatUsd(row.SizeUSD) +
			" | " + formatPrice(row.EntryPrice) +
			" | " + formatPrice(row.MarkPrice) +
			" | " + formatPrice(row.LiqPrice))
		if withDistance {
			sb.WriteString(fmt.Sprintf(" | %.2f%%", row.DistancePct))
		}
		sb.WriteString(fmt.Sprintf(" | %.0fx | %s |\n", row.Leverage, formatUsd(row.UnrealizedPnL)))
	}
	return sb.String()
}

// formatPrice renders a price with enough precision to stay readable across
// instruments spanning six orders of magnitude.
func formatPrice(v float64) string {
	switch {
	case v == 0:
		return "-"
	case v >= 1000:
		return fmt.Sprintf("%.1f", v)
	case v >= 1:
		return fmt.Sprintf("%.4f", v)
	default:
		return fmt.Sprintf("%.6f", v)
	}
}

// hyperDataRows converts an upstream position list into table rows, capped.
func hyperDataRows(payload *HyperDataPositions, limit int) []PositioningRow {
	if payload == nil || len(payload.Positions) == 0 {
		return nil
	}
	rows := make([]PositioningRow, 0, min(len(payload.Positions), limit))
	for _, position := range payload.Positions {
		if len(rows) >= limit {
			break
		}
		symbol := strings.ToUpper(strings.TrimSpace(position.Symbol))
		if symbol == "" {
			continue
		}
		rows = append(rows, PositioningRow{
			Symbol:        symbol,
			Side:          strings.TrimSpace(position.Side),
			SizeUSD:       position.SizeUSD,
			EntryPrice:    position.EntryPrice,
			MarkPrice:     position.CurrentPrice,
			LiqPrice:      position.LiqPrice,
			DistancePct:   position.DistancePct,
			Leverage:      position.Leverage,
			UnrealizedPnL: position.UnrealizedPnL,
		})
	}
	return rows
}
