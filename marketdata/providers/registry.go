// Package providers ships the built-in marketdata.Provider implementations.
//
// This package is the single wiring point for market intelligence. To add a new
// data source: implement marketdata.Provider in its own file, then append it to
// defaultProviders(). Nothing else in the codebase changes — the strategy engine,
// the trading loop and the prompt builder only ever talk to a marketdata.Registry.
//
// Built-in sources, in prompt order:
//
//	directional_signal     per-instrument bias, strength and change timeline (free)
//	hyperliquid_flow       cross-instrument capital flow (free, no key)
//	hyperliquid_leverage   open-interest structure and crowding (free, no key)
//	hyperdata_orderflow    aggressive order flow via a local sidecar (opt-in)
//	hyperdata_positioning  tracked positions and liquidation distance (opt-in)
//	coinank_liquidation    price-bucketed liquidation clusters (opt-in, needs a key)
//
// The two hyperdata_* sources consume the HyperData Terminal REST API rather
// than a Go library. That is deliberate: it lets the sidecar be upgraded on its
// own schedule, and a sidecar that is down costs only its own blocks. See
// docs/architecture/market-data-providers.md.
package providers

import (
	"sync"

	"nofx/marketdata"
)

// defaultRowLimit bounds how many instruments a ranking source emits when the
// strategy leaves the row count unset.
const defaultRowLimit = 10

var (
	defaultOnce     sync.Once
	defaultRegistry *marketdata.Registry
)

// Default returns the process-wide provider registry. It is a singleton so the
// cached upstream clients are shared by every running trader.
func Default() *marketdata.Registry {
	defaultOnce.Do(func() {
		defaultRegistry = marketdata.NewRegistry(defaultProviders()...)
	})
	return defaultRegistry
}

// defaultProviders is the built-in catalogue, in the order their blocks appear in
// the prompt.
//
// The direction block leads because it is the only source that states a verdict;
// the descriptive sources that follow are what a reader uses to check it.
func defaultProviders() []marketdata.Provider {
	return []marketdata.Provider{
		NewDirectionalSignalProvider(),
		NewHyperliquidFlowProvider(),
		NewHyperliquidLeverageProvider(),
		NewHyperDataOrderflowProvider(),
		NewHyperDataPositioningProvider(),
		NewCoinankLiquidationProvider(),
	}
}
