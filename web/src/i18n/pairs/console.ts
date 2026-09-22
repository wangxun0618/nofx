import type { PairMap } from '../translation-pair'

/**
 * Copy owned by the settings/console screens and the trader configuration
 * surfaces — toasts, field labels, status badges and confirm buttons.
 */
export const consolePairs: PairMap = {
  // ── Settings page: load failures ───────────────────────────────────────
  'settings.errLoadExchangeBalances': {
    en: 'Failed to load exchange balances',
    zh: '加载交易所余额失败',
  },
  'settings.errLoadAiModels': { en: 'Failed to load AI models', zh: '加载 AI 模型失败' },
  'settings.errLoadExchanges': { en: 'Failed to load exchanges', zh: '加载交易所失败' },

  // ── Settings page: password ────────────────────────────────────────────
  'settings.passwordTooShort': {
    en: 'Password must be at least 8 characters',
    zh: '密码至少需要 8 个字符',
  },
  'settings.errUpdatePassword': { en: 'Failed to update password', zh: '更新密码失败' },
  'settings.passwordUpdated': { en: 'Password updated successfully', zh: '密码已更新' },
  'settings.updatePassword': { en: 'Update Password', zh: '更新密码' },
  'settings.updating': { en: 'Updating...', zh: '更新中…' },

  // ── Settings page: model config ────────────────────────────────────────
  'settings.modelNotFound': { en: 'Model not found', zh: '未找到该模型' },
  'settings.modelSaved': { en: 'Model config saved', zh: '模型配置已保存' },
  'settings.errSaveModel': { en: 'Failed to save model config', zh: '保存模型配置失败' },
  'settings.modelRemoved': { en: 'Model config removed', zh: '模型配置已移除' },
  'settings.errRemoveModel': { en: 'Failed to remove model config', zh: '移除模型配置失败' },

  // ── Settings page: exchange config ─────────────────────────────────────
  'settings.exchangeUpdated': { en: 'Exchange config updated', zh: '交易所配置已更新' },
  'settings.exchangeCreated': { en: 'Exchange account created', zh: '交易所账户已创建' },
  'settings.errSaveExchange': { en: 'Failed to save exchange config', zh: '保存交易所配置失败' },
  'settings.exchangeDeleted': { en: 'Exchange account deleted', zh: '交易所账户已删除' },
  'settings.errDeleteExchange': {
    en: 'Failed to delete exchange account',
    zh: '删除交易所账户失败',
  },

  // ── Settings page: labels and badges ───────────────────────────────────
  'settings.baseUrl': { en: 'Base URL', zh: '接口地址' },
  'settings.tabAiModels': { en: 'AI Models', zh: 'AI 模型' },
  'settings.badgeApiKey': { en: 'API Key', zh: 'API 密钥' },
  'settings.badgeCustomModel': { en: 'Custom Model', zh: '自定义模型' },
  'settings.refreshBalances': { en: 'Refresh Balances', zh: '刷新余额' },
  'settings.refreshing': { en: 'Refreshing…', zh: '刷新中…' },
  'settings.badgeAsterKey': { en: 'Aster Key', zh: 'Aster 密钥' },
  'settings.badgeLighterKey': { en: 'Lighter Key', zh: 'Lighter 密钥' },
  'settings.hidden': { en: 'Hidden', zh: '已隐藏' },
  'settings.shown': { en: 'Shown', zh: '已显示' },

  // ── Model / exchange credential flows ──────────────────────────────────
  'cred.savedReenter': {
    en: 'Saved. Re-enter to replace.',
    zh: '已保存。重新输入可替换。',
  },
  'cred.invalidKey': { en: 'Invalid key', zh: '密钥无效' },
  'cred.validationFailed': { en: 'Validation request failed', zh: '校验请求失败' },
  'cred.keyConfigured': { en: 'API Key configured', zh: '已配置 API 密钥' },
  'cred.keyNotConfigured': { en: 'API Key not configured', zh: '未配置 API 密钥' },
  'cred.apiKeyLabel': { en: 'API Key *', zh: 'API 密钥 *' },

  // ── Key obfuscation modal ──────────────────────────────────────────────
  'key.obfuscationCopied': {
    en: 'Obfuscation string copied to clipboard',
    zh: '混淆字符串已复制到剪贴板',
  },
  'key.copyFailedManual': {
    en: 'Copy failed, please copy the obfuscation string manually',
    zh: '复制失败，请手动复制混淆字符串',
  },
  'key.copyUnsupported': {
    en: 'This browser does not support automatic copy, please copy manually',
    zh: '当前浏览器不支持自动复制，请手动复制',
  },

  // ── Traders list / history ─────────────────────────────────────────────
  'traders.shownInArena': { en: 'Shown in arena', zh: '已在竞技场展示' },
  'traders.hiddenFromArena': { en: 'Hidden from arena', zh: '已从竞技场隐藏' },
  'history.winRate': { en: 'Win Rate', zh: '胜率' },
  'history.errLoad': { en: 'Failed to load history', zh: '加载历史失败' },
  'dashboard.noStrategy': { en: 'No Strategy', zh: '暂无策略' },

  // ── Trader config ──────────────────────────────────────────────────────
  'traderCfg.oiTop': { en: 'OI Top', zh: '持仓量最高' },
  'traderCfg.oiLow': { en: 'OI Low', zh: '持仓量最低' },

  // ── Decision card ──────────────────────────────────────────────────────
  'decision.systemPrompt': { en: 'System Prompt', zh: '系统提示词' },
  'decision.userPrompt': { en: 'User Prompt', zh: '用户提示词' },
  'decision.copied': { en: '{label} copied!', zh: '{label} 已复制！' },

  // ── Auth ───────────────────────────────────────────────────────────────
  'auth.hidePassword': { en: 'Hide password', zh: '隐藏密码' },
  'auth.showPassword': { en: 'Show password', zh: '显示密码' },
  'auth.betaCodeRequired': {
    en: 'A beta code is required to register during the closed beta',
    zh: '内测期间需要邀请码才能注册',
  },
  'auth.registrationServerError': {
    en: 'Registration failed due to server error',
    zh: '服务器错误导致注册失败',
  },

  // ── Autopilot launch panel ─────────────────────────────────────────────
  'autopilot.copied': { en: '{label} copied', zh: '{label} 已复制' },
  'autopilot.copyFailed': { en: 'Copy failed', zh: '复制失败' },
  'autopilot.running': { en: 'NOFX Autopilot is running', zh: 'NOFX Autopilot 正在运行' },
  'autopilot.depositHyperliquid': {
    en: 'Deposit USDC to your Hyperliquid account, the balance check updates automatically.',
    zh: '向你的 Hyperliquid 账户充值 USDC，余额检查会自动更新。',
  },

  // ── Terminal: risk radar ───────────────────────────────────────────────
  'risk.leverage': { en: 'Leverage', zh: '杠杆' },
  'risk.concentration': { en: 'Concentration', zh: '集中度' },
  'risk.drawdown': { en: 'Drawdown', zh: '回撤' },
  'risk.radarSubtitle': {
    en: 'Risk radar · live position-risk check',
    zh: '风险雷达 · 实时持仓风险检查',
  },
  'risk.avg': { en: 'avg', zh: '均值' },
  'risk.peak': { en: 'peak', zh: '峰值' },
  'risk.cap': { en: 'cap', zh: '上限' },
  'risk.ofEquity': { en: 'of equity', zh: '占净值' },
  'risk.topPositionShare': { en: 'top-position share', zh: '最大单仓占比' },
  'risk.peakDrawdown': { en: 'peak drawdown', zh: '峰值回撤' },
  'risk.heldCap': { en: 'held / cap', zh: '已持仓 / 上限' },
  'risk.long': { en: 'long', zh: '多头' },
  'risk.short': { en: 'short', zh: '空头' },
  'risk.net': { en: 'net', zh: '净额' },
  'risk.mark': { en: 'mark', zh: '标记价' },
  'risk.positionCount': { en: 'positions', zh: '个持仓' },
  'risk.liqLevels': { en: 'liq levels', zh: '个强平价' },
  'risk.flat': { en: 'Flat', zh: '空仓' },
  'risk.longLean': { en: 'Long-lean', zh: '偏多' },
  'risk.shortLean': { en: 'Short-lean', zh: '偏空' },
  'risk.balanced': { en: 'Balanced', zh: '均衡' },
  'risk.risky': { en: 'Risky', zh: '危险' },
  'risk.high': { en: 'High', zh: '偏高' },
  'risk.safe': { en: 'Safe', zh: '安全' },
  'risk.tight': { en: 'Tight', zh: '偏紧' },
  'risk.ample': { en: 'Ample', zh: '充裕' },
  'risk.concentrated': { en: 'Concentrated', zh: '集中' },
  'risk.spread': { en: 'Spread', zh: '分散' },
  'risk.calm': { en: 'Calm', zh: '平稳' },
  'risk.deep': { en: 'Deep', zh: '深' },
  'risk.caution': { en: 'Caution', zh: '注意' },
  'risk.full': { en: 'Full', zh: '已满' },
  'risk.room': { en: 'Room', zh: '有余' },
  'risk.netExposure': { en: 'Net exposure', zh: '净敞口' },
  'risk.marginUsed': { en: 'Margin used', zh: '保证金占用' },
  'risk.positions': { en: 'Positions', zh: '持仓数' },
  'risk.unrealizedPnl': { en: 'Unrealized PnL', zh: '未实现盈亏' },

  // ── Model catalog blurbs (plain data module → tg) ──────────────────────
  'model.blurbMostCapableContext': { en: 'Most capable · 1.05M context', zh: '能力最强 · 1.05M 上下文' },
  'model.blurbPreviousFlagship': { en: 'Previous flagship', zh: '上一代旗舰' },
  'model.blurbMostCapable': { en: 'Most capable', zh: '能力最强' },
  'model.blurbCodingFlagship': { en: 'Coding & agents flagship', zh: '编码与 Agent 旗舰' },
  'model.blurbFastGeneral': { en: 'Fast general model', zh: '快速通用模型' },
  'model.blurbAdvancedReasoning': { en: 'Advanced reasoning', zh: '高级推理' },
  'model.blurbDeepReasoning': { en: 'Deep reasoning flagship', zh: '深度推理旗舰' },

  // ── Charts ─────────────────────────────────────────────────────────────
  'chart.bollingerBands': { en: 'Bollinger Bands', zh: '布林带' },
  'chart.bbUpper': { en: 'BB Upper', zh: '布林上轨' },
  'chart.bbMiddle': { en: 'BB Middle', zh: '布林中轨' },
  'chart.bbLower': { en: 'BB Lower', zh: '布林下轨' },
  'chart.errFetchKline': { en: 'Failed to fetch kline data', zh: '获取 K 线数据失败' },
  'chart.errFetchKlineOurs': {
    en: 'Failed to fetch kline data from our service',
    zh: '从本站服务获取 K 线数据失败',
  },
  'chart.errFetchKlinesOurs': {
    en: 'Failed to fetch klines from our service',
    zh: '从本站服务获取 K 线数据失败',
  },
  'chart.errLoadChartData': { en: 'Failed to load chart data', zh: '加载图表数据失败' },
  'chart.errLoadData': { en: 'Failed to load data', zh: '加载数据失败' },
  'chart.errInitChart': { en: 'Failed to initialize chart', zh: '初始化图表失败' },

  // ── Navigation chrome ─────────────────────────────────────────────────
  'nav.closeNavigation': { en: 'Close navigation', zh: '关闭导航' },
  'nav.openNavigation': { en: 'Open navigation', zh: '打开导航' },

  // ── Misc ───────────────────────────────────────────────────────────────
  'branding.tagline': { en: 'AI Trading Platform', zh: 'AI 交易平台' },
  'footer.issues': { en: 'Issues', zh: '问题反馈' },
}
