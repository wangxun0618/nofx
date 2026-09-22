import { useEffect, useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import type { CSSProperties } from 'react'
import useSWR, { mutate } from 'swr'
import { api } from '../../lib/api'
import { confirmToast, notify } from '../../lib/notify'
import type {
  SystemStatus,
  AccountInfo,
  Position,
  DecisionRecord,
  TraderInfo,
} from '../../types'
import { OrchestrationTopology } from './OrchestrationTopology'
import { OrderBook } from './OrderBook'
import { KlineChart } from './KlineChart'
import { ExecutionLog } from './ExecutionLog'
import { RiskRadar } from './RiskRadar'
import { EdgeProfile } from './EdgeProfile'
import { useDemoEngine } from '../../lib/demo/useDemoEngine'

// fixed height for the row-1 panels so the row stays balanced at any width
const ROW1_H = 500
import { t } from '../../i18n/translations'
import { useLanguage } from '../../contexts/LanguageContext'
import './terminal.css'

interface TerminalDashboardProps {
  selectedTrader?: TraderInfo
  traders?: TraderInfo[]
  selectedTraderId?: string
  onTraderSelect: (traderId: string) => void
  status?: SystemStatus
  account?: AccountInfo
  positions?: Position[]
  decisions?: DecisionRecord[]
}

function fmtUsd(n: number | undefined, signed = false): string {
  if (n == null || Number.isNaN(n)) return '—'
  const sign = signed && n > 0 ? '+' : n < 0 ? '-' : ''
  return `${sign}$${Math.abs(n).toLocaleString('en-US', { maximumFractionDigits: 2 })}`
}
function fmtPct(n: number | undefined): string {
  if (n == null || Number.isNaN(n)) return '—'
  return `${n >= 0 ? '+' : ''}${n.toFixed(2)}%`
}
/** Price with magnitude-aware precision: 64,416 · 184.2 · 2.3775 · 0.0067 */
function fmtPx(n: number | undefined): string {
  if (n == null || Number.isNaN(n) || n === 0) return '—'
  const dp = n >= 1000 ? 0 : n >= 100 ? 1 : n >= 1 ? 2 : 4
  return n.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
}
function baseLabel(raw?: string): string {
  if (!raw) return ''
  return raw.toUpperCase().replace(/^XYZ:/, '').replace(/[-_]/g, '').replace(/(USDT|USDC|USD)$/, '')
}
function parseScanMinutes(scan?: string): number {
  if (!scan) return 15
  const m = scan.match(/(\d+)\s*m/i)
  if (m) return parseInt(m[1], 10)
  const h = scan.match(/(\d+)\s*h/i)
  if (h) return parseInt(h[1], 10) * 60
  const n = parseInt(scan, 10)
  return Number.isFinite(n) && n > 0 ? n : 15
}
function fmtTime(raw?: string | number): string {
  if (raw == null || raw === '') return ''
  let n = typeof raw === 'number' ? raw : Number(raw)
  if (Number.isFinite(n)) {
    if (n < 1e12) n *= 1000
    return new Date(n).toLocaleString('en-GB', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })
  }
  const d = new Date(raw as string)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleString('en-GB', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })
}

/** Hold duration from entry/exit epoch-ms as a compact 45m / 2h10 / 1d3h. */
function fmtHold(entry?: number, exit?: number): string {
  if (!entry || !exit || exit <= entry) return '—'
  const mins = Math.round((exit - entry) / 60000)
  if (mins < 60) return `${mins}m`
  const h = Math.floor(mins / 60)
  const m = mins % 60
  if (h < 24) return m ? `${h}h${m}` : `${h}h`
  const d = Math.floor(h / 24)
  return `${d}d${h % 24}h`
}

function useTick(ms = 1000) {
  const [, set] = useState(0)
  useEffect(() => {
    const id = setInterval(() => set((n) => n + 1), ms)
    return () => clearInterval(id)
  }, [ms])
}

export function TerminalDashboard({
  selectedTrader,
  traders,
  selectedTraderId,
  onTraderSelect,
  status: propStatus,
  account: propAccount,
  positions: propPositions,
  decisions: propDecisions,
}: TerminalDashboardProps) {
  const { language } = useLanguage()
  const traderId = selectedTrader?.trader_id || selectedTraderId
  useTick(1000)
  const clock = new Date().toLocaleTimeString('en-GB', { hour12: false })
  const [closing, setClosing] = useState<string | null>(null)

  async function closePositionRow(symbol: string, side: 'LONG' | 'SHORT') {
    if (!traderId || closing) return
    const ok = await confirmToast(
      t('terminal.marketCloseOne', language, { symbol, side }),
      {
        title: t('terminal.closePosition', language),
        okText: t('terminal.close', language),
        cancelText: t('common.cancel', language),
      }
    )
    if (!ok) return
    setClosing(symbol)
    try {
      await api.closePosition(traderId, symbol, side)
      notify.success(`${symbol} ${side} closed`)
      await Promise.all([
        mutate(`positions-${traderId}`),
        mutate(`account-${traderId}`),
      ])
    } catch (err) {
      notify.error(
        err instanceof Error ? err.message : t('terminal.closeFailed', language)
      )
    } finally {
      setClosing(null)
    }
  }

  async function closeAllPositions(open: Position[]) {
    if (!traderId || closing || open.length === 0) return
    const ok = await confirmToast(
      t('terminal.marketCloseAll', language, { count: open.length }),
      {
        title: t('terminal.flattenBook', language),
        okText: t('terminal.closeAll', language),
        cancelText: t('common.cancel', language),
      }
    )
    if (!ok) return
    setClosing('__all__')
    let failed = 0
    // Sequential: parallel closes race on exchange nonces / rate limits.
    for (const p of open) {
      const side = /long|buy/i.test(p.side) ? 'LONG' : 'SHORT'
      try {
        await api.closePosition(traderId, p.symbol, side)
      } catch {
        failed++
      }
    }
    await Promise.all([
      mutate(`positions-${traderId}`),
      mutate(`account-${traderId}`),
    ])
    if (failed === 0) notify.success(t('terminal.allPositionsClosed', language))
    else
      notify.error(
        t('terminal.closesFailed', language, {
          failed,
          total: open.length,
        })
      )
    setClosing(null)
  }

  const { data: realFullStats } = useSWR(
    traderId ? ['full-stats', traderId] : null,
    () => api.getFullStats(traderId!, true),
    { refreshInterval: 30000, shouldRetryOnError: false }
  )
  const { data: realHistory } = useSWR(
    traderId ? ['pos-history', traderId] : null,
    () => api.getPositionHistory(traderId!, 50, true),
    { refreshInterval: 60000, shouldRetryOnError: false }
  )
  const { data: realConfig } = useSWR(
    traderId ? ['trader-config', traderId] : null,
    () => api.getTraderConfig(traderId!, true),
    { refreshInterval: 120000, shouldRetryOnError: false }
  )

  // Demo / showcase mode for product walkthroughs. Toggle with Shift+D (or the
  // discreet corner dot). Generates a fast, profitable-looking US-equity dataset
  // entirely client-side — it never touches the backend or any real account.
  // When off, real data flows through unchanged.
  const [demo, setDemo] = useState(false)
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.shiftKey && (e.key === 'D' || e.key === 'd')) {
        const el = document.activeElement
        if (el && /^(input|textarea|select)$/i.test(el.tagName)) return
        setDemo((v) => !v)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])
  const sim = useDemoEngine(demo)
  const on = demo && !!sim

  const status = on ? sim!.status : propStatus
  const account = on ? sim!.account : propAccount
  const positions = on ? sim!.positions : propPositions
  const decisions = on ? sim!.decisions : propDecisions
  const fullStats = on ? sim!.fullStats : realFullStats
  const history = on ? sim!.history : realHistory
  const config = on ? (sim!.config as unknown as typeof realConfig) : realConfig

  const latest = decisions && decisions.length > 0 ? decisions[0] : undefined
  const candidateCoins = latest?.candidate_coins ?? []

  // the price panels follow the instrument the bot is ACTUALLY holding (first
  // open position, else this cycle's first candidate) so they match the real
  // traded symbol; only fall back to SP500 if the bot holds nothing.
  const defaultSym = useMemo(() => {
    const heldBases = [...(positions ?? []).map((p) => p.symbol), ...candidateCoins].map(baseLabel).filter(Boolean)
    return heldBases[0] || 'SP500'
  }, [positions, candidateCoins])
  const activeSym = defaultSym.toUpperCase()

  const pnl = account?.total_pnl ?? 0
  const pnlPct = account?.total_pnl_pct ?? 0
  const up = pnl >= 0
  const running = status?.is_running

  // direction per symbol — priority: AI's actual decision > long (never blindly
  // fabricate a short when the bot has no opinion).
  const dirFor = useMemo(() => {
    const dec = new Map<string, 'long' | 'short'>()
    ;(latest?.decisions ?? []).forEach((d) => {
      const b = baseLabel(d.symbol)
      if (d.action === 'open_long' || d.action === 'close_short') dec.set(b, 'long')
      else if (d.action === 'open_short' || d.action === 'close_long') dec.set(b, 'short')
    })
    return (sym: string): 'long' | 'short' => dec.get(baseLabel(sym)) ?? 'long'
  }, [latest])

  const scanMin = config?.scan_interval_minutes || parseScanMinutes(status?.scan_interval)
  const nextCycleMs = useMemo(() => {
    if (!latest?.timestamp) return null
    return new Date(latest.timestamp).getTime() + scanMin * 60_000
  }, [latest?.timestamp, scanMin])
  let countdown = '—'
  if (nextCycleMs) {
    const ms = nextCycleMs - Date.now()
    if (ms <= 0) countdown = 'due now'
    else {
      const s = Math.floor(ms / 1000)
      countdown = `${Math.floor(s / 60)}m ${s % 60}s`
    }
  }

  const recentTrades = (history?.positions ?? []).slice(0, 8)
  const symbolStats = useMemo(
    () => (history?.symbol_stats ?? []).slice().sort((a, b) => b.total_trades - a.total_trades).slice(0, 6),
    [history]
  )
  const maxSymTrades = symbolStats.reduce((m, s) => Math.max(m, s.total_trades), 1)

  const sc: CSSProperties = { padding: '10px 14px' }
  const cellBorder = '1px solid var(--tm-hair)'

  // Portal the trader selector + run status into the global nav so the app has
  // a single top bar (no separate dashboard titlebar).
  const [navSlot, setNavSlot] = useState<HTMLElement | null>(null)
  useEffect(() => {
    setNavSlot(document.getElementById('dash-header-slot'))
  }, [])

  return (
    <div className="nofx-terminal" style={{ minHeight: '100vh', padding: 0 }}>
      {/* discreet, unlabelled showcase toggle (Shift+D also works) */}
      <button
        type="button"
        onClick={() => setDemo((v) => !v)}
        aria-label={t('terminal.togglePresentation', language)}
        style={{
          position: 'fixed',
          right: 10,
          bottom: 10,
          zIndex: 9999,
          width: 12,
          height: 12,
          padding: 0,
          borderRadius: '50%',
          border: 'none',
          cursor: 'pointer',
          background: on ? 'var(--tm-up)' : 'rgba(26,24,19,0.2)',
          opacity: on ? 0.55 : 0.18,
        }}
      />
      {/* centered, capped content column — no border (keeps it from feeling
          embedded) but bounded so the aspect-ratio SVGs don't balloon on wide screens */}
      {navSlot &&
        createPortal(
          <span className="nofx-terminal" style={{ background: 'transparent', display: 'flex', alignItems: 'center', gap: 12, marginLeft: 16, paddingLeft: 16, borderLeft: '1px solid rgba(26,24,19,0.15)', fontSize: 11 }}>
            <span className="tm-sc" style={{ color: 'var(--tm-muted)' }}>orchestration</span>
            {traders && traders.length > 0 && (
              <select value={traderId || ''} onChange={(e) => onTraderSelect(e.target.value)} className="tm-mono"
                style={{ background: 'var(--tm-panel)', color: 'var(--tm-ink)', border: '1px solid var(--tm-hair)', borderRadius: 0, fontSize: 11, padding: '3px 6px' }}>
                {traders.map((t) => (<option key={t.trader_id} value={t.trader_id} style={{ color: '#111' }}>{t.trader_name}</option>))}
              </select>
            )}
            <span style={{ color: running ? 'var(--tm-up)' : 'var(--tm-muted)' }}>{running ? '● running' : '○ stopped'}</span>
            <span className="tm-sc" style={{ color: 'var(--tm-muted)' }}>cycle</span><span className="tm-mono" style={{ color: 'var(--tm-ink)' }}>{status?.call_count ?? '—'}</span>
            <span className="tm-px" style={{ fontSize: 12, color: 'var(--tm-ink)' }}>{clock}</span>
          </span>,
          navSlot,
        )}
      <div className="tm-box" style={{ maxWidth: 1280, margin: '0 auto', border: 'none' }}>
        {/* runtime health banner — safe mode would otherwise only be visible in
            server logs while the bot silently idles */}
        {!on && status?.safe_mode && (
          <div className="tm-mono" style={{ display: 'flex', gap: 10, alignItems: 'center', margin: '8px 14px 0', padding: '8px 12px', fontSize: 11, border: '1px solid var(--tm-down)', color: 'var(--tm-down)', background: 'rgba(200,60,40,0.06)', flexWrap: 'wrap' }}>
            <span style={{ fontWeight: 600 }}>{t('terminal.safeModeBanner', language)}</span>
            <span style={{ color: 'var(--tm-ink-2)' }}>{status.safe_mode_reason || ''}</span>
          </div>
        )}
        {/* first-run reassurance — a fresh autopilot looks idle for its first
            minute (the AI is reading the market); tell newcomers what to expect */}
        {!on && status?.is_running && (status.call_count ?? 0) <= 1 && !status.safe_mode && (
          <div className="tm-mono" style={{ display: 'flex', gap: 10, alignItems: 'center', margin: '8px 14px 0', padding: '8px 12px', fontSize: 11, border: '1px solid var(--tm-up)', color: 'var(--tm-ink)', background: 'rgba(40,140,80,0.06)', flexWrap: 'wrap' }}>
            <span style={{ fontWeight: 600, color: 'var(--tm-up)' }}>{t('terminal.liveTitle', language)}</span>
            <span style={{ color: 'var(--tm-ink-2)' }}>
              {t('terminal.firstRunHint', language)}
            </span>
          </div>
        )}
        {/* config / identity strip — first row, flows directly under the global nav */}
        <div className="tm-mono" style={{ display: 'flex', gap: 16, padding: '6px 14px', fontSize: 11, color: 'var(--tm-ink-2)', flexWrap: 'wrap' }}>
          <span style={{ fontWeight: 500 }}>{selectedTrader?.trader_name ?? 'NOFX'}</span>
          <span><span className="tm-sc">model </span>{(() => {
            const raw = config?.ai_model || status?.ai_model || ''
            if (!raw) return '—'
            return raw.length > 16 ? raw.slice(0, 16).toUpperCase() : raw.toUpperCase()
          })()}</span>
          <span><span className="tm-sc">strategy </span>{config?.strategy_name || selectedTrader?.strategy_name || '—'}</span>
          <span><span className="tm-sc">lev </span>{config?.btc_eth_leverage ?? '—'}× / {config?.altcoin_leverage ?? '—'}×</span>
          <span><span className="tm-sc">scan </span>{scanMin}m</span>
          <span><span className="tm-sc">universe </span>{candidateCoins.length}</span>
          <span><span className="tm-sc">positions </span>{positions?.length ?? 0}</span>
          <span style={{ marginLeft: 'auto' }}><span className="tm-sc">{t('terminal.nextCycle', language)}</span>{countdown}</span>
        </div>
        <div className="tm-rule" />

        {/* metric row — "Total P/L" is equity-based (includes unrealized);
            "Realized P/L" is closed-trades only and matches PF/win-rate/sharpe,
            so the two never read as contradicting each other */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)' }}>
          {[
            { l: t('terminal.equityLabel', language), v: fmtUsd(account?.total_equity), c: 'var(--tm-ink)' },
            { l: t('terminal.totalPnl', language), v: `${fmtUsd(pnl, true)} (${fmtPct(pnlPct)})`, c: up ? 'var(--tm-up)' : 'var(--tm-dn)' },
            {
              l: t('terminal.realizedPnl', language),
              v: fullStats != null ? fmtUsd(fullStats.total_pnl, true) : '—',
              c: fullStats != null && fullStats.total_pnl >= 0 ? 'var(--tm-up)' : 'var(--tm-dn)',
            },
            { l: t('terminal.profitFactor', language), v: fullStats != null ? fullStats.profit_factor.toFixed(2) : '—', c: 'var(--tm-ink)' },
            // max_drawdown_pct is already a percent (18.5 = -18.5%)
            { l: t('terminal.maxDrawdown', language), v: fullStats != null ? `-${fullStats.max_drawdown_pct.toFixed(1)}%` : '—', c: 'var(--tm-dn)' },
          ].map((m, i) => (
            <div key={m.l} style={{ padding: '12px 14px', borderRight: i < 4 ? cellBorder : 'none' }}>
              <div className="tm-sc">{m.l}</div>
              <div className="tm-mono" style={{ fontSize: 17, fontWeight: 500, color: m.c, marginTop: 3 }}>{m.v}</div>
            </div>
          ))}
        </div>
        <div className="tm-rule" />

        {/* trades summary */}
        {fullStats != null && (
          <>
            <div className="tm-mono" style={{ display: 'flex', gap: 18, padding: '6px 14px', fontSize: 11, color: 'var(--tm-ink-2)', flexWrap: 'wrap' }}>
              <span className="tm-sc">trades <b style={{ color: 'var(--tm-ink)' }}>{fullStats.total_trades}</b></span>
              <span className="tm-sc tm-up">win {fullStats.win_trades} ({fullStats.win_rate.toFixed(1)}%)</span>
              <span className="tm-sc tm-dn">loss {fullStats.loss_trades}</span>
              {/* fee-drag chain: gross realized − fees = net realized */}
              <span className="tm-sc">gross <b style={{ color: 'var(--tm-ink)' }}>{fmtUsd(fullStats.total_pnl + fullStats.total_fee, true)}</b></span>
              <span className="tm-sc">fees <b style={{ color: 'var(--tm-ink)' }}>-{fmtUsd(fullStats.total_fee)}</b></span>
              <span className="tm-sc">net <b style={{ color: fullStats.total_pnl >= 0 ? 'var(--tm-up)' : 'var(--tm-dn)' }}>{fmtUsd(fullStats.total_pnl, true)}</b></span>
              <span className="tm-sc">sharpe/trade <b style={{ color: 'var(--tm-ink)' }}>{fullStats.sharpe_ratio.toFixed(2)}</b></span>
              <span className="tm-sc">{t('terminal.avgWinLoss', language)}<b style={{ color: 'var(--tm-ink)' }}>{fullStats.avg_win.toFixed(2)}/{fullStats.avg_loss.toFixed(2)}</b></span>
            </div>
            <div className="tm-rule" />
          </>
        )}

        {/* ── row 1: live L2 order book · live K-line (instrument selector)
              both columns are locked to one fixed height so the row is always
              balanced ── */}
        <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0,1fr) minmax(0,1.05fr)' }}>
          <div style={{ ...sc, borderRight: cellBorder, height: ROW1_H, overflow: 'hidden' }}>
            <OrderBook symbol={activeSym} demo={on} markPrice={positions?.find((p) => baseLabel(p.symbol) === activeSym)?.entry_price} />
          </div>
          <div style={{ ...sc, height: ROW1_H, display: 'flex', flexDirection: 'column', minHeight: 0 }}>
            <div style={{ flex: 1, minHeight: 0 }}>
              <KlineChart symbol={activeSym} fill demo={on} />
            </div>
          </div>
        </div>
        <div className="tm-rule" />

        {/* orchestration topology — second row, full width (the agent workflow) */}
        <div style={sc}>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 10, marginBottom: 4 }}>
            <span className="tm-px" style={{ fontSize: 12 }}>{t('terminal.orchestrationTopology', language)}</span>
            <span className="tm-sc">{t('terminal.orchestrationFlow', language)}</span>
          </div>
          <OrchestrationTopology
            layers={[
              {
                // every candidate the AI actually judged this cycle (its full decision set)
                key: 'decision',
                title: t('terminal.stageDecision', language),
                items: candidateCoins.map((c) => ({ symbol: c, dir: dirFor(c) })),
              },
              {
                // executed & live: every open position is an executed order, so
                // EXECUTE mirrors the live book (this cycle's fills plus anything
                // still open from prior cycles) and flows straight into HOLD
                key: 'exec',
                title: t('terminal.stageExecute', language),
                items: (positions ?? []).map((p) => ({
                  symbol: p.symbol,
                  dir: (p.side || '').toLowerCase().includes('short') ? ('short' as const) : ('long' as const),
                })),
              },
              {
                key: 'hold',
                title: t('terminal.stageHold', language),
                items: (positions ?? []).map((p) => ({
                  symbol: p.symbol,
                  dir: (p.side || '').toLowerCase().includes('short') ? ('short' as const) : ('long' as const),
                })),
              },
            ]}
          />
        </div>
        <div className="tm-rule" />

        {/* ── row 3: execution log · risk radar · recent trades ── */}
        <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0,1.1fr) minmax(0,1fr) minmax(0,1fr)' }}>
          <div style={{ ...sc, borderRight: cellBorder }}>
            <ExecutionLog decisions={decisions} height={432} />
          </div>
          <div style={{ ...sc, borderRight: cellBorder }}>
            <RiskRadar positions={positions} account={account} config={config} fullStats={fullStats} />
          </div>
          <div style={sc}>
            {/* live open positions (the book right now) */}
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 6 }}>
              <span className="tm-px" style={{ fontSize: 11 }}>{t('terminal.positions', language)}</span>
              <span className="tm-sc">{t('terminal.currentPositionsLive', language)}</span>
              <span className="tm-sc" style={{ marginLeft: 'auto' }}>{positions?.length ?? 0} open</span>
              {traderId && !on && positions && positions.length > 0 && (
                <button
                  type="button"
                  onClick={() => void closeAllPositions(positions)}
                  disabled={closing !== null}
                  className="tm-mono"
                  style={{
                    background: 'transparent',
                    border: '1px solid var(--tm-dn)',
                    color: 'var(--tm-dn)',
                    borderRadius: 3,
                    fontSize: 9,
                    padding: '1px 6px',
                    cursor: closing ? 'not-allowed' : 'pointer',
                    opacity: closing ? 0.5 : 1,
                  }}
                >
                  {closing === '__all__' ? 'closing…' : 'close all'}
                </button>
              )}
            </div>
            {positions && positions.length > 0 ? (
              <table className="tm-mono" style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11 }}>
                <thead>
                  <tr className="tm-sc" style={{ fontSize: 9 }}>
                    <td style={{ padding: '0 0 3px' }}>symbol</td>
                    <td style={{ padding: '0 0 3px' }}>side·lev</td>
                    <td style={{ padding: '0 0 3px', textAlign: 'right' }}>entry</td>
                    <td style={{ padding: '0 0 3px', textAlign: 'right' }}>size</td>
                    <td style={{ padding: '0 0 3px', textAlign: 'right' }}>PnL</td>
                    <td style={{ padding: '0 0 3px', textAlign: 'right' }}>return%</td>
                    {traderId && !on && <td style={{ padding: '0 0 3px' }} />}
                  </tr>
                </thead>
                <tbody>
                  {positions.map((p, i) => {
                    const long = /long|buy/i.test(p.side)
                    const win = (p.unrealized_pnl ?? 0) >= 0
                    const notional = Math.abs(p.quantity ?? 0) * (p.mark_price || p.entry_price || 0)
                    return (
                      <tr key={`${p.symbol}-${i}`} style={{ borderTop: '1px solid var(--tm-hair)' }}>
                        <td style={{ padding: '5px 0', fontWeight: 500 }}>{baseLabel(p.symbol)}</td>
                        <td style={{ padding: '5px 0' }} className={long ? 'tm-up' : 'tm-dn'}>{long ? 'long' : 'short'} <span style={{ color: 'var(--tm-muted)' }}>{p.leverage}×</span></td>
                        <td style={{ padding: '5px 0', textAlign: 'right', color: 'var(--tm-ink-2)' }}>{fmtPx(p.entry_price)}</td>
                        <td style={{ padding: '5px 0', textAlign: 'right', color: 'var(--tm-ink-2)' }}>{fmtUsd(notional)}</td>
                        <td style={{ padding: '5px 0', textAlign: 'right' }} className={win ? 'tm-up' : 'tm-dn'}>{fmtUsd(p.unrealized_pnl, true)}</td>
                        <td style={{ padding: '5px 0', textAlign: 'right' }} className={win ? 'tm-up' : 'tm-dn'}>{(p.unrealized_pnl_pct ?? 0).toFixed(2)}%</td>
                        {traderId && !on && (
                          <td style={{ padding: '5px 0 5px 8px', textAlign: 'right', width: 1 }}>
                            <button
                              type="button"
                              onClick={() => void closePositionRow(p.symbol, long ? 'LONG' : 'SHORT')}
                              disabled={closing !== null}
                              title={`Close ${p.symbol}`}
                              className="tm-mono"
                              style={{
                                background: 'transparent',
                                border: '1px solid var(--tm-dn)',
                                color: 'var(--tm-dn)',
                                borderRadius: 3,
                                fontSize: 9,
                                padding: '1px 5px',
                                cursor: closing ? 'not-allowed' : 'pointer',
                                opacity: closing ? 0.5 : 1,
                              }}
                            >
                              {closing === p.symbol ? '…' : 'close'}
                            </button>
                          </td>
                        )}
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            ) : <div className="tm-sc" style={{ padding: '8px 0' }}>{t('terminal.noOpenPositions', language)}</div>}

            <div className="tm-rule" style={{ margin: '12px 0 10px' }} />

            <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 6 }}>
              <span className="tm-px" style={{ fontSize: 11 }}>{t('terminal.recentTrades', language)}</span>
              <span className="tm-sc">{t('terminal.recentCloses', language)}</span>
            </div>
            {recentTrades.length > 0 ? (
              <table className="tm-mono" style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11 }}>
                <thead>
                  <tr className="tm-sc" style={{ fontSize: 9 }}>
                    <td style={{ padding: '0 0 3px' }}>symbol</td>
                    <td style={{ padding: '0 0 3px' }}>side</td>
                    <td style={{ padding: '0 0 3px', textAlign: 'right' }}>hold</td>
                    <td style={{ padding: '0 0 3px' }}> closed</td>
                    <td style={{ padding: '0 0 3px', textAlign: 'right' }}>PnL</td>
                  </tr>
                </thead>
                <tbody>
                  {recentTrades.map((p) => {
                    const win = p.realized_pnl >= 0
                    return (
                      <tr key={p.id} style={{ borderTop: '1px solid var(--tm-hair)' }}>
                        <td style={{ padding: '5px 0', fontWeight: 500 }}>{baseLabel(p.symbol)}</td>
                        <td style={{ padding: '5px 0' }} className={p.side === 'long' || p.side === 'LONG' ? 'tm-up' : 'tm-dn'}>{p.side.toLowerCase()}</td>
                        <td style={{ padding: '5px 0', textAlign: 'right', color: 'var(--tm-ink-2)' }}>{fmtHold(p.entry_time, p.exit_time)}</td>
                        <td style={{ padding: '5px 0 5px 6px', color: 'var(--tm-muted)' }}>{fmtTime(p.exit_time)}</td>
                        <td style={{ padding: '5px 0', textAlign: 'right' }} className={win ? 'tm-up' : 'tm-dn'}>{fmtUsd(p.realized_pnl, true)}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            ) : <div className="tm-sc" style={{ padding: '8px 0' }}>{t('terminal.noClosedTrades', language)}</div>}
          </div>
        </div>
        <div className="tm-rule" />

        {/* by-symbol history · edge profile — footer */}
        <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0,1fr) minmax(0,1fr)' }}>
          <div style={{ ...sc, borderRight: cellBorder }}>
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 8 }}>
              <span className="tm-px" style={{ fontSize: 11 }}>{t('terminal.bySymbol', language)}</span>
              <span className="tm-sc">{t('terminal.bySymbolHistory', language)}</span>
            </div>
            {symbolStats.length > 0 ? symbolStats.map((s) => (
              <div key={s.symbol} style={{ marginBottom: 7 }}>
                <div className="tm-mono" style={{ display: 'flex', fontSize: 11, marginBottom: 2 }}>
                  <span style={{ fontWeight: 500 }}>{baseLabel(s.symbol)}</span>
                  <span className="tm-sc" style={{ marginLeft: 8 }}>{s.total_trades} trades · {s.win_rate.toFixed(0)}% win</span>
                  <span className={s.total_pnl >= 0 ? 'tm-up' : 'tm-dn'} style={{ marginLeft: 'auto' }}>{fmtUsd(s.total_pnl, true)}</span>
                </div>
                <div style={{ height: 4, background: 'var(--tm-hair)' }}>
                  <div style={{ height: 4, width: `${(s.total_trades / maxSymTrades) * 100}%`, background: s.total_pnl >= 0 ? 'var(--tm-up)' : 'var(--tm-dn)' }} />
                </div>
              </div>
            )) : <div className="tm-sc">{t('terminal.noSymbolHistory', language)}</div>}
          </div>
          <div style={sc}>
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 8 }}>
              <span className="tm-px" style={{ fontSize: 11 }}>{t('terminal.edgeProfile', language)}</span>
              <span className="tm-sc">{t('terminal.edgeProfileDetail', language)}</span>
            </div>
            <EdgeProfile positions={history?.positions} />
          </div>
        </div>
      </div>
    </div>
  )
}

export default TerminalDashboard
