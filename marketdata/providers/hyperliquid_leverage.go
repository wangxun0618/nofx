package providers

import (
	"context"
	"fmt"
	"strings"

	"nofx/marketdata"
	"nofx/provider/hyperliquid"
	"nofx/store"
)

// LeverageRow is the structured form of one instrument's position structure.
type LeverageRow struct {
	Symbol          string  `json:"symbol"`
	Dex             string  `json:"dex,omitempty"`
	OpenInterestUsd float64 `json:"open_interest_usd"`
	Volume24h       float64 `json:"volume_24h"`
	MaxLeverage     int     `json:"max_leverage"`
	FundingAnnual   float64 `json:"funding_annual_pct"`
	PremiumPct      float64 `json:"premium_pct"`
}

// LeveragePayload is the structured position-structure board.
type LeveragePayload struct {
	LargestOpenInterest []LeverageRow `json:"largest_open_interest"`
	LongCrowded         []LeverageRow `json:"long_crowded"`
	ShortCrowded        []LeverageRow `json:"short_crowded"`
	Candidates          []LeverageRow `json:"candidates"`
	UniverseSize        int           `json:"universe_size"`
}

// HyperliquidLeverageProvider maps where positions are large and which side is
// over-paying to hold them.
//
// Scope note, stated plainly because it matters for how the model should use it:
// the free Hyperliquid feed publishes aggregate open interest and funding, not a
// price-bucketed liquidation order book. This provider therefore approximates
// liquidation *pressure* from position size, funding cost and mark-vs-oracle
// dislocation. For true per-price liquidation clusters, enable the optional
// CoinAnk provider.
type HyperliquidLeverageProvider struct{}

// NewHyperliquidLeverageProvider builds the position-structure source.
func NewHyperliquidLeverageProvider() *HyperliquidLeverageProvider {
	return &HyperliquidLeverageProvider{}
}

// Name implements marketdata.Provider.
func (p *HyperliquidLeverageProvider) Name() string { return "hyperliquid_leverage" }

// Description implements marketdata.Provider.
func (p *HyperliquidLeverageProvider) Description() string {
	return "Open-interest concentration, funding cost and mark-vs-oracle dislocation — a free approximation of liquidation pressure."
}

// RequiresAPIKey reports false: the Hyperliquid info endpoint is public.
func (p *HyperliquidLeverageProvider) RequiresAPIKey() bool { return false }

// Enabled implements marketdata.Provider. Public and unmetered, like the flow source.
func (p *HyperliquidLeverageProvider) Enabled(store.IndicatorConfig) bool { return true }

// Fetch implements marketdata.Provider.
func (p *HyperliquidLeverageProvider) Fetch(ctx context.Context, req marketdata.Request, _ store.IndicatorConfig) (*marketdata.Insight, error) {
	board, err := hyperliquid.GetAssetContexts(ctx)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid asset contexts: %w", err)
	}
	if len(board) == 0 {
		return nil, nil
	}

	universe := rankBy(board, leverageUniverseSize, func(a, b hyperliquid.CoinInfo) bool {
		return a.Volume24h > b.Volume24h
	})

	limit := req.NormalizeLimit(defaultRowLimit)
	briefLimit := leverageBriefRows
	if limit < briefLimit {
		briefLimit = limit
	}

	largestOI := rankBy(universe, limit, func(a, b hyperliquid.CoinInfo) bool {
		return a.OpenInterestUsd > b.OpenInterestUsd
	})
	longCrowded := rankByWithPositiveFunding(universe, briefLimit, true)
	shortCrowded := rankByWithPositiveFunding(universe, briefLimit, false)

	candidates := candidatesFromBoard(board, req.Symbols)

	var sb strings.Builder
	sb.WriteString("## Open-Interest Structure & Leverage Crowding (Hyperliquid)\n\n")
	sb.WriteString("This is a leverage map, not a liquidation order book. Read it as \"where positions are large and who is over-paying to hold them\": high absolute funding plus a mark price away from the oracle marks a stretched side. Publish-free data does not include per-price liquidation size.\n\n")
	sb.WriteString(baselineFundingNote)
	sb.WriteString("\n")

	sb.WriteString("### Largest open interest\n\n")
	sb.WriteString("| # | Instrument | Open interest | 24h volume | Max lev | Funding (ann.) | Mark vs oracle |\n")
	sb.WriteString("| -: | :--- | ---: | ---: | ---: | ---: | ---: |\n")
	for i, c := range largestOI {
		sb.WriteString(fmt.Sprintf("| %d | %s | $%s | $%s | %dx | %+.1f%% | %+.3f%% |\n",
			i+1, displaySymbol(c), formatUsd(c.OpenInterestUsd), formatUsd(c.Volume24h),
			c.MaxLeverage, annualizeFunding(c.FundingRate), c.Premium*100))
	}
	sb.WriteString("\n")

	sb.WriteString("### Stretched sides (annualised funding)\n\n")
	sb.WriteString("- Longs over-paying: " + joinBriefs(longCrowded, fundingBrief) + "\n")
	sb.WriteString("- Shorts over-paying: " + joinBriefs(shortCrowded, fundingBrief) + "\n")

	if len(candidates) > 0 {
		sb.WriteString("\n### Your candidate instruments\n\n")
		sb.WriteString("| Instrument | Open interest | 24h volume | Funding (ann.) | Mark vs oracle |\n")
		sb.WriteString("| :--- | ---: | ---: | ---: | ---: |\n")
		for _, c := range candidates {
			sb.WriteString(fmt.Sprintf("| %s | $%s | $%s | %+.1f%% | %+.3f%% |\n",
				displaySymbol(c), formatUsd(c.OpenInterestUsd), formatUsd(c.Volume24h),
				annualizeFunding(c.FundingRate), c.Premium*100))
		}
	}

	return &marketdata.Insight{
		Provider: p.Name(),
		Title:    "Open-interest structure",
		Markdown: sb.String(),
		Payload: LeveragePayload{
			LargestOpenInterest: toLeverageRows(largestOI),
			LongCrowded:         toLeverageRows(longCrowded),
			ShortCrowded:        toLeverageRows(shortCrowded),
			Candidates:          toLeverageRows(candidates),
			UniverseSize:        len(board),
		},
	}, nil
}

const (
	// leverageUniverseSize bounds the board before ranking.
	leverageUniverseSize = 120
	// leverageBriefRows is how many instruments the one-line lists show.
	leverageBriefRows = 5
	// maxCandidateRows caps the per-candidate table so a large candidate pool
	// cannot bloat the prompt.
	maxCandidateRows = 12
)

// candidatesFromBoard resolves the requested symbols against the live board,
// preserving request order and dropping duplicates.
func candidatesFromBoard(board []hyperliquid.CoinInfo, symbols []string) []hyperliquid.CoinInfo {
	rows := make([]hyperliquid.CoinInfo, 0, len(symbols))
	seen := make(map[string]bool, len(symbols))

	for _, want := range symbols {
		coin, ok := findInBoard(board, want)
		if !ok || seen[coin.Symbol] {
			continue
		}
		seen[coin.Symbol] = true
		rows = append(rows, coin)
		if len(rows) >= maxCandidateRows {
			break
		}
	}
	return rows
}

// findInBoard matches a caller-supplied symbol against the board, accepting the
// raw board name, the normalized form and the xyz alias form.
func findInBoard(board []hyperliquid.CoinInfo, symbol string) (hyperliquid.CoinInfo, bool) {
	want := hyperliquid.NormalizeCoin(symbol)
	wantBase := hyperliquid.NormalizeCoinBase(symbol)

	for _, c := range board {
		if c.Symbol == symbol || hyperliquid.NormalizeCoin(c.Symbol) == want {
			return c, true
		}
		if wantBase != "" && hyperliquid.NormalizeCoinBase(c.Symbol) == wantBase {
			return c, true
		}
	}
	return hyperliquid.CoinInfo{}, false
}

// toLeverageRows converts board entries into the API payload shape.
func toLeverageRows(coins []hyperliquid.CoinInfo) []LeverageRow {
	rows := make([]LeverageRow, 0, len(coins))
	for _, c := range coins {
		rows = append(rows, LeverageRow{
			Symbol:          c.Symbol,
			Dex:             c.Dex,
			OpenInterestUsd: c.OpenInterestUsd,
			Volume24h:       c.Volume24h,
			MaxLeverage:     c.MaxLeverage,
			FundingAnnual:   annualizeFunding(c.FundingRate),
			PremiumPct:      c.Premium * 100,
		})
	}
	return rows
}
