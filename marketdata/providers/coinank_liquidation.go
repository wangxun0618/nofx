package providers

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"nofx/logger"
	"nofx/marketdata"
	"nofx/provider/coinank"
	"nofx/store"
)

// coinankBaseURL is the CoinAnk open-api host, matching coinank_api.MainApiUrl.
const coinankBaseURL = "https://api.coinank.com"

// LiqBin is realised liquidation notional for one price bucket.
type LiqBin struct {
	StartPrice float64 `json:"bucket_start_price"`
	EndPrice   float64 `json:"bucket_end_price"`
	LongLiq    float64 `json:"long_liq_usd"`
	ShortLiq   float64 `json:"short_liq_usd"`
}

// LiqSymbolClusters groups the buckets of one instrument.
type LiqSymbolClusters struct {
	Symbol string   `json:"symbol"`
	Bins   []LiqBin `json:"bins"`
}

// LiquidationPayload is the structured liquidation-cluster board.
type LiquidationPayload struct {
	Symbols []LiqSymbolClusters `json:"symbols"`
}

// CoinankLiquidationProvider adds real price-bucketed liquidation clusters.
//
// The built-in Hyperliquid sources approximate liquidation pressure from open
// interest and funding, because the free feed does not publish per-price
// liquidation size. CoinAnk does, so this provider closes that gap — at the cost
// of a third-party account. It reports itself disabled without a key, which keeps
// the default deployment dependency-free.
type CoinankLiquidationProvider struct{}

// NewCoinankLiquidationProvider builds the optional liquidation source.
func NewCoinankLiquidationProvider() *CoinankLiquidationProvider {
	return &CoinankLiquidationProvider{}
}

// Name implements marketdata.Provider.
func (p *CoinankLiquidationProvider) Name() string { return "coinank_liquidation" }

// Description implements marketdata.Provider.
func (p *CoinankLiquidationProvider) Description() string {
	return "Price-bucketed liquidation clusters, rebuilt from realised liquidation orders. Optional: requires a CoinAnk API key."
}

// RequiresAPIKey reports true: CoinAnk needs a registered key.
func (p *CoinankLiquidationProvider) RequiresAPIKey() bool { return true }

// Enabled reports true only when the strategy carries a CoinAnk key.
func (p *CoinankLiquidationProvider) Enabled(cfg store.IndicatorConfig) bool {
	return strings.TrimSpace(cfg.CoinankAPIKey) != ""
}

// Fetch implements marketdata.Provider.
func (p *CoinankLiquidationProvider) Fetch(ctx context.Context, req marketdata.Request, cfg store.IndicatorConfig) (*marketdata.Insight, error) {
	key := strings.TrimSpace(cfg.CoinankAPIKey)
	if key == "" {
		return nil, nil
	}

	targets := coinankTargets(req)
	if len(targets) == 0 {
		return nil, nil
	}

	client := coinank.NewCoinankClient(coinankBaseURL, key)

	var (
		sections []string
		payload  []LiqSymbolClusters
		failures int
	)

	for _, symbol := range targets {
		orders, err := client.LiquidationOrders(ctx, symbol, "", "", 0, 0)
		if err != nil {
			// A missing instrument on CoinAnk is expected for Hyperliquid-only
			// equities; log at info level and keep going.
			logger.Infof("⏭️  CoinAnk liquidations unavailable for %s: %v", symbol, err)
			failures++
			continue
		}
		bins := bucketLiquidations(orders)
		if len(bins) == 0 {
			continue
		}
		payload = append(payload, LiqSymbolClusters{Symbol: symbol, Bins: bins})
		sections = append(sections, renderLiquidationBins(symbol, bins))
	}

	if len(sections) == 0 {
		if failures > 0 {
			return nil, fmt.Errorf("coinank returned no usable liquidation data for %d instrument(s)", failures)
		}
		return nil, nil
	}

	var sb strings.Builder
	sb.WriteString("## Liquidation Clusters (CoinAnk)\n\n")
	sb.WriteString("Realised liquidations bucketed by price. The heaviest zones mark where forced selling (long liq) or forced buying (short liq) has concentrated — a nearby cluster is a magnet, a cleared one is a vacuum.\n\n")
	for _, section := range sections {
		sb.WriteString(section)
	}

	return &marketdata.Insight{
		Provider: p.Name(),
		Title:    "Liquidation clusters",
		Markdown: sb.String(),
		Payload:  LiquidationPayload{Symbols: payload},
	}, nil
}

const (
	// maxLiquidationSymbols caps how many instruments are queried, because each
	// one costs a separate upstream call inside the provider timeout.
	maxLiquidationSymbols = 3
	// liquidationBucketCount is the price resolution of the cluster histogram.
	liquidationBucketCount = 12
)

// coinankTargets picks the instruments worth querying: open positions first (they
// carry live risk), then candidates, de-duplicated and capped.
func coinankTargets(req marketdata.Request) []string {
	ordered := append([]string{}, req.Positions...)
	ordered = append(ordered, req.Symbols...)

	seen := make(map[string]bool, len(ordered))
	targets := make([]string, 0, maxLiquidationSymbols)
	for _, symbol := range ordered {
		symbol = strings.TrimSpace(symbol)
		if symbol == "" {
			continue
		}
		// CoinAnk keys instruments by the plain base coin; skip the xyz: prefix
		// form outright rather than querying with an unknown contract code.
		if idx := strings.LastIndex(symbol, ":"); idx >= 0 {
			symbol = symbol[idx+1:]
		}
		symbol = strings.ToUpper(symbol)
		if seen[symbol] {
			continue
		}
		seen[symbol] = true
		targets = append(targets, symbol)
		if len(targets) >= maxLiquidationSymbols {
			break
		}
	}
	return targets
}

// bucketLiquidations turns individual liquidation orders into a price histogram,
// which is the shape the model reasons about best.
func bucketLiquidations(orders []coinank.LiquidationOrdersResponse) []LiqBin {
	usable := make([]coinank.LiquidationOrdersResponse, 0, len(orders))
	low, high := 0.0, 0.0

	for _, order := range orders {
		if order.Price <= 0 || order.TradeTurnover <= 0 {
			continue
		}
		usable = append(usable, order)
		if len(usable) == 1 {
			low, high = order.Price, order.Price
			continue
		}
		if order.Price < low {
			low = order.Price
		}
		if order.Price > high {
			high = order.Price
		}
	}
	if len(usable) == 0 || high <= low {
		return nil
	}

	step := (high - low) / liquidationBucketCount
	bins := make([]LiqBin, liquidationBucketCount)
	for i := range bins {
		bins[i] = LiqBin{StartPrice: low + float64(i)*step, EndPrice: low + float64(i+1)*step}
	}

	for _, order := range usable {
		index := int((order.Price - low) / step)
		switch {
		case index < 0:
			index = 0
		case index >= liquidationBucketCount:
			index = liquidationBucketCount - 1
		}
		switch strings.ToLower(order.PosSide) {
		case "long":
			bins[index].LongLiq += order.TradeTurnover
		case "short":
			bins[index].ShortLiq += order.TradeTurnover
		}
	}

	// Heaviest zones first: the model does not need the price-ordered form.
	sort.SliceStable(bins, func(i, j int) bool {
		return bins[i].LongLiq+bins[i].ShortLiq > bins[j].LongLiq+bins[j].ShortLiq
	})
	return bins
}

// renderLiquidationBins formats one instrument's clusters as a markdown table.
func renderLiquidationBins(symbol string, bins []LiqBin) string {
	shown := bins
	if len(shown) > 5 {
		shown = shown[:5]
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### %s\n\n", symbol))
	sb.WriteString("| Price zone | Long liq | Short liq | Dominant |\n")
	sb.WriteString("| :--- | ---: | ---: | :--- |\n")
	for _, bin := range shown {
		sb.WriteString(fmt.Sprintf("| %.6g – %.6g | $%s | $%s | %s |\n",
			bin.StartPrice, bin.EndPrice, formatUsd(bin.LongLiq), formatUsd(bin.ShortLiq), dominantSide(bin)))
	}
	sb.WriteString("\n")
	return sb.String()
}

// dominantSide names whichever liquidation side carries more notional in a bucket.
func dominantSide(bin LiqBin) string {
	switch {
	case bin.LongLiq == 0 && bin.ShortLiq == 0:
		return "none"
	case bin.LongLiq >= bin.ShortLiq:
		return "long liq"
	default:
		return "short liq"
	}
}
