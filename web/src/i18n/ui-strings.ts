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
  'modelCfg.createWallet': { en: '🔑 Create Wallet', zh: '🔑 创建钱包' },
  'modelCfg.backupNow': {
    en: 'Important: Backup your private key NOW!',
    zh: '重要：请立即备份你的私钥！',
  },
  'modelCfg.privateKeyWarning': {
    en: 'This is your wallet private key. If lost, it cannot be recovered and all assets will be permanently lost. Copy and save it securely.',
    zh: '这是你的钱包私钥。一旦丢失将无法找回，链上资产会永久损失。请复制并妥善保存。',
  },
  'modelCfg.doNotScreenshot': {
    en: 'Do NOT screenshot or share with anyone',
    zh: '请勿截图，也不要分享给任何人',
  },
  'modelCfg.saveToPasswordManager': {
    en: 'Save to a password manager (1Password / Bitwarden)',
    zh: '保存到密码管理器（1Password / Bitwarden）',
  },
  'modelCfg.writeItDown': {
    en: 'Or write it down and store it safely',
    zh: '或手写记录并安全存放',
  },
  'modelCfg.confirmWalletAddress': {
    en: 'Please confirm this is your wallet address (verify in MetaMask)',
    zh: '请确认这是你的钱包地址（可在 MetaMask 中核对）',
  },
  'modelCfg.depositTitle': {
    en: 'Deposit USDC (Base Chain)',
    zh: '充值 USDC（Base 链）',
  },
  'modelCfg.scanQrOrCopy': {
    en: 'Scan QR or copy address to transfer',
    zh: '扫描二维码或复制地址进行转账',
  },
  'modelCfg.scanQrApp': {
    en: 'Scan QR with exchange app to transfer',
    zh: '用交易所 App 扫码转账',
  },
  'modelCfg.chooseBase': {
    en: 'Choose Base network when withdrawing',
    zh: '提现时请选择 Base 网络',
  },
  'modelCfg.orBridge': { en: 'Or bridge: ', zh: '或跨链：' },
  'modelCfg.minUsdc': { en: 'Min $1 USDC to start', zh: '最低 1 USDC 即可开始' },

  // ── Trader configuration ──────────────────────────────────────────────
  'traderCfg.fixedUsStocks': { en: 'Fixed US stocks', zh: '固定美股列表' },
  'traderCfg.vergexSignalBoard': {
    en: 'Vergex signal board',
    zh: 'Vergex 信号榜',
  },
  'traderCfg.claw402Board': { en: 'Claw402 board', zh: 'Claw402 榜单' },
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
  'terminal.marketNetInflow': { en: 'Market net inflow', zh: '市场净流入' },
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
  'terminal.costLiqMap': { en: 'Cost / Liq map', zh: '成本 / 强平地图' },
  'terminal.longCost': { en: 'Long cost', zh: '多头成本' },
  'terminal.shortCost': { en: 'Short cost', zh: '空头成本' },
  'terminal.longLiq': { en: 'Long liq', zh: '多头强平' },
  'terminal.shortLiq': { en: 'Short liq', zh: '空头强平' },
  'terminal.loadingCostLiq': {
    en: 'Loading cost/liquidation map…',
    zh: '正在加载成本/强平地图…',
  },
  'terminal.costLine': { en: 'Cost line', zh: '成本线' },
  'terminal.noSignalData': {
    en: 'No signal data (claw402).',
    zh: '暂无信号数据（claw402）。',
  },
  'terminal.bullish': { en: 'Bullish', zh: '看多' },
  'terminal.bearish': { en: 'Bearish', zh: '看空' },
  'terminal.neutral': { en: 'Neutral', zh: '中性' },
  'terminal.clickToSwitch': { en: 'click to switch ▸', zh: '点击切换 ▸' },
  'terminal.signalMatrix': { en: 'Signal matrix · vergex', zh: '信号矩阵 · vergex' },
  'terminal.noNetFlow': {
    en: 'No net-flow data (claw402 payment required).',
    zh: '暂无净流入数据（需要 claw402 付费）。',
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
  'onboarding.quickstart': { en: 'Quickstart', zh: '快速上手' },
  'onboarding.followSteps': {
    en: 'Follow these 4 steps to get started fast',
    zh: '按这 4 步快速开始',
  },
  'onboarding.baseNetwork': { en: 'Base network', zh: 'Base 网络' },
  'onboarding.continueSetup': { en: 'Continue setup', zh: '继续设置' },
  'onboarding.step1Title': { en: '1. Fast AI', zh: '1. 快速上手 AI' },
  'onboarding.step1Desc': {
    en: 'Start with Claw402 + DeepSeek. No model picking needed for the first run.',
    zh: '先用 Claw402 + DeepSeek 起步，首次运行无需挑选模型。',
  },
  'onboarding.step2Title': { en: '2. Add Exchange', zh: '2. 添加交易所' },
  'onboarding.step2Desc': {
    en: 'Connect an exchange so the AI can actually place trades.',
    zh: '连接交易所，AI 才能真正下单交易。',
  },
  'onboarding.step3Title': { en: '3. Pick Strategy', zh: '3. 选择策略' },
  'onboarding.step3Desc': {
    en: 'You can start with a default strategy and fine-tune later.',
    zh: '可以先使用默认策略，后续再逐步调优。',
  },
  'onboarding.step4Title': { en: '4. Create Trader', zh: '4. 创建交易员' },
  'onboarding.step4Desc': {
    en: 'Last step: bind your model and exchange, then start running.',
    zh: '最后一步：绑定模型与交易所，然后启动运行。',
  },
  'onboarding.walletLabel': { en: 'Wallet', zh: '钱包' },
  'onboarding.payPerCall': {
    en: 'Pay per call with Base USDC',
    zh: '使用 Base 链 USDC 按次付费',
  },
  'onboarding.oneClickSetup': { en: 'One-click setup', zh: '一键配置' },
  'onboarding.strategyReady': { en: 'Strategy ready', zh: '策略已就绪' },
  'onboarding.optionalWorthLook': {
    en: 'Optional, but worth a quick look',
    zh: '可选，但建议快速看一下',
  },
  'onboarding.openStrategy': { en: 'Open strategy', zh: '打开策略' },
  'onboarding.traderCreated': {
    en: 'Trader created, you can add more',
    zh: '交易员已创建，可继续添加',
  },
  'onboarding.readyToCreate': { en: 'Ready to create', zh: '可以创建了' },
  'onboarding.finishFirstThree': {
    en: 'Finish the first three steps first',
    zh: '请先完成前三步',
  },
  'onboarding.createAnother': { en: 'Create another', zh: '再创建一个' },
  'onboarding.createNow': { en: 'Create now', zh: '立即创建' },
  'onboarding.exchangeOptions': {
    en: 'Binance / OKX / Bybit / Hyperliquid',
    zh: '币安 / OKX / Bybit / Hyperliquid',
  },

  // ── Onboarding mode selector ──────────────────────────────────────────
  'mode.experience': { en: 'Experience', zh: '体验模式' },
  'mode.beginnerTitle': { en: 'Beginner Mode', zh: '新手模式' },
  'mode.beginnerBadge': { en: 'Recommended', zh: '推荐' },
  'mode.beginnerDesc': {
    en: 'Generate a Base wallet automatically and start with Claw402 + GLM by default.',
    zh: '自动生成 Base 钱包，默认使用 Claw402 + GLM 起步。',
  },
  'mode.advancedTitle': { en: 'Advanced Mode', zh: '高级模式' },
  'mode.advancedDesc': {
    en: 'Keep the full manual flow and configure models, wallets, and exchanges yourself.',
    zh: '保留完整手动流程，自行配置模型、钱包与交易所。',
  },

  // ── Beginner wallet onboarding overlay ────────────────────────────────
  'onboarding.beginnerGuard': { en: 'Beginner Guard', zh: '新手守护' },
  'onboarding.walletReady': {
    en: 'Your wallet is ready',
    zh: '你的钱包已就绪',
  },
  'onboarding.payPerCallTitle': { en: 'Pay per call', zh: '按次付费' },
  'onboarding.preparingWallet': {
    en: 'Preparing your Base wallet...',
    zh: '正在准备你的 Base 钱包…',
  },
  'onboarding.depositAddress': {
    en: 'Deposit address (Base USDC)',
    zh: '充值地址（Base USDC）',
  },
  'onboarding.refreshBalance': { en: 'Refresh balance', zh: '刷新余额' },
  'onboarding.balanceHint': {
    en: '$5–$10 usually lasts a long time · balance updates by itself after you deposit',
    zh: '5–10 美元通常可用很久 · 充值后余额会自动更新',
  },
  'onboarding.noUsdcYet': { en: "Don't have USDC yet?", zh: '还没有 USDC？' },
  'onboarding.buyUsdcLead': {
    en: 'Buy USDC on Binance, OKX or Coinbase, then withdraw it to the address above — and pick the',
    zh: '可在币安、OKX 或 Coinbase 购买 USDC，然后提现到上方地址 — 提现时请选择',
  },
  'onboarding.buyUsdcTail': {
    en: 'when the exchange asks. It usually arrives in about a minute. Only send USDC on Base.',
    zh: '网络。通常约 1 分钟到账。仅支持通过 Base 网络转入 USDC。',
  },
  'onboarding.walletAddress': { en: 'Wallet address', zh: '钱包地址' },
  'onboarding.address': { en: 'Address', zh: '地址' },
  'onboarding.copyAddress': { en: 'Copy address', zh: '复制地址' },
  'onboarding.privateKeyBackup': {
    en: 'Private key, back it up now',
    zh: '私钥，请立即备份',
  },
  'onboarding.privateKey': { en: 'Private key', zh: '私钥' },
  'onboarding.copyPrivateKey': { en: 'Copy private key', zh: '复制私钥' },
  'onboarding.walletUsageNote': {
    en: 'This wallet only pays for model calls. It does not fund your exchange automatically. The private key cannot be recovered, and you should only deposit Base USDC.',
    zh: '该钱包仅用于支付模型调用费用，不会自动为交易所入金。私钥无法找回，且只应存入 Base 链上的 USDC。',
  },
  'onboarding.copySuccess': { en: '{label} copied', zh: '{label} 已复制' },
  'onboarding.copyFailed': { en: 'Copy failed', zh: '复制失败' },
  'onboarding.skip': { en: 'Skip', zh: '跳过' },
  'onboarding.envSaved': {
    en: 'Wallet details were also saved to {path}',
    zh: '钱包信息也已保存到 {path}',
  },
  'onboarding.prepareWalletFailed': {
    en: 'Failed to prepare beginner wallet',
    zh: '创建新手钱包失败',
  },

  // ── Auth screens (extras) ─────────────────────────────────────────────
  'auth.abortRegistration': { en: 'ABORT_REGISTRATION', zh: '中止注册' },
  'auth.registerIntro': {
    en: 'This account owns your NOFX instance. Next step: a guided launch — about $13 and five minutes to your first AI trade.',
    zh: '该账户是你 NOFX 实例的所有者。下一步是引导式启动：约 13 美元、5 分钟即可完成第一笔 AI 交易。',
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
  'terminal.aiWalletEmpty': {
    en: 'AI fee wallet is out of USDC — decisions are failing.',
    zh: 'AI 费用钱包的 USDC 已耗尽 — 决策正在失败。',
  },
  'terminal.aiWalletLow': {
    en: 'AI fee wallet is low ({amount} USDC) — top up soon.',
    zh: 'AI 费用钱包余额偏低（{amount} USDC）— 请尽快充值。',
  },
  'terminal.safeModeBanner': {
    en: 'Safe mode: AI failed repeatedly, no new positions are being opened.',
    zh: '安全模式：AI 连续失败，暂不开新仓。',
  },
  'terminal.depositToRecover': {
    en: 'Deposit Base USDC to the Claw402 wallet, the trader recovers automatically.',
    zh: '向 Claw402 钱包充值 Base 链 USDC 后，交易程序会自动恢复。',
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
  'terminal.stageFlow': { en: 'FLOW', zh: '资金流' },
  'terminal.stageSignal': { en: 'SIGNAL', zh: '信号' },
  'terminal.stageDecision': { en: 'DECISION', zh: '决策' },
  'terminal.stageExecute': { en: 'EXECUTE', zh: '执行' },
  'terminal.stageHold': { en: 'HOLD', zh: '持仓' },
  'terminal.live': { en: 'live', zh: '实时' },
  'terminal.flat': { en: 'flat', zh: '空仓' },
  'terminal.sync': { en: 'sync', zh: '同步中' },
  'terminal.bars': { en: 'bars', zh: '根' },
  'terminal.liq': { en: 'liq', zh: '强平' },
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
