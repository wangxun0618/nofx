package providers

import (
	"fmt"
	"sort"

	"nofx/provider/hyperliquid"
)

// annualizeFunding converts Hyperliquid's hourly funding rate into an annualised
// percentage. A per-hour decimal is hard to compare across instruments; the
// annualised form makes "who is paying to hold this position" immediately legible.
func annualizeFunding(hourly float64) float64 {
	return hourly * 24 * 365 * 100
}

// baselineFundingNote calibrates the funding column for the model.
//
// Hyperliquid's funding formula includes a fixed base component (0.01% per 8h),
// which annualises to about +11%. Most majors therefore print ~+11.0% day after
// day with no crowding at all. Without this warning the model reads the baseline
// as a strong long-crowding signal — the single most misleading number in the feed.
const baselineFundingNote = "Note on funding: Hyperliquid anchors funding at roughly +11% annualised, so a reading near +11.0% is the base rate rather than crowding. Only the deviations from it are informative — well above +11% means longs are paying up, negative means shorts are.\n"

// formatUsd renders a USD amount compactly (1.23B / 456.7M / 12.3K).
func formatUsd(v float64) string {
	abs := v
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= 1e9:
		return fmt.Sprintf("%.2fB", v/1e9)
	case abs >= 1e6:
		return fmt.Sprintf("%.1fM", v/1e6)
	case abs >= 1e3:
		return fmt.Sprintf("%.1fK", v/1e3)
	default:
		return fmt.Sprintf("%.2f", v)
	}
}

// displaySymbol labels equities-board instruments so the model never confuses
// them with crypto pairs of a similar ticker.
func displaySymbol(c hyperliquid.CoinInfo) string {
	if c.Dex == "xyz" {
		return c.Symbol + " (eq)"
	}
	return c.Symbol
}

// rankBy returns up to limit instruments ordered by less. The input slice is not
// mutated, so several rankings can be derived from a single board fetch.
func rankBy(coins []hyperliquid.CoinInfo, limit int, less func(a, b hyperliquid.CoinInfo) bool) []hyperliquid.CoinInfo {
	out := make([]hyperliquid.CoinInfo, len(coins))
	copy(out, coins)
	sort.SliceStable(out, func(i, j int) bool { return less(out[i], out[j]) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
