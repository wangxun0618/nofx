package kernel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"nofx/logger"
	"nofx/market"
	"nofx/marketdata"
	"nofx/marketdata/providers"
	"nofx/provider/hyperliquid"
	"nofx/provider/nofxos"
	"nofx/security"
	"nofx/store"
	"sort"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// Type Definitions
// ============================================================================

// PositionInfo position information
type PositionInfo struct {
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"` // "long" or "short"
	EntryPrice       float64 `json:"entry_price"`
	MarkPrice        float64 `json:"mark_price"`
	Quantity         float64 `json:"quantity"`
	Leverage         int     `json:"leverage"`
	UnrealizedPnL    float64 `json:"unrealized_pnl"`
	UnrealizedPnLPct float64 `json:"unrealized_pnl_pct"`
	PeakPnLPct       float64 `json:"peak_pnl_pct"` // Historical peak profit percentage
	LiquidationPrice float64 `json:"liquidation_price"`
	MarginUsed       float64 `json:"margin_used"`
	UpdateTime       int64   `json:"update_time"` // Position update timestamp (milliseconds)
}

// AccountInfo account information
type AccountInfo struct {
	TotalEquity      float64 `json:"total_equity"`      // Account equity
	AvailableBalance float64 `json:"available_balance"` // Available balance
	UnrealizedPnL    float64 `json:"unrealized_pnl"`    // Unrealized profit/loss
	TotalPnL         float64 `json:"total_pnl"`         // Total profit/loss
	TotalPnLPct      float64 `json:"total_pnl_pct"`     // Total profit/loss percentage
	MarginUsed       float64 `json:"margin_used"`       // Used margin
	MarginUsedPct    float64 `json:"margin_used_pct"`   // Margin usage rate
	PositionCount    int     `json:"position_count"`    // Number of positions
}

// CandidateCoin candidate coin (from coin pool)
type CandidateCoin struct {
	Symbol  string   `json:"symbol"`
	Sources []string `json:"sources"` // e.g. "screener", "oi_top", "static"
	// Score is the ranking score behind the selection, when the source computes
	// one. It travels to the prompt so the model can see *why* an instrument is
	// in front of it instead of treating the list as arbitrary.
	Score float64 `json:"score,omitempty"`
}

// OITopData open interest growth top data (for AI decision reference)
type OITopData struct {
	Rank              int     // OI Top ranking
	OIDeltaPercent    float64 // Open interest change percentage (1 hour)
	OIDeltaValue      float64 // Open interest change value
	PriceDeltaPercent float64 // Price change percentage
}

// TradingStats trading statistics (for AI input)
type TradingStats struct {
	TotalTrades    int     `json:"total_trades"`     // Total number of trades (closed)
	WinRate        float64 `json:"win_rate"`         // Win rate (%)
	ProfitFactor   float64 `json:"profit_factor"`    // Profit factor
	SharpeRatio    float64 `json:"sharpe_ratio"`     // Sharpe ratio
	TotalPnL       float64 `json:"total_pnl"`        // Total profit/loss
	AvgWin         float64 `json:"avg_win"`          // Average win
	AvgLoss        float64 `json:"avg_loss"`         // Average loss
	MaxDrawdownPct float64 `json:"max_drawdown_pct"` // Maximum drawdown (%)
}

// RecentOrder recently completed order (for AI input)
type RecentOrder struct {
	Symbol       string  `json:"symbol"`        // Trading pair
	Side         string  `json:"side"`          // long/short
	EntryPrice   float64 `json:"entry_price"`   // Entry price
	ExitPrice    float64 `json:"exit_price"`    // Exit price
	RealizedPnL  float64 `json:"realized_pnl"`  // Realized profit/loss
	PnLPct       float64 `json:"pnl_pct"`       // Profit/loss percentage
	EntryTime    string  `json:"entry_time"`    // Entry time
	ExitTime     string  `json:"exit_time"`     // Exit time
	HoldDuration string  `json:"hold_duration"` // Hold duration, e.g. "2h30m"
}

// Context trading context (complete information passed to AI)
type Context struct {
	CurrentTime    string                             `json:"current_time"`
	RuntimeMinutes int                                `json:"runtime_minutes"`
	CallCount      int                                `json:"call_count"`
	Account        AccountInfo                        `json:"account"`
	Positions      []PositionInfo                     `json:"positions"`
	CandidateCoins []CandidateCoin                    `json:"candidate_coins"`
	PromptVariant  string                             `json:"prompt_variant,omitempty"`
	TradingStats   *TradingStats                      `json:"trading_stats,omitempty"`
	RecentOrders   []RecentOrder                      `json:"recent_orders,omitempty"`
	MarketDataMap  map[string]*market.Data            `json:"-"`
	MultiTFMarket  map[string]map[string]*market.Data `json:"-"`
	OITopDataMap   map[string]*OITopData              `json:"-"`
	QuantDataMap   map[string]*QuantData              `json:"-"`
	// Insights carries the market-wide context collected from the pluggable
	// marketdata providers. Ordering follows provider registration so the
	// generated prompt stays stable between cycles.
	Insights        []*marketdata.Insight `json:"-"`
	BTCETHLeverage  int                   `json:"-"`
	AltcoinLeverage int                   `json:"-"`
	Timeframes      []string              `json:"-"`
	// CoinSourceNotes records why the candidate pool fell back or shrank this
	// cycle ("ai500 unavailable: ..."). They are rendered into the prompt so the
	// model can see the universe might be incomplete instead of inferring that
	// the market is.
	CoinSourceNotes []string `json:"-"`
}

// Decision AI trading decision
type Decision struct {
	Symbol string `json:"symbol"`
	Action string `json:"action"` // Standard: "open_long", "open_short", "close_long", "close_short", "hold", "wait"
	// Grid actions: "place_buy_limit", "place_sell_limit", "cancel_order", "cancel_all_orders", "pause_grid", "resume_grid", "adjust_grid"

	// Opening position parameters
	Leverage        int     `json:"leverage,omitempty"`
	PositionSizeUSD float64 `json:"position_size_usd,omitempty"`
	StopLoss        float64 `json:"stop_loss,omitempty"`
	TakeProfit      float64 `json:"take_profit,omitempty"`

	// Grid trading parameters
	Price      float64 `json:"price,omitempty"`       // Limit order price (for grid)
	Quantity   float64 `json:"quantity,omitempty"`    // Order quantity (for grid)
	LevelIndex int     `json:"level_index,omitempty"` // Grid level index
	OrderID    string  `json:"order_id,omitempty"`    // Order ID (for cancel)

	// Common parameters
	Confidence int     `json:"confidence,omitempty"` // Confidence level (0-100)
	RiskUSD    float64 `json:"risk_usd,omitempty"`   // Maximum USD risk
	Reasoning  string  `json:"reasoning"`
}

// FullDecision AI's complete decision (including chain of thought)
type FullDecision struct {
	SystemPrompt        string     `json:"system_prompt"`
	UserPrompt          string     `json:"user_prompt"`
	CoTTrace            string     `json:"cot_trace"`
	Decisions           []Decision `json:"decisions"`
	RawResponse         string     `json:"raw_response"`
	Timestamp           time.Time  `json:"timestamp"`
	AIRequestDurationMs int64      `json:"ai_request_duration_ms,omitempty"`
}

// QuantData quantitative data structure (fund flow, position changes, price changes)
type QuantData struct {
	Symbol      string             `json:"symbol"`
	Price       float64            `json:"price"`
	Netflow     *NetflowData       `json:"netflow,omitempty"`
	OI          map[string]*OIData `json:"oi,omitempty"`
	PriceChange map[string]float64 `json:"price_change,omitempty"`
}

type NetflowData struct {
	Institution *FlowTypeData `json:"institution,omitempty"`
	Personal    *FlowTypeData `json:"personal,omitempty"`
}

type FlowTypeData struct {
	Future map[string]float64 `json:"future,omitempty"`
	Spot   map[string]float64 `json:"spot,omitempty"`
}

type OIData struct {
	CurrentOI float64                 `json:"current_oi"`
	Delta     map[string]*OIDeltaData `json:"delta,omitempty"`
}

type OIDeltaData struct {
	OIDelta        float64 `json:"oi_delta"`
	OIDeltaValue   float64 `json:"oi_delta_value"`
	OIDeltaPercent float64 `json:"oi_delta_percent"`
}

// ============================================================================
// StrategyEngine - Core Strategy Execution Engine
// ============================================================================

// StrategyEngine strategy execution engine
type StrategyEngine struct {
	config       *store.StrategyConfig
	nofxosClient *nofxos.Client
	// insights is the pluggable market-intelligence registry. The engine never
	// references a concrete data source: it asks the registry to collect
	// whatever the strategy enabled.
	insights *marketdata.Registry
	// notesMu guards notes below. Candidate-pool construction runs once per
	// cycle on the trader's own engine instance, but the same engine is also
	// reachable from the strategy-preview API.
	notesMu sync.Mutex
	// notes accumulates candidate-pool degradations for the current cycle until
	// TakeCoinSourceNotes drains them into the prompt context.
	notes []string
}

// NoteCoinSource records a degradation reason for the prompt.
func (e *StrategyEngine) NoteCoinSource(format string, args ...any) {
	note := fmt.Sprintf(format, args...)
	if e == nil {
		logger.Warnf("⚠️  %s", note)
		return
	}
	e.notesMu.Lock()
	e.notes = append(e.notes, note)
	e.notesMu.Unlock()
	logger.Warnf("⚠️  %s", note)
}

// TakeCoinSourceNotes drains this cycle's degradation notes. Draining rather
// than exposing the field keeps each note attached to the cycle that produced
// it instead of accumulating across cycles.
func (e *StrategyEngine) TakeCoinSourceNotes() []string {
	if e == nil {
		return nil
	}
	e.notesMu.Lock()
	defer e.notesMu.Unlock()
	if len(e.notes) == 0 {
		return nil
	}
	out := e.notes
	e.notes = nil
	return out
}

// NewStrategyEngine creates strategy execution engine.
func NewStrategyEngine(config *store.StrategyConfig) *StrategyEngine {
	// Create NofxOS client with API key from config
	apiKey := config.Indicators.NofxOSAPIKey
	if apiKey == "" {
		apiKey = nofxos.DefaultAuthKey
	}
	client := nofxos.NewClient(nofxos.DefaultBaseURL, apiKey)

	return &StrategyEngine{
		config:       config,
		nofxosClient: client,
		insights:     providers.Default(),
	}
}

// MarketInsightProviders lists every registered market-intelligence source,
// regardless of whether the current strategy enables it. The API exposes this so
// the UI can render the catalogue the user picks from.
func (e *StrategyEngine) MarketInsightProviders() []marketdata.Provider {
	if e == nil || e.insights == nil {
		return nil
	}
	return e.insights.All()
}

// CollectInsights gathers the enabled market-intelligence sources for one
// trading cycle. Candidate and position symbols are passed through so per-symbol
// sources can scope their queries.
//
// Every provider fails soft: the registry logs and skips a broken source, so a
// missing insight can never abort a cycle.
func (e *StrategyEngine) CollectInsights(candidateSymbols, positionSymbols []string, language string) []*marketdata.Insight {
	if e == nil || e.insights == nil {
		return nil
	}

	cfg := e.config.Indicators
	if !cfg.EnableMarketInsights {
		return nil
	}

	req := marketdata.Request{
		Symbols:   uniqueNonEmpty(candidateSymbols...),
		Positions: uniqueNonEmpty(positionSymbols...),
		Language:  language,
		Limit:     cfg.MarketInsightLimit,
	}

	started := time.Now()
	insights := e.insights.Collect(context.Background(), req, cfg)
	if len(insights) == 0 {
		logger.Infof("⏭️  No market insights collected (enabled sources: %d)", len(e.insights.Active(cfg)))
		return nil
	}

	names := make([]string, 0, len(insights))
	for _, ins := range insights {
		names = append(names, ins.Provider)
	}
	logger.Infof("📊 Market insights ready in %s: %s",
		time.Since(started).Round(time.Millisecond), strings.Join(names, ", "))

	return insights
}

func (e *StrategyEngine) usesHyperliquidNativeUniverse() bool {
	if e == nil || e.config == nil {
		return false
	}
	source := e.config.CoinSource
	if source.SourceType == "hyper_all" || source.SourceType == "hyper_main" || source.SourceType == "hyper_rank" || source.UseHyperAll || source.UseHyperMain {
		return true
	}
	for _, symbol := range source.StaticCoins {
		if market.IsXyzDexAsset(symbol) {
			return true
		}
	}
	return false
}

// GetRiskControlConfig gets risk control configuration
func (e *StrategyEngine) GetRiskControlConfig() store.RiskControlConfig {
	return e.config.RiskControl
}

// GetLanguage returns the language from config or falls back to auto-detection
func (e *StrategyEngine) GetLanguage() Language {
	switch e.config.Language {
	case "zh":
		return LangChinese
	case "en":
		return LangEnglish
	default:
		// Fall back to auto-detection from prompt content for backward compatibility
		return detectLanguage(e.config.PromptSections.RoleDefinition)
	}
}

// GetConfig gets complete strategy configuration
func (e *StrategyEngine) GetConfig() *store.StrategyConfig {
	return e.config
}

// ============================================================================
// Candidate Coins
// ============================================================================

// GetCandidateCoins gets candidate coins based on strategy configuration
func (e *StrategyEngine) GetCandidateCoins() ([]CandidateCoin, error) {
	var candidates []CandidateCoin
	symbolSources := make(map[string][]string)

	coinSource := e.config.CoinSource

	switch coinSource.SourceType {
	case "static":
		for _, symbol := range coinSource.StaticCoins {
			symbol = market.Normalize(symbol)
			candidates = append(candidates, CandidateCoin{
				Symbol:  symbol,
				Sources: []string{"static"},
			})
		}

		return e.filterExcludedCoins(candidates), nil

	case "ai500":
		// The retired NofxOS AI500 endpoint answers 402, so the source is served
		// by the local screener: same slot in the product, honest name in the
		// candidate's source list.
		if !coinSource.UseAI500 {
			logger.Infof("⚠️  source_type is 'ai500' but use_ai500 is false, falling back to static coins")
			return e.filterExcludedCoins(e.staticCandidates()), nil
		}
		coins, err := e.getScreenerCoins(coinSource.AI500Limit)
		if err != nil {
			return e.filterExcludedCoins(e.degradedCandidates("ai500", err)), nil
		}
		return e.filterExcludedCoins(coins), nil

	case "oi_top":
		// Check use_oi_top flag; if false, fall back to static coins
		if !coinSource.UseOITop {
			logger.Infof("⚠️  source_type is 'oi_top' but use_oi_top is false, falling back to static coins")
			return e.filterExcludedCoins(e.staticCandidates()), nil
		}
		coins, err := e.getOITopCoins(coinSource.OITopLimit)
		if err != nil {
			return e.filterExcludedCoins(e.degradedCandidates("oi_top", err)), nil
		}
		return e.filterExcludedCoins(coins), nil

	case "oi_low":
		// OI decrease ranking, suitable for short positions
		if !coinSource.UseOILow {
			logger.Infof("⚠️  source_type is 'oi_low' but use_oi_low is false, falling back to static coins")
			return e.filterExcludedCoins(e.staticCandidates()), nil
		}
		coins, err := e.getOILowCoins(coinSource.OILowLimit)
		if err != nil {
			return e.filterExcludedCoins(e.degradedCandidates("oi_low", err)), nil
		}
		return e.filterExcludedCoins(coins), nil

	case "hyper_all":
		// All Hyperliquid perp coins
		if !coinSource.UseHyperAll {
			logger.Infof("⚠️  source_type is 'hyper_all' but use_hyper_all is false, falling back to static coins")
			return e.filterExcludedCoins(e.staticCandidates()), nil
		}
		coins, err := e.getHyperAllCoins()
		if err != nil {
			return e.filterExcludedCoins(e.degradedCandidates("hyper_all", err)), nil
		}
		return e.filterExcludedCoins(coins), nil

	case "hyper_main":
		// Top N Hyperliquid coins by 24h volume
		if !coinSource.UseHyperMain {
			logger.Infof("⚠️  source_type is 'hyper_main' but use_hyper_main is false, falling back to static coins")
			return e.filterExcludedCoins(e.staticCandidates()), nil
		}
		coins, err := e.getHyperMainCoins(coinSource.HyperMainLimit)
		if err != nil {
			return e.filterExcludedCoins(e.degradedCandidates("hyper_main", err)), nil
		}
		return e.filterExcludedCoins(coins), nil

	case "hyper_rank":
		coins, err := e.getHyperRankCoins(coinSource.HyperRankCategory, coinSource.HyperRankDirection, coinSource.HyperRankLimit)
		if err != nil {
			return e.filterExcludedCoins(e.degradedCandidates("hyper_rank", err)), nil
		}
		return e.filterExcludedCoins(coins), nil

	case "mixed":
		mixedScores := make(map[string]float64)
		if coinSource.UseAI500 {
			poolCoins, err := e.getScreenerCoins(coinSource.AI500Limit)
			if err != nil {
				e.NoteCoinSource("Screener unavailable in mixed pool (%v)", err)
			} else {
				mixedScores = mergeMixedScores(mixedScores, poolCoins)
				for _, coin := range poolCoins {
					symbolSources[coin.Symbol] = append(symbolSources[coin.Symbol], "screener")
				}
			}
		}

		if coinSource.UseOITop {
			oiCoins, err := e.getOITopCoins(coinSource.OITopLimit)
			if err != nil {
				e.NoteCoinSource("OI-top ranking unavailable in mixed pool (%v)", err)
			} else {
				mixedScores = mergeMixedScores(mixedScores, oiCoins)
				for _, coin := range oiCoins {
					symbolSources[coin.Symbol] = append(symbolSources[coin.Symbol], "oi_top")
				}
			}
		}

		if coinSource.UseOILow {
			oiLowCoins, err := e.getOILowCoins(coinSource.OILowLimit)
			if err != nil {
				e.NoteCoinSource("OI-low ranking unavailable in mixed pool (%v)", err)
			} else {
				mixedScores = mergeMixedScores(mixedScores, oiLowCoins)
				for _, coin := range oiLowCoins {
					symbolSources[coin.Symbol] = append(symbolSources[coin.Symbol], "oi_low")
				}
			}
		}

		if coinSource.UseHyperAll {
			hyperCoins, err := e.getHyperAllCoins()
			if err != nil {
				e.NoteCoinSource("Hyperliquid universe unavailable in mixed pool (%v)", err)
			} else {
				for _, coin := range hyperCoins {
					symbolSources[coin.Symbol] = append(symbolSources[coin.Symbol], "hyper_all")
				}
			}
		}

		if coinSource.UseHyperMain {
			hyperMainCoins, err := e.getHyperMainCoins(coinSource.HyperMainLimit)
			if err != nil {
				e.NoteCoinSource("Hyperliquid main ranking unavailable in mixed pool (%v)", err)
			} else {
				for _, coin := range hyperMainCoins {
					symbolSources[coin.Symbol] = append(symbolSources[coin.Symbol], "hyper_main")
				}
			}
		}

		for _, symbol := range coinSource.StaticCoins {
			symbol = market.Normalize(symbol)
			if _, exists := symbolSources[symbol]; !exists {
				symbolSources[symbol] = []string{"static"}
			} else {
				symbolSources[symbol] = append(symbolSources[symbol], "static")
			}
		}

		for symbol, sources := range symbolSources {
			candidates = append(candidates, CandidateCoin{
				Symbol:  symbol,
				Sources: sources,
				Score:   mixedScores[symbol],
			})
		}
		return e.filterExcludedCoins(candidates), nil

	default:
		return nil, fmt.Errorf("unknown coin source type: %s", coinSource.SourceType)
	}
}

// staticCandidates builds the always-available static list. Every source falls
// back to it, which is what makes a data-source outage survivable.
func (e *StrategyEngine) staticCandidates() []CandidateCoin {
	if e == nil || len(e.config.CoinSource.StaticCoins) == 0 {
		return nil
	}
	candidates := make([]CandidateCoin, 0, len(e.config.CoinSource.StaticCoins))
	for _, symbol := range e.config.CoinSource.StaticCoins {
		candidates = append(candidates, CandidateCoin{
			Symbol:  market.Normalize(symbol),
			Sources: []string{"static"},
		})
	}
	return candidates
}

// degradedCandidates keeps the cycle alive after a source failure.
//
// An unavailable ranking used to abort the whole cycle; now it shrinks the
// universe to whatever is still known and says so. Without the note the model
// would read a two-coin universe as a deliberate market view rather than an
// outage, which is exactly the silent failure this replaces.
func (e *StrategyEngine) degradedCandidates(source string, cause error) []CandidateCoin {
	e.NoteCoinSource("Candidate source %q unavailable this cycle (%v)", source, cause)
	fallback := e.staticCandidates()
	if len(fallback) == 0 {
		e.NoteCoinSource("No static coins configured, so the candidate pool is empty for this cycle")
		return nil
	}
	e.NoteCoinSource("Fell back to %d static candidate coin(s)", len(fallback))
	return fallback
}

// filterExcludedCoins removes excluded coins from the candidates list
func (e *StrategyEngine) filterExcludedCoins(candidates []CandidateCoin) []CandidateCoin {
	if len(e.config.CoinSource.ExcludedCoins) == 0 {
		return candidates
	}

	// Build excluded set for O(1) lookup
	excluded := make(map[string]bool)
	for _, coin := range e.config.CoinSource.ExcludedCoins {
		normalized := market.Normalize(coin)
		excluded[normalized] = true
	}

	// Filter out excluded coins
	filtered := make([]CandidateCoin, 0, len(candidates))
	for _, c := range candidates {
		if !excluded[c.Symbol] {
			filtered = append(filtered, c)
		} else {
			logger.Infof("🚫 Excluded coin: %s", c.Symbol)
		}
	}

	return filtered
}

// getScreenerCoins serves the "ai500" product slot with the local attention
// ranking. The retired vendor list is not reproducible from public data, so the
// label carried on each candidate is "screener": what the model sees matches
// where the number came from.
func (e *StrategyEngine) getScreenerCoins(limit int) ([]CandidateCoin, error) {
	if limit <= 0 {
		limit = 30
	}

	rows, err := Screener(context.Background(), limit)
	if err != nil {
		return nil, err
	}

	candidates := make([]CandidateCoin, 0, len(rows))
	for _, row := range rows {
		candidates = append(candidates, CandidateCoin{
			Symbol:  row.Symbol,
			Sources: []string{"screener"},
			Score:   row.Score,
		})
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("screener returned no instruments")
	}
	logger.Infof("✅ Local screener selected %d/%d candidates (ai500 replacement)", len(candidates), limit)
	return candidates, nil
}

// getOITopCoins returns the largest open-interest increases over the last hour.
func (e *StrategyEngine) getOITopCoins(limit int) ([]CandidateCoin, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := OIChangeRanking(context.Background(), limit, false)
	if err != nil {
		return nil, err
	}
	return oiCandidates(rows, "oi_top"), nil
}

// getOILowCoins returns the largest open-interest decreases over the last hour.
func (e *StrategyEngine) getOILowCoins(limit int) ([]CandidateCoin, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := OIChangeRanking(context.Background(), limit, true)
	if err != nil {
		return nil, err
	}
	return oiCandidates(rows, "oi_low"), nil
}

// oiCandidates converts the local ranking into candidates.
func oiCandidates(rows []OIRow, source string) []CandidateCoin {
	candidates := make([]CandidateCoin, 0, len(rows))
	for _, row := range rows {
		candidates = append(candidates, CandidateCoin{
			Symbol:  row.Symbol,
			Sources: []string{source},
		})
	}
	if len(candidates) > 0 {
		logger.Infof("✅ Local open-interest ranking selected %d %s candidates over %dm",
			len(candidates), source, rows[0].WindowMinutes)
	}
	return candidates
}

// mergeMixedScores carries each source's own score into the merged pool. When a
// symbol arrives from several sources the largest score wins, so the number next
// to it is the strongest claim rather than whichever source happened to run last.
func mergeMixedScores(dst map[string]float64, coins []CandidateCoin) map[string]float64 {
	for _, coin := range coins {
		if existing, ok := dst[coin.Symbol]; !ok || coin.Score > existing {
			dst[coin.Symbol] = coin.Score
		}
	}
	return dst
}

// getHyperAllCoins returns all available Hyperliquid perpetual coins
func (e *StrategyEngine) getHyperAllCoins() ([]CandidateCoin, error) {
	ctx := context.Background()
	symbols, err := hyperliquid.GetAllCoinSymbols(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Hyperliquid coins: %w", err)
	}

	var candidates []CandidateCoin
	for _, symbol := range symbols {
		// Add USDT suffix for compatibility
		normalizedSymbol := market.Normalize(symbol + "USDT")
		candidates = append(candidates, CandidateCoin{
			Symbol:  normalizedSymbol,
			Sources: []string{"hyper_all"},
		})
	}
	logger.Infof("✅ Loaded %d Hyperliquid coins (hyper_all)", len(candidates))
	return candidates, nil
}

// getHyperMainCoins returns top N Hyperliquid coins by 24h volume
func (e *StrategyEngine) getHyperMainCoins(limit int) ([]CandidateCoin, error) {
	if limit <= 0 {
		limit = 20
	}

	ctx := context.Background()
	symbols, err := hyperliquid.GetMainCoinSymbols(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get Hyperliquid main coins: %w", err)
	}

	var candidates []CandidateCoin
	for _, symbol := range symbols {
		// Add USDT suffix for compatibility
		normalizedSymbol := market.Normalize(symbol + "USDT")
		candidates = append(candidates, CandidateCoin{
			Symbol:  normalizedSymbol,
			Sources: []string{"hyper_main"},
		})
	}
	logger.Infof("✅ Loaded %d Hyperliquid main coins (hyper_main) by 24h volume", len(candidates))
	return candidates, nil
}

func clampHyperRankLimit(limit int) int {
	if limit <= 0 {
		return 5
	}
	if limit > 10 {
		return 10
	}
	return limit
}

func (e *StrategyEngine) getHyperRankCoins(category, direction string, limit int) ([]CandidateCoin, error) {
	category = strings.ToLower(strings.TrimSpace(category))
	if category == "" {
		category = "stock"
	}
	direction = strings.ToLower(strings.TrimSpace(direction))
	if direction == "" {
		direction = "gainers"
	}
	limit = clampHyperRankLimit(limit)

	ctx := context.Background()
	var ranked []struct {
		symbol string
		info   hyperliquid.CoinInfo
		cat    string
	}

	if category == "crypto" || category == "all" {
		coins, err := hyperliquid.GetPerpDexCoins(ctx, "")
		if err != nil {
			return nil, fmt.Errorf("failed to get Hyperliquid crypto ranking: %w", err)
		}
		for _, coin := range coins {
			ranked = append(ranked, struct {
				symbol string
				info   hyperliquid.CoinInfo
				cat    string
			}{symbol: market.Normalize(coin.Symbol + "USDT"), info: coin, cat: "crypto"})
		}
	}

	if category != "crypto" {
		coins, err := hyperliquid.GetPerpDexCoins(ctx, "xyz")
		if err != nil {
			return nil, fmt.Errorf("failed to get Hyperliquid XYZ ranking: %w", err)
		}
		for _, coin := range coins {
			base := strings.TrimPrefix(coin.Symbol, "xyz:")
			cat := hyperliquid.XYZCategory(base)
			if category != "all" && cat != category {
				continue
			}
			ranked = append(ranked, struct {
				symbol string
				info   hyperliquid.CoinInfo
				cat    string
			}{symbol: hyperliquid.FormatCoinForAPI("xyz:" + base), info: coin, cat: cat})
		}
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		switch direction {
		case "losers":
			return ranked[i].info.Change24hPct < ranked[j].info.Change24hPct
		case "volume":
			return ranked[i].info.Volume24h > ranked[j].info.Volume24h
		default:
			return ranked[i].info.Change24hPct > ranked[j].info.Change24hPct
		}
	})

	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	candidates := make([]CandidateCoin, 0, len(ranked))
	source := fmt.Sprintf("hyper_rank_%s_%s", category, direction)
	for _, item := range ranked {
		candidates = append(candidates, CandidateCoin{Symbol: item.symbol, Sources: []string{source}})
	}
	logger.Infof("✅ Loaded %d Hyperliquid rank coins (%s/%s, capped at %d)", len(candidates), category, direction, limit)
	return candidates, nil
}

// ============================================================================
// External & Quant Data
// ============================================================================

// FetchMarketData fetches market data based on strategy configuration
func (e *StrategyEngine) FetchMarketData(symbol string) (*market.Data, error) {
	return market.Get(symbol)
}

// FetchExternalData fetches external data sources
func (e *StrategyEngine) FetchExternalData() (map[string]interface{}, error) {
	externalData := make(map[string]interface{})

	for _, source := range e.config.Indicators.ExternalDataSources {
		data, err := e.fetchSingleExternalSource(source)
		if err != nil {
			logger.Infof("⚠️  Failed to fetch external data source [%s]: %v", source.Name, err)
			continue
		}
		externalData[source.Name] = data
	}

	return externalData, nil
}

func (e *StrategyEngine) fetchSingleExternalSource(source store.ExternalDataSource) (interface{}, error) {
	// SSRF Protection: Validate URL before making request
	if err := security.ValidateURL(source.URL); err != nil {
		return nil, fmt.Errorf("external source URL validation failed: %w", err)
	}

	timeout := time.Duration(source.RefreshSecs) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Use SSRF-safe HTTP client
	client := security.SafeHTTPClient(timeout)

	req, err := http.NewRequest(source.Method, source.URL, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range source.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if source.DataPath != "" {
		result = extractJSONPath(result, source.DataPath)
	}

	return result, nil
}

func extractJSONPath(data interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}

	return current
}

// FetchQuantData fetches quantitative data for a single coin
func (e *StrategyEngine) FetchQuantData(symbol string) (*QuantData, error) {
	if !e.config.Indicators.EnableQuantData {
		return nil, nil
	}
	if e.usesHyperliquidNativeUniverse() || market.IsXyzDexAsset(symbol) {
		logger.Infof("⏭️  Skipping NofxOS quant data for Hyperliquid symbol %s; using native Hyperliquid klines/mark data only", symbol)
		return nil, nil
	}

	// Use nofxos client with unified API key
	include := "oi,price"
	if e.config.Indicators.EnableQuantNetflow {
		include = "netflow,oi,price"
	}

	nofxosData, err := e.nofxosClient.GetCoinData(symbol, include)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quant data: %w", err)
	}

	if nofxosData == nil {
		return nil, nil
	}

	// Convert nofxos.QuantData to kernel.QuantData
	quantData := &QuantData{
		Symbol:      nofxosData.Symbol,
		Price:       nofxosData.Price,
		PriceChange: nofxosData.PriceChange,
	}

	// Convert OI data
	if nofxosData.OI != nil {
		quantData.OI = make(map[string]*OIData)
		for exchange, oiData := range nofxosData.OI {
			if oiData != nil {
				kData := &OIData{
					CurrentOI: oiData.CurrentOI,
				}
				if oiData.Delta != nil {
					kData.Delta = make(map[string]*OIDeltaData)
					for dur, delta := range oiData.Delta {
						if delta != nil {
							kData.Delta[dur] = &OIDeltaData{
								OIDelta:        delta.OIDelta,
								OIDeltaValue:   delta.OIDeltaValue,
								OIDeltaPercent: delta.OIDeltaPercent,
							}
						}
					}
				}
				quantData.OI[exchange] = kData
			}
		}
	}

	// Convert Netflow data
	if nofxosData.Netflow != nil {
		quantData.Netflow = &NetflowData{}
		if nofxosData.Netflow.Institution != nil {
			quantData.Netflow.Institution = &FlowTypeData{
				Future: nofxosData.Netflow.Institution.Future,
				Spot:   nofxosData.Netflow.Institution.Spot,
			}
		}
		if nofxosData.Netflow.Personal != nil {
			quantData.Netflow.Personal = &FlowTypeData{
				Future: nofxosData.Netflow.Personal.Future,
				Spot:   nofxosData.Netflow.Personal.Spot,
			}
		}
	}

	return quantData, nil
}

// FetchQuantDataBatch batch fetches quantitative data
func (e *StrategyEngine) FetchQuantDataBatch(symbols []string) map[string]*QuantData {
	result := make(map[string]*QuantData)

	if !e.config.Indicators.EnableQuantData {
		return result
	}

	for _, symbol := range symbols {
		data, err := e.FetchQuantData(symbol)
		if err != nil {
			logger.Infof("⚠️  Failed to fetch quantitative data for %s: %v", symbol, err)
			continue
		}
		if data != nil {
			result[symbol] = data
		}
	}

	return result
}

func uniqueNonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func uniqueValues(values ...string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

// ============================================================================
// Helper Functions
// ============================================================================

// detectLanguage detects language from text content
// Returns LangChinese if text contains Chinese characters, otherwise LangEnglish
func detectLanguage(text string) Language {
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			return LangChinese
		}
	}
	return LangEnglish
}
