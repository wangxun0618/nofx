import useSWR from 'swr'
import type { CSSProperties } from 'react'
import { api } from '../../lib/api'
import { t } from '../../i18n/translations'
import { useLanguage } from '../../contexts/LanguageContext'
import type {
  MarketDirectionChange,
  MarketDirectionPayload,
  MarketDirectionRow,
  MarketFlowPayload,
  MarketFlowRow,
  MarketInsightFailure,
  MarketLeveragePayload,
  MarketLeverageRow,
  MarketOrderflowPayload,
  MarketPositioningPayload,
  MarketPositioningRow,
} from '../../lib/api/data'
import './terminal.css'

// The upstream board is cached server-side for five minutes, so polling faster
// than that would only re-render identical numbers.
const POLL_MS = 120_000

/** Every provider the registry can emit, plus the synthetic coverage block. */
export type MarketInsightProvider =
  | 'directional_signal'
  | 'hyperliquid_flow'
  | 'hyperliquid_leverage'
  | 'hyperdata_orderflow'
  | 'hyperdata_positioning'
  | 'data_coverage'

interface MarketInsightsPanelProps {
  /** Which registered provider to render. */
  provider: MarketInsightProvider
  height?: number
  /**
   * Request the sidecar-backed sources. Panels for HyperData providers need
   * this; the others work without it because their inputs are free endpoints.
   */
  hyperdata?: boolean
}

function fmtUsdCompact(n: number | undefined): string {
  if (n == null || Number.isNaN(n)) return '—'
  const abs = Math.abs(n)
  if (abs >= 1e9) return `$${(n / 1e9).toFixed(2)}B`
  if (abs >= 1e6) return `$${(n / 1e6).toFixed(1)}M`
  if (abs >= 1e3) return `$${(n / 1e3).toFixed(1)}K`
  return `$${n.toFixed(2)}`
}

function fmtSigned(n: number | undefined, digits = 2): string {
  if (n == null || Number.isNaN(n)) return '—'
  return `${n >= 0 ? '+' : ''}${n.toFixed(digits)}%`
}

/** Signed z-score, which reads better with an explicit minus sign. */
function fmtScore(n: number | undefined): string {
  if (n == null || Number.isNaN(n)) return '—'
  return `${n >= 0 ? '+' : ''}${n.toFixed(2)}`
}

/** Equities-board instruments get a suffix so a ticker is never ambiguous. */
function fmtSym(symbol: string, dex?: string): string {
  const base = symbol.replace(/^xyz:/i, '').toUpperCase()
  return dex === 'xyz' ? `${base}·eq` : base
}

function toneClass(n: number | undefined): string {
  if (n == null || Number.isNaN(n) || n === 0) return ''
  return n > 0 ? 'tm-up' : 'tm-dn'
}

/**
 * Direction tone. A bullish verdict is an upward read, so it takes the same
 * colour as a price rise.
 */
function biasClass(bias: string | undefined): string {
  if (bias === 'bullish') return 'tm-up'
  if (bias === 'bearish') return 'tm-dn'
  return ''
}

const headerCell: CSSProperties = {
  fontSize: 10,
  letterSpacing: '0.08em',
  textTransform: 'uppercase',
  color: 'var(--tm-muted)',
}

const rowStyle: CSSProperties = {
  display: 'grid',
  gap: 6,
  alignItems: 'center',
  padding: '3px 0',
  fontSize: 11,
  borderTop: '1px solid var(--tm-rule)',
}

const ellipsis: CSSProperties = {
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
}

function TableHead({ labels, columns }: { labels: string[]; columns: string }) {
  return (
    <div style={{ ...rowStyle, gridTemplateColumns: columns, borderTop: 'none' }}>
      {labels.map((label, index) => (
        <span key={label} style={index === 0 ? headerCell : { ...headerCell, textAlign: 'right' }}>
          {label}
        </span>
      ))}
    </div>
  )
}

function Empty() {
  const { language } = useLanguage()
  return <div className="tm-sc">{t('terminal.marketInsightsEmpty', language)}</div>
}

/** Cross-market fund flow: turnover concentration plus funding crowding. */
function FlowTable({ payload }: { payload: MarketFlowPayload }) {
  const rows: MarketFlowRow[] = payload?.most_traded || []

  if (rows.length === 0) return <Empty />

  const columns = 'minmax(0,1.1fr) repeat(4, minmax(0,1fr))'

  return (
    <div style={{ overflow: 'auto', minHeight: 0 }}>
      <TableHead labels={['instrument', '24h vol', '24h chg', 'open int.', 'funding ann.']} columns={columns} />

      {rows.map((row) => (
        <div key={`${row.dex || 'main'}:${row.symbol}`} style={{ ...rowStyle, gridTemplateColumns: columns }}>
          <span className="tm-mono" style={{ color: 'var(--tm-ink)' }}>{fmtSym(row.symbol, row.dex)}</span>
          <span className="tm-mono" style={{ textAlign: 'right', color: 'var(--tm-ink-2)' }}>{fmtUsdCompact(row.volume_24h)}</span>
          <span className={`tm-mono ${toneClass(row.change_24h_pct)}`} style={{ textAlign: 'right' }}>{fmtSigned(row.change_24h_pct)}</span>
          <span className="tm-mono" style={{ textAlign: 'right', color: 'var(--tm-ink-2)' }}>{fmtUsdCompact(row.open_interest_usd)}</span>
          <span className={`tm-mono ${toneClass(row.funding_annual_pct)}`} style={{ textAlign: 'right' }}>{fmtSigned(row.funding_annual_pct, 1)}</span>
        </div>
      ))}
    </div>
  )
}

/** Open-interest structure: where positions are large and which side is stretched. */
function LeverageTable({ payload }: { payload: MarketLeveragePayload }) {
  const rows: MarketLeverageRow[] = payload?.largest_open_interest || []

  if (rows.length === 0) return <Empty />

  const columns = 'minmax(0,1.1fr) repeat(4, minmax(0,1fr))'

  return (
    <div style={{ overflow: 'auto', minHeight: 0 }}>
      <TableHead labels={['instrument', 'open int.', '24h vol', 'max lev', 'vs oracle']} columns={columns} />

      {rows.map((row) => (
        <div key={`${row.dex || 'main'}:${row.symbol}`} style={{ ...rowStyle, gridTemplateColumns: columns }}>
          <span className="tm-mono" style={{ color: 'var(--tm-ink)' }}>{fmtSym(row.symbol, row.dex)}</span>
          <span className="tm-mono" style={{ textAlign: 'right', color: 'var(--tm-ink-2)' }}>{fmtUsdCompact(row.open_interest_usd)}</span>
          <span className="tm-mono" style={{ textAlign: 'right', color: 'var(--tm-ink-2)' }}>{fmtUsdCompact(row.volume_24h)}</span>
          <span className="tm-mono" style={{ textAlign: 'right', color: 'var(--tm-ink-2)' }}>{row.max_leverage ? `${row.max_leverage}×` : '—'}</span>
          <span className={`tm-mono ${toneClass(row.premium_pct)}`} style={{ textAlign: 'right' }}>{fmtSigned(row.premium_pct, 3)}</span>
        </div>
      ))}
    </div>
  )
}

/**
 * The direction board.
 *
 * Shows the verdict, its strength, and which inputs voted — the point of carrying
 * the components is that a verdict can be checked rather than trusted.
 */
function DirectionTable({ payload }: { payload: MarketDirectionPayload }) {
  const { language } = useLanguage()

  // Prefer the instruments in play; fall back to the strongest reads on each side
  // so the panel still says something when no candidates have been resolved yet.
  const rows: MarketDirectionRow[] = payload?.instruments?.length
    ? payload.instruments
    : [...(payload?.top_bullish || []), ...(payload?.top_bearish || [])]

  const changes: MarketDirectionChange[] = payload?.recent_changes || []

  if (rows.length === 0 && changes.length === 0) return <Empty />

  const columns = 'minmax(0,1fr) minmax(0,0.7fr) minmax(0,0.55fr) minmax(0,1.6fr)'
  const missingFlow = !(payload?.components || []).includes('flow')

  return (
    <div style={{ overflow: 'auto', minHeight: 0, display: 'flex', flexDirection: 'column', gap: 8 }}>
      {rows.length > 0 ? (
        <div>
          <TableHead labels={['instrument', 'bias', 'score', 'evidence']} columns={columns} />
          {rows.map((row) => (
            <div key={row.symbol} style={{ ...rowStyle, gridTemplateColumns: columns }}>
              <span className="tm-mono" style={{ color: 'var(--tm-ink)' }}>{fmtSym(row.symbol)}</span>
              <span className={`tm-mono ${biasClass(row.bias)}`} style={{ textAlign: 'right' }}>{row.bias}</span>
              <span className={`tm-mono ${toneClass(row.score)}`} style={{ textAlign: 'right' }}>{fmtScore(row.score)}</span>
              <span className="tm-sc" style={{ ...ellipsis, textAlign: 'right' }} title={(row.components || []).map((c) => `${c.name} ${c.detail}`).join(' · ')}>
                {(row.components || []).map((c) => `${c.name} ${c.detail}`).join(' · ') || '—'}
              </span>
            </div>
          ))}
        </div>
      ) : null}

      {missingFlow ? (
        <div className="tm-sc" style={{ fontSize: 10 }}>
          {t('terminal.directionNoFlow', language)}
        </div>
      ) : null}

      {changes.length > 0 ? (
        <div>
          <div className="tm-sc" style={{ ...headerCell, paddingBottom: 4 }}>
            {t('terminal.directionChanges', language)}
          </div>
          {changes.map((change) => (
            <div
              key={`${change.symbol}:${change.changed_at}`}
              className="tm-sc"
              style={{ fontSize: 10, padding: '2px 0', borderTop: '1px solid var(--tm-rule)' }}
              title={change.reason}
            >
              <span className="tm-mono" style={{ color: 'var(--tm-ink)' }}>{change.symbol}</span>{' '}
              <span className={biasClass(change.from_bias)}>{change.from_bias}</span>
              {' → '}
              <span className={biasClass(change.to_bias)}>{change.to_bias}</span>
              {' · '}
              {new Date(change.changed_at).toLocaleTimeString()}
              {' · '}
              <span style={ellipsis}>{change.reason}</span>
            </div>
          ))}
        </div>
      ) : null}
    </div>
  )
}

/**
 * Multi-venue order flow.
 *
 * Venue coverage is rendered above the numbers on purpose: a single-venue delta
 * is weaker evidence, and the point of this source is to say so rather than to
 * present one venue's tape as the whole market's.
 */
function OrderflowTable({ payload }: { payload: MarketOrderflowPayload }) {
  const { language } = useLanguage()
  const rows = payload?.rows || []
  const venues = payload?.venue_status || []
  const unavailable = payload?.unavailable_symbols || []
  const longShort = payload?.long_short || {}

  if (rows.length === 0 && Object.keys(longShort).length === 0) return <Empty />

  const columns = 'minmax(0,1fr) repeat(4, minmax(0,1fr))'

  return (
    <div style={{ overflow: 'auto', minHeight: 0, display: 'flex', flexDirection: 'column', gap: 8 }}>
      {venues.length > 0 ? (
        <div className="tm-sc" style={{ fontSize: 10 }}>{venues.join(' · ')}</div>
      ) : null}

      {payload?.version_warning ? (
        <div className="tm-sc" style={{ fontSize: 10 }}>{payload.version_warning}</div>
      ) : null}

      {rows.length > 0 ? (
        <div>
          <TableHead labels={['instrument', 'cvd', 'HL', 'BN', 'trades/s']} columns={columns} />
          {rows.map((row) => (
            <div key={row.symbol} style={{ ...rowStyle, gridTemplateColumns: columns }}>
              <span className="tm-mono" style={{ color: 'var(--tm-ink)' }}>{fmtSym(row.symbol)}</span>
              <span className={`tm-mono ${toneClass(row.cumulative_cvd)}`} style={{ textAlign: 'right' }}>{fmtUsdCompact(row.cumulative_cvd)}</span>
              <span className="tm-mono" style={{ textAlign: 'right', color: 'var(--tm-ink-2)' }}>{fmtUsdCompact(row.cvd_hyperliquid)}</span>
              <span className="tm-mono" style={{ textAlign: 'right', color: 'var(--tm-ink-2)' }}>{fmtUsdCompact(row.cvd_binance)}</span>
              <span className="tm-mono" style={{ textAlign: 'right', color: 'var(--tm-ink-2)' }}>{row.trades_per_second?.toFixed(1) ?? '—'}</span>
            </div>
          ))}
        </div>
      ) : null}

      {Object.keys(longShort).length > 0 ? (
        <div>
          <div className="tm-sc" style={{ ...headerCell, paddingBottom: 4 }}>
            {t('terminal.leverageSentiment', language)}
          </div>
          {Object.entries(longShort).map(([symbol, entry]) => (
            <div key={symbol} style={{ ...rowStyle, gridTemplateColumns: 'minmax(0,1fr) repeat(2, minmax(0,1fr))', fontSize: 10 }}>
              <span className="tm-mono" style={{ color: 'var(--tm-ink)' }}>{symbol}</span>
              <span className="tm-mono" style={{ textAlign: 'right', color: 'var(--tm-ink-2)' }}>{entry.long_ratio?.toFixed(1)}% L</span>
              <span className="tm-mono" style={{ textAlign: 'right', color: 'var(--tm-ink-2)' }}>{entry.long_short_ratio?.toFixed(2)}×</span>
            </div>
          ))}
        </div>
      ) : null}

      {unavailable.length > 0 ? (
        <div className="tm-sc" style={{ fontSize: 10 }}>
          {t('terminal.orderflowUntracked', language)} {unavailable.join(', ')}
        </div>
      ) : null}
    </div>
  )
}

/** Tracked positions and how close they sit to liquidation. */
function PositioningTable({ payload }: { payload: MarketPositioningPayload }) {
  const { language } = useLanguage()
  const rows: MarketPositioningRow[] = payload?.near_liquidation?.length
    ? payload.near_liquidation
    : payload?.largest || []

  if (rows.length === 0) return <Empty />

  const columns = 'minmax(0,1fr) minmax(0,0.6fr) repeat(3, minmax(0,0.9fr))'
  const showDistance = (payload?.near_liquidation?.length || 0) > 0

  return (
    <div style={{ overflow: 'auto', minHeight: 0, display: 'flex', flexDirection: 'column', gap: 6 }}>
      <div className="tm-sc" style={{ fontSize: 10 }}>
        {t('terminal.positionSampleNotice', language)}
        {payload?.scan_age_seconds != null ? ` · ${Math.round(payload.scan_age_seconds)}s` : ''}
      </div>

      <TableHead
        labels={showDistance ? ['instrument', 'side', 'size', 'liq dist', 'lev'] : ['instrument', 'side', 'size', 'lev', 'uPnL']}
        columns={columns}
      />

      {rows.map((row, index) => (
        <div key={`${row.symbol}:${row.side}:${index}`} style={{ ...rowStyle, gridTemplateColumns: columns }}>
          <span className="tm-mono" style={{ color: 'var(--tm-ink)' }}>{fmtSym(row.symbol)}</span>
          <span className={`tm-mono ${row.side?.toLowerCase() === 'short' ? 'tm-dn' : 'tm-up'}`} style={{ textAlign: 'right' }}>{row.side}</span>
          <span className="tm-mono" style={{ textAlign: 'right', color: 'var(--tm-ink-2)' }}>{fmtUsdCompact(row.size_usd)}</span>
          {showDistance ? (
            <span className="tm-mono" style={{ textAlign: 'right', color: 'var(--tm-ink-2)' }}>{row.distance_pct?.toFixed(2)}%</span>
          ) : (
            <span className="tm-mono" style={{ textAlign: 'right', color: 'var(--tm-ink-2)' }}>{row.leverage ? `${row.leverage}×` : '—'}</span>
          )}
          {showDistance ? (
            <span className="tm-mono" style={{ textAlign: 'right', color: 'var(--tm-ink-2)' }}>{row.leverage ? `${row.leverage}×` : '—'}</span>
          ) : (
            <span className={`tm-mono ${toneClass(row.unrealized_pnl)}`} style={{ textAlign: 'right' }}>{fmtUsdCompact(row.unrealized_pnl)}</span>
          )}
        </div>
      ))}
    </div>
  )
}

/**
 * Sources that were selected but produced nothing.
 *
 * Rendered rather than hidden because a gap in the prompt is indistinguishable
 * from a quiet market unless it is stated.
 */
function CoverageList({ failures }: { failures: MarketInsightFailure[] }) {
  const { language } = useLanguage()

  if (failures.length === 0) {
    return <div className="tm-sc">{t('terminal.coverageComplete', language)}</div>
  }

  return (
    <div style={{ overflow: 'auto', minHeight: 0 }}>
      {failures.map((failure) => (
        <div key={failure.provider} style={{ ...rowStyle, gridTemplateColumns: 'minmax(0,1fr) minmax(0,2fr)' }}>
          <span className="tm-mono" style={{ color: 'var(--tm-ink)' }}>{failure.provider}</span>
          <span className="tm-sc" style={{ ...ellipsis, textAlign: 'right' }} title={failure.reason}>{failure.reason}</span>
        </div>
      ))}
    </div>
  )
}

/**
 * Renders one market-intelligence source from the pluggable registry.
 *
 * The data comes from the same backend registry the strategy engine reads, so
 * what is shown here is exactly what the AI reasons over. All panels on a page
 * share one SWR key, which keeps several of them to a single request.
 */
export function MarketInsightsPanel({ provider, height = 300, hyperdata = false }: MarketInsightsPanelProps) {
  const { language } = useLanguage()

  const needsSidecar = provider === 'hyperdata_orderflow' || provider === 'hyperdata_positioning'

  const { data, error, isLoading } = useSWR(
    needsSidecar ? 'market-insights:sidecar' : 'market-insights',
    () => api.getMarketInsights({ silent: true, hyperdata: needsSidecar || hyperdata }),
    { refreshInterval: POLL_MS, revalidateOnFocus: false }
  )

  const insight = data?.insights?.find((item) => item.provider === provider)

  // A missing sidecar-backed panel is the expected state when the feature is off,
  // so it says what is missing instead of implying the market is quiet.
  const emptyMessage = (() => {
    if (error instanceof Error) return error.message
    if (needsSidecar) return t('terminal.sidecarUnavailable', language)
    return t('terminal.marketInsightsEmpty', language)
  })()

  const titles: Record<MarketInsightProvider, string> = {
    directional_signal: t('terminal.marketDirection', language),
    hyperliquid_flow: t('terminal.marketFlow', language),
    hyperliquid_leverage: t('terminal.oiStructure', language),
    hyperdata_orderflow: t('terminal.orderflow', language),
    hyperdata_positioning: t('terminal.positioning', language),
    data_coverage: t('terminal.dataCoverage', language),
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height, minHeight: 0 }}>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 6 }}>
        <span className="tm-px" style={{ fontSize: 11 }}>{titles[provider]}</span>
        <span className="tm-sc">{t('terminal.live', language)}</span>
        <span className="tm-sc" style={{ marginLeft: 'auto' }}>
          {insight ? new Date(insight.fetched_at).toLocaleTimeString() : '—'}
        </span>
      </div>

      {isLoading && !insight ? <div className="tm-sc">…</div> : null}

      {!isLoading && !insight ? (
        <div className="tm-sc" style={{ padding: '12px 0' }}>{emptyMessage}</div>
      ) : null}

      {insight && provider === 'directional_signal' ? (
        <DirectionTable payload={insight.payload as MarketDirectionPayload} />
      ) : null}
      {insight && provider === 'hyperliquid_flow' ? (
        <FlowTable payload={insight.payload as MarketFlowPayload} />
      ) : null}
      {insight && provider === 'hyperliquid_leverage' ? (
        <LeverageTable payload={insight.payload as MarketLeveragePayload} />
      ) : null}
      {insight && provider === 'hyperdata_orderflow' ? (
        <OrderflowTable payload={insight.payload as MarketOrderflowPayload} />
      ) : null}
      {insight && provider === 'hyperdata_positioning' ? (
        <PositioningTable payload={insight.payload as MarketPositioningPayload} />
      ) : null}
      {insight && provider === 'data_coverage' ? (
        <CoverageList failures={(insight.payload as MarketInsightFailure[]) || []} />
      ) : null}
    </div>
  )
}
