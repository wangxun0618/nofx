package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"nofx/logger"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	hyperliquidInfoURL = "https://api.hyperliquid.xyz/info"
	cacheDuration      = 24 * time.Hour // Cache for 24 hours
)

// CoinInfo represents basic Hyperliquid market information.
type CoinInfo struct {
	Symbol       string  `json:"symbol"`
	Volume24h    float64 `json:"volume_24h"` // 24h notional volume in USD
	MarkPrice    float64 `json:"mark_price"`
	PrevDayPrice float64 `json:"prev_day_price,omitempty"`
	Change24hPct float64 `json:"change_24h_pct,omitempty"`
	MaxLeverage  int     `json:"max_leverage,omitempty"`
	SzDecimals   int     `json:"sz_decimals,omitempty"`
	Dex          string  `json:"dex,omitempty"` // perp dex the asset belongs to ("" = main, "xyz" = equities board)

	// Derivatives context, sourced from the same free `metaAndAssetCtxs` call.
	// These power the market-insight providers and are also shown in the
	// symbol picker, so they are part of the public payload.
	FundingRate     float64 `json:"funding_rate,omitempty"`      // current hourly funding rate (decimal, e.g. 0.0000125)
	OpenInterest    float64 `json:"open_interest,omitempty"`     // open interest in coin units
	OpenInterestUsd float64 `json:"open_interest_usd,omitempty"` // open interest notional in USD
	Premium         float64 `json:"premium,omitempty"`           // mark price premium over oracle price (decimal)
	OraclePrice     float64 `json:"oracle_price,omitempty"`
	MidPrice        float64 `json:"mid_price,omitempty"`
}

// XYZCategory returns the NOFX product category for a Hyperliquid XYZ base symbol.
func XYZCategory(baseSymbol string) string {
	baseSymbol = strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(baseSymbol, "xyz:")))
	switch baseSymbol {
	case "TSLA", "NVDA", "AAPL", "MSFT", "GOOGL", "GOOG", "AMZN", "META", "NFLX", "AMD", "INTC", "COIN", "MSTR", "PLTR", "HOOD", "CRCL", "SNDK", "MU", "SMSN", "DRAM", "SKHX", "BABA", "ASML", "AVGO", "IONQ", "RGTI", "RKLB", "SMCI", "MARA", "RIOT", "MRVL", "SNOW", "CRM", "ORCL", "ADBE", "PYPL", "SHOP", "UBER", "SPOT", "ABNB", "RDDT", "ARM", "SOFI", "XYZ", "LVMH", "PDD", "NVO", "SONY", "DIS", "WMT", "NKE", "JPM", "BAC", "V", "MA", "JNJ", "PG", "UNH", "HD", "XOM", "CVX", "TM", "RACE", "VOW3", "BMW", "MBG":
		return "stock"
	case "GOLD", "SILVER", "COPPER", "NATGAS", "URANIUM", "ALUMINIUM", "PLATINUM", "PALLADIUM", "BRENTOIL", "CL", "CORN", "WHEAT", "TTF":
		return "commodity"
	case "SPX", "NDX", "DJI", "VIX", "DAX", "FTSE", "NIKKEI", "HSI", "CSI300", "XYZ100", "XYZ25", "XYZ50":
		return "index"
	case "EUR", "GBP", "JPY", "AUD", "CAD", "CHF", "MXN", "BRL", "TRY", "ZAR", "CNH", "KRW":
		return "forex"
	case "OPENAI", "ANTHROPIC", "SPACEX", "STRIPE", "FIGMA", "DATBRICKS", "PERPLEXITY", "XAI", "BYTEDANCE", "REVOLUT":
		return "pre_ipo"
	default:
		return "stock"
	}
}

// CoinProvider provides Hyperliquid coin lists
type CoinProvider struct {
	mu          sync.RWMutex
	allCoins    []CoinInfo
	mainCoins   []CoinInfo
	lastUpdated time.Time
	httpClient  *http.Client
}

var (
	defaultProvider *CoinProvider
	providerOnce    sync.Once
)

// GetProvider returns the singleton CoinProvider instance
func GetProvider() *CoinProvider {
	providerOnce.Do(func() {
		defaultProvider = &CoinProvider{
			httpClient: &http.Client{Timeout: 30 * time.Second},
		}
	})
	return defaultProvider
}

// metaResponse represents the response from Hyperliquid meta endpoint
type metaResponse struct {
	Universe []struct {
		Name        string `json:"name"`
		SzDecimals  int    `json:"szDecimals"`
		MaxLeverage int    `json:"maxLeverage"`
	} `json:"universe"`
}

// assetCtx represents asset context with market data. Field set mirrors the
// free `metaAndAssetCtxs` response; everything declared here is available
// without an API key, so nothing that the insight providers need is dropped.
type assetCtx struct {
	Funding      string `json:"funding"`      // hourly funding rate (decimal, e.g. 0.0000125)
	OpenInterest string `json:"openInterest"` // open interest in coin units
	DayNtlVlm    string `json:"dayNtlVlm"`    // 24h notional volume in USD
	DayBaseVlm   string `json:"dayBaseVlm"`   // 24h base volume in coin units
	MarkPx       string `json:"markPx"`
	MidPx        string `json:"midPx"`
	OraclePx     string `json:"oraclePx"`
	PrevDayPx    string `json:"prevDayPx"`
	Premium      string `json:"premium"` // mark premium over oracle (decimal)
}

func fetchPerpDexCoins(ctx context.Context, client *http.Client, dex string) ([]CoinInfo, error) {
	reqPayload := map[string]string{"type": "metaAndAssetCtxs"}
	if dex != "" {
		reqPayload["dex"] = dex
	}
	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", hyperliquidInfoURL,
		bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch coin data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Response is an array: [meta, [assetCtxs...]]
	var rawResp []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(rawResp) < 2 {
		return nil, fmt.Errorf("unexpected response format")
	}

	var meta metaResponse
	if err := json.Unmarshal(rawResp[0], &meta); err != nil {
		return nil, fmt.Errorf("failed to parse meta: %w", err)
	}

	var ctxs []assetCtx
	if err := json.Unmarshal(rawResp[1], &ctxs); err != nil {
		return nil, fmt.Errorf("failed to parse asset contexts: %w", err)
	}

	coins := make([]CoinInfo, 0, len(meta.Universe))
	for i, u := range meta.Universe {
		var vol, mark, prevDay, change24hPct float64
		var funding, oi, oiUsd, premium, oraclePx, midPx float64
		if i < len(ctxs) {
			vol, _ = strconv.ParseFloat(ctxs[i].DayNtlVlm, 64)
			mark, _ = strconv.ParseFloat(ctxs[i].MarkPx, 64)
			prevDay, _ = strconv.ParseFloat(ctxs[i].PrevDayPx, 64)
			funding, _ = strconv.ParseFloat(ctxs[i].Funding, 64)
			oi, _ = strconv.ParseFloat(ctxs[i].OpenInterest, 64)
			premium, _ = strconv.ParseFloat(ctxs[i].Premium, 64)
			oraclePx, _ = strconv.ParseFloat(ctxs[i].OraclePx, 64)
			midPx, _ = strconv.ParseFloat(ctxs[i].MidPx, 64)
			// `openInterest` is denominated in coin units; convert to a USD
			// notional so different assets can be ranked on one scale.
			if oi > 0 && mark > 0 {
				oiUsd = oi * mark
			}
			if prevDay > 0 && mark > 0 {
				change24hPct = ((mark - prevDay) / prevDay) * 100
			}
		}
		coins = append(coins, CoinInfo{
			Symbol:          u.Name,
			Volume24h:       vol,
			MarkPrice:       mark,
			PrevDayPrice:    prevDay,
			Change24hPct:    change24hPct,
			MaxLeverage:     u.MaxLeverage,
			SzDecimals:      u.SzDecimals,
			Dex:             dex,
			FundingRate:     funding,
			OpenInterest:    oi,
			OpenInterestUsd: oiUsd,
			Premium:         premium,
			OraclePrice:     oraclePx,
			MidPrice:        midPx,
		})
	}

	sort.Slice(coins, func(i, j int) bool {
		return coins[i].Volume24h > coins[j].Volume24h
	})
	return coins, nil
}

// perpDexCacheTTL bounds how often the perp-dex symbol board is re-fetched.
// The tradable symbol list changes rarely; prices/volume on the board are
// display hints, so short staleness is far better than hammering the
// Hyperliquid API (which rate-limits with 429) on every panel render.
const perpDexCacheTTL = 5 * time.Minute

type perpDexCacheEntry struct {
	coins     []CoinInfo
	fetchedAt time.Time
}

type perpDexCacheStore struct {
	mu      sync.Mutex
	entries map[string]perpDexCacheEntry
}

var perpDexCoinCache = &perpDexCacheStore{entries: map[string]perpDexCacheEntry{}}

// fetchPerpDexCoinsFn is swappable in tests.
var fetchPerpDexCoinsFn = fetchPerpDexCoins

// GetPerpDexCoins returns current tradable USDC perp assets for a given
// Hyperliquid dex, served from a TTL cache. When the upstream fetch fails
// (e.g. HTTP 429 rate limiting) and stale data exists, the stale board is
// served instead of an error so the UI keeps working.
func GetPerpDexCoins(ctx context.Context, dex string) ([]CoinInfo, error) {
	perpDexCoinCache.mu.Lock()
	defer perpDexCoinCache.mu.Unlock()

	entry, hasCache := perpDexCoinCache.entries[dex]
	if hasCache && time.Since(entry.fetchedAt) < perpDexCacheTTL {
		return copyCoins(entry.coins), nil
	}

	coins, err := fetchPerpDexCoinsFn(ctx, &http.Client{Timeout: 30 * time.Second}, dex)
	if err != nil {
		if hasCache {
			logger.Infof("⚠️ Hyperliquid perp-dex fetch failed (%v); serving cached board for dex %q from %s",
				err, dex, entry.fetchedAt.Format(time.RFC3339))
			return copyCoins(entry.coins), nil
		}
		return nil, err
	}

	perpDexCoinCache.entries[dex] = perpDexCacheEntry{coins: coins, fetchedAt: time.Now()}
	return copyCoins(coins), nil
}

// copyCoins returns a defensive copy so callers cannot mutate the cache.
func copyCoins(coins []CoinInfo) []CoinInfo {
	out := make([]CoinInfo, len(coins))
	copy(out, coins)
	return out
}

// fetchCoins fetches all default Hyperliquid crypto coins and sorts by volume
func (p *CoinProvider) fetchCoins(ctx context.Context) error {
	coins, err := fetchPerpDexCoins(ctx, p.httpClient, "")
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.allCoins = coins
	// Main coins are top 20 by volume
	if len(coins) > 20 {
		p.mainCoins = coins[:20]
	} else {
		p.mainCoins = coins
	}
	p.lastUpdated = time.Now()

	logger.Infof("✅ Hyperliquid coin list updated: %d total coins, top 20 by volume cached", len(coins))

	return nil
}

// ensureUpdated checks if cache is stale and refreshes if needed
func (p *CoinProvider) ensureUpdated(ctx context.Context) error {
	p.mu.RLock()
	needsUpdate := time.Since(p.lastUpdated) > cacheDuration || len(p.allCoins) == 0
	p.mu.RUnlock()

	if needsUpdate {
		return p.fetchCoins(ctx)
	}
	return nil
}

// GetAllCoins returns all available Hyperliquid perp coins
func (p *CoinProvider) GetAllCoins(ctx context.Context) ([]CoinInfo, error) {
	if err := p.ensureUpdated(ctx); err != nil {
		return nil, err
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return a copy to avoid mutation
	result := make([]CoinInfo, len(p.allCoins))
	copy(result, p.allCoins)
	return result, nil
}

// GetMainCoins returns top N coins by 24h volume
func (p *CoinProvider) GetMainCoins(ctx context.Context, limit int) ([]CoinInfo, error) {
	if err := p.ensureUpdated(ctx); err != nil {
		return nil, err
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if limit <= 0 {
		limit = 20
	}

	// Return top N coins
	count := limit
	if count > len(p.allCoins) {
		count = len(p.allCoins)
	}

	result := make([]CoinInfo, count)
	copy(result, p.allCoins[:count])
	return result, nil
}

// GetCoinSymbols returns just the symbol names (for compatibility)
func GetAllCoinSymbols(ctx context.Context) ([]string, error) {
	coins, err := GetProvider().GetAllCoins(ctx)
	if err != nil {
		return nil, err
	}

	symbols := make([]string, len(coins))
	for i, c := range coins {
		symbols[i] = c.Symbol
	}
	return symbols, nil
}

// GetMainCoinSymbols returns top N coin symbols by volume
func GetMainCoinSymbols(ctx context.Context, limit int) ([]string, error) {
	coins, err := GetProvider().GetMainCoins(ctx, limit)
	if err != nil {
		return nil, err
	}

	symbols := make([]string, len(coins))
	for i, c := range coins {
		symbols[i] = c.Symbol
	}
	return symbols, nil
}

// ForceRefresh forces a refresh of the coin cache
func (p *CoinProvider) ForceRefresh(ctx context.Context) error {
	return p.fetchCoins(ctx)
}

// xyzPerpDex is the Hyperliquid perp dex that hosts tokenized equities and other
// non-crypto instruments. The main crypto dex is addressed with an empty string.
const xyzPerpDex = "xyz"

// GetAssetContexts returns every tradable Hyperliquid perp — main crypto dex plus
// the "xyz" equities dex — together with its derivatives context: funding rate,
// open interest (coin units and USD notional) and mark-vs-oracle premium.
//
// Everything here comes from the free `metaAndAssetCtxs` endpoint, so no API key
// is involved. The result is volume-sorted and served through the same short TTL
// cache as GetPerpDexCoins, which means a rate-limited upstream degrades to stale
// data rather than to an error.
func GetAssetContexts(ctx context.Context) ([]CoinInfo, error) {
	merged := make([]CoinInfo, 0, 256)
	var firstErr error

	for _, dex := range []string{"", xyzPerpDex} {
		coins, err := GetPerpDexCoins(ctx, dex)
		if err != nil {
			// One dex failing must not hide the other; remember the first error
			// and only surface it when nothing at all came back.
			if firstErr == nil {
				firstErr = err
			}
			logger.Infof("⚠️ Hyperliquid asset contexts unavailable for dex %q: %v", dex, err)
			continue
		}
		merged = append(merged, coins...)
	}

	if len(merged) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, fmt.Errorf("no asset contexts returned")
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Volume24h > merged[j].Volume24h
	})
	return merged, nil
}

// FindAssetContext returns the derivatives context for a single symbol, matching
// on the raw board name, the normalized symbol and the xyz alias form so callers
// can pass whatever the rest of the system uses.
func FindAssetContext(ctx context.Context, symbol string) (*CoinInfo, error) {
	contexts, err := GetAssetContexts(ctx)
	if err != nil {
		return nil, err
	}

	want := NormalizeCoin(symbol)
	wantBase := NormalizeCoinBase(symbol)

	for i := range contexts {
		coin := &contexts[i]
		if coin.Symbol == symbol || NormalizeCoin(coin.Symbol) == want {
			return coin, nil
		}
		if wantBase != "" && NormalizeCoinBase(coin.Symbol) == wantBase {
			return coin, nil
		}
	}
	return nil, nil
}
