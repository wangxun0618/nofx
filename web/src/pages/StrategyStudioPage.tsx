import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Bot,
  Check,
  Loader2,
  Plus,
  RefreshCw,
  Save,
  Shield,
  Sparkles,
  Trash2,
} from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'
import { DeepVoidBackground } from '../components/common/DeepVoidBackground'
import { api } from '../lib/api'
import { confirmToast, notify } from '../lib/notify'
import type {
  AIStrategyConfig,
  CoinSourceConfig,
  IndicatorConfig,
  RiskControlConfig,
  Strategy,
  StrategyConfig,
} from '../types'
import { launchAutopilot } from '../lib/launch/launchAutopilot'
import type { MarketSymbol } from '../lib/api/data'
import { buildDashboardPath, ROUTES } from '../router/paths'
import { t, type Language } from '../i18n/translations'

const API_BASE = import.meta.env.VITE_API_BASE || ''

type Scope =
  'all' | 'crypto' | 'stock' | 'commodity' | 'index' | 'forex' | 'pre_ipo'

const scopeOptions: Array<{ value: Scope; zh: string; en: string }> = [
  { value: 'all', zh: '全部', en: 'All' },
  { value: 'stock', zh: '美股', en: 'US Stocks' },
  { value: 'crypto', zh: '加密', en: 'Crypto' },
  { value: 'commodity', zh: '大宗商品', en: 'Commodities' },
  { value: 'index', zh: '指数', en: 'Indices' },
  { value: 'forex', zh: '外汇', en: 'FX' },
  { value: 'pre_ipo', zh: 'Pre-IPO', en: 'Pre-IPO' },
]

const timeframeOptions = ['5m', '15m', '30m', '1h', '4h', '1d']
const barCountOptions = [20, 30, 50]
const topNOptions = [5, 8, 10, 20, 30]
const confidenceOptions = [65, 75, 82]

const text = (language: string, zh: string, en: string) =>
  language === 'zh' ? zh : en

type Profile = 'careful' | 'balanced' | 'active'

const profileOptions: Array<{
  value: Profile
  zh: string
  en: string
  zhNote: string
  enNote: string
  maxPositions: number
  leverage: number
  confidence: number
  topN: number
  timeframe: string
  bars: number
  margin: number
  promptZh: string
  promptEn: string
}> = [
  {
    value: 'careful',
    zh: '稳健',
    en: 'Careful',
    zhNote: '交易更少，仅在信号一致时入场',
    enNote: 'Fewer trades, only aligned signals',
    maxPositions: 1,
    leverage: 10,
    confidence: 82,
    topN: 5,
    timeframe: '1h',
    bars: 30,
    margin: 1.0,
    promptZh:
      '稳健模式：跟随 Hyperliquid 成交量靠前的候选品种，结合行情 / 持仓量 / 资金费率生成决策；仅在信号一致时入场。',
    promptEn:
      'Careful mode: follow the Hyperliquid volume leaders and combine market data, open interest, and funding rates to form decisions; only enter on aligned signals.',
  },
  {
    value: 'balanced',
    zh: '均衡',
    en: 'Balanced',
    zhNote: '机会与风险的推荐平衡',
    enNote: 'Recommended balance of opportunity and risk',
    maxPositions: 2,
    leverage: 10,
    confidence: 75,
    topN: 5,
    timeframe: '15m',
    bars: 30,
    margin: 1.0,
    promptZh:
      '均衡模式：优先选择 Hyperliquid 成交量靠前的候选品种，结合行情 / 持仓量 / 资金费率生成决策，并在其趋势保持不变期间持续持有。',
    promptEn:
      'Balanced mode: prioritize the Hyperliquid volume leaders and combine market data, open interest, and funding rates to form decisions, holding while the trend stays intact.',
  },
  {
    value: 'active',
    zh: '激进',
    en: 'Active',
    zhNote: '更快捕捉趋势，持仓更多',
    enNote: 'Faster trend capture with more positions',
    maxPositions: 3,
    leverage: 10,
    confidence: 68,
    topN: 8,
    timeframe: '5m',
    bars: 50,
    margin: 1.0,
    promptZh:
      '激进模式：在 Hyperliquid 成交量靠前的候选品种中快速捕捉趋势，结合行情 / 持仓量 / 资金费率生成决策，并持有直到趋势反转或标的离开候选池。',
    promptEn:
      'Active mode: capture trends quickly among the Hyperliquid volume leaders, combining market data, open interest, and funding rates to form decisions, and hold until the trend reverses or the symbol leaves the universe.',
  },
]

function getAIConfig(config: StrategyConfig): AIStrategyConfig | null {
  if (config.ai_config) return config.ai_config
  if (config.coin_source && config.indicators && config.risk_control) {
    return {
      coin_source: config.coin_source,
      indicators: config.indicators,
      risk_control: config.risk_control,
      prompt_sections: config.prompt_sections,
      custom_prompt: config.custom_prompt,
    }
  }
  return null
}

function defaultCoinSource(
  source?: Partial<CoinSourceConfig>
): CoinSourceConfig {
  const staticCoins = source?.static_coins || []
  // With pinned symbols the strategy trades only those; otherwise it follows the
  // free Hyperliquid native universe (24h volume Top N).
  const useStatic = staticCoins.length > 0
  return {
    source_type: useStatic ? 'static' : 'hyper_main',
    static_coins: staticCoins,
    excluded_coins: [],
    use_ai500: false,
    use_oi_top: false,
    use_oi_low: false,
    use_hyper_all: false,
    use_hyper_main: !useStatic,
    hyper_main_limit: source?.hyper_main_limit || 30,
    hyper_rank_category: source?.hyper_rank_category || 'all',
    hyper_rank_direction: source?.hyper_rank_direction || 'gainers',
    hyper_rank_limit: source?.hyper_rank_limit || 0,
  }
}

function defaultIndicators(
  indicators?: Partial<IndicatorConfig>
): IndicatorConfig {
  const klines = indicators?.klines || {
    primary_timeframe: '15m',
    primary_count: 30,
    enable_multi_timeframe: false,
  }

  return {
    klines: {
      primary_timeframe: klines.primary_timeframe || '15m',
      primary_count: klines.primary_count || 30,
      longer_timeframe: '',
      longer_count: 0,
      enable_multi_timeframe: false,
      selected_timeframes: [klines.primary_timeframe || '15m'],
    },
    enable_raw_klines: true,
    enable_ema: false,
    enable_macd: false,
    enable_rsi: false,
    enable_atr: false,
    enable_boll: false,
    enable_volume: false,
    enable_oi: false,
    enable_funding_rate: false,
    nofxos_api_key: '',
    enable_quant_data: false,
    enable_quant_oi: false,
    enable_quant_netflow: false,
    enable_oi_ranking: false,
    enable_netflow_ranking: false,
    enable_price_ranking: false,
  }
}

function defaultRisk(risk?: Partial<RiskControlConfig>): RiskControlConfig {
  const leverage =
    risk?.altcoin_max_leverage || risk?.btc_eth_max_leverage || 10
  return {
    max_positions: risk?.max_positions || 8,
    btc_eth_max_leverage: leverage,
    altcoin_max_leverage: leverage,
    btc_eth_max_position_value_ratio:
      risk?.btc_eth_max_position_value_ratio || 5,
    altcoin_max_position_value_ratio:
      risk?.altcoin_max_position_value_ratio || 5,
    max_margin_usage: risk?.max_margin_usage || 1.0,
    min_position_size: risk?.min_position_size || 12,
    min_risk_reward_ratio: risk?.min_risk_reward_ratio || 3,
    min_confidence: risk?.min_confidence || 78,
  }
}

function simplifyConfig(
  config: StrategyConfig | null | undefined
): StrategyConfig {
  const ai = config ? getAIConfig(config) : null
  return {
    strategy_type: 'ai_trading',
    language: config?.language || 'zh',
    ai_config: {
      coin_source: defaultCoinSource(ai?.coin_source),
      indicators: defaultIndicators(ai?.indicators),
      risk_control: defaultRisk(ai?.risk_control),
      custom_prompt: ai?.custom_prompt || '',
      prompt_sections: ai?.prompt_sections,
    },
    grid_config: null,
    publish_config: config?.publish_config,
  }
}

function normalizeSymbol(symbol: string) {
  return symbol
    .trim()
    .toUpperCase()
    .replace(/^XYZ:/, '')
    .replace(/-USDC$/, '')
}

function categoryLabel(category: string | undefined, language: Language) {
  const option = scopeOptions.find((item) => item.value === category)
  if (!option) return category || t('strategy.tradeFi', language)
  return text(language, option.zh, option.en)
}

function profileFromRisk(risk: RiskControlConfig | null | undefined): Profile {
  if (!risk) return 'balanced'
  if (risk.min_confidence >= 80 || risk.max_positions <= 1) return 'careful'
  if (risk.altcoin_max_leverage >= 5 || risk.max_positions >= 3) return 'active'
  return 'balanced'
}

function formatChange(value?: number) {
  if (typeof value !== 'number' || Number.isNaN(value)) return ''
  const sign = value > 0 ? '+' : ''
  return `${sign}${value.toFixed(2)}%`
}

export function StrategyStudioPage() {
  const { token } = useAuth()
  const { language } = useLanguage()
  const navigate = useNavigate()
  const [strategies, setStrategies] = useState<Strategy[]>([])
  const [selectedStrategy, setSelectedStrategy] = useState<Strategy | null>(
    null
  )
  const [editingConfig, setEditingConfig] = useState<StrategyConfig | null>(
    null
  )
  const [symbols, setSymbols] = useState<MarketSymbol[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [symbolsLoading, setSymbolsLoading] = useState(false)
  const [symbolsError, setSymbolsError] = useState('')
  const [hasChanges, setHasChanges] = useState(false)

  const aiConfig = editingConfig?.ai_config || null
  const coinSource = aiConfig?.coin_source
  const indicators = aiConfig?.indicators
  const risk = aiConfig?.risk_control
  const selectedSymbols = coinSource?.static_coins || []
  const activeProfile = profileFromRisk(risk)

  const visibleSymbols = useMemo(
    () =>
      symbols
        .filter((item) => item.category !== 'crypto')
        .slice()
        .sort((a, b) => (b.volume_24h || 0) - (a.volume_24h || 0)),
    [symbols]
  )

  const selectedSet = useMemo(
    () => new Set(selectedSymbols.map(normalizeSymbol)),
    [selectedSymbols]
  )

  const loadStrategies = useCallback(
    async (preferredStrategyId?: string) => {
      if (!token) return
      setLoading(true)
      try {
        const result = await api.getStrategies()
        setStrategies(result)
        const next =
          (preferredStrategyId
            ? result.find((item) => item.id === preferredStrategyId)
            : null) ||
          result.find((item) => item.is_active) ||
          result[0] ||
          null
        setSelectedStrategy(next)
        setEditingConfig(next ? simplifyConfig(next.config) : null)
        setHasChanges(false)
      } catch (err) {
        notify.error(
          err instanceof Error
            ? err.message
            : t('strategy.errLoadStrategies', language)
        )
      } finally {
        setLoading(false)
      }
    },
    [token]
  )

  const loadSymbols = useCallback(async () => {
    setSymbolsLoading(true)
    setSymbolsError('')
    try {
      const result = await api.getSymbols('hyperliquid-xyz')
      setSymbols(result.symbols || [])
    } catch (err) {
      setSymbolsError(
        err instanceof Error ? err.message : t('strategy.errSymbolList', language)
      )
    } finally {
      setSymbolsLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadStrategies()
    void loadSymbols()
  }, [loadStrategies, loadSymbols])

  const patchAI = (patch: Partial<AIStrategyConfig>) => {
    setEditingConfig((prev) => {
      const base = simplifyConfig(prev)
      return {
        ...base,
        language: language as 'zh' | 'en',
        ai_config: {
          ...base.ai_config!,
          ...patch,
        },
      }
    })
    setHasChanges(true)
  }

  const patchCoinSource = (patch: Partial<CoinSourceConfig>) => {
    patchAI({
      coin_source: defaultCoinSource({
        ...coinSource,
        ...patch,
      }),
    })
  }

  const patchIndicators = (patch: Partial<IndicatorConfig>) => {
    patchAI({
      indicators: defaultIndicators({
        ...indicators,
        ...patch,
      }),
    })
  }

  const patchRisk = (patch: Partial<RiskControlConfig>) => {
    patchAI({
      risk_control: defaultRisk({
        ...risk,
        ...patch,
      }),
    })
  }

  const createStrategy = async () => {
    if (!token) return
    try {
      const response = await fetch(
        `${API_BASE}/api/strategies/default-config?lang=${language}`,
        { headers: { Authorization: `Bearer ${token}` } }
      )
      const defaultConfig = response.ok
        ? simplifyConfig(await response.json())
        : simplifyConfig(null)
      defaultConfig.language = language as 'zh' | 'en'
      defaultConfig.ai_config = {
        ...defaultConfig.ai_config!,
        coin_source: defaultCoinSource({
          ...defaultConfig.ai_config?.coin_source,
          static_coins: [],
        }),
        indicators: defaultIndicators({
          ...defaultConfig.ai_config?.indicators,
          klines: {
            primary_timeframe: '15m',
            primary_count: 30,
            enable_multi_timeframe: false,
            selected_timeframes: ['15m'],
          },
        }),
        risk_control: defaultRisk({
          ...defaultConfig.ai_config?.risk_control,
          max_positions: 8,
          btc_eth_max_leverage: 10,
          altcoin_max_leverage: 10,
          btc_eth_max_position_value_ratio: 5,
          altcoin_max_position_value_ratio: 5,
          max_margin_usage: 1.0,
          min_confidence: 78,
          min_risk_reward_ratio: 3,
        }),
        custom_prompt: t('strategy.createCustomPrompt1', language),
        prompt_sections: undefined,
      }
      const created = await api.createStrategy({
        name: t('strategy.autoName', language),
        description: t('strategy.autoDesc', language),
        config: defaultConfig,
      })
      await loadStrategies(created.id)
      setHasChanges(false)
    } catch (err) {
      notify.error(
        err instanceof Error
          ? err.message
          : t('strategy.errCreateStrategy', language)
      )
    }
  }

  const saveStrategy = async (
    activateAfter = false,
    overrideConfig?: StrategyConfig,
    successMessage?: string
  ) => {
    if (!token || !selectedStrategy || (!editingConfig && !overrideConfig))
      return
    setSaving(true)
    try {
      const config = simplifyConfig(overrideConfig || editingConfig)
      config.language = language as 'zh' | 'en'
      const response = await fetch(
        `${API_BASE}/api/strategies/${selectedStrategy.id}`,
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({
            name: selectedStrategy.name,
            description: selectedStrategy.description,
            config,
            is_public: selectedStrategy.is_public,
            config_visible: selectedStrategy.config_visible,
          }),
        }
      )
      if (!response.ok) throw new Error(t('strategy.errSaveStrategy', language))
      if (activateAfter) {
        await api.activateStrategy(selectedStrategy.id)
      }
      setHasChanges(false)
      notify.success(
        successMessage ||
          (activateAfter
            ? t('strategy.savedActivated', language)
            : t('strategy.saved', language))
      )
      await loadStrategies(selectedStrategy.id)
    } catch (err) {
      notify.error(
        err instanceof Error
          ? err.message
          : t('strategy.errSaveStrategy', language)
      )
    } finally {
      setSaving(false)
    }
  }

  const buildAutopilotConfig = (): StrategyConfig => {
    const base = simplifyConfig(editingConfig)
    base.language = language as 'zh' | 'en'
    base.ai_config = {
      ...base.ai_config!,
      coin_source: defaultCoinSource({
        ...base.ai_config?.coin_source,
        static_coins: [],
      }),
      indicators: defaultIndicators({
        ...base.ai_config?.indicators,
        klines: {
          primary_timeframe: '15m',
          primary_count: 30,
          enable_multi_timeframe: false,
          selected_timeframes: ['15m'],
        },
      }),
      risk_control: defaultRisk({
        ...base.ai_config?.risk_control,
        max_positions: 8,
        btc_eth_max_leverage: 10,
        altcoin_max_leverage: 10,
        // Five times equity is the hard per-position cap; allocation is dynamic.
        btc_eth_max_position_value_ratio: 5,
        altcoin_max_position_value_ratio: 5,
        max_margin_usage: 1.0,
        min_confidence: 78,
        min_risk_reward_ratio: 3,
      }),
      custom_prompt: t('strategy.createCustomPrompt2', language),
      prompt_sections: undefined,
    }
    return base
  }

  const startAutopilotAgent = async () => {
    if (!selectedStrategy) return

    setSaving(true)
    try {
      // The shared launcher runs the server-side preflight (fresh wallet and
      // exchange balances) BEFORE ensureStrategy, so a failed launch never
      // mutates or activates the strategy as a side effect.
      const outcome = await launchAutopilot({
        scanIntervalMinutes: 15,
        ensureStrategy: async () => {
          const config = buildAutopilotConfig()
          setEditingConfig(config)
          await api.updateStrategy(selectedStrategy.id, {
            name: selectedStrategy.name,
            description:
              selectedStrategy.description ||
              t('strategy.autopilotSub', language),
            config,
          })
          await api.activateStrategy(selectedStrategy.id)
          return selectedStrategy.id
        },
      })

      if (!outcome.ok) {
        notify.error(outcome.message)
        const setupTarget = outcome.kind === 'error' ? null : outcome.setupTarget
        if (setupTarget) {
          navigate(`${ROUTES.traders}?setup=${setupTarget}`)
        }
        return
      }

      if (outcome.warning) {
        notify.warning(outcome.warning)
      }
      notify.success(t('strategy.autopilotStarted', language))
      setHasChanges(false)
      await loadStrategies(selectedStrategy.id)
      navigate(buildDashboardPath(outcome.traderId))
    } finally {
      setSaving(false)
    }
  }

  const activateStrategy = async () => {
    if (!selectedStrategy) return
    try {
      await api.activateStrategy(selectedStrategy.id)
      notify.success(text(language, '策略已启用', 'Strategy activated'))
      await loadStrategies(selectedStrategy.id)
    } catch (err) {
      notify.error(
        err instanceof Error
          ? err.message
          : t('strategy.errActivateStrategy', language)
      )
    }
  }

  const deleteStrategy = async () => {
    if (!selectedStrategy || selectedStrategy.is_active) return
    const ok = await confirmToast(t('strategy.deleteConfirm', language), {
      title: t('strategy.confirmDelete', language),
      okText: t('strategy.delete', language),
      cancelText: t('strategy.cancel', language),
    })
    if (!ok) return
    try {
      await api.deleteStrategy(selectedStrategy.id)
      notify.success(t('strategy.deleted', language))
      await loadStrategies()
    } catch (err) {
      notify.error(
        err instanceof Error
          ? err.message
          : t('strategy.errDeleteStrategy', language)
      )
    }
  }

  const toggleSymbol = (symbol: string) => {
    const normalized = normalizeSymbol(symbol)
    const next = selectedSet.has(normalized)
      ? selectedSymbols.filter((item) => normalizeSymbol(item) !== normalized)
      : [...selectedSymbols, symbol].slice(0, 30)
    patchCoinSource({ static_coins: next })
  }

  const setTimeframe = (timeframe: string) => {
    patchIndicators({
      klines: {
        primary_timeframe: timeframe,
        primary_count: indicators?.klines.primary_count || 30,
        enable_multi_timeframe: false,
        selected_timeframes: [timeframe],
      },
    })
  }

  const setBarCount = (count: number) => {
    patchIndicators({
      klines: {
        primary_timeframe: indicators?.klines.primary_timeframe || '15m',
        primary_count: count,
        enable_multi_timeframe: false,
        selected_timeframes: [indicators?.klines.primary_timeframe || '15m'],
      },
    })
  }

  const setLeverage = (leverage: number) => {
    patchRisk({
      btc_eth_max_leverage: leverage,
      altcoin_max_leverage: leverage,
    })
  }

  const applyProfile = (profile: (typeof profileOptions)[number]) => {
    setEditingConfig((prev) => {
      const base = simplifyConfig(prev)
      const currentAI = base.ai_config!
      return {
        ...base,
        language: language as 'zh' | 'en',
        ai_config: {
          ...currentAI,
          coin_source: defaultCoinSource({
            ...currentAI.coin_source,
            hyper_main_limit: profile.topN,
          }),
          indicators: defaultIndicators({
            ...currentAI.indicators,
            klines: {
              primary_timeframe: profile.timeframe,
              primary_count: profile.bars,
              enable_multi_timeframe: false,
              selected_timeframes: [profile.timeframe],
            },
          }),
          risk_control: defaultRisk({
            ...currentAI.risk_control,
            max_positions: profile.maxPositions,
            btc_eth_max_leverage: profile.leverage,
            altcoin_max_leverage: profile.leverage,
            max_margin_usage: profile.margin,
            min_confidence: profile.confidence,
          }),
          custom_prompt: text(language, profile.promptZh, profile.promptEn),
          prompt_sections: undefined,
        },
      }
    })
    setHasChanges(true)
  }

  if (loading) {
    return (
      <div className="flex min-h-[70vh] items-center justify-center">
        <Loader2 className="h-7 w-7 animate-spin text-nofx-gold" />
      </div>
    )
  }

  return (
    <DeepVoidBackground className="min-h-[calc(100vh-64px)] bg-nofx-bg">
      <div className="border-b border-[rgba(26,24,19,0.14)] bg-nofx-bg/75 px-5 py-4 backdrop-blur">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold text-nofx-text">
              {text(language, 'NOFX Autopilot', 'NOFX Autopilot')}
            </h1>
            <p className="mt-1 text-sm text-nofx-text-muted">
              {t('strategy.autopilotSub', language)}
            </p>
          </div>
          <button
            type="button"
            onClick={startAutopilotAgent}
            disabled={saving || !selectedStrategy}
            className="inline-flex items-center gap-2 rounded-lg bg-nofx-gold px-4 py-2 text-sm font-semibold text-nofx-bg hover:bg-nofx-gold-highlight"
          >
            {saving ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Bot className="h-4 w-4" />
            )}
            {text(language, '启动 Autopilot', 'Launch Autopilot')}
          </button>
        </div>
      </div>

      <div className="grid min-h-[calc(100vh-137px)] grid-cols-1">
        <aside className="hidden border-r border-[rgba(26,24,19,0.14)] bg-nofx-bg-deeper p-3">
          <div className="mb-2 px-2 text-xs font-medium uppercase tracking-wide text-nofx-text-muted">
            {text(language, '我的策略', 'My strategies')}
          </div>
          <div className="space-y-2">
            {strategies.map((strategy) => (
              <button
                key={strategy.id}
                type="button"
                onClick={() => {
                  setSelectedStrategy(strategy)
                  setEditingConfig(simplifyConfig(strategy.config))
                  setHasChanges(false)
                }}
                className={`w-full rounded-lg border px-3 py-3 text-left transition ${
                  selectedStrategy?.id === strategy.id
                    ? 'border-nofx-gold bg-nofx-gold/10'
                    : 'border-[rgba(26,24,19,0.14)] bg-nofx-bg-lighter hover:border-[rgba(26,24,19,0.24)]'
                }`}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="line-clamp-2 text-sm font-medium text-nofx-text">
                    {strategy.name}
                  </span>
                  {strategy.is_active ? (
                    <span className="rounded bg-nofx-success/15 px-1.5 py-0.5 text-[10px] text-nofx-success">
                      {text(language, '已启用', 'Active')}
                    </span>
                  ) : null}
                </div>
                {strategy.description ? (
                  <div className="mt-1 line-clamp-2 text-xs text-nofx-text-muted">
                    {strategy.description}
                  </div>
                ) : null}
              </button>
            ))}
          </div>
        </aside>

        <main className="overflow-y-auto p-5">
          {selectedStrategy && aiConfig && coinSource && indicators && risk ? (
            <div className="mx-auto max-w-7xl space-y-4">
              <section className="hidden rounded-lg border border-[rgba(26,24,19,0.14)] bg-nofx-bg-lighter p-4">
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div className="min-w-0 flex-1">
                    <input
                      value={selectedStrategy.name}
                      onChange={(event) => {
                        setSelectedStrategy({
                          ...selectedStrategy,
                          name: event.target.value,
                        })
                        setHasChanges(true)
                      }}
                      className="w-full bg-transparent text-lg font-semibold text-nofx-text outline-none"
                    />
                    <input
                      value={selectedStrategy.description || ''}
                      onChange={(event) => {
                        setSelectedStrategy({
                          ...selectedStrategy,
                          description: event.target.value,
                        })
                        setHasChanges(true)
                      }}
                      placeholder={t('strategy.oneLineNote', language)}
                      className="mt-1 w-full bg-transparent text-sm text-nofx-text-muted outline-none placeholder:text-nofx-text-muted/50"
                    />
                    {hasChanges ? (
                      <div className="mt-2 text-xs text-nofx-gold">
                        {text(language, '有未保存的修改', 'Unsaved changes')}
                      </div>
                    ) : null}
                  </div>
                  <div className="hidden flex-wrap gap-2">
                    <button
                      type="button"
                      onClick={() => saveStrategy(true)}
                      disabled={saving}
                      className="inline-flex items-center gap-2 rounded-lg bg-nofx-success px-3 py-2 text-sm font-semibold text-nofx-bg disabled:cursor-not-allowed disabled:opacity-45"
                    >
                      {saving ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Check className="h-4 w-4" />
                      )}
                      {text(language, '保存并启用', 'Save and use')}
                    </button>
                    <button
                      type="button"
                      onClick={() => saveStrategy()}
                      disabled={saving || !hasChanges}
                      className="inline-flex items-center gap-2 rounded-lg bg-nofx-gold px-3 py-2 text-sm font-semibold text-nofx-bg disabled:cursor-not-allowed disabled:opacity-45"
                    >
                      {saving ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Save className="h-4 w-4" />
                      )}
                      {text(language, '保存', 'Save')}
                    </button>
                    {!selectedStrategy.is_active ? (
                      <button
                        type="button"
                        onClick={activateStrategy}
                        className="inline-flex items-center gap-2 rounded-lg border border-nofx-success/30 bg-nofx-success/10 px-3 py-2 text-sm text-nofx-success hover:bg-nofx-success/15"
                      >
                        <Check className="h-4 w-4" />
                        {text(language, '仅启用', 'Activate only')}
                      </button>
                    ) : null}
                    {!selectedStrategy.is_active ? (
                      <button
                        type="button"
                        onClick={deleteStrategy}
                        className="inline-flex items-center gap-2 rounded-lg border border-nofx-danger/25 bg-nofx-danger/10 px-3 py-2 text-sm text-nofx-danger hover:bg-nofx-danger/15"
                      >
                        <Trash2 className="h-4 w-4" />
                        {text(language, '删除', 'Delete')}
                      </button>
                    ) : null}
                  </div>
                </div>
              </section>

              <section className="rounded-lg border border-[rgba(26,24,19,0.14)] bg-nofx-bg-lighter p-4">
                <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <div className="flex items-center gap-2 text-sm font-semibold text-nofx-text">
                      <Sparkles className="h-4 w-4 text-nofx-gold" />
                      {text(language, '候选池', 'Candidate universe')}
                    </div>
                    <div className="mt-1 text-xs text-nofx-text-muted">
                      {text(
                        language,
                        '默认使用 Hyperliquid 原生数据源（24h 成交量 Top 30，免费无需 API Key），也可手动锁定标的。',
                        'Defaults to the Hyperliquid native universe (24h volume Top 30, free, no API key). You can also pin symbols manually.'
                      )}
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={() => {
                      void loadSymbols()
                    }}
                    disabled={symbolsLoading}
                    className="inline-flex items-center gap-2 rounded-lg border border-[rgba(26,24,19,0.14)] bg-nofx-bg-deeper px-3 py-2 text-xs text-nofx-text-muted hover:text-nofx-text disabled:opacity-50"
                  >
                    <RefreshCw
                      className={`h-3.5 w-3.5 ${symbolsLoading ? 'animate-spin' : ''}`}
                    />
                    {t('common.refresh', language)}
                  </button>
                </div>

                <div className="mb-4 grid gap-3 md:grid-cols-2">
                  <button
                    type="button"
                    onClick={() => patchCoinSource({ static_coins: [] })}
                    className={`rounded-lg border p-4 text-left transition ${
                      selectedSymbols.length === 0
                        ? 'border-nofx-success bg-nofx-success/10'
                        : 'border-[rgba(26,24,19,0.14)] bg-nofx-bg-deeper hover:border-[rgba(26,24,19,0.24)]'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div className="text-sm font-semibold text-nofx-text">
                        {text(
                          language,
                          'Hyperliquid 原生候选池',
                          'Hyperliquid native universe'
                        )}
                      </div>
                      {selectedSymbols.length === 0 ? (
                        <Check className="h-4 w-4 text-nofx-success" />
                      ) : null}
                    </div>
                    <div className="mt-2 text-xs text-nofx-text-muted">
                      {text(
                        language,
                        `运行时使用 Hyperliquid 24h 成交量前 ${coinSource.hyper_main_limit || 30} 的品种，结合行情 / 持仓量 / 资金费率生成决策。`,
                        `At runtime, trade the Hyperliquid 24h-volume top ${coinSource.hyper_main_limit || 30}, combining market data, open interest, and funding rates.`
                      )}
                    </div>
                  </button>

                  <div
                    className={`rounded-lg border p-4 text-left transition ${
                      selectedSymbols.length > 0
                        ? 'border-nofx-gold bg-nofx-gold/10'
                        : 'border-[rgba(26,24,19,0.14)] bg-nofx-bg-deeper'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div className="text-sm font-semibold text-nofx-text">
                        {text(language, '手动选币', 'Manual picks')}
                      </div>
                      {selectedSymbols.length > 0 ? (
                        <Check className="h-4 w-4 text-nofx-gold" />
                      ) : null}
                    </div>
                    <div className="mt-2 text-xs text-nofx-text-muted">
                      {selectedSymbols.length > 0
                        ? text(
                            language,
                            `已固定 ${selectedSymbols.length} 个标的，仅交易这些。`,
                            `${selectedSymbols.length} symbols pinned; trade only these.`
                          )
                        : text(
                            language,
                            '从下方列表中挑选标的，选中后仅交易这些标的。',
                            'Pick symbols from the list below; once selected only those are traded.'
                          )}
                    </div>
                  </div>
                </div>

                <div className="mb-4 flex flex-wrap items-center gap-3">
                  {selectedSymbols.length === 0 ? (
                    <>
                      <span className="text-sm text-nofx-text-muted">
                        {text(language, '候选数量', 'Universe size')}
                      </span>
                      <select
                        value={coinSource.hyper_main_limit || 30}
                        onChange={(event) =>
                          patchCoinSource({
                            hyper_main_limit: Number(event.target.value),
                          })
                        }
                        className="rounded-lg border border-[rgba(26,24,19,0.14)] bg-nofx-bg px-3 py-2 text-sm text-nofx-text"
                      >
                        {topNOptions.map((value) => (
                          <option key={value} value={value}>
                            {text(language, `前 ${value}`, `Top ${value}`)}
                          </option>
                        ))}
                      </select>
                    </>
                  ) : (
                    <>
                      <span className="text-sm text-nofx-text-muted">
                        {text(
                          language,
                          `已选 ${selectedSymbols.length} 个`,
                          `${selectedSymbols.length} selected`
                        )}
                      </span>
                      <button
                        type="button"
                        onClick={() => patchCoinSource({ static_coins: [] })}
                        className="rounded-lg border border-[rgba(26,24,19,0.14)] px-3 py-2 text-xs text-nofx-text-muted hover:text-nofx-text"
                      >
                        {text(language, '清除所选', 'Clear selected')}
                      </button>
                    </>
                  )}
                </div>

                {symbolsError ? (
                  <div className="mb-4 rounded-lg border border-nofx-gold/20 bg-nofx-gold/10 px-3 py-2 text-xs text-nofx-gold">
                    {symbolsError}
                  </div>
                ) : null}

                <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                  {visibleSymbols.map((item) => {
                    const symbol = normalizeSymbol(item.symbol)
                    const selected = selectedSet.has(symbol)
                    return (
                      <button
                        key={`${item.exchange}-${symbol}`}
                        type="button"
                        onClick={() => toggleSymbol(symbol)}
                        className={`rounded-lg border p-3 text-left transition ${
                          selected
                            ? 'border-nofx-gold bg-nofx-gold/10'
                            : 'border-[rgba(26,24,19,0.14)] bg-nofx-bg-deeper hover:border-[rgba(26,24,19,0.24)]'
                        }`}
                      >
                        <div className="flex items-center justify-between gap-2">
                          <span className="font-mono text-sm font-semibold text-nofx-text">
                            {symbol}
                          </span>
                          <span className="text-[10px] text-nofx-text-muted">
                            {formatChange(item.change_24h_pct)}
                          </span>
                        </div>
                        <div className="mt-2 flex items-center justify-between gap-2 text-[11px] text-nofx-text-muted">
                          <span>{categoryLabel(item.category, language)}</span>
                          <span>
                            {item.mark_price
                              ? `$${item.mark_price.toFixed(2)}`
                              : 'ready'}
                          </span>
                        </div>
                      </button>
                    )
                  })}
                </div>

                {visibleSymbols.length === 0 && !symbolsLoading ? (
                  <div className="rounded-lg border border-[rgba(26,24,19,0.14)] bg-nofx-bg-deeper px-3 py-3 text-sm text-nofx-text-muted">
                    {text(language, '暂无可用市场。', 'No markets available.')}
                  </div>
                ) : null}
              </section>

              <details className="hidden rounded-lg border border-[rgba(26,24,19,0.14)] bg-nofx-bg-deeper p-4">
                <summary className="cursor-pointer text-sm font-semibold text-nofx-text">
                  {text(language, '高级设置', 'Advanced settings')}
                </summary>
                <div className="mt-4 rounded-lg border border-[rgba(26,24,19,0.14)] bg-nofx-bg-lighter p-4">
                  <div className="mb-3 text-sm font-semibold text-nofx-text">
                    {text(language, '交易风格', 'Trading style')}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {profileOptions.map((profile) => (
                      <button
                        key={profile.value}
                        type="button"
                        onClick={() => applyProfile(profile)}
                        className={`rounded-lg border px-3 py-2 text-sm transition ${
                          activeProfile === profile.value
                            ? 'border-nofx-gold bg-nofx-gold/10 text-nofx-gold'
                            : 'border-[rgba(26,24,19,0.14)] text-nofx-text-muted hover:text-nofx-text'
                        }`}
                      >
                        {text(language, profile.zh, profile.en)}
                      </button>
                    ))}
                  </div>
                </div>
                <div className="mt-4 grid gap-4 lg:grid-cols-2">
                  <div className="rounded-lg border border-[rgba(26,24,19,0.14)] bg-nofx-bg-lighter p-4">
                    <div className="mb-4 flex items-center gap-2 text-sm font-semibold text-nofx-text">
                      <Sparkles className="h-4 w-4 text-nofx-gold" />
                      {text(language, '原始 K 线', 'Raw candles')}
                    </div>
                    <div className="space-y-4">
                      <div>
                        <div className="mb-2 text-xs text-nofx-text-muted">
                          {text(language, '时间周期', 'Timeframe')}
                        </div>
                        <div className="flex flex-wrap gap-2">
                          {timeframeOptions.map((timeframe) => (
                            <button
                              key={timeframe}
                              type="button"
                              onClick={() => setTimeframe(timeframe)}
                              className={`rounded-lg border px-3 py-2 text-sm ${
                                indicators.klines.primary_timeframe ===
                                timeframe
                                  ? 'border-nofx-gold bg-nofx-gold/10 text-nofx-gold'
                                  : 'border-[rgba(26,24,19,0.14)] text-nofx-text-muted hover:text-nofx-text'
                              }`}
                            >
                              {timeframe}
                            </button>
                          ))}
                        </div>
                      </div>
                      <div>
                        <div className="mb-2 text-xs text-nofx-text-muted">
                          {text(language, 'K 线根数', 'Bars')}
                        </div>
                        <div className="flex flex-wrap gap-2">
                          {barCountOptions.map((count) => (
                            <button
                              key={count}
                              type="button"
                              onClick={() => setBarCount(count)}
                              className={`rounded-lg border px-3 py-2 text-sm ${
                                indicators.klines.primary_count === count
                                  ? 'border-nofx-gold bg-nofx-gold/10 text-nofx-gold'
                                  : 'border-[rgba(26,24,19,0.14)] text-nofx-text-muted hover:text-nofx-text'
                              }`}
                            >
                              {count}
                            </button>
                          ))}
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="rounded-lg border border-[rgba(26,24,19,0.14)] bg-nofx-bg-lighter p-4">
                    <div className="mb-4 flex items-center gap-2 text-sm font-semibold text-nofx-text">
                      <Shield className="h-4 w-4 text-nofx-success" />
                      {text(language, '交易参数', 'Trading parameters')}
                    </div>
                    <div className="grid gap-4 sm:grid-cols-3">
                      <label className="space-y-2">
                        <span className="text-xs text-nofx-text-muted">
                          {text(language, '最大持仓数', 'Max positions')}
                        </span>
                        <select
                          value={risk.max_positions}
                          onChange={(event) =>
                            patchRisk({
                              max_positions: Number(event.target.value),
                            })
                          }
                          className="w-full rounded-lg border border-[rgba(26,24,19,0.14)] bg-nofx-bg px-3 py-2 text-sm text-nofx-text"
                        >
                          {[1, 2, 3, 4, 5, 6, 7, 8].map((value) => (
                            <option key={value} value={value}>
                              {value}
                            </option>
                          ))}
                        </select>
                      </label>
                      <label className="space-y-2">
                        <span className="text-xs text-nofx-text-muted">
                          {text(language, '杠杆倍数', 'Leverage')}
                        </span>
                        <select
                          value={risk.altcoin_max_leverage}
                          onChange={(event) =>
                            setLeverage(Number(event.target.value))
                          }
                          className="w-full rounded-lg border border-[rgba(26,24,19,0.14)] bg-nofx-bg px-3 py-2 text-sm text-nofx-text"
                        >
                          {[1, 2, 3, 5, 8, 10].map((value) => (
                            <option key={value} value={value}>
                              {value}x
                            </option>
                          ))}
                        </select>
                      </label>
                      <label className="space-y-2">
                        <span className="text-xs text-nofx-text-muted">
                          {t('strategy.entryConfidence', language)}
                        </span>
                        <select
                          value={risk.min_confidence}
                          onChange={(event) =>
                            patchRisk({
                              min_confidence: Number(event.target.value),
                            })
                          }
                          className="w-full rounded-lg border border-[rgba(26,24,19,0.14)] bg-nofx-bg px-3 py-2 text-sm text-nofx-text"
                        >
                          {confidenceOptions.map((value) => (
                            <option key={value} value={value}>
                              {value}%
                            </option>
                          ))}
                        </select>
                      </label>
                    </div>
                  </div>
                </div>

                <div className="mt-4 rounded-lg border border-[rgba(26,24,19,0.14)] bg-nofx-bg-lighter p-4">
                  <div className="mb-2 text-sm font-semibold text-nofx-text">
                    {t('strategy.strategyNote', language)}
                  </div>
                  <textarea
                    value={aiConfig.custom_prompt || ''}
                    onChange={(event) =>
                      patchAI({ custom_prompt: event.target.value })
                    }
                    placeholder={t('strategy.notePlaceholder', language)}
                    className="h-28 w-full resize-none rounded-lg border border-[rgba(26,24,19,0.14)] bg-nofx-bg px-3 py-2 text-sm text-nofx-text outline-none placeholder:text-nofx-text-muted/50"
                  />
                </div>
              </details>
            </div>
          ) : (
            <div className="flex h-full items-center justify-center">
              <button
                type="button"
                onClick={createStrategy}
                className="inline-flex items-center gap-2 rounded-lg bg-nofx-gold px-4 py-2 text-sm font-semibold text-nofx-bg hover:bg-nofx-gold-highlight"
              >
                <Plus className="h-4 w-4" />
                {t('strategy.initializeAutopilot', language)}
              </button>
            </div>
          )}
        </main>
      </div>
    </DeepVoidBackground>
  )
}

export default StrategyStudioPage
