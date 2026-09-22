import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  AlertCircle,
  ArrowRight,
  CheckCircle2,
  ExternalLink,
  Loader2,
  RefreshCw,
  ShieldCheck,
  Wallet,
  Zap,
} from 'lucide-react'
import { toast } from 'sonner'
import { buildDashboardPath } from '../../router/paths'
import {
  AUTOPILOT_TRADER_NAME,
  ensureAutopilotStrategy,
  isAutopilotStrategyName,
  launchAutopilot,
} from '../../lib/launch/launchAutopilot'
import { runLaunchPreflight } from '../../lib/launch/preflight'
import type { LaunchPreflightResult } from '../../lib/launch/types'
import type {
  AIModel,
  Exchange,
  ExchangeAccountState,
  TraderInfo,
} from '../../types'
import { HyperliquidWalletConnect } from '../common/HyperliquidWalletConnect'
import { t, type Language } from '../../i18n/translations'

type LaunchStepStatus = 'ready' | 'action' | 'blocked'

interface AutopilotLaunchPanelProps {
  models: AIModel[]
  exchanges: Exchange[]
  exchangeAccountStates: Record<string, ExchangeAccountState>
  traders?: TraderInfo[]
  isLoggedIn: boolean
  language: Language
  onRefresh: () => Promise<void>
  onOpenModelConfig?: () => void
  onOpenHyperliquidConfig?: () => void
}

const MIN_TRADING_USDC = 12

function parseNumber(value?: string | number) {
  if (typeof value === 'number') return Number.isFinite(value) ? value : 0
  if (!value) return 0
  const parsed = Number(value.replace(/[,$\s]/g, ''))
  return Number.isFinite(parsed) ? parsed : 0
}

function shortAddress(address?: string) {
  if (!address) return '--'
  return `${address.slice(0, 6)}…${address.slice(-4)}`
}

function formatUSDC(value: number) {
  return new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value)
}

export function AutopilotLaunchPanel({
  models,
  exchanges,
  exchangeAccountStates,
  traders = [],
  isLoggedIn,
  language,
  onRefresh,
  onOpenModelConfig,
  onOpenHyperliquidConfig,
}: AutopilotLaunchPanelProps) {
  const navigate = useNavigate()
  const [launching, setLaunching] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const isZh = language === 'zh'

  // The configured AI model is the only AI prerequisite: any enabled provider
  // with an API key on file can drive the autopilot.
  const aiModel = useMemo(
    () =>
      models.find(
        (model) => model.enabled && (model.has_api_key || model.apiKey)
      ) || null,
    [models]
  )
  const modelReady = Boolean(aiModel)

  const hyperliquidExchange = useMemo(
    () =>
      exchanges.find(
        (exchange) =>
          exchange.exchange_type === 'hyperliquid' &&
          exchange.enabled &&
          Boolean(exchange.hyperliquidWalletAddr) &&
          Boolean(exchange.hyperliquidBuilderApproved)
      ) || null,
    [exchanges]
  )

  // Any hyperliquid account (even partially configured) is enough for the
  // server preflight — it reports exactly which prerequisite is missing.
  const preflightExchange = useMemo(
    () =>
      hyperliquidExchange ||
      exchanges.find((exchange) => exchange.exchange_type === 'hyperliquid') ||
      null,
    [exchanges, hyperliquidExchange]
  )

  // Server-side preflight is the source of truth for balances: it queries the
  // chain / exchange live (30s server cache) instead of trusting the balance
  // snapshot cached in the model object. Poll while the panel is visible so
  // deposits show up without a manual refresh.
  const [preflight, setPreflight] = useState<LaunchPreflightResult | null>(null)
  const aiModelId = aiModel?.id
  const preflightExchangeId = preflightExchange?.id
  useEffect(() => {
    if (!isLoggedIn || !aiModelId || !preflightExchangeId) {
      setPreflight(null)
      return
    }
    let cancelled = false
    const check = async () => {
      try {
        const result = await runLaunchPreflight({
          ai_model_id: aiModelId,
          exchange_id: preflightExchangeId,
        })
        if (!cancelled) setPreflight(result)
      } catch {
        // keep the last known result; client-derived fallbacks still render
      }
    }
    void check()
    const timer = setInterval(() => void check(), 20000)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [isLoggedIn, aiModelId, preflightExchangeId])

  const preflightCheck = (id: string) =>
    preflight?.checks.find((check) => check.id === id)

  const hyperliquidConnected = Boolean(hyperliquidExchange)
  const exchangeState = hyperliquidExchange
    ? exchangeAccountStates[hyperliquidExchange.id]
    : undefined
  const accountCheck = preflightCheck('exchange_account')
  const tradingFundsCheck = preflightCheck('exchange_funds')
  const tradingBalance =
    tradingFundsCheck?.actual ??
    parseNumber(exchangeState?.available_balance ?? exchangeState?.total_equity)
  const minTradingUSDC = preflight?.min_trading_usdc ?? MIN_TRADING_USDC
  const tradingBalanceReady =
    hyperliquidConnected &&
    (accountCheck && tradingFundsCheck
      ? accountCheck.status === 'ok' && tradingFundsCheck.status !== 'failed'
      : exchangeState?.status === 'ok' && tradingBalance >= minTradingUSDC)

  const autopilotTrader = useMemo(
    () =>
      traders.find((trader) => trader.trader_name === AUTOPILOT_TRADER_NAME) ||
      traders.find((trader) => isAutopilotStrategyName(trader.strategy_name)) ||
      null,
    [traders]
  )

  const allReady = modelReady && hyperliquidConnected && tradingBalanceReady

  const refreshEverything = async () => {
    setRefreshing(true)
    try {
      await onRefresh()
    } finally {
      setRefreshing(false)
    }
  }

  const handleLaunch = async () => {
    if (!aiModel || !hyperliquidExchange) return
    setLaunching(true)
    try {
      // Shared launch path (same as Strategy Studio): server preflight with
      // fresh balances first, then strategy provisioning, then create/start.
      const outcome = await launchAutopilot({
        ensureStrategy: ensureAutopilotStrategy,
        scanIntervalMinutes: 5,
      })

      if (!outcome.ok) {
        toast.error(outcome.message)
        if (outcome.kind === 'preflight') {
          setPreflight(outcome.preflight)
        }
        if (outcome.kind !== 'error') {
          if (outcome.setupTarget === 'ai-model') {
            onOpenModelConfig?.()
          } else if (outcome.setupTarget === 'hyperliquid') {
            onOpenHyperliquidConfig?.()
          }
        }
        await refreshEverything()
        return
      }

      if (outcome.warning) {
        toast.warning(outcome.warning)
      }
      await onRefresh()
      toast.success(t('autopilot.running', language))
      navigate(buildDashboardPath(outcome.traderId))
    } finally {
      setLaunching(false)
    }
  }

  const steps: Array<{
    title: string
    detail: string
    status: LaunchStepStatus
    meta?: string
    action?: JSX.Element
  }> = [
    {
      title: t('hlw.panelStep1Title', language),
      detail: t('hlw.panelStep1Detail', language),
      status: modelReady ? 'ready' : 'action',
      meta: aiModel
        ? `${aiModel.name} · ${aiModel.provider}`
        : t('noModelsConfigured', language),
      action: (
        <button
          type="button"
          onClick={() => onOpenModelConfig?.()}
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-nofx-gold hover:text-nofx-accent"
        >
          <ArrowRight className="h-3.5 w-3.5" />
          {t('onboarding.oneClickSetup', language)}
        </button>
      ),
    },
    {
      title: t('hlw.panelStep2Title', language),
      detail: t('hlw.panelStep2Detail', language),
      status: hyperliquidConnected ? 'ready' : 'action',
      meta: hyperliquidExchange?.hyperliquidWalletAddr
        ? `${shortAddress(hyperliquidExchange.hyperliquidWalletAddr)} · authorized`
        : t('hlw.panelStep2Meta', language),
      action: (
        <button
          type="button"
          onClick={() => onOpenHyperliquidConfig?.()}
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-nofx-gold hover:text-nofx-accent"
        >
          <Wallet className="h-3.5 w-3.5" />
          {t('hlw.open', language)}
        </button>
      ),
    },
    {
      title: t('hlw.panelStep3Title', language),
      detail: t('hlw.panelStep3Detail', language),
      status: tradingBalanceReady
        ? 'ready'
        : hyperliquidConnected
          ? 'action'
          : 'blocked',
      meta: hyperliquidConnected
        ? `${formatUSDC(tradingBalance)} USDC available${
            tradingBalanceReady ? '' : ` · needs ≥ ${minTradingUSDC} USDC`
          }`
        : t('hlw.panelStep3MetaFinish', language),
    },
    {
      title: t('hlw.panelStep4Title', language),
      detail: t('hlw.panelStep4Detail', language),
      status: allReady ? 'ready' : 'blocked',
      meta: autopilotTrader?.is_running
        ? t('hlw.panelStep4Running', language)
        : autopilotTrader
          ? t('hlw.panelStep4Ready', language)
          : allReady
            ? t('hlw.panelStep4EverythingReady', language)
            : t('hlw.panelStep4Unlocks', language),
    },
  ]

  const renderPrimaryAction = () => {
    if (!modelReady) {
      return (
        <button
          type="button"
          onClick={() => onOpenModelConfig?.()}
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-nofx-gold px-4 py-3 text-sm font-bold text-white hover:bg-nofx-accent"
        >
          {t('hlw.panelSetupModel', language)}
          <ArrowRight className="h-4 w-4" />
        </button>
      )
    }

    if (!hyperliquidConnected) {
      return (
        <button
          type="button"
          onClick={() => {
            if (onOpenHyperliquidConfig) {
              onOpenHyperliquidConfig()
            } else {
              document
                .getElementById('hyperliquid-quick-connect')
                ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
            }
          }}
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-nofx-gold px-4 py-3 text-sm font-bold text-white hover:bg-nofx-accent"
        >{t('hlw.connect', language)}<ArrowRight className="h-4 w-4" />
        </button>
      )
    }

    if (!tradingBalanceReady) {
      return (
        <a
          href="https://app.hyperliquid.xyz/"
          target="_blank"
          rel="noreferrer"
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-nofx-gold px-4 py-3 text-sm font-bold text-white hover:bg-nofx-accent"
        >
          {t('hlw.panelDepositUsdc', language)}
          <ExternalLink className="h-4 w-4" />
        </a>
      )
    }

    if (autopilotTrader?.is_running) {
      return (
        <button
          type="button"
          onClick={() =>
            navigate(buildDashboardPath(autopilotTrader.trader_id))
          }
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-nofx-success px-4 py-3 text-sm font-bold text-white hover:bg-nofx-success/80"
        >
          {t('hlw.panelOpenDashboard', language)}
          <ArrowRight className="h-4 w-4" />
        </button>
      )
    }

    return (
      <button
        type="button"
        onClick={() => void handleLaunch()}
        disabled={launching || !allReady}
        className="inline-flex items-center justify-center gap-2 rounded-lg bg-nofx-gold px-4 py-3 text-sm font-bold text-white hover:bg-nofx-accent disabled:cursor-not-allowed disabled:opacity-60"
      >
        {launching ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <Zap className="h-4 w-4" />
        )}
        {t('hlw.panelStartAutopilot', language)}
      </button>
    )
  }

  return (
    <section
      id="autopilot-launch-panel"
      className="overflow-hidden rounded-xl border border-nofx-gold/20 bg-nofx-bg-lighter"
    >
      <div className="grid gap-0 xl:grid-cols-[1.05fr_0.95fr]">
        <div className="p-5 md:p-6">
          <div className="mb-5 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <div className="mb-2 inline-flex items-center gap-2 rounded-full border border-nofx-gold/25 bg-nofx-gold/10 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-nofx-gold">
                <ShieldCheck className="h-3.5 w-3.5" />
                {t('hlw.panelGuidedLaunch', language)}
              </div>
              <h2 className="text-2xl font-bold tracking-tight text-nofx-text md:text-3xl">
                {t('hlw.panelTitle', language)}
              </h2>
              <p className="mt-2 max-w-2xl text-sm leading-6 text-nofx-text-muted">
                {t('hlw.panelSubtitle', language)}
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                onClick={() => void refreshEverything()}
                disabled={refreshing}
                className="inline-flex items-center justify-center gap-2 rounded-lg border border-nofx-gold/20 bg-nofx-bg-deeper px-3 py-2 text-xs font-semibold text-nofx-text-muted hover:text-nofx-text disabled:opacity-60"
              >
                <RefreshCw
                  className={`h-3.5 w-3.5 ${refreshing ? 'animate-spin' : ''}`}
                />{t('common.refresh', language)}</button>
              {renderPrimaryAction()}
            </div>
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            {steps.map((step, index) => (
              <div
                key={step.title}
                className="rounded-lg border border-nofx-gold/20 bg-nofx-bg p-4"
              >
                <div className="flex items-start gap-3">
                  <div
                    className={`mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border text-sm font-bold ${
                      step.status === 'ready'
                        ? 'border-nofx-success/30 bg-nofx-success/15 text-nofx-success'
                        : step.status === 'action'
                          ? 'border-nofx-gold/30 bg-nofx-gold/15 text-nofx-gold'
                          : 'border-nofx-gold/20 bg-nofx-bg-deeper text-nofx-text-muted'
                    }`}
                  >
                    {step.status === 'ready' ? (
                      <CheckCircle2 className="h-4 w-4" />
                    ) : step.status === 'action' ? (
                      index + 1
                    ) : (
                      <AlertCircle className="h-4 w-4" />
                    )}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <h3 className="font-semibold text-nofx-text">
                        {step.title}
                      </h3>
                      {step.action}
                    </div>
                    <p className="mt-1 text-xs leading-5 text-nofx-text-muted">
                      {step.detail}
                    </p>
                    <div className="mt-3 font-mono text-xs text-nofx-gold/90">
                      {step.meta}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        <aside className="border-t border-nofx-gold/20 bg-nofx-bg p-5 md:p-6 xl:border-l xl:border-t-0">
            <div className="mb-4 flex items-center gap-2 text-sm font-semibold text-nofx-text">
            <Wallet className="h-4 w-4 text-nofx-gold" />
            {t('hlw.panelHyperliquidSetup', language)}
          </div>
          {hyperliquidConnected ? (
            <div className="rounded-lg border border-nofx-success/25 bg-nofx-success/10 p-4">
              <div className="flex items-center gap-2 text-sm font-semibold text-nofx-success">
                <CheckCircle2 className="h-4 w-4" />
                {t('hlw.panelAuthReady', language)}
              </div>
              <div className="mt-2 font-mono text-xs text-nofx-success/90">
                {shortAddress(hyperliquidExchange?.hyperliquidWalletAddr)}
              </div>
              <p className="mt-3 text-xs leading-5 text-nofx-text-muted">
                {t('hlw.panelFundsStay', language)}
              </p>
            </div>
          ) : (
            <div>
              <div id="hyperliquid-quick-connect">
                <HyperliquidWalletConnect
                  language={isZh ? 'zh' : 'en'}
                  isLoggedIn={isLoggedIn}
                  variant="inline"
                  onSaved={refreshEverything}
                />
              </div>
            </div>
          )}
        </aside>
      </div>
    </section>
  )
}
