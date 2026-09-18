import type { PairMap } from '../translation-pair'

// Populated by the strategy i18n pass.
export const strategyPairs: PairMap = {
  // Bull/Bear Radar + Cost/Liquidation Heatmap
  'strategy.bullBearNotLoaded': {
    en: 'Bull/Bear Radar has not loaded yet.',
    zh: '牛熊雷达尚未加载。',
  },
  'strategy.bullBearRadar': { en: 'Bull/Bear Radar', zh: '牛熊雷达' },
  'strategy.directionChanges': { en: 'Direction changes', zh: '方向变化' },
  'strategy.noDirectionHistory': {
    en: 'No direction-change history returned.',
    zh: '暂无方向变化历史记录。',
  },
  'strategy.heatmapNotLoaded': {
    en: 'Cost/liquidation heatmap has not loaded yet.',
    zh: '持仓成本 / 强平热力图尚未加载。',
  },
  'strategy.costLiquidationHeatmap': {
    en: 'Cost / Liquidation Heatmap',
    zh: '持仓成本 / 强平热力图',
  },
  'strategy.costDistribution': {
    en: 'position cost distribution · liquidation clusters',
    zh: '持仓成本分布 · 清算密集区',
  },
  'strategy.liqSnapshotLag': {
    en: 'Liquidation prices use latest snapshot; incremental trades can lag.',
    zh: '强平价采用最新快照，增量成交可能存在延迟。',
  },
  'strategy.longLiquidation': { en: 'Long liquidation', zh: '多单清算' },
  'strategy.shortLiquidation': { en: 'Short liquidation', zh: '空单清算' },
  'strategy.flushFuelNote': {
    en: 'Long liquidations can force sell into downside breaks.',
    zh: '多单清算可能在下跌破位时触发强制卖出。',
  },
  'strategy.squeezeFuelNote': {
    en: 'Short liquidations can force buy into upside breaks.',
    zh: '空单清算可能在上涨破位时触发强制买入。',
  },
  'strategy.activeBinsNote': {
    en: '{count} active price bins returned.',
    zh: '共返回 {count} 个有效价格区间。',
  },
  'strategy.costPositions': {
    en: '{count} cost positions',
    zh: '{count} 个持仓成本',
  },
  'strategy.liqPrices': {
    en: '{count} liquidation prices',
    zh: '{count} 个强平价',
  },
  'strategy.mainnet': { en: 'mainnet', zh: '主网' },
  'strategy.bandLabel': { en: '±{band}% band', zh: '±{band}% 区间' },
  'strategy.loadingDirectionHeatmap': {
    en: 'Loading direction history and heatmap...',
    zh: '正在加载方向历史与热力图……',
  },
  'strategy.noClawMarkets': {
    en: 'No Claw402 markets available.',
    zh: '暂无可用的 Claw402 市场。',
  },
  'strategy.noMarkets': {
    en: 'No markets available.',
    zh: '暂无可用市场。',
  },

  // Signal Board section
  'strategy.signalBoard': { en: 'Signal Board', zh: '信号看板' },
  'strategy.signalBoardSub': {
    en: 'Live direction board · direction history · liquidation map',
    zh: '实时方向看板 · 方向历史 · 清算地图',
  },
  'strategy.loadClawBoard': {
    en: 'Load Claw402 board',
    zh: '加载 Claw402 看板',
  },
  'strategy.clawBoard': { en: 'Claw402 board', zh: 'Claw402 看板' },
  'strategy.symbolPool': { en: 'Symbol pool', zh: '币种池' },
  'strategy.clearSelected': { en: 'Clear selected', zh: '清除所选' },
  'strategy.followClawDynamically': {
    en: 'Follow Claw402.ai board dynamically',
    zh: '动态跟随 Claw402.ai 看板',
  },
  'strategy.followClawDesc': {
    en: 'At runtime, trade the current range Top {top}; the board refreshes each cycle.',
    zh: '运行时会交易当前区间排名前 {top} 的标的，看板每个周期刷新。',
  },
  'strategy.pinnedUniverse': { en: 'Pinned universe', zh: '锁定标的池' },
  'strategy.pinnedFixed': {
    en: '{count} symbols fixed; trade only these.',
    zh: '已固定 {count} 个标的，仅交易这些。',
  },
  'strategy.pinnedDefault': {
    en: 'Autopilot uses the live Claw402 board by default.',
    zh: '自动领航默认使用实时 Claw402 看板。',
  },
  'strategy.selectedCount': {
    en: '{count} selected',
    zh: '已选 {count} 个',
  },
  'strategy.withoutPicks': {
    en: 'Without manual picks, runtime uses Claw402.ai Top {top} in this range',
    zh: '未手动选择时，运行时使用本区间 Claw402.ai 排名前 {top} 的标的',
  },
  'strategy.topLabel': { en: 'Top {value}', zh: '前 {value}' },
  'strategy.loadSignalBoard': {
    en: 'Load Signal Board',
    zh: '加载信号看板',
  },
  'strategy.noBoardContext': {
    en: 'NOFX Autopilot follows the Claw402 direction board and uses liquidation structure and raw candles as context.',
    zh: 'NOFX 自动领航跟随 Claw402 方向看板，并将清算结构与原始 K 线作为上下文。',
  },

  // Strategy CRUD / actions
  'strategy.autoName': {
    en: 'NOFX Claw402 Auto Strategy',
    zh: 'NOFX Claw402 自动策略',
  },
  'strategy.autoDesc': {
    en: 'The single built-in strategy: read the Claw402.ai board, fetch per-symbol details, then trade with raw candles.',
    zh: '内置单一策略：读取 Claw402.ai 看板，获取单个标的详情，再结合原始 K 线进行交易。',
  },
  'strategy.savedActivated': {
    en: 'Strategy saved and activated',
    zh: '策略已保存并激活',
  },
  'strategy.saved': { en: 'Strategy saved', zh: '策略已保存' },
  'strategy.activated': { en: 'Strategy activated', zh: '策略已激活' },
  'strategy.deleted': { en: 'Strategy deleted', zh: '策略已删除' },
  'strategy.deleteConfirm': {
    en: 'Delete this strategy?',
    zh: '删除该策略？',
  },
  'strategy.confirmDelete': { en: 'Confirm delete', zh: '确认删除' },
  'strategy.delete': { en: 'Delete', zh: '删除' },
  'strategy.cancel': { en: 'Cancel', zh: '取消' },
  'strategy.autopilotTitle': { en: 'NOFX Autopilot', zh: 'NOFX 自动领航' },
  'strategy.autopilotSub': {
    en: 'Autonomous market selection powered by the live Claw402.ai direction board.',
    zh: '由实时 Claw402.ai 方向看板驱动的自主选币。',
  },
  'strategy.launchAutopilot': {
    en: 'Launch Autopilot',
    zh: '启动自动领航',
  },
  'strategy.myStrategies': { en: 'My strategies', zh: '我的策略' },
  'strategy.active': { en: 'Active', zh: '运行中' },
  'strategy.oneLineNote': {
    en: 'One-line strategy note',
    zh: '一句话策略备注',
  },
  'strategy.unsavedChanges': {
    en: 'Unsaved changes',
    zh: '未保存的更改',
  },
  'strategy.saveAndUse': { en: 'Save and use', zh: '保存并使用' },
  'strategy.save': { en: 'Save', zh: '保存' },
  'strategy.activateOnly': { en: 'Activate only', zh: '仅激活' },
  'strategy.advancedSettings': {
    en: 'Advanced settings',
    zh: '高级设置',
  },
  'strategy.tradingStyle': { en: 'Trading style', zh: '交易风格' },
  'strategy.rawCandles': { en: 'Raw candles', zh: '原始 K 线' },
  'strategy.timeframe': { en: 'Timeframe', zh: '时间周期' },
  'strategy.bars': { en: 'Bars', zh: 'K 线数量' },
  'strategy.tradingParameters': {
    en: 'Trading parameters',
    zh: '交易参数',
  },
  'strategy.maxPositions': { en: 'Max positions', zh: '最大持仓数' },
  'strategy.leverage': { en: 'Leverage', zh: '杠杆' },
  'strategy.entryConfidence': {
    en: 'Entry confidence',
    zh: '入场置信度',
  },
  'strategy.strategyNote': { en: 'Strategy note', zh: '策略备注' },
  'strategy.notePlaceholder': {
    en: 'Example: only trade clean trends; skip entries when board signals conflict with candles.',
    zh: '示例：只交易清晰的趋势；当看板信号与 K 线冲突时跳过入场。',
  },
  'strategy.initializeAutopilot': {
    en: 'Initialize Autopilot',
    zh: '初始化自动领航',
  },

  // Default custom prompts (also stored into strategy config)
  'strategy.createCustomPrompt1': {
    en: 'NOFX Autopilot reads the Claw402.ai direction board each cycle, loads direction history and cost/liquidation structure, and holds each position until its board direction changes.',
    zh: 'NOFX 自动领航每个周期读取 Claw402.ai 方向看板，加载方向历史与持仓成本 / 强平结构，并持有每个仓位直到其看板方向发生变化。',
  },
  'strategy.createCustomPrompt2': {
    en: 'Run NOFX Autopilot: use the Claw402.ai direction board as the candidate universe, open in the board direction, and hold until that direction changes, becomes neutral, or disappears.',
    zh: '运行 NOFX 自动领航：以 Claw402.ai 方向看板作为候选标的池，按看板方向开仓，并持有直到该方向改变、转为中性或消失。',
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
  'strategy.errBoardUnavailable': {
    en: 'Claw402.ai board unavailable',
    zh: 'Claw402.ai 看板不可用',
  },
  'strategy.errCreateStrategy': {
    en: 'Failed to create strategy',
    zh: '创建策略失败',
  },
  'strategy.autopilotStarted': {
    en: 'NOFX Autopilot started',
    zh: 'NOFX Autopilot 已启动',
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
  'strategy.errCurrentDir': {
    en: 'Current direction: {msg}',
    zh: '当前方向：{msg}',
  },
  'strategy.errDirHistory': {
    en: 'Direction history: {msg}',
    zh: '方向历史：{msg}',
  },
  'strategy.errHeatmap': { en: 'Heatmap: {msg}', zh: '热力图：{msg}' },

  // Bias labels
  'strategy.biasLong': { en: 'Long Bias', zh: '看多' },
  'strategy.biasShort': { en: 'Short Bias', zh: '看空' },
  'strategy.biasNeutral': { en: 'Neutral', zh: '中性' },
  'strategy.detailsReady': { en: 'details ready', zh: '详情就绪' },
  'strategy.tradeFi': { en: 'TradeFi', zh: 'TradeFi' },
}
