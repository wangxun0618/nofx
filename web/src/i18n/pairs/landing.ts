import type { PairMap } from '../translation-pair'

/**
 * Landing-page copy: the marketing hero, the deployment walkthrough and the
 * feature grid. Brand names (NOFX, Hyperliquid, exchange and model names) are
 * deliberately left untranslated.
 */
export const landingPairs: PairMap = {
  // ── Hero ───────────────────────────────────────────────────────────────
  'landing2.heroSubtitle': {
    en: 'Professional AI trading agents for US stocks, commodities, FX and Pre-IPO synthetic markets. Build institutional-grade strategies by chatting in plain English.',
    zh: '面向美股、大宗商品、外汇与 Pre-IPO 合成市场的专业 AI 交易代理。用自然语言对话，即可构建机构级策略。',
  },
  'landing2.seeItWork': { en: 'SEE IT WORK', zh: '查看实况' },
  'landing2.startTrading': { en: 'Start Trading', zh: '开始交易' },
  'landing2.sourceLink': { en: 'Source', zh: '源码' },
  'landing2.heroPromise': {
    en: 'Self-hosted & open source · about $12 is enough to start · bring your own AI API key · first trade in minutes',
    zh: '自托管、开源 · 约 12 美元即可起步 · 使用你自己的 AI API Key · 几分钟内完成首次交易',
  },

  // Terminal-style network log in the hero mock
  'landing2.logConnecting': {
    en: '> CONNECTING TO MARKET DATA... OK',
    zh: '> 正在连接行情数据... OK',
  },
  'landing2.logSyncingVenues': { en: '> SYNCING VENUES (424/424)... OK', zh: '> 正在同步交易场所（424/424）... OK' },
  'landing2.logLoadingUniverse': {
    en: '> LOADING MULTI-ASSET UNIVERSE... DONE',
    zh: '> 正在加载多资产标的池... DONE',
  },
  'landing2.logAwaitingInput': { en: '> AWAITING USER INPUT_', zh: '> 等待用户输入_' },

  // ── Brand hero ─────────────────────────────────────────────────────────
  'landing2.marquee': {
    en: 'NOFX AI TRADING • AUTOMATED WEALTH • DECENTRALIZED INTELLIGENCE • PUNK ETHOS •',
    zh: 'NOFX AI 交易 • 自动化财富 • 去中心化智能 • PUNK 精神 •',
  },
  'landing2.brandTagline': {
    en: 'Autonomous trading agents. High-frequency execution.',
    zh: '自主交易代理。高频执行。',
  },
  'landing2.brandIntro': {
    en: 'Institutional-grade strategies for the',
    zh: '为',
  },
  'landing2.systemOnline': { en: 'SYSTEM ONLINE', zh: '系统在线' },

  // ── Brand agent terminal ───────────────────────────────────────────────
  'landing2.terminalLive': { en: 'Live', zh: '实时' },
  'landing2.terminalInflow': { en: '24h inflow', zh: '24 小时净流入' },
  'landing2.terminalEnv': { en: 'mainnet • v2.4.0', zh: '主网 • v2.4.0' },

  // ── Deployment hub ─────────────────────────────────────────────────────
  'landing2.systemDeployment': { en: 'System Deployment', zh: '系统部署' },
  'landing2.deployWord': { en: 'DEPLOY', zh: '即刻' },
  'landing2.instantlyWord': { en: 'INSTANTLY', zh: '部署' },
  'landing2.deployIntro': {
    en: 'One command on your laptop or any server installs everything. Open the address it prints, create your account, connect an AI model with your own API key, and the guided launch takes you to your first AI trade in about five minutes — around $12 of trading funds is enough to start.',
    zh: '在你的笔记本或任意服务器上执行一行命令即可完成安装。打开它输出的地址、创建账户，用你自己的 API Key 接入 AI 模型，引导式启动流程会在约五分钟内带你完成首次 AI 交易 — 约 12 美元交易资金即可起步。',
  },
  'landing2.stepRegister': {
    en: 'Register — the first account owns this instance.',
    zh: '注册 — 第一个账户将成为本实例的所有者。',
  },
  'landing2.stepFund': {
    en: 'Add $12+ of trading funds on Hyperliquid — guided, with a deposit link.',
    zh: '在 Hyperliquid 存入 12 美元以上的交易资金 — 有引导，附充值入口。',
  },
  'landing2.stepPressStart': {
    en: 'Press Start — the AI trades on its own; stop it anytime.',
    zh: '点击启动 — AI 会自动交易，随时可以停止。',
  },
  'landing2.oneLineInstall': { en: 'One-Line Install', zh: '一行命令安装' },
  'landing2.oneLineInstallDesc': {
    en: 'Docker handles every dependency',
    zh: '依赖全部由 Docker 处理',
  },
  'landing2.keysStayHome': { en: 'Your Keys Stay Home', zh: '密钥留在本地' },
  'landing2.keysStayHomeDesc': {
    en: 'Runs on your machine, keys encrypted locally',
    zh: '运行在你自己机器上，密钥本地加密',
  },
  'landing2.initProtocol': { en: '# Initialize NoFX Core Protocol', zh: '# 初始化 NoFX 核心协议' },

  // ── Brand features ─────────────────────────────────────────────────────
  'landing2.coreProtocol': { en: 'Core Protocol', zh: '核心协议' },
  'landing2.specs': { en: 'Specs', zh: '规格' },
  'landing2.specsSubtitle': {
    en: 'Next generation infrastructure for algorithmic dominance.',
    zh: '为算法交易优势而生的新一代基础设施。',
  },
  'landing2.featureAiDriven': { en: 'AI DRIVEN', zh: 'AI 驱动' },
  'landing2.featureAiDrivenDesc': {
    en: 'Powered by advanced LLMs (Claude, GPT-4, DeepSeek) to analyze market sentiment and technicals in real-time.',
    zh: '由先进大模型（Claude、GPT-4、DeepSeek）驱动，实时分析市场情绪与技术面。',
  },
  'landing2.featureAutonomous': { en: 'AUTONOMOUS', zh: '自主运行' },
  'landing2.featureAutonomousDesc': {
    en: 'Fully automated trading loops. From data ingestion to order execution without human intervention.',
    zh: '全自动交易闭环。从数据接入到下单执行，全程无需人工干预。',
  },
  'landing2.featurePunkSocial': { en: 'PUNK SOCIAL', zh: 'PUNK 社交' },
  'landing2.featurePunkSocialDesc': {
    en: 'Follow and copy AI traders. A social layer built for the post-human economy.',
    zh: '关注并跟单 AI 交易员。为后人类经济打造的社交层。',
  },
  'landing2.featureNonCustodial': { en: 'NON-CUSTODIAL', zh: '非托管' },
  'landing2.featureNonCustodialDesc': {
    en: 'Your funds, your keys. Connect via API keys or decentralized wallets. We never touch your assets.',
    zh: '资金和密钥都在你手中。可通过 API 密钥或去中心化钱包接入，我们绝不接触你的资产。',
  },
  'landing2.featureHighFrequency': { en: 'HIGH FREQUENCY', zh: '高频' },
  'landing2.featureHighFrequencyDesc': {
    en: 'Event-driven architecture capable of processing thousands of market signals per second.',
    zh: '事件驱动架构，每秒可处理数千条市场信号。',
  },
  'landing2.featureOpenSource': { en: 'OPEN SOURCE', zh: '开源' },
  'landing2.featureOpenSourceDesc': {
    en: 'Auditable codebase. Community driven strategies. Build your own trader upon our core.',
    zh: '代码可审计，策略由社区驱动。你可以在我们的内核之上构建自己的交易员。',
  },

  // ── Agent grid ─────────────────────────────────────────────────────────
  'landing2.agentLargeCap': {
    en: 'Large-cap momentum and breakout trading.',
    zh: '大盘股动量与突破交易。',
  },
  'landing2.agentFx': {
    en: 'FX trend and macro regime allocation.',
    zh: '外汇趋势与宏观周期配置。',
  },
  'landing2.agentPrivate': {
    en: 'Private-market momentum basket engine.',
    zh: '私募市场动量篮子引擎。',
  },
  'landing2.agentGridPitch': {
    en: 'CREATE TRADERS FOR US STOCKS, COMMODITIES, FX AND PRE-IPO MARKETS. DESCRIBE THE STRATEGY IN ONE SENTENCE.',
    zh: '为美股、大宗商品、外汇与 Pre-IPO 市场创建交易员。用一句话描述你的策略。',
  },
  'landing2.winRateShort': { en: 'Win %', zh: '胜率' },

  // ── Footer ─────────────────────────────────────────────────────────────
  'landing2.pullRequests': { en: 'Pull Requests', zh: '拉取请求' },
}
