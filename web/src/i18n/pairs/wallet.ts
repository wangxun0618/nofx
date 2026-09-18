import type { PairMap } from '../translation-pair'

// Wallet / Hyperliquid / Autopilot launch copy. Both languages are required on
// every key. Brand names (Hyperliquid, NOFX, Claw402, MetaMask, Rabby, OKX,
// Coinbase, Base, Arbitrum, USDC) are intentionally preserved.
export const walletPairs: PairMap = {
  // ---------- HyperliquidWalletConnect ----------
  'hlw.noAccountReturned': {
    en: 'Wallet returned no account',
    zh: '钱包未返回任何账户',
  },
  'hlw.walletNotConnected': {
    en: 'Wallet is not connected',
    zh: '钱包尚未连接',
  },
  'hlw.copyFailed': { en: 'Copy failed', zh: '复制失败' },
  'hlw.mainWalletCopied': { en: 'Main wallet copied', zh: '主钱包已复制' },
  'hlw.agentWalletCopied': { en: 'Agent wallet copied', zh: '代理钱包已复制' },
  'hlw.walletAccountChanged': {
    en: 'Wallet account changed. Review and restart authorization.',
    zh: '钱包账户已变更，请检查并重新发起授权。',
  },
  'hlw.walletDisconnected': {
    en: 'Wallet disconnected. Connect a wallet to continue.',
    zh: '钱包已断开连接，请连接钱包以继续。',
  },
  'hlw.failedLoadBalance': {
    en: 'Failed to load Hyperliquid balance',
    zh: '加载 Hyperliquid 余额失败',
  },
  'hlw.connectToBegin': {
    en: 'Connect your wallet to begin',
    zh: '连接钱包以开始',
  },
  'hlw.prepareAccess': {
    en: 'Prepare secure trading access',
    zh: '准备安全交易访问',
  },
  'hlw.approveTradeOnly1': {
    en: 'Approve trade-only access · 1 of 2',
    zh: '授权仅交易访问 · 1/2',
  },
  'hlw.finishAuth2': {
    en: 'Finish authorization · 2 of 2',
    zh: '完成授权 · 2/2',
  },
  'hlw.verifySavedConnection': {
    en: 'Verify the saved connection',
    zh: '验证已保存的连接',
  },
  'hlw.hyperliquidReady': { en: 'Hyperliquid is ready', zh: 'Hyperliquid 已就绪' },
  'hlw.chooseWalletExtension': {
    en: 'Choose the wallet extension you want to connect.',
    zh: '请选择要连接的钱包扩展。',
  },
  'hlw.walletConnectionFailed': {
    en: 'Wallet connection failed',
    zh: '钱包连接失败',
  },
  'hlw.noAccountAvailable': {
    en: 'No account is available in this wallet. Create or import an account in the selected wallet, then try again.',
    zh: '该钱包中没有可用账户，请在所选钱包中创建或导入账户后重试。',
  },
  'hlw.failedGenerateAgent': {
    en: 'Failed to generate agent wallet',
    zh: '生成代理钱包失败',
  },
  'hlw.agentWalletGenerated': {
    en: 'NOFX agent wallet generated',
    zh: 'NOFX 代理钱包已生成',
  },
  'hlw.connectionSavedRefresh': {
    en: 'Connection saved. Refresh the page to update dashboard data.',
    zh: '连接已保存，请刷新页面以更新看板数据。',
  },
  'hlw.agentApproved': {
    en: 'Hyperliquid agent approved',
    zh: 'Hyperliquid 代理已授权',
  },
  'hlw.renewRequiresSignIn': {
    en: 'Renewal requires signing in: Hyperliquid forbids reusing the same agent, so renewal creates a new agent and updates the stored key.',
    zh: '续期需要登录：Hyperliquid 禁止复用同一代理，因此续期会创建新代理并更新已存储的密钥。',
  },
  'hlw.multipleConfigsRenew': {
    en: 'Multiple enabled Hyperliquid configs match this wallet. Select the exact saved connection before renewing.',
    zh: '有多个已启用的 Hyperliquid 配置匹配此钱包，请在续期前选择确切的已保存连接。',
  },
  'hlw.noConfigRenew': {
    en: 'No matching enabled NOFX config was found. Save the connection before renewing its agent.',
    zh: '未找到匹配的已启用 NOFX 配置，请在续期代理前先保存连接。',
  },
  'hlw.agentRenewed': {
    en: 'Agent renewed (new agent, valid 180 days)',
    zh: '代理已续期（新代理，有效期 180 天）',
  },
  'hlw.authSucceededNotSaved': {
    en: 'Wallet authorization succeeded, but NOFX could not save the connection. Use "Save connection" to retry.',
    zh: '钱包授权成功，但 NOFX 无法保存连接。请使用“保存连接”重试。',
  },
  'hlw.pleaseSignInBeforeSave': {
    en: 'Please sign in before saving the agent wallet for trading.',
    zh: '保存代理钱包用于交易前请先登录。',
  },
  'hlw.multipleConfigsSave': {
    en: 'Multiple enabled Hyperliquid configs match this wallet. Select the exact connection before saving.',
    zh: '有多个已启用的 Hyperliquid 配置匹配此钱包，请在保存前选择确切的连接。',
  },
  'hlw.accountUpdatedInNofx': {
    en: 'Hyperliquid account updated in NOFX',
    zh: 'Hyperliquid 账户已在 NOFX 更新',
  },
  'hlw.accountAuthUpdated': {
    en: 'Existing Hyperliquid account authorization updated',
    zh: '现有 Hyperliquid 账户授权已更新',
  },
  'hlw.generateAgentBeforeSave': {
    en: 'Generate and authorize a new agent wallet before saving',
    zh: '保存前请先生成并授权新的代理钱包',
  },
  'hlw.accountSavedToNofx': {
    en: 'Hyperliquid account saved to NOFX',
    zh: 'Hyperliquid 账户已保存到 NOFX',
  },
  'hlw.failedSaveAccount': {
    en: 'Failed to save Hyperliquid account',
    zh: '保存 Hyperliquid 账户失败',
  },
  'hlw.tradeOnlyAccess': {
    en: 'NOFX receives trade-only access; it can never withdraw your funds.',
    zh: 'NOFX 仅获得交易权限，永远无法提取你的资金。',
  },
  'hlw.twoApprovals': {
    en: 'Two wallet approvals: authorize the NOFX Agent, then approve a maximum 0.05% builder fee. Your main wallet key never leaves your wallet.',
    zh: '两次钱包授权：先授权 NOFX 代理，再批准最高 0.05% 的 Builder 费用。你的主钱包私钥永远不会离开你的钱包。',
  },
  'hlw.chooseWallet': { en: 'Choose wallet', zh: '选择钱包' },
  'hlw.reauthorizeTradeOnly': {
    en: 'Re-authorize trade-only access',
    zh: '重新授权仅交易访问',
  },
  'hlw.approveTradeOnly': {
    en: 'Approve trade-only access',
    zh: '授权仅交易访问',
  },
  'hlw.agentLabel': { en: 'Agent', zh: '代理钱包' },
  'hlw.mainLabel': { en: 'Main', zh: '主钱包' },
  'hlw.openHyperliquid': { en: 'Open Hyperliquid', zh: '打开 Hyperliquid' },
  'hlw.reset': { en: 'Reset', zh: '重置' },
  'hlw.progressWallet': { en: 'Wallet', zh: '钱包' },
  'hlw.progressAuthorize': { en: 'Authorize', zh: '授权' },
  'hlw.progressReady': { en: 'Ready', zh: '就绪' },
  'hlw.expected': { en: 'Expected', zh: '预期' },
  'hlw.loading': { en: 'Loading…', zh: '加载中…' },
  'hlw.tradingAuthFinalized': {
    en: 'Trading authorization finalized',
    zh: '交易授权已完成',
  },
  'hlw.agentApprovalFailed': {
    en: 'Agent approval failed',
    zh: '代理授权失败',
  },
  'hlw.agentRenewalFailed': {
    en: 'Agent renewal failed',
    zh: '代理续期失败',
  },
  'hlw.tradingAuthFailed': {
    en: 'Trading authorization failed',
    zh: '交易授权失败',
  },
  'hlw.walletMismatch': {
    en: 'Connected wallet {wallet} does not match the configured Hyperliquid account {expected}. Switch the active account in your wallet extension and retry.',
    zh: '连接的钱包 {wallet} 与已配置的 Hyperliquid 账户 {expected} 不匹配，请在钱包扩展中切换活跃账户后重试。',
  },

  // ---------- TraderLaunchGuestPage ----------
  'hlw.guestCreateAccountTitle': {
    en: 'Create your NOFX account',
    zh: '创建你的 NOFX 账户',
  },
  'hlw.guestCreateAccountDetail': {
    en: 'Your account keeps the Autopilot configuration, wallet authorization state, and trading dashboard in one place.',
    zh: '你的账户将集中托管自动策略配置、钱包授权状态与交易看板，统一于一处。',
  },
  'hlw.guestCreateAccountAction': { en: 'Create account', zh: '创建账户' },
  'hlw.guestFundFeeTitle': { en: 'Fund the AI fee wallet', zh: '为 AI 费用钱包充值' },
  'hlw.guestFundFeeDetail': {
    en: 'NOFX prepares a Base USDC wallet for Claw402.ai data and model calls. This wallet is separate from trading collateral.',
    zh: 'NOFX 会为你准备一个 Base 链上的 USDC 钱包，用于 Claw402.ai 的数据与模型调用。该钱包与交易保证金相互独立。',
  },
  'hlw.guestOpenDepositQr': { en: 'Open deposit QR', zh: '打开充值二维码' },
  'hlw.guestAuthorizeTitle': { en: 'Authorize Hyperliquid', zh: '授权 Hyperliquid' },
  'hlw.guestAuthorizeDetail': {
    en: 'Connect your trading wallet, approve the NOFX Agent, and approve the builder fee. Funds remain in your Hyperliquid account.',
    zh: '连接你的交易钱包，批准 NOFX 代理，并批准 Builder 费用。资金始终保留在你的 Hyperliquid 账户中。',
  },
  'hlw.guestConnectExchange': { en: 'Connect exchange', zh: '连接交易所' },
  'hlw.guestDepositTitle': { en: 'Deposit trading USDC', zh: '存入交易 USDC' },
  'hlw.guestDepositDetail': {
    en: 'Add USDC on Hyperliquid, then start NOFX Autopilot. The strategy is created and launched automatically.',
    zh: '在 Hyperliquid 上存入 USDC，随后启动 NOFX Autopilot。策略将自动创建并运行。',
  },
  'hlw.guestHeadline': {
    en: 'One strategy. Four setup steps. Then it trades.',
    zh: '一套策略，四步设置，然后它开始交易。',
  },
  'hlw.guestIntro': {
    en: 'NOFX runs a single Claw402-driven strategy: board, per-market details, liquidation structure, candles, execution. No strategy picker, no manual symbol picking required.',
    zh: 'NOFX 运行由 Claw402 驱动的统一策略：行情板、逐标的详情、清算结构、K 线与执行。无需选择策略，也无需手动挑选币种。',
  },
  'hlw.guestStartSetup': { en: 'Start setup', zh: '开始设置' },
  'hlw.guestNoWalletTitle': { en: 'No trading wallet yet?', zh: '还没有交易钱包？' },
  'hlw.guestNoWalletDetail': {
    en: 'NOFX does not need your main-wallet private key. Install or unlock an EVM wallet, fund Hyperliquid with USDC, then authorize the NOFX Agent after sign-in.',
    zh: 'NOFX 不需要你的主钱包私钥。安装或解锁一个 EVM 钱包，向 Hyperliquid 充值 USDC，登录后再授权 NOFX 代理。',
  },
  'hlw.guestRabbyDetail': {
    en: 'Create or import an EVM wallet before connecting to Hyperliquid.',
    zh: '在连接 Hyperliquid 之前，请先创建或导入一个 EVM 钱包。',
  },
  'hlw.guestMetaMaskDetail': {
    en: 'Already use MetaMask? Unlock it, then continue setup inside NOFX.',
    zh: '已经在使用 MetaMask？解锁它，然后在 NOFX 内继续设置。',
  },
  'hlw.guestDepositHyperliquidDetail': {
    en: 'Deposit USDC there. Trading funds stay in your Hyperliquid account.',
    zh: '在那里存入 USDC，交易资金保留在你的 Hyperliquid 账户中。',
  },
  'hlw.guestAfterLaunchTitle': { en: 'What runs after launch', zh: '启动后运行什么' },
  'hlw.guestAfterLaunchDetail': {
    en: 'The same production path runs every cycle. The interface only asks you to fund, authorize, and start.',
    zh: '每个周期都运行相同的生产流程，界面只要求你充值、授权并启动。',
  },
  'hlw.guestPipeline1': {
    en: 'Read the live Claw402.ai board, with US stocks prioritized before crypto.',
    zh: '读取 Claw402.ai 实时行情板，美股优先于加密资产。',
  },
  'hlw.guestPipeline2': {
    en: 'Load current direction, direction history, and cost/liquidation structure for each candidate.',
    zh: '加载每个候选标的的当前方向、方向历史，以及成本与清算结构。',
  },
  'hlw.guestPipeline3': {
    en: 'Confirm with raw OHLCV candles, then trade full-size 10x only when the setup is strong enough.',
    zh: '结合原始 OHLCV K 线确认，仅在形态足够强时才以 10 倍满仓交易。',
  },

  // ---------- AutopilotLaunchPanel ----------
  'hlw.panelGuidedLaunch': { en: 'Guided Launch', zh: '引导式启动' },
  'hlw.panelTitle': {
    en: 'Start NOFX Autopilot in minutes',
    zh: '几分钟启动 NOFX Autopilot',
  },
  'hlw.panelSubtitle': {
    en: 'Four small steps, about $13 total. No API keys, no config files — the AI trades for you, and you can stop it anytime.',
    zh: '四个简单步骤，总计约 13 美元。无需 API 密钥、无需配置文件——AI 为你交易，随时可停止。',
  },
  'hlw.deposit': { en: 'Deposit', zh: '充值' },
  'hlw.create': { en: 'Create', zh: '创建' },
  'hlw.panelStep1Title': {
    en: 'Step 1 · Fund the AI wallet ($1+)',
    zh: '步骤 1 · 为 AI 钱包充值（$1 起）',
  },
  'hlw.panelStep1Detail': {
    en: 'The AI pays a tiny fee each time it thinks. Send $1 or more of USDC on the Base network to this address — from Binance, OKX, Coinbase or any wallet. Separate from your trading money.',
    zh: 'AI 每次思考都会支付极少量费用。请通过 Base 网络向此地址转入 $1 或更多 USDC——可从 Binance、OKX、Coinbase 或任意钱包转入。与你的交易资金相互独立。',
  },
  'hlw.panelStep1TakesMinute': {
    en: 'Takes 1 minute — we create the wallet for you',
    zh: '约需 1 分钟——钱包由我们为你创建',
  },
  'hlw.panelStep2Title': {
    en: 'Step 2 · Connect Hyperliquid',
    zh: '步骤 2 · 连接 Hyperliquid',
  },
  'hlw.panelStep2Detail': {
    en: 'Approve NOFX once with your crypto wallet (Rabby or MetaMask). This lets the AI place trades for you — it can never withdraw your money.',
    zh: '用你的加密钱包（Rabby 或 MetaMask）授权 NOFX 一次，即可让 AI 代为下单——但永远无法提取你的资金。',
  },
  'hlw.panelStep2Meta': {
    en: 'A few clicks + 3 wallet signatures',
    zh: '几次点击 + 3 次钱包签名',
  },
  'hlw.open': { en: 'Open', zh: '打开' },
  'hlw.panelStep3Title': {
    en: 'Step 3 · Add trading money ($12+)',
    zh: '步骤 3 · 存入交易资金（$12 起）',
  },
  'hlw.panelStep3Detail': {
    en: 'Deposit USDC into your Hyperliquid account (app.hyperliquid.xyz → Deposit, USDC on Arbitrum). This is what the AI trades with — start small, you can add more anytime.',
    zh: '向你的 Hyperliquid 账户充值 USDC（app.hyperliquid.xyz → 充值，使用 Arbitrum 上的 USDC）。这是 AI 用于交易的资金——先小额开始，后续可随时追加。',
  },
  'hlw.panelStep3MetaFinish': {
    en: 'Finish step 2 first',
    zh: '请先完成步骤 2',
  },
  'hlw.panelStep4Title': { en: 'Step 4 · Press start', zh: '步骤 4 · 点击启动' },
  'hlw.panelStep4Detail': {
    en: 'The AI reads the market every few minutes, picks its trades, and manages them on its own. Watch every decision live on the dashboard — stop it with one click anytime.',
    zh: 'AI 每隔几分钟读取行情、自行选择并管理交易。可在看板上实时查看每一次决策——随时一键停止。',
  },
  'hlw.panelStep4Running': {
    en: 'Running — open the dashboard to watch',
    zh: '运行中——打开看板查看',
  },
  'hlw.panelStep4Ready': { en: 'Ready to start', zh: '准备启动' },
  'hlw.panelStep4EverythingReady': {
    en: 'Everything is ready — press the button',
    zh: '一切就绪——点击按钮即可',
  },
  'hlw.panelStep4Unlocks': {
    en: 'Unlocks when steps 1–3 are green',
    zh: '当步骤 1–3 全部完成时解锁',
  },
  'hlw.panelSetupAiWallet': { en: 'Set up the AI wallet', zh: '设置 AI 钱包' },
  'hlw.panelDepositUsdc': {
    en: 'Deposit USDC on Hyperliquid',
    zh: '在 Hyperliquid 上存入 USDC',
  },
  'hlw.panelOpenDashboard': { en: 'Open dashboard', zh: '打开看板' },
  'hlw.panelStartAutopilot': {
    en: 'Start NOFX Autopilot',
    zh: '启动 NOFX Autopilot',
  },
  'hlw.panelHyperliquidSetup': { en: 'Hyperliquid setup', zh: 'Hyperliquid 设置' },
  'hlw.panelAuthReady': {
    en: 'Trading authorization is ready',
    zh: '交易授权已就绪',
  },
  'hlw.panelFundsStay': {
    en: 'Funds stay in your Hyperliquid account. NOFX only stores the authorized Agent key required for automated execution.',
    zh: '资金保留在你的 Hyperliquid 账户中。NOFX 仅保存用于自动执行的已授权代理密钥。',
  },
}
