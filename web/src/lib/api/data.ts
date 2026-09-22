import type {
  SystemStatus,
  AccountInfo,
  Position,
  DecisionRecord,
  Statistics,
  TraderFullStats,
  CompetitionData,
  PositionHistoryResponse,
} from '../../types'
import { API_BASE, httpClient } from './helpers'

import { tg } from '../../i18n/translations'

export interface Kline {
  openTime: number
  open: number
  high: number
  low: number
  close: number
  volume: number
  closeTime: number
}

export interface MarketSymbol {
  symbol: string
  display?: string
  name: string
  category: 'crypto' | 'stock' | 'forex' | 'commodity' | 'index' | string
  exchange?: string
  volume_24h?: number
  mark_price?: number
  change_24h_pct?: number
  prev_day_price?: number
  maxLeverage?: number
  sz_decimals?: number
}

export interface SymbolListResponse {
  exchange: string
  symbols: MarketSymbol[]
  count: number
}

// ── Market insights (pluggable data sources) ────────────────────────────────
// Mirrors marketdata.Provider on the backend. The catalogue comes from the
// registry, so a newly registered source appears here without a frontend change.

export interface MarketInsightSource {
  name: string
  description: string
  requires_key: boolean
  /**
   * Names an external process the source needs. Locally hosted sources need no
   * credential but are still unavailable until the user starts something, which
   * `requires_key` cannot express.
   */
  requires_service?: string
}

/** One instrument's cross-market flow contribution. */
export interface MarketFlowRow {
  symbol: string
  dex?: string
  volume_24h: number
  change_24h_pct: number
  open_interest_usd: number
  funding_rate_1h: number
  funding_annual_pct: number
}

export interface MarketFlowPayload {
  most_traded: MarketFlowRow[]
  most_crowded_long: MarketFlowRow[]
  most_crowded_short: MarketFlowRow[]
  gainers: MarketFlowRow[]
  losers: MarketFlowRow[]
  universe_size: number
}

/** One instrument's open-interest and leverage structure. */
export interface MarketLeverageRow {
  symbol: string
  dex?: string
  open_interest_usd: number
  volume_24h: number
  max_leverage: number
  funding_annual_pct: number
  premium_pct: number
}

export interface MarketLeveragePayload {
  largest_open_interest: MarketLeverageRow[]
  long_crowded: MarketLeverageRow[]
  short_crowded: MarketLeverageRow[]
  candidates: MarketLeverageRow[]
  universe_size: number
}

/** One realised-liquidation price bucket. */
export interface MarketLiquidationBin {
  bucket_start_price: number
  bucket_end_price: number
  long_liq_usd: number
  short_liq_usd: number
}

export interface MarketLiquidationPayload {
  symbols: { symbol: string; bins: MarketLiquidationBin[] }[]
}

/** One auditable input behind a directional verdict. */
export interface MarketDirectionComponent {
  /** Stable component id: momentum, premium or flow. */
  name: string
  vote: 'bullish' | 'bearish' | 'neutral'
  /** Human-readable magnitude, e.g. "+1.90σ" or "-0.050%". */
  detail: string
}

export interface MarketDirectionRow {
  symbol: string
  bias: 'bullish' | 'bearish' | 'neutral'
  score: number
  bullish: number
  bearish: number
  neutral: number
  components: MarketDirectionComponent[]
}

/** A recorded bias transition plus the component evidence that caused it. */
export interface MarketDirectionChange {
  symbol: string
  from_bias: string
  to_bias: string
  from_score: number
  to_score: number
  reason: string
  changed_at: string
}

export interface MarketDirectionPayload {
  top_bullish: MarketDirectionRow[]
  top_bearish: MarketDirectionRow[]
  instruments: MarketDirectionRow[]
  recent_changes?: MarketDirectionChange[]
  universe_size: number
  components: string[]
}

/** One instrument's aggressive order flow. */
export interface MarketOrderflowRow {
  symbol: string
  cumulative_cvd: number
  cvd_hyperliquid: number
  cvd_binance: number
  net_volume?: number
  net_volume_window?: string
  trades_per_second: number
  aggregate_signal?: string
  venues?: string[]
  metrics_available: boolean
}

export interface MarketOrderflowPayload {
  rows: MarketOrderflowRow[]
  long_short?: Record<
    string,
    { long_ratio: number; short_ratio: number; long_short_ratio: number; timestamp: number }
  >
  basis?: Record<
    string,
    { spot_price: number; perp_price: number; basis_pct: number; timestamp: number }
  >
  venue_status?: string[]
  unavailable_symbols?: string[]
  hyperdata_version?: string
  version_warning?: string
}

/** One tracked position, relative to its liquidation price. */
export interface MarketPositioningRow {
  symbol: string
  side: string
  size_usd: number
  entry_price: number
  mark_price: number
  liq_price: number
  distance_pct: number
  leverage: number
  unrealized_pnl: number
}

export interface MarketPositioningPayload {
  largest: MarketPositioningRow[]
  near_liquidation: MarketPositioningRow[]
  danger_threshold_pct?: number
  scan_age_seconds?: number
  tracked_positions: number
  hyperdata_version?: string
}

/**
 * A source that was selected but produced nothing this cycle. The backend emits
 * these as a `data_coverage` block so a missing feed is never mistaken for a
 * quiet market.
 */
export interface MarketInsightFailure {
  provider: string
  reason: string
}

export interface MarketInsight {
  provider: string
  title: string
  markdown: string
  markdown_zh?: string
  payload?:
    | MarketFlowPayload
    | MarketLeveragePayload
    | MarketLiquidationPayload
    | MarketDirectionPayload
    | MarketOrderflowPayload
    | MarketPositioningPayload
    | MarketInsightFailure[]
    | Record<string, unknown>
  fetched_at: string
}

export interface MarketInsightsResponse {
  insights: MarketInsight[]
  fetched_at: string
}

export const dataApi = {
  async getSymbols(exchange = 'hyperliquid-xyz'): Promise<SymbolListResponse> {
    const result = await httpClient.get<SymbolListResponse>(
      `${API_BASE}/symbols?exchange=${encodeURIComponent(exchange)}`
    )
    if (!result.success) throw new Error(tg('lib.fetchSymbolList'))
    return result.data || { exchange, symbols: [], count: 0 }
  },

  async getStatus(traderId?: string, silent?: boolean): Promise<SystemStatus> {
    const url = traderId
      ? `${API_BASE}/status?trader_id=${traderId}`
      : `${API_BASE}/status`
    const result = await httpClient.request<SystemStatus>(url, { silent })
    if (!result.success) throw new Error(tg('lib.fetchSystemStatus'))
    return result.data!
  },

  async getAccount(traderId?: string, silent?: boolean): Promise<AccountInfo> {
    const url = traderId
      ? `${API_BASE}/account?trader_id=${traderId}`
      : `${API_BASE}/account`
    const result = await httpClient.request<AccountInfo>(url, { silent })
    if (!result.success) throw new Error(tg('lib.fetchAccountInfo'))
    return result.data!
  },

  async getPositions(traderId?: string, silent?: boolean): Promise<Position[]> {
    const url = traderId
      ? `${API_BASE}/positions?trader_id=${traderId}`
      : `${API_BASE}/positions`
    const result = await httpClient.request<Position[]>(url, { silent })
    if (!result.success) throw new Error(tg('lib.fetchPositions'))
    return result.data!
  },

  async getDecisions(traderId?: string): Promise<DecisionRecord[]> {
    const url = traderId
      ? `${API_BASE}/decisions?trader_id=${traderId}`
      : `${API_BASE}/decisions`
    const result = await httpClient.get<DecisionRecord[]>(url)
    if (!result.success) throw new Error(tg('lib.fetchDecisionLogs'))
    return result.data!
  },

  async getLatestDecisions(
    traderId?: string,
    limit: number = 5,
    silent?: boolean
  ): Promise<DecisionRecord[]> {
    const params = new URLSearchParams()
    if (traderId) {
      params.append('trader_id', traderId)
    }
    params.append('limit', limit.toString())

    const result = await httpClient.request<DecisionRecord[]>(
      `${API_BASE}/decisions/latest?${params}`,
      { silent }
    )
    if (!result.success) throw new Error(tg('lib.fetchLatestDecisions'))
    return result.data!
  },

  async getStatistics(
    traderId?: string,
    silent?: boolean
  ): Promise<Statistics> {
    const url = traderId
      ? `${API_BASE}/statistics?trader_id=${traderId}`
      : `${API_BASE}/statistics`
    const result = await httpClient.request<Statistics>(url, { silent })
    if (!result.success) throw new Error(tg('lib.fetchStatistics'))
    return result.data!
  },

  async getFullStats(
    traderId?: string,
    silent?: boolean
  ): Promise<TraderFullStats> {
    const url = traderId
      ? `${API_BASE}/statistics/full?trader_id=${traderId}`
      : `${API_BASE}/statistics/full`
    const result = await httpClient.request<TraderFullStats>(url, { silent })
    if (!result.success) throw new Error(tg('lib.fetchFullStatistics'))
    return result.data!
  },

  async getKlines(
    symbol: string,
    interval = '5m',
    exchange = 'hyperliquid',
    limit = 60,
    silent?: boolean
  ): Promise<Kline[]> {
    const params = new URLSearchParams({
      symbol,
      interval,
      exchange,
      limit: String(limit),
    })
    const result = await httpClient.request<Kline[]>(
      `${API_BASE}/klines?${params}`,
      { silent }
    )
    if (!result.success) throw new Error(tg('lib.fetchKlines'))
    return result.data!
  },

  async getEquityHistory(traderId?: string, silent?: boolean): Promise<any[]> {
    const url = traderId
      ? `${API_BASE}/equity-history?trader_id=${traderId}`
      : `${API_BASE}/equity-history`
    const result = await httpClient.request<any[]>(url, { silent })
    if (!result.success) throw new Error(tg('lib.fetchEquityHistory'))
    return result.data!
  },

  async getEquityHistoryBatch(
    traderIds: string[],
    hours?: number
  ): Promise<any> {
    const result = await httpClient.post<any>(
      `${API_BASE}/equity-history-batch`,
      { trader_ids: traderIds, hours: hours || 0 }
    )
    if (!result.success) throw new Error(tg('lib.fetchBatchEquityHistory'))
    return result.data!
  },

  async getTopTraders(): Promise<any[]> {
    const result = await httpClient.get<any[]>(`${API_BASE}/top-traders`)
    if (!result.success) throw new Error(tg('lib.fetchTopTraders'))
    return result.data!
  },

  async getPublicTraderConfig(traderId: string): Promise<any> {
    const result = await httpClient.get<any>(
      `${API_BASE}/traders/${traderId}/public-config`
    )
    if (!result.success) throw new Error(tg('lib.fetchPublicTraderConfig'))
    return result.data!
  },

  async getCompetition(): Promise<CompetitionData> {
    const result = await httpClient.get<CompetitionData>(
      `${API_BASE}/competition`
    )
    if (!result.success) throw new Error(tg('lib.fetchCompetitionData'))
    return result.data!
  },

  async getPositionHistory(
    traderId: string,
    limit: number = 100,
    silent?: boolean
  ): Promise<PositionHistoryResponse> {
    const result = await httpClient.request<PositionHistoryResponse>(
      `${API_BASE}/positions/history?trader_id=${traderId}&limit=${limit}`,
      { silent }
    )
    if (!result.success) throw new Error(tg('lib.fetchPositionHistory'))
    return result.data!
  },

  /**
   * Lists every market-intelligence source the backend ships, regardless of
   * whether the active strategy enables it.
   */
  async getMarketInsightSources(
    silent?: boolean
  ): Promise<MarketInsightSource[]> {
    const result = await httpClient.request<{ sources: MarketInsightSource[] }>(
      `${API_BASE}/market-insights/providers`,
      { silent }
    )
    if (!result.success) throw new Error(tg('lib.fetchMarketInsightSources'))
    return result.data?.sources || []
  },

  /**
   * Collects the market-wide insight board. This is the same registry the
   * strategy engine reads, so the terminal shows exactly what the AI sees.
   */
  async getMarketInsights(params?: {
    lang?: string
    sources?: string[]
    /** Requests the sidecar-backed sources, which need HyperData running. */
    hyperdata?: boolean
    hyperdataUrl?: string
    silent?: boolean
  }): Promise<MarketInsightsResponse> {
    const query = new URLSearchParams()
    if (params?.lang) query.set('lang', params.lang)
    if (params?.sources?.length) query.set('sources', params.sources.join(','))
    if (params?.hyperdata) {
      query.set('hyperdata', '1')
      if (params.hyperdataUrl) query.set('hyperdata_url', params.hyperdataUrl)
    }
    const suffix = query.toString() ? `?${query.toString()}` : ''

    const result = await httpClient.request<MarketInsightsResponse>(
      `${API_BASE}/market-insights${suffix}`,
      { silent: params?.silent }
    )
    if (!result.success) throw new Error(tg('lib.fetchMarketInsights'))
    return result.data || { insights: [], fetched_at: '' }
  },
}
