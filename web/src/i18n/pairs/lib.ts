import type { PairMap } from '../translation-pair'

/**
 * Copy owned by the non-React layer: the HTTP client, the API modules, the
 * launch helpers and the small utility functions. These files sit outside the
 * React tree, so they read the ambient language through `tg(key)` from
 * `../translations` rather than receiving a `language` argument.
 *
 * Console-only diagnostics are deliberately absent — they are developer-facing
 * and stay in English.
 */
export const libPairs: PairMap = {
  // ── HTTP client ────────────────────────────────────────────────────────
  'lib.reqTimeout': { en: 'Request timed out', zh: '请求超时' },
  'lib.networkError': { en: 'Network error', zh: '网络错误' },
  'lib.timeoutDetail': {
    en: 'The upstream service took too long to respond',
    zh: '上游服务响应时间过长',
  },
  'lib.unreachableDetail': {
    en: 'Unable to reach the server',
    zh: '无法连接到服务器',
  },
  'lib.sessionExpired': { en: 'Session expired', zh: '登录状态已过期' },
  'lib.permissionDeniedTitle': { en: 'Permission Denied', zh: '权限不足' },
  'lib.permissionDeniedDetail': {
    en: 'You do not have permission to access this resource',
    zh: '你没有访问该资源的权限',
  },
  'lib.permissionDenied': { en: 'Permission denied', zh: '权限不足' },
  'lib.apiNotFoundTitle': { en: 'API Not Found', zh: '接口不存在' },
  'lib.apiNotFoundDetail': {
    en: 'The requested endpoint does not exist (404)',
    zh: '请求的接口不存在（404）',
  },
  'lib.apiNotFound': { en: 'API not found', zh: '接口不存在' },
  'lib.serverErrorTitle': { en: 'Server Error', zh: '服务器错误' },
  'lib.serverErrorDetail': {
    en: 'Please try again later or contact support',
    zh: '请稍后重试，或联系技术支持',
  },
  'lib.serverError': { en: 'Server error', zh: '服务器错误' },
  'lib.operationFailed': { en: 'Operation failed', zh: '操作失败' },
  'lib.requestFailed': { en: 'Request failed', zh: '请求失败' },

  // ── api/data.ts ────────────────────────────────────────────────────────
  'lib.fetchSymbolList': { en: 'Failed to fetch symbol list', zh: '获取交易对列表失败' },
  'lib.fetchSystemStatus': { en: 'Failed to fetch system status', zh: '获取系统状态失败' },
  'lib.fetchAccountInfo': { en: 'Failed to fetch account info', zh: '获取账户信息失败' },
  'lib.fetchPositions': { en: 'Failed to fetch positions', zh: '获取持仓失败' },
  'lib.fetchDecisionLogs': { en: 'Failed to fetch decision logs', zh: '获取决策日志失败' },
  'lib.fetchLatestDecisions': { en: 'Failed to fetch latest decisions', zh: '获取最新决策失败' },
  'lib.fetchStatistics': { en: 'Failed to fetch statistics', zh: '获取统计数据失败' },
  'lib.fetchFullStatistics': { en: 'Failed to fetch full statistics', zh: '获取完整统计数据失败' },
  'lib.fetchKlines': { en: 'Failed to fetch klines', zh: '获取 K 线数据失败' },
  'lib.fetchEquityHistory': { en: 'Failed to fetch equity history', zh: '获取净值历史失败' },
  'lib.fetchBatchEquityHistory': {
    en: 'Failed to fetch batch equity history',
    zh: '获取批量净值历史失败',
  },
  'lib.fetchTopTraders': { en: 'Failed to fetch top traders', zh: '获取排行榜交易员失败' },
  'lib.fetchPublicTraderConfig': {
    en: 'Failed to fetch public trader config',
    zh: '获取公开交易员配置失败',
  },
  'lib.fetchCompetitionData': { en: 'Failed to fetch competition data', zh: '获取竞赛数据失败' },
  'lib.fetchPositionHistory': { en: 'Failed to fetch position history', zh: '获取持仓历史失败' },

  // ── api/config.ts ──────────────────────────────────────────────────────
  'lib.fetchModelConfigs': { en: 'Failed to fetch model configs', zh: '获取模型配置失败' },
  'lib.fetchSupportedModels': { en: 'Failed to fetch supported models', zh: '获取支持的模型列表失败' },
  'lib.fetchPromptTemplates': { en: 'Failed to fetch prompt templates', zh: '获取提示词模板失败' },
  'lib.updateModelConfigs': { en: 'Failed to update model configs', zh: '更新模型配置失败' },
  'lib.fetchExchangeConfigs': { en: 'Failed to fetch exchange configs', zh: '获取交易所配置失败' },
  'lib.fetchExchangeAccountStates': {
    en: 'Failed to fetch exchange account states',
    zh: '获取交易所账户状态失败',
  },
  'lib.fetchSupportedExchanges': {
    en: 'Failed to fetch supported exchanges',
    zh: '获取支持的交易所列表失败',
  },
  'lib.updateExchangeConfigs': { en: 'Failed to update exchange configs', zh: '更新交易所配置失败' },
  'lib.createExchangeAccount': { en: 'Failed to create exchange account', zh: '创建交易所账户失败' },
  'lib.deleteExchangeAccount': { en: 'Failed to delete exchange account', zh: '删除交易所账户失败' },
  'lib.fetchServerIp': { en: 'Failed to fetch server IP', zh: '获取服务器 IP 失败' },

  // ── api/traders.ts ─────────────────────────────────────────────────────
  'lib.fetchTraderList': { en: 'Failed to fetch trader list', zh: '获取交易员列表失败' },
  'lib.fetchPublicTraderList': {
    en: 'Failed to fetch public trader list',
    zh: '获取公开交易员列表失败',
  },
  'lib.createTrader': { en: 'Failed to create trader', zh: '创建交易员失败' },
  'lib.deleteTrader': { en: 'Failed to delete trader', zh: '删除交易员失败' },
  'lib.startTrader': { en: 'Failed to start trader', zh: '启动交易员失败' },
  'lib.stopTrader': { en: 'Failed to stop trader', zh: '停止交易员失败' },
  'lib.updateCompetitionVisibility': {
    en: 'Failed to update competition visibility',
    zh: '更新竞赛可见性失败',
  },
  'lib.closePosition': { en: 'Failed to close position', zh: '平仓失败' },
  'lib.updateCustomPrompt': { en: 'Failed to update custom prompt', zh: '更新自定义提示词失败' },
  'lib.fetchTraderConfig': { en: 'Failed to fetch trader config', zh: '获取交易员配置失败' },
  'lib.updateTrader': { en: 'Failed to update trader', zh: '更新交易员失败' },

  // ── api/strategies.ts ──────────────────────────────────────────────────
  'lib.fetchStrategyList': { en: 'Failed to fetch strategy list', zh: '获取策略列表失败' },
  'lib.fetchStrategy': { en: 'Failed to fetch strategy', zh: '获取策略失败' },
  'lib.fetchActiveStrategy': { en: 'Failed to fetch active strategy', zh: '获取已启用策略失败' },
  'lib.fetchDefaultStrategyConfig': {
    en: 'Failed to fetch default strategy config',
    zh: '获取默认策略配置失败',
  },
  'lib.createStrategy': { en: 'Failed to create strategy', zh: '创建策略失败' },
  'lib.updateStrategy': { en: 'Failed to update strategy', zh: '更新策略失败' },
  'lib.deleteStrategy': { en: 'Failed to delete strategy', zh: '删除策略失败' },
  'lib.activateStrategy': { en: 'Failed to activate strategy', zh: '启用策略失败' },
  'lib.duplicateStrategy': { en: 'Failed to duplicate strategy', zh: '复制策略失败' },

  // ── api/telegram.ts ────────────────────────────────────────────────────
  'lib.fetchTelegramConfig': { en: 'Failed to fetch Telegram config', zh: '获取 Telegram 配置失败' },
  'lib.saveTelegramConfig': { en: 'Failed to save Telegram config', zh: '保存 Telegram 配置失败' },
  'lib.unbindTelegram': { en: 'Failed to unbind Telegram', zh: '解绑 Telegram 失败' },
  'lib.updateTelegramModel': { en: 'Failed to update Telegram model', zh: '更新 Telegram 模型失败' },

  // ── launch helpers ─────────────────────────────────────────────────────
  'lib.noEnabledModel': {
    en: 'No AI model is ready. Add an API key for a supported provider first.',
    zh: '没有可用的 AI 模型。请先为受支持的服务商添加 API Key。',
  },
  'lib.launchNotReady': {
    en: 'Launch prerequisites are not ready yet.',
    zh: '启动前置条件尚未就绪。',
  },
  'lib.launchAutopilotFailed': {
    en: 'Failed to launch NOFX Autopilot',
    zh: '启动 NOFX Autopilot 失败',
  },
  'lib.autopilotStrategyName': {
    en: 'NOFX Auto Strategy',
    zh: 'NOFX 自动交易策略',
  },
  'lib.autopilotStrategyDesc': {
    en: 'Built-in strategy: each cycle it reads the top Hyperliquid instruments by 24h volume, then decides with your AI model.',
    zh: '内置策略：每轮读取 Hyperliquid 24 小时成交量靠前的品种，再交由你的 AI 模型做决策。',
  },
  'lib.createAutopilotStrategyFailed': {
    en: 'Failed to create the autopilot strategy',
    zh: '创建自动交易策略失败',
  },
  'lib.runPreflightFailed': {
    en: 'Failed to run launch preflight',
    zh: '运行启动预检失败',
  },
  'lib.noHyperliquidAccount': {
    en: 'No Hyperliquid account is connected. Connect Hyperliquid and authorize the NOFX agent first.',
    zh: '未连接 Hyperliquid 账户。请先连接 Hyperliquid 并授权 NOFX 代理。',
  },
  'lib.hyperliquidDisabled': {
    en: 'The Hyperliquid account is disabled. Enable it first.',
    zh: 'Hyperliquid 账户已停用，请先启用。',
  },
  'lib.hyperliquidAgentKeyMissing': {
    en: 'The Hyperliquid agent key is missing. Reconnect Hyperliquid and save the agent wallet.',
    zh: '缺少 Hyperliquid 代理密钥。请重新连接 Hyperliquid 并保存代理钱包。',
  },
  'lib.hyperliquidBuilderIncomplete': {
    en: 'Hyperliquid builder authorization is not complete. Finish wallet authorization first.',
    zh: 'Hyperliquid builder 授权尚未完成。请先完成钱包授权。',
  },
  'lib.hyperliquidAddressMissing': {
    en: 'The Hyperliquid wallet address is missing. Reconnect Hyperliquid first.',
    zh: '缺少 Hyperliquid 钱包地址。请先重新连接 Hyperliquid。',
  },

  // ── wallet helpers ─────────────────────────────────────────────────────
  'lib.browserWallet': { en: 'Browser wallet', zh: '浏览器钱包' },
  'lib.walletRequestRejected': {
    en: 'The wallet request was rejected.',
    zh: '钱包请求已被拒绝。',
  },
  'lib.invalidSignatureLength': {
    en: 'Invalid wallet signature length',
    zh: '钱包签名长度无效',
  },
  'lib.invalidChainId': {
    en: 'Wallet returned an invalid chain id',
    zh: '钱包返回的链 ID 无效',
  },
  'lib.invalidSignature': {
    en: 'Wallet returned an invalid signature',
    zh: '钱包返回的签名无效',
  },

  // ── small utilities ────────────────────────────────────────────────────
  'lib.copied': { en: 'Copied', zh: '已复制' },
  'lib.copyFailed': { en: 'Copy failed', zh: '复制失败' },
  'lib.invalidPemPublicKey': {
    en: 'Invalid PEM formatted public key',
    zh: 'PEM 公钥格式无效',
  },
  'lib.cryptoNotInitialized': {
    en: 'Crypto service not initialized. Call initialize() first.',
    zh: '加密服务尚未初始化。请先调用 initialize()。',
  },
  'lib.unknownError': { en: 'Unknown error', zh: '未知错误' },
  'lib.fetchGitHubStats': {
    en: 'Failed to fetch GitHub stats',
    zh: '获取 GitHub 数据失败',
  },

  // ── demo walkthrough engine ────────────────────────────────────────────
  'demo.noteBoardUnchanged': {
    en: 'Momentum is unchanged this cycle; support and resistance still line up with the position.',
    zh: '本周期动能未变；支撑与阻力仍与当前持仓吻合。',
  },
  'demo.noteUsEquityBid': {
    en: 'US-equity tape is broadly bid: SP500 and semis (NVDA, MU, TSM) stay bullish. Positions stay open until the trend breaks.',
    zh: '美股整体买盘积极：SP500 与半导体板块（NVDA、MU、TSM）维持看多。在趋势破坏前，持仓保持不变。',
  },

  // ── auth ───────────────────────────────────────────────────────────────
  'lib.unexpectedLoginResponse': { en: 'Unexpected login response', zh: '登录响应异常' },
  'lib.loginFailedRetry': { en: 'Login failed, please try again', zh: '登录失败，请重试' },
  'lib.loginFailed': { en: 'Login failed', zh: '登录失败' },
  'lib.registrationFailed': { en: 'Registration failed', zh: '注册失败' },
  'lib.detailedServerError': { en: 'Detailed server error', zh: '服务器返回的详细错误' },
  'lib.fetchSystemConfig': { en: 'Failed to fetch system config', zh: '获取系统配置失败' },
}
