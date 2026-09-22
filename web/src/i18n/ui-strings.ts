// Dedicated dictionary for UI strings that previously bypassed `t()` — either
// hardcoded English in JSX, or `language === 'zh' ? 'English' : 'English'`
// ternaries left behind by the English-only conversion.
//
// Every key holds the English and Chinese copy side by side so translation
// parity is structural: adding a key to one language without the other is a
// type error (see the `TranslationPair` type below).
//
// Lookup goes through `t()` in ./translations, which consults this dictionary
// when a key is not found in the main translation table.

import type { TranslationPair } from './translation-pair'
import { consolePairs } from './pairs/console'
import { faqPairs } from './pairs/faq'
import { landingPairs } from './pairs/landing'
import { libPairs } from './pairs/lib'
import { metricsPairs } from './pairs/metrics'
import { strategyPairs } from './pairs/strategy'
import { walletPairs } from './pairs/wallet'

export type { TranslationPair }

type Dict = { [key: string]: string | Dict }

function build<T extends Record<string, TranslationPair>>(pairs: T) {
  const en: Dict = {}
  const zh: Dict = {}
  for (const [key, pair] of Object.entries(pairs)) {
    const [group, leaf] = key.includes('.') ? key.split('.', 2) : ['_root', key]
    if (group === '_root') {
      en[leaf] = pair.en
      zh[leaf] = pair.zh
    } else {
      if (!en[group]) {
        en[group] = {}
        zh[group] = {}
      }
      ;(en[group] as Dict)[leaf] = pair.en
      ;(zh[group] as Dict)[leaf] = pair.zh
    }
  }
  return { en, zh }
}

// ---------------------------------------------------------------------------
// Flat source of truth: one line per string, both languages adjacent.
// ---------------------------------------------------------------------------
export const uiStringPairs = {
  // ── Shared vocabulary ──────────────────────────────────────────────────
  'common.hide': { en: 'Hide', zh: '收起' },
  'common.show': { en: 'Show', zh: '展开' },
  'common.copy': { en: 'Copy', zh: '复制' },
  'common.refresh': { en: 'Refresh', zh: '刷新' },
  'common.copyAddress': { en: '📋 Copy Address', zh: '📋 复制地址' },
  'common.copied': { en: '✅ Copied', zh: '✅ 已复制' },
  'common.copyToClipboard': {
    en: 'Copy to clipboard',
    zh: '复制到剪贴板',
  },
  'common.downloadAsFile': { en: 'Download as file', zh: '下载为文件' },
  'common.perPage': { en: 'Per page', zh: '每页' },
  'common.noDescription': { en: 'No description', zh: '暂无描述' },
  'common.symbol': { en: 'Symbol', zh: '币种' },
  'common.grids': { en: 'Grids', zh: '网格数' },
  'common.searchSymbol': { en: 'Search symbol...', zh: '搜索币种…' },
  'common.deposit': { en: '💳 Deposit', zh: '💳 充值' },
  'common.network': { en: 'Network', zh: '网络' },
  'common.settings': { en: 'Settings', zh: '设置' },

  // Shared buttons and field labels reused across many screens.
  'common.cancel': { en: 'Cancel', zh: '取消' },
  'common.close': { en: 'Close', zh: '关闭' },
  'common.confirm': { en: 'Confirm', zh: '确认' },
  'common.save': { en: 'Save', zh: '保存' },
  'common.delete': { en: 'Delete', zh: '删除' },
  'common.retry': { en: 'Retry', zh: '重试' },
  'common.unknown': { en: 'Unknown', zh: '未知' },
  'common.email': { en: 'Email', zh: '邮箱' },
  'common.password': { en: 'Password', zh: '密码' },
  'common.role': { en: 'Role', zh: '角色' },
  'common.back': { en: 'Back', zh: '返回' },
  'common.next': { en: 'Next', zh: '下一步' },
  'common.done': { en: 'Done', zh: '完成' },
  'common.optional': { en: 'Optional', zh: '可选' },
  'common.required': { en: 'Required', zh: '必填' },
  'common.loading': { en: 'Loading…', zh: '加载中…' },
  'common.search': { en: 'Search', zh: '搜索' },
  'common.requestFailed': { en: 'Request failed', zh: '请求失败' },
  'common.copyFailed': { en: 'Copy failed', zh: '复制失败' },
  'common.unknownError': { en: 'Unknown error', zh: '未知错误' },
  'common.enter': { en: 'Enter', zh: '回车' },
  'common.atLeast8Chars': { en: 'At least 8 characters', zh: '至少 8 位字符' },

  // ── Status chips ───────────────────────────────────────────────────────
  'status.notChecked': { en: 'NOT CHECKED', zh: '未检查' },
  'status.disabled': { en: 'DISABLED', zh: '已停用' },
  'status.incomplete': { en: 'INCOMPLETE', zh: '信息不全' },
  'status.invalidKeys': { en: 'INVALID KEYS', zh: '密钥无效' },
  'status.noPermission': { en: 'NO PERMISSION', zh: '无权限' },
  'status.unavailable': { en: 'UNAVAILABLE', zh: '不可用' },
  'status.standby': { en: 'STANDBY', zh: '待命' },
  'status.checking': { en: 'CHECKING...', zh: '检查中…' },
  'status.ready': { en: 'Ready', zh: '已就绪' },
  'status.pending': { en: 'Pending', zh: '待完成' },
  'status.configured': { en: 'Configured', zh: '已配置' },
  'status.manage': { en: 'Manage', zh: '管理' },
  'status.configure': { en: 'Configure', zh: '去配置' },

  // ── Navigation ─────────────────────────────────────────────────────────
  'nav.data': { en: 'Data', zh: '数据' },
  'nav.market': { en: 'Market', zh: '策略市场' },
  'nav.loginRequired': { en: 'LOGIN_REQ', zh: '需登录' },
  'nav.openNavigation': { en: 'Open navigation', zh: '打开导航' },
  'nav.closeNavigation': { en: 'Close navigation', zh: '关闭导航' },

  // ── Language switcher ──────────────────────────────────────────────────
  'lang.label': { en: 'Language', zh: '语言' },
  'lang.zh': { en: '中文', zh: '中文' },
  'lang.en': { en: 'EN', zh: 'EN' },

  // ── Auth screens ───────────────────────────────────────────────────────
  'auth.terminalOnline': { en: 'Terminal Online', zh: '终端在线' },
  'auth.heroTitleA': { en: 'AI-Powered', zh: 'AI 驱动' },
  'auth.heroTitleB': {
    en: 'Multi-Market Trading Terminal',
    zh: '多市场交易终端',
  },
  'auth.heroTitleC': { en: 'Trading Terminal', zh: '交易终端' },
  'auth.heroDescription': {
    en: 'Plug into 10+ exchanges including Hyperliquid, OKX, Aster, and 7 LLM models. Deploy 24/7 automated strategies with natural language.',
    zh: '接入 Hyperliquid、OKX、Aster 等 10+ 交易所与 7 种大模型，用自然语言部署 24 小时自动运行的交易策略。',
  },
  'auth.exchanges': { en: 'Exchanges', zh: '交易所' },
  'auth.aiModels': { en: 'AI Models', zh: 'AI 模型' },
  'auth.alwaysOn': { en: 'Always On', zh: '全天候运行' },
  'auth.continueWithEmail': {
    en: 'Continue with your email',
    zh: '使用邮箱继续',
  },
  'auth.createYourAccount': { en: 'CREATE YOUR ACCOUNT', zh: '创建你的账户' },
  'auth.initializing': { en: 'INITIALIZING...', zh: '初始化中…' },
  'auth.createAccountTag': { en: 'CREATE_ACCOUNT', zh: '创建账户' },
  'auth.encryption': { en: 'ENCRYPTION: AES-256', zh: '加密算法：AES-256' },
  'auth.secureRegistry': { en: 'SECURE_REGISTRY', zh: '安全注册表' },
  'auth.closeReturnHome': { en: 'Close / Return Home', zh: '关闭 / 返回首页' },
  'auth.noFxLogo': { en: 'NOFX Logo', zh: 'NOFX 标识' },

  // ── Hyperliquid wallet connect ─────────────────────────────────────────
  'hlw.title': { en: 'Hyperliquid Wallet', zh: 'Hyperliquid 钱包' },
  'hlw.connect': { en: 'Connect Hyperliquid', zh: '连接 Hyperliquid' },
  'hlw.connected': { en: 'Connected', zh: '已连接' },
  'hlw.connectWallet': { en: 'Connect your wallet', zh: '连接你的钱包' },
  'hlw.createKey': {
    en: 'Create a trading key for NOFX',
    zh: '为 NOFX 创建交易密钥',
  },
  'hlw.approveInWallet': {
    en: 'Approve it in your wallet (trade-only, cannot withdraw)',
    zh: '在钱包中确认授权（仅可交易，无法提取资金）',
  },
  'hlw.approveBuilderFee': {
    en: 'Approve the small per-trade builder fee',
    zh: '确认每笔交易的小额 builder 费用',
  },
  'hlw.save': { en: 'Save to NOFX — done', zh: '保存到 NOFX — 完成' },
  'hlw.done': {
    en: 'All set — trading authorized',
    zh: '全部就绪 — 已获得交易授权',
  },
  'hlw.balance': { en: 'Hyperliquid balance', zh: 'Hyperliquid 余额' },
  'hlw.withdrawable': { en: 'Withdrawable', zh: '可提取' },
  'hlw.equity': { en: 'Equity', zh: '净值' },
  'hlw.marginUsed': { en: 'Margin used', zh: '已用保证金' },
  'hlw.unrealizedPnl': { en: 'Unrealized PnL', zh: '未实现盈亏' },
  'hlw.noCustody': {
    en: 'Funds stay in your Hyperliquid account; NOFX only stores the authorized agent wallet.',
    zh: '资金始终留在你的 Hyperliquid 账户中，NOFX 只保存已授权的代理钱包。',
  },
  'hlw.agentExpiry': { en: 'Agent authorization expires', zh: '代理授权到期时间' },
  'hlw.expired': { en: 'Expired', zh: '已过期' },
  'hlw.agentNoAuth': {
    en: 'No NOFX agent authorization found',
    zh: '未找到 NOFX 的代理授权',
  },
  'hlw.renewAgent': {
    en: 'Renew agent authorization (+180d)',
    zh: '续期代理授权（+180 天）',
  },
  'hlw.renewHint': {
    en: 'Hyperliquid forbids reusing an agent, so renewal creates a new agent approved for 180 days, then updates the stored key in NOFX (sign-in required).',
    zh: 'Hyperliquid 不允许复用同一个代理，因此续期会创建一个新的 180 天代理，并更新 NOFX 中保存的密钥（需要登录）。',
  },
  'hlw.noWalletTitle': { en: 'No EVM wallet detected', zh: '未检测到 EVM 钱包' },
  'hlw.noWalletDetail': {
    en: 'Install Rabby or MetaMask, create or import a wallet, then return here to connect Hyperliquid.',
    zh: '请先安装 Rabby 或 MetaMask，创建或导入钱包，然后回到这里连接 Hyperliquid。',
  },
  'hlw.noWalletDetailShort': {
    en: 'No EVM wallet detected. Install MetaMask, Rabby, OKX or Coinbase Wallet.',
    zh: '未检测到 EVM 钱包。请安装 MetaMask、Rabby、OKX 或 Coinbase Wallet。',
  },
  'hlw.installRabby': { en: 'Install Rabby', zh: '安装 Rabby' },
  'hlw.installMetaMask': { en: 'Install MetaMask', zh: '安装 MetaMask' },
  'hlw.reAuthorize': { en: 'Re-authorize trading', zh: '重新授权交易' },
  'hlw.manualDisabled': {
    en: 'Manual private-key/API-key entry is disabled. Use MetaMask, Rabby, OKX, Coinbase Wallet or another EVM wallet to connect, authorize the agent, and approve the builder fee.',
    zh: '已停用手动填写私钥 / API Key。请使用 MetaMask、Rabby、OKX、Coinbase Wallet 等 EVM 钱包完成连接、代理授权与 builder 费用确认。',
  },
  'hlw.connectionProgress': { en: 'Connection progress', zh: '连接进度' },
  'hlw.walletExtension': { en: 'Wallet extension', zh: '钱包插件' },
  'hlw.connectStep': { en: 'Connect wallet', zh: '连接钱包' },
  'hlw.prepareStep': { en: 'Prepare secure access', zh: '准备安全访问' },
  'hlw.saveStep': { en: 'Save connection', zh: '保存连接' },
  'hlw.approveFeeStep': { en: 'Approve fee & finish', zh: '确认费用并完成' },

  // ── Model / wallet configuration modal ────────────────────────────────

  // ── Trader configuration ──────────────────────────────────────────────
  'traderCfg.fixedUsStocks': { en: 'Fixed US stocks', zh: '固定美股列表' },
  'traderCfg.hyperRankBoard': {
    en: 'Hyperliquid rank board',
    zh: 'Hyperliquid 排行榜',
  },
  'traderCfg.hlAllMarkets': {
    en: 'Hyperliquid all markets',
    zh: 'Hyperliquid 全市场',
  },
  'traderCfg.hlMainMarkets': {
    en: 'Hyperliquid main markets',
    zh: 'Hyperliquid 主流市场',
  },
  'traderCfg.noDescription': { en: 'No description', zh: '暂无描述' },
  'traderCfg.symbol': { en: 'Symbol', zh: '币种' },
  'traderCfg.grids': { en: 'Grids', zh: '网格数' },
  'traderCfg.perPage': { en: 'Per page', zh: '每页' },
  'traderCfg.hide': { en: 'Hide', zh: '收起' },
  'traderCfg.show': { en: 'Show', zh: '展开' },
  'traderCfg.copy': { en: 'Copy', zh: '复制' },
  'traderCfg.hideTrader': { en: 'Hide trader', zh: '隐藏该交易员' },
  'traderCfg.showTrader': { en: 'Show trader', zh: '显示该交易员' },

  // ── Exchange configuration ────────────────────────────────────────────
  'exchangeCfg.walletAuthRequired': {
    en: 'Hyperliquid requires wallet authorization',
    zh: 'Hyperliquid 需要通过钱包授权',
  },
  'exchangeCfg.useWalletFlow': {
    en: 'Use the wallet authorization flow to connect Hyperliquid.',
    zh: '请使用钱包授权流程连接 Hyperliquid。',
  },
  'exchangeCfg.notManualKeys': {
    en: 'Hyperliquid must be connected through wallet authorization, not manual keys.',
    zh: 'Hyperliquid 必须通过钱包授权连接，不能使用手动填写密钥的方式。',
  },

  // ── Terminal dashboard ────────────────────────────────────────────────
  'terminal.liveTitle': { en: 'Your AI is live.', zh: '你的 AI 正在运行。' },
  'terminal.nextCycle': { en: 'next cycle', zh: '下一轮' },
  'terminal.avgWinLoss': { en: 'avg win/loss', zh: '平均盈/亏' },
  'terminal.orchestrationTopology': {
    en: 'Orchestration topology',
    zh: '编排拓扑',
  },
  'terminal.orchestrationFlow': {
    en: 'Orchestration topology · net inflow → signal → execute → hold',
    zh: '编排拓扑 · 净流入 → 信号 → 执行 → 持仓',
  },
  'terminal.positions': { en: 'Positions', zh: '持仓' },
  'terminal.currentPositionsLive': {
    en: 'Current positions · live',
    zh: '当前持仓 · 实时',
  },
  'terminal.noOpenPositions': { en: 'No open positions.', zh: '当前没有持仓。' },
  'terminal.recentTrades': { en: 'Recent trades', zh: '最近成交' },
  'terminal.recentCloses': {
    en: 'Recent closes · symbol/side/hold/pnl',
    zh: '最近平仓 · 币种/方向/持仓时长/盈亏',
  },
  'terminal.noClosedTrades': { en: 'No closed trades yet.', zh: '暂无已平仓记录。' },
  'terminal.bySymbol': { en: 'By symbol', zh: '按币种' },
  'terminal.bySymbolHistory': {
    en: 'By-symbol history · trades/win/pnl',
    zh: '币种历史 · 成交/胜率/盈亏',
  },
  'terminal.noSymbolHistory': { en: 'No symbol history.', zh: '暂无币种历史。' },
  'terminal.edgeProfile': { en: 'Edge profile', zh: '优势画像' },
  'terminal.edgeProfileDetail': {
    en: 'Net by hold time & side · after fees',
    zh: '按持仓时长与方向统计净收益 · 已扣手续费',
  },
  'terminal.togglePresentation': {
    en: 'toggle presentation mode',
    zh: '切换演示模式',
  },
  'terminal.orderBook': { en: 'Order book', zh: '订单簿' },
  'terminal.connectingHyperliquid': {
    en: 'Connecting to Hyperliquid…',
    zh: '正在连接 Hyperliquid…',
  },
  'terminal.symbol': { en: 'SYMBOL', zh: '币种' },
  'terminal.buySell': { en: 'BUY/SELL', zh: '买/卖' },
  'terminal.trades': { en: 'TRADES', zh: '成交笔数' },
  'terminal.executionLog': { en: 'Execution log', zh: '执行日志' },
  'terminal.noExecutionEvents': {
    en: 'No execution events yet.',
    zh: '暂无执行事件。',
  },
  'terminal.riskRadar': { en: 'Risk radar', zh: '风险雷达' },
  'terminal.noRiskData': { en: 'No live risk data.', zh: '暂无实时风险数据。' },
  'terminal.systemStatusOnline': {
    en: 'SYSTEM_STATUS::ONLINE',
    zh: '系统状态::在线',
  },
  'terminal.aiModel': { en: 'AI Model:', zh: 'AI 模型：' },
  'terminal.exchange': { en: 'Exchange:', zh: '交易所：' },
  'terminal.strategy': { en: 'Strategy:', zh: '策略：' },
  'terminal.cycles': { en: 'Cycles:', zh: '轮次：' },
  'terminal.runtime': { en: 'Runtime:', zh: '运行时长：' },
  'terminal.modelsConfig': { en: 'MODELS_CONFIG', zh: '模型配置' },
  'terminal.exchangeKeys': { en: 'EXCHANGE_KEYS', zh: '交易所密钥' },
  'terminal.telegramBot': { en: 'TELEGRAM_BOT', zh: 'Telegram 机器人' },

  // ── Landing page ──────────────────────────────────────────────────────
  'landing.kernelLatency': { en: 'KERNEL_LATENCY', zh: '内核延迟' },
  'landing.memoryIntegrity': { en: 'MEMORY_INTEGRITY', zh: '内存完整性' },
  'landing.uptime': { en: 'UPTIME', zh: '在线时长' },
  'landing.securityProtocols': { en: 'SECURITY PROTOCOLS', zh: '安全协议' },
  'landing.level3Activate': { en: 'LEVEL 3 ACTIVATE', zh: '三级权限已激活' },
  'landing.agentOs': {
    en: 'NOFX PROFESSIONAL MULTI-ASSET AGENT OS',
    zh: 'NOFX 专业多资产智能体操作系统',
  },
  'landing.trading': { en: 'TRADING', zh: '交易' },
  'landing.createTraderCmd': {
    en: 'create US stock trader --idea="breakouts"',
    zh: '创建美股交易员 --思路="突破"',
  },
  'landing.globalMarketAccess': {
    en: 'GLOBAL MARKET ACCESS',
    zh: '全球市场接入',
  },
  'landing.multiAssetRouting': {
    en: 'MULTI-ASSET ROUTING ENABLED',
    zh: '多资产路由已启用',
  },
  'landing.lowLatencyLink': {
    en: 'LOW LATENCY LINK: 12ms',
    zh: '低延迟链路：12ms',
  },
  'landing.aiModelLine': {
    en: 'AI MODEL: Claude Opus 4.6',
    zh: 'AI 模型：Claude Opus 4.6',
  },
  'landing.traderTerminal': { en: 'NOFX Trader Terminal', zh: 'NOFX 交易员终端' },
  'landing.portfolioPnl': { en: 'Portfolio PnL', zh: '组合盈亏' },
  'landing.netflow': { en: 'Netflow', zh: '净流入' },
  'landing.lsRatio': { en: 'L/S Ratio', zh: '多空比' },
  'landing.orderBook': { en: 'Order Book', zh: '订单簿' },
  'landing.spread': { en: 'Spread:', zh: '点差：' },
  'landing.positions': { en: 'Positions', zh: '持仓' },
  'landing.latency': { en: 'Latency: 12ms', zh: '延迟：12ms' },
  'landing.wsConnStable': { en: 'WS_CONN: STABLE', zh: 'WS 连接：稳定' },
  'landing.tps': { en: 'TPS: 48,291', zh: 'TPS：48,291' },
  'landing.deployInstantly': { en: 'DEPLOY INSTANTLY', zh: '即刻部署' },
  'landing.evolved': { en: 'EVOLVED', zh: '进化版' },
  'landing.degenerates': { en: 'DEGENERATES', zh: '堕落者' },
  'landing.marquee': {
    en: 'NOFX AI TRADING • AUTOMATED WEALTH • DECENTRALIZED INTELLIGENCE • PUNK ETHOS •',
    zh: 'NOFX AI 交易 • 自动化财富 • 去中心化智能 • 朋克精神 •',
  },
  'landing.documentation': { en: 'Documentation', zh: '文档' },

  // ── Charts ────────────────────────────────────────────────────────────
  'chart.binanceKline': { en: 'Binance Kline Data', zh: '币安 K 线数据' },
  'chart.historicalOrders': { en: 'Historical Order Data', zh: '历史订单数据' },
  'chart.status': { en: 'Status', zh: '状态' },
  'chart.viewChart': { en: 'Click to view chart', zh: '点击查看图表' },
  'chart.candlestick': { en: 'Candlestick chart', zh: 'K 线图' },

  // ── Strategy studio ───────────────────────────────────────────────────
  'strategyStudio.flushFuelBelow': { en: 'Flush fuel below', zh: '下方扫除燃料' },
  'strategyStudio.squeezeFuelAbove': {
    en: 'Squeeze fuel above',
    zh: '上方挤压燃料',
  },
  'strategyStudio.binStep': { en: 'Bin step', zh: '价格档位' },

  // ── Onboarding / beginner flow ────────────────────────────────────────
  'onboarding.oneClickSetup': { en: 'One-click setup', zh: '一键配置' },

  // ── Auth screens (extras) ─────────────────────────────────────────────
  'auth.abortRegistration': { en: 'ABORT_REGISTRATION', zh: '中止注册' },
  'auth.registerIntro': {
    en: 'This account owns your NOFX instance. Next step: a guided launch — bring an AI API key, then about $12 and five minutes to your first AI trade.',
    zh: '该账户是你 NOFX 实例的所有者。下一步是引导式启动：准备一个 AI API Key，再约 12 美元、5 分钟即可完成第一笔 AI 交易。',
  },
  'auth.whitelistHeading': { en: 'RESTRICTED', zh: '受限' },
  'auth.accessWord': { en: 'ACCESS', zh: '访问' },

  // ── 404 page ──────────────────────────────────────────────────────────
  'notFound.signalLost': { en: 'SIGNAL_LOST', zh: '信号丢失' },
  'notFound.description': {
    en: 'The requested coordinates do not exist in the current sector. The page may have been moved, deleted, or never existed in this timeline.',
    zh: '请求的坐标在当前扇区中不存在。该页面可能已被移动、删除，或从未存在。',
  },
  'notFound.returnBase': { en: 'RETURN_BASE', zh: '返回首页' },

  // ── Terminal extras ───────────────────────────────────────────────────
  'terminal.loadingMarket': { en: 'Loading market…', zh: '正在加载行情…' },
  'terminal.funnelMatrixAria': {
    en: 'Decision funnel matrix: flow, signal, decision, execute, hold',
    zh: '决策漏斗矩阵：资金流、信号、决策、执行、持仓',
  },

  // Manual position management
  'terminal.closePosition': { en: 'Close position', zh: '平仓' },
  'terminal.close': { en: 'Close', zh: '平仓' },
  'terminal.closeAll': { en: 'Close all', zh: '全部平仓' },
  'terminal.flattenBook': { en: 'Flatten book', zh: '一键平仓' },
  'terminal.closeFailed': { en: 'Close failed', zh: '平仓失败' },
  'terminal.allPositionsClosed': {
    en: 'All positions closed',
    zh: '已平掉全部持仓',
  },
  'terminal.marketCloseOne': {
    en: 'Market-close {symbol} {side}?',
    zh: '市价平掉 {symbol} {side}？',
  },
  'terminal.marketCloseAll': {
    en: 'Market-close ALL {count} open positions?',
    zh: '市价平掉全部 {count} 个持仓？',
  },
  'terminal.closesFailed': {
    en: '{failed}/{total} closes failed',
    zh: '{failed}/{total} 笔平仓失败',
  },

  // Runtime health banners
  'terminal.safeModeBanner': {
    en: 'Safe mode: AI failed repeatedly, no new positions are being opened.',
    zh: '安全模式：AI 连续失败，暂不开新仓。',
  },
  'terminal.firstRunHint': {
    en: 'It reads the whole market before acting — the first decision usually lands within a minute or two and will appear in the Execution Log below. You can stop it anytime from the Config page.',
    zh: '它会先读取全市场数据再行动 — 首个决策通常在一两分钟内产生，并显示在下方执行日志中。你可以随时在配置页停止它。',
  },

  // Metric strip
  'terminal.equityLabel': { en: 'Equity', zh: '净值' },
  'terminal.totalPnl': { en: 'Total P/L · incl. unrealized', zh: '总盈亏 · 含未实现' },
  'terminal.realizedPnl': {
    en: 'Realized P/L · closed trades',
    zh: '已实现盈亏 · 已平仓',
  },
  'terminal.profitFactor': { en: 'Profit factor', zh: '利润因子' },
  'terminal.maxDrawdown': { en: 'Max drawdown', zh: '最大回撤' },

  // Decision funnel stages. The topology renders `title` in English and `zh`
  // for Chinese, so both sides must be real.
  'terminal.stageDecision': { en: 'DECISION', zh: '决策' },
  'terminal.stageExecute': { en: 'EXECUTE', zh: '执行' },
  'terminal.stageHold': { en: 'HOLD', zh: '持仓' },
  'terminal.live': { en: 'live', zh: '实时' },
  'terminal.flat': { en: 'flat', zh: '空仓' },
  'terminal.sync': { en: 'sync', zh: '同步中' },
  'terminal.bars': { en: 'bars', zh: '根' },
  'terminal.liveCandles': { en: 'Live candles', zh: '实时 K 线' },
  'chart.kline': { en: 'Kline', zh: 'K 线' },
} satisfies Record<string, TranslationPair>

const built = build({
  ...uiStringPairs,
  ...faqPairs,
  ...strategyPairs,
  ...walletPairs,
  ...consolePairs,
  ...metricsPairs,
  ...landingPairs,
  ...libPairs,
})

export const uiStrings: { en: Dict; zh: Dict } = { en: built.en, zh: built.zh }

export type UiStrings = typeof uiStrings
