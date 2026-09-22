package providers

import (
	"context"
	"fmt"
	"strings"

	"nofx/marketdata"
	"nofx/provider/hyperliquid"
	"nofx/store"
)

// FlowRow is the structured form of one instrument's flow contribution. The HTTP
// API and the web terminal consume it directly.
type FlowRow struct {
	Symbol          string  `json:"symbol"`
	Dex             string  `json:"dex,omitempty"`
	Volume24h       float64 `json:"volume_24h"`
	Change24hPct    float64 `json:"change_24h_pct"`
	OpenInterestUsd float64 `json:"open_interest_usd"`
	FundingRate1h   float64 `json:"funding_rate_1h"`
	FundingAnnual   float64 `json:"funding_annual_pct"`
}

// FlowPayload is the structured cross-market flow board.
type FlowPayload struct {
	MostTraded       []FlowRow `json:"most_traded"`
	MostCrowdedLong  []FlowRow `json:"most_crowded_long"`
	MostCrowdedShort []FlowRow `json:"most_crowded_short"`
	Gainers          []FlowRow `json:"gainers"`
	Losers           []FlowRow `json:"losers"`
	UniverseSize     int       `json:"universe_size"`
}

// HyperliquidFlowProvider surfaces cross-instrument capital flow on Hyperliquid:
// where 24h turnover is concentrated, which side is paying funding to hold, and
// what moved. It is the free replacement for the retired paid flow rankings and
// requires no API key.
type HyperliquidFlowProvider struct{}

// NewHyperliquidFlowProvider builds the cross-market flow source.
func NewHyperliquidFlowProvider() *HyperliquidFlowProvider {
	return &HyperliquidFlowProvider{}
}

// Name implements marketdata.Provider.
func (p *HyperliquidFlowProvider) Name() string { return "hyperliquid_flow" }

// Description implements marketdata.Provider.
func (p *HyperliquidFlowProvider) Description() string {
	return "Cross-instrument fund flow: 24h turnover concentration, movers and funding-rate crowding. Free and keyless."
}

// RequiresAPIKey reports false: the Hyperliquid info endpoint is public.
func (p *HyperliquidFlowProvider) RequiresAPIKey() bool { return false }

// Enabled always reports true. The underlying Hyperliquid endpoint is public and
// unmetered, so usage is governed centrally by enable_market_insights plus the
// source allow-list rather than by per-source credentials.
func (p *HyperliquidFlowProvider) Enabled(store.IndicatorConfig) bool { return true }

// Fetch implements marketdata.Provider.
func (p *HyperliquidFlowProvider) Fetch(ctx context.Context, req marketdata.Request, _ store.IndicatorConfig) (*marketdata.Insight, error) {
	board, err := hyperliquid.GetAssetContexts(ctx)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid asset contexts: %w", err)
	}
	if len(board) == 0 {
		return nil, nil
	}

	// Restrict every ranking to instruments that actually trade: an exotic pair
	// with a wild funding print is noise, not information.
	universe := rankBy(board, flowUniverseSize, func(a, b hyperliquid.CoinInfo) bool {
		return a.Volume24h > b.Volume24h
	})

	limit := req.NormalizeLimit(defaultRowLimit)
	briefLimit := flowBriefRows
	if limit < briefLimit {
		briefLimit = limit
	}

	mostTraded := rankBy(universe, limit, func(a, b hyperliquid.CoinInfo) bool {
		return a.Volume24h > b.Volume24h
	})
	crowdedLong := rankByWithPositiveFunding(universe, briefLimit, true)
	crowdedShort := rankByWithPositiveFunding(universe, briefLimit, false)
	gainers := rankBy(universe, briefLimit, func(a, b hyperliquid.CoinInfo) bool {
		return a.Change24hPct > b.Change24hPct
	})
	losers := rankBy(universe, briefLimit, func(a, b hyperliquid.CoinInfo) bool {
		return a.Change24hPct < b.Change24hPct
	})

	var sb strings.Builder
	sb.WriteString("## Cross-Market Fund Flow (Hyperliquid, 24h)\n\n")
	sb.WriteString("Where capital is active and which side is paying funding to hold. Treat this as a liquidity and crowding map, not as a standalone direction signal.\n\n")
	sb.WriteString(baselineFundingNote)
	sb.WriteString("\n")

	sb.WriteString("### Highest turnover\n\n")
	sb.WriteString("| # | Instrument | 24h volume | 24h chg | Open interest | Funding (ann.) |\n")
	sb.WriteString("| -: | :--- | ---: | ---: | ---: | ---: |\n")
	for i, c := range mostTraded {
		sb.WriteString(fmt.Sprintf("| %d | %s | $%s | %+.2f%% | $%s | %+.1f%% |\n",
			i+1, displaySymbol(c), formatUsd(c.Volume24h), c.Change24hPct,
			formatUsd(c.OpenInterestUsd), annualizeFunding(c.FundingRate)))
	}
	sb.WriteString("\n")

	sb.WriteString("### Funding crowding (annualised)\n\n")
	sb.WriteString("- Longs paying most to stay in: " + joinBriefs(crowdedLong, fundingBrief) + "\n")
	sb.WriteString("- Shorts paying most to stay in: " + joinBriefs(crowdedShort, fundingBrief) + "\n\n")

	sb.WriteString("### 24h movers\n\n")
	sb.WriteString("- Up: " + joinBriefs(gainers, changeBrief) + "\n")
	sb.WriteString("- Down: " + joinBriefs(losers, changeBrief) + "\n")

	return &marketdata.Insight{
		Provider: p.Name(),
		Title:    "Cross-market fund flow",
		Markdown: sb.String(),
		Payload: FlowPayload{
			MostTraded:       toFlowRows(mostTraded),
			MostCrowdedLong:  toFlowRows(crowdedLong),
			MostCrowdedShort: toFlowRows(crowdedShort),
			Gainers:          toFlowRows(gainers),
			Losers:           toFlowRows(losers),
			UniverseSize:     len(board),
		},
	}, nil
}

const (
	// flowUniverseSize bounds the board before ranking, so a thin market cannot
	// dominate a crowding or mover list.
	flowUniverseSize = 120
	// flowBriefRows is how many instruments the compact one-line lists show.
	flowBriefRows = 5
)

// rankByWithPositiveFunding returns the instruments paying the most funding on
// the requested side. Positive funding means longs pay shorts, so the "longs are
// crowded" list needs positive rates and the "shorts are crowded" list needs the
// most negative ones — both are ordered by strength of the print.
func rankByWithPositiveFunding(coins []hyperliquid.CoinInfo, limit int, longsCrowded bool) []hyperliquid.CoinInfo {
	candidates := make([]hyperliquid.CoinInfo, 0, len(coins))
	for _, c := range coins {
		if longsCrowded && c.FundingRate > 0 {
			candidates = append(candidates, c)
		}
		if !longsCrowded && c.FundingRate < 0 {
			candidates = append(candidates, c)
		}
	}
	if longsCrowded {
		return rankBy(candidates, limit, func(a, b hyperliquid.CoinInfo) bool {
			return a.FundingRate > b.FundingRate
		})
	}
	return rankBy(candidates, limit, func(a, b hyperliquid.CoinInfo) bool {
		return a.FundingRate < b.FundingRate
	})
}

// fundingBrief renders one instrument as "SYM +12.3%".
func fundingBrief(c hyperliquid.CoinInfo) string {
	return fmt.Sprintf("%+.1f%%", annualizeFunding(c.FundingRate))
}

// changeBrief renders one instrument as "SYM +4.2%".
func changeBrief(c hyperliquid.CoinInfo) string {
	return fmt.Sprintf("%+.2f%%", c.Change24hPct)
}

// joinBriefs renders a compact comma-separated list of instruments.
func joinBriefs(coins []hyperliquid.CoinInfo, value func(hyperliquid.CoinInfo) string) string {
	if len(coins) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(coins))
	for _, c := range coins {
		parts = append(parts, displaySymbol(c)+" "+value(c))
	}
	return strings.Join(parts, ", ")
}

// toFlowRows converts board entries into the API payload shape.
func toFlowRows(coins []hyperliquid.CoinInfo) []FlowRow {
	rows := make([]FlowRow, 0, len(coins))
	for _, c := range coins {
		rows = append(rows, FlowRow{
			Symbol:          c.Symbol,
			Dex:             c.Dex,
			Volume24h:       c.Volume24h,
			Change24hPct:    c.Change24hPct,
			OpenInterestUsd: c.OpenInterestUsd,
			FundingRate1h:   c.FundingRate,
			FundingAnnual:   annualizeFunding(c.FundingRate),
		})
	}
	return rows
}
