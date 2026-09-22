import type { PairMap } from '../translation-pair'

// Strategy studio + built-in autopilot copy.
export const strategyPairs: PairMap = {
  // Built-in autopilot preset
  'strategy.autoName': {
    en: 'NOFX Auto Strategy',
    zh: 'NOFX 自动策略',
  },
  'strategy.autoDesc': {
    en: 'The built-in strategy: each cycle it reads the top Hyperliquid instruments by 24h volume, then decides with the AI model you connected.',
    zh: '内置策略：每轮读取 Hyperliquid 24 小时成交量靠前的品种，再交由你接入的 AI 模型做决策。',
  },
  'strategy.autopilotStarted': {
    en: 'NOFX Autopilot started',
    zh: 'NOFX Autopilot 已启动',
  },
  'strategy.autopilotSub': {
    en: 'Autonomous market selection driven by Hyperliquid 24h volume and your AI model.',
    zh: '由 Hyperliquid 24 小时成交量与你接入的 AI 模型共同驱动的自主选币。',
  },
  'strategy.initializeAutopilot': {
    en: 'Initialize Autopilot',
    zh: '初始化自动领航',
  },

  // Strategy CRUD / actions
  'strategy.savedActivated': {
    en: 'Strategy saved and activated',
    zh: '策略已保存并激活',
  },
  'strategy.saved': { en: 'Strategy saved', zh: '策略已保存' },
  'strategy.deleted': { en: 'Strategy deleted', zh: '策略已删除' },
  'strategy.deleteConfirm': {
    en: 'Delete this strategy?',
    zh: '删除该策略？',
  },
  'strategy.confirmDelete': { en: 'Confirm delete', zh: '确认删除' },
  'strategy.delete': { en: 'Delete', zh: '删除' },
  'strategy.cancel': { en: 'Cancel', zh: '取消' },

  // Strategy form
  'strategy.entryConfidence': {
    en: 'Entry confidence',
    zh: '入场置信度',
  },
  'strategy.oneLineNote': {
    en: 'One-line strategy note',
    zh: '一句话策略备注',
  },
  'strategy.strategyNote': { en: 'Strategy note', zh: '策略备注' },
  'strategy.notePlaceholder': {
    en: 'Example: only trade clean trends; skip entries when the model signal conflicts with the candles.',
    zh: '示例：只交易清晰的趋势；当模型信号与 K 线冲突时跳过入场。',
  },
  'strategy.tradeFi': { en: 'TradeFi', zh: 'TradeFi' },

  // Default custom prompts (also stored into strategy config)
  'strategy.createCustomPrompt1': {
    en: 'NOFX Autopilot reads the top Hyperliquid instruments by 24h volume each cycle, loads raw candles and position context, and holds each position until its entry thesis no longer holds.',
    zh: 'NOFX 自动领航每轮读取 Hyperliquid 24 小时成交量靠前的品种，加载原始 K 线与持仓上下文，并持有每个仓位直到入场依据不再成立。',
  },
  'strategy.createCustomPrompt2': {
    en: 'Run NOFX Autopilot: use the top Hyperliquid instruments by 24h volume as the candidate universe, and manage each position until its entry thesis no longer holds.',
    zh: '运行 NOFX 自动领航：以 Hyperliquid 24 小时成交量靠前的品种作为候选标的池，并管理每个仓位直到入场依据不再成立。',
  },

  // Errors / toasts
  'strategy.errLoadStrategies': {
    en: 'Failed to load strategies',
    zh: '加载策略失败',
  },
  'strategy.errSymbolList': {
    en: 'Symbol list unavailable',
    zh: '币种列表不可用',
  },
  'strategy.errCreateStrategy': {
    en: 'Failed to create strategy',
    zh: '创建策略失败',
  },
  'strategy.errSaveStrategy': {
    en: 'Failed to save strategy',
    zh: '保存策略失败',
  },
  'strategy.errActivateStrategy': {
    en: 'Failed to activate strategy',
    zh: '激活策略失败',
  },
  'strategy.errDeleteStrategy': {
    en: 'Failed to delete strategy',
    zh: '删除策略失败',
  },
}
