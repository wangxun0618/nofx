# 更新日志

NOFX 项目的所有重要更改都将记录在此文件中。

本文件格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

**语言:** [English](CHANGELOG.md) | [中文](CHANGELOG.zh-CN.md)

---

## [未发布]

### 新增
- 多语言文档系统（英文/中文/俄语/乌克兰语）
- 完整的快速开始指南（Docker、自定义 API）
- 架构文档，包含系统设计细节
- 用户指南，包含 FAQ 和故障排除
- 社区文档，包含悬赏计划
- **可插拔市场情报层**（`marketdata`）：`Provider` / `Registry` / `Insight` 抽象，新增数据源只需实现接口并登记到 `marketdata/providers/registry.go`，引擎、交易循环与提示词构建器均无需改动
- 两个完全免费、无需 API Key 的内置数据源：`hyperliquid_flow`（跨市场资金流：成交额、涨跌幅、资金费率拥挤度）与 `hyperliquid_leverage`（持仓量结构、杠杆拥挤度、mark 对 oracle 偏离）
- 可选数据源 `coinank_liquidation`：由逐笔强平记录在本地重建**价格分桶强平簇**，替代原 Vergex 强平热力图；填入 CoinAnk API Key 后自动启用
- 新增 API：`GET /api/market-insights/providers`（列出全部数据源）与 `GET /api/market-insights`（抓取数据）
- 策略工作室「市场数据源」配置区：总开关、数据源卡片（含「免费 / 需要 API Key」徽章）、榜单行数与凭据输入
- 终端新增「跨市场资金流」与「持仓结构」面板，渲染的正是 AI 决策时读取的同一份数据
- **方向信号源 `directional_signal`**：由价格动量、永续溢价与激进订单流三路投票，本地计算每个品种的 `bias` 与信号强度 `score`，并把**每一个分量逐条列出**供审计——用于替代原 Vergex 方向榜，但**不声明为权威结论**
- **方向变更时间线**：`marketdata.DirectionTracker` 记录 bias 翻转（含 X→Y 与由分量票 diff 算出的原因），落盘到 `data/direction_history.json` 并跨重启存活
- **HyperData Terminal 侧车接入**：新增 `hyperdata_orderflow`（多交易所 CVD 逐所归因、账户多空比、基差）与 `hyperdata_positioning`（最大被追踪仓位、距强平距离）两个可选数据源，通过 HTTP 消费该独立开源项目的 REST API
- **上游升级机制**：`deploy/hyperdata/` 下提供钉住 commit 的 `upstream.env`、`sidecar.sh`（install / update / status / contract / run）、会拒绝构建偏离 pin 代码的 `Dockerfile`；`docker-compose.hyperdata.yml` 可把侧车作为独立服务并行部署
- **活体上游漂移报告**（`TestHyperDataUpstreamHasNotDrifted`）：探测运行中的侧车，把与录制 fixture 的每处差异分类为 `MISSING` / `TYPE CHANGED`（破坏性）或 `ADDED`（信息性），使升级在采纳前就能被分诊。它做成一个 Go 测试而不是第二个探测脚本，因此 HTTP 契约只有一份定义，而不是两份可能互相矛盾的东西
- **上游版本钉是被强制的，而不只是文档**：`TestUpstreamPinIsStillMeaningful` 会在「钉住的 commit、`hyperDataTestedMajor`、录制的 health fixture」三者主版本不一致时让构建失败，因此只改一半的版本推进无法悄悄通过
- **数据源契约测试**：以 fixture 钉住上游响应形状，并单独覆盖「允许的漂移」（新增字段、类型放宽、`null`），上游改字段会在测试里失败而不是在实盘周期里静默降级
- **数据源覆盖度块**：本轮未贡献数据的源会以 `data_coverage` 块写进提示词，避免模型把「源挂了」误读成「市场很安静」
- 新增 `marketdata.ServiceBacked` 可选接口，用于表达「需要另一个进程在运行但不需要凭证」的数据源

### 变更
- 重组文档结构为逻辑分类
- 更新所有 README 文件，添加适当的导航链接
- AI 推理改为直连 8 家原生服务商（DeepSeek、OpenAI、Claude、Qwen、Gemini、Grok、Kimi、MiniMax），使用你自己的 API Key，中间不再有任何网关
- 内置自动交易策略的默认候选池改为 Hyperliquid 原生成交量榜（`hyper_main`：24 小时成交量前 30），不再依赖已下线的信号看板
- 落地页、启动流程、策略工作室与终端的产品文案不再出现「按次计费」表述
- 市场级数据由四条硬编码链路（`Context` 字段类型强绑、引擎单一 client、交易循环三段 if、提示词直接调格式化函数）重构为单一的 `Registry.Collect` 调用
- 数据源失败从「静默丢弃」改为「记录并在提示词末尾报告」，因为静默丢弃会让模型无法区分「无事发生」与「数据源已死」
- `IndicatorConfig` 的 token 估算改为按实际启用的数据源数量计算，并把「说明性文字」与「表格行」分开预算——旧公式只按行数估算，会显著低估方向信号块

### 移除
- Claw402 / x402 按次付费网关：模型路由、方向看板数据、USDC 钱包包、按次计费记录、启动预检余额门槛与新手引导钱包流程
- Vergex 信号榜、方向变化排行榜与持仓成本/强平热力图数据源，及对应的终端与策略工作室界面（其中方向榜与方向变更历史已由 `directional_signal` 重建，强平热力图已由 `coinank_liquidation`、`hyperliquid_leverage` 与 `hyperdata_positioning` 替代；`signal_managed_exit` 出场模式**未**恢复，因为它属于交易层语义而非数据源能力）
- nofxos 三个排行能力：`OIRankingData` 与 `GetOIRanking`、`netflow.go`、`price.go` 及其格式化函数（对应端点已返回 402）
- 前端死路由与死组件：`/data`、`/strategy-market`、新手引导钱包页、新手引导卡片与新手/进阶模式选择器
- nofxos 剩余的排行接口（已无任何调用方）：`GetAI500List` / `GetTopRatedCoins` / `GetAvailableCoins`、AI500 的 TTL 缓存及其测试、OI 增减榜的类型与方法、`GetCoinDataBatch`、`FormatQuantDataForAI` 以及未被使用的 auth key 访问器。这些端点均返回 402，只有 `GetCoinData` 仍可达，因此该包保留
- 前端整条死图表链——约 4.5k 行、没有任何外部引入，只能互相引用：`AdvancedChart`、`ChartTabs`、`EquityChart`、`ChartWithOrders`、`ChartWithOrdersSimple`、`TradingViewChart`、`TraderDashboardPage`、`PageNotFound`、`BrandHero`、`BrandStats`、`stores/` 目录、`useCounterAnimation`、`ui/input.tsx`、`lib/text.ts`、`utils/indicators.ts`，以及只有它们才会读取的 i18n 段落（`advancedChart`、`chartWithOrders`、`traderDashboard`、`notFound`）
- 未接线的 Go 辅助代码：`safe` 包（零引入）、`kernel/prompt_builder.go`（已被 `engine_prompt.go` 取代，仅自身测试引用）、`trader/position_snapshot.go`、`trader/helpers.go`，以及 `market/data_indicators.go` 中仅供测试用的 `ExportCalculate*` 包装
- 五个无人引用的 npm 依赖：`date-fns`、`@radix-ui/react-slot`、`class-variance-authority`、`lightweight-charts`、`zustand`

### 修复
- 四处界面文案取用了并不存在的 key，中文环境下会直接显示原始 key 或英文兜底：登录按钮、图表加载态、对比图表的 PnL 标签与 Lighter API 密钥导入提示，现已分别对应 `auth.loggingIn`、`chart.loadingChartData`、`chart.leadPnL`、`exchangeCfg.lighterApiKeyImported`
- **账户级熔断从"有个名字"变成真的**：`MaxDailyLoss` / `MaxDrawdown` 只声明从不读取，`stopUntil` 从无赋值（暂停分支不可达），`dailyPnL` 只被重置——上报的 `daily_pnl` 恒为 0。现在交易循环每周期按市值测量权益，相对当日开盘权益跌破 `risk_control.max_daily_loss_pct`、或相对运行峰值回撤达到 `risk_control.max_drawdown_pct` 时暂停交易，并上报它所依据的数字。新策略默认 10% / 20% / 240 分钟。暂停只拦开仓、绝不强平：交易所侧止损与浮盈回吐保护照常运行
- **匿名统计改为显式同意**：`EXPERIENCE_IMPROVEMENT` 过去默认开启，等于全新安装在用户同意之前就已把交易所、币种、下单金额与杠杆发往 Google Analytics。现在必须显式设为 `true`
- **`signal_managed_exit` 有入口了，且能跨重启**：策略编辑器与 API 文档暴露 `exit_mode` / `signal_score_floor`，每笔持仓的开仓依据落盘到 `data/signal_book_<trader>.json`。此前该簿只存在于内存，每次重启后都要拿一个编造的 `0.00` 入场分作比较，可能在重启后第一个周期清掉整个仓位，而文档还写着该模式"未恢复"
- 前端调用的 `GET /api/prompt-templates` 在本服务从未注册过；这个无调用方的 helper 已删除
- `indicators.enable_quant_data` 过去每周期为每个币种重试一个已失效端点：现在遇到权限类应答（401/402/403）即在本进程内停用该数据源并只说明一次，走与候选池降级相同的提示通道
- `klines.longer_count` 是个不起作用的旋钮——抓取路径一直只用 `primary_count`。已从结构体、API 与编辑器中移除，不再留作陷阱
- README 写"九家交易所"而实际适配器有十家；市场数据文档里关于已退役 `signal_managed_exit` 模式的条目写着该模式未恢复

---

## [3.0.0] - 2025-10-30

### 新增 - 重大架构变革 🚀

**系统完全重新设计 - 基于 Web 的配置平台**

这是一个**重大破坏性更新**，将 NOFX 从基于静态配置的系统完全转变为现代化的 Web 交易平台。

#### 数据库驱动架构
- SQLite 集成，取代静态 JSON 配置
- 持久化存储，自动时间戳
- 外键关系和触发器确保数据一致性
- 为 AI 模型、交易所、交易员和系统配置分离表结构

#### 基于 Web 的配置界面
- 完整的 Web 配置管理（无需编辑 JSON）
- 通过 Web 界面设置 AI 模型（DeepSeek/Qwen API 密钥）
- 交易所管理（Binance/Hyperliquid 凭证）
- 动态创建交易员（结合任意 AI 模型和交易所）
- 实时控制（无需重启即可启动/停止交易员）

#### 灵活架构
- 关注点分离（AI 模型和交易所独立）
- 混合搭配能力（无限组合）
- 可扩展设计（支持无限交易员）
- 清洁起点（无默认交易员）

#### 增强的 API 层
- RESTful 设计，完整的 CRUD 操作
- 新端点：
  - `GET/PUT /api/models` - AI 模型配置
  - `GET/PUT /api/exchanges` - 交易所配置
  - `POST/DELETE /api/traders` - 交易员管理
  - `POST /api/traders/:id/start|stop` - 交易员控制
- 更新所有 API 端点文档

#### 现代化代码库
- 类型安全，适当分离配置类型
- 数据库抽象，使用预处理语句
- 全面的错误处理和验证
- 更好的代码组织（数据库、API、业务逻辑）

### 变更
- **破坏性变更**：不再使用旧的 `config.json` 文件
- 必须通过 Web 界面进行配置
- 设置更简单，用户体验更好
- 配置更改无需重启服务器

### 为什么重要
- 🎯 **用户体验**：配置和管理更容易
- 🔧 **灵活性**：创建 AI 模型和交易所的任意组合
- 📊 **可扩展性**：支持复杂的多交易员设置
- 🔒 **可靠性**：数据库确保数据持久性和一致性
- 🚀 **面向未来**：为高级功能奠定基础

---

## [2.0.2] - 2025-10-29

### 修复 - 关键错误修复：交易历史和性能分析

#### 盈亏计算 - 重大错误修复
- **修复**：盈亏现在计算为实际 USDT 金额，而不是仅百分比
- 之前忽略了仓位大小和杠杆（例如，100 USDT @ 5% = 1000 USDT @ 5%）
- 现在：`盈亏 (USDT) = 仓位价值 × 价格变化 % × 杠杆`
- 影响：胜率、盈利因子和夏普比率现在准确

#### 仓位跟踪 - 缺失关键数据
- **修复**：持仓记录现在存储数量和杠杆
- 之前只存储价格和时间
- 这对准确的盈亏计算至关重要

#### 仓位键逻辑 - 多空冲突
- **修复**：从 `symbol` 改为 `symbol_side` 格式
- 现在正确区分多头和空头仓位
- 示例：`BTCUSDT_long` vs `BTCUSDT_short`

#### 夏普比率计算 - 代码优化
- **变更**：用 `math.Sqrt` 替换自定义牛顿法
- 更可靠、可维护和高效

### 为什么重要
- 历史交易统计现在显示真实的 USDT 盈亏
- 不同杠杆交易之间的性能比较准确
- AI 自学习机制接收正确的反馈
- 多仓位跟踪（同时多空）正常工作

---

## [2.0.2] - 2025-10-29

### 修复 - Aster 交易所精度错误

- 修复 Aster 交易所精度错误（代码 -1111）
- 改进价格和数量格式化以匹配交易所要求
- 添加详细的精度处理日志用于调试
- 增强所有订单函数的精度处理

#### 技术细节
- 添加 `formatFloatWithPrecision` 函数
- 根据交易所规范格式化价格和数量
- 删除尾随零以优化 API 请求

---

## [2.0.1] - 2025-10-29

### 修复 - ComparisonChart 数据处理

- 修复 ComparisonChart 数据处理逻辑
- 从 cycle_number 切换到时间戳分组
- 解决后端重启时图表冻结问题
- 改进图表数据显示（按时间顺序显示所有历史数据）
- 增强调试日志

---

## [2.0.0] - 2025-10-28

### 新增 - 重大更新

- AI 自学习机制（历史反馈、性能分析）
- 多交易员竞赛模式（Qwen vs DeepSeek）
- 币安风格 UI（完整界面仿制）
- 性能比较图表（实时 ROI 比较）
- 风险控制优化（每币种仓位限制调整）

### 修复

- 修复硬编码初始余额问题
- 修复多交易员数据同步问题
- 优化图表数据对齐（使用 cycle_number）

---

## [1.0.0] - 2025-10-27

### 新增 - 初始版本

- 基础 AI 交易功能
- 决策日志系统
- 简单的 Web 界面
- 支持币安合约
- DeepSeek 和 Qwen AI 模型集成

---

## 如何使用本更新日志

### 用户
- 查看 [未发布] 部分了解即将推出的功能
- 查看版本部分了解变更内容
- 遵循破坏性变更的迁移指南

### 贡献者
进行更改时，将它们添加到 [未发布] 部分的相应类别下：
- **新增** - 新功能
- **变更** - 现有功能的变更
- **弃用** - 即将删除的功能
- **移除** - 已删除的功能
- **修复** - 错误修复
- **安全** - 安全修复

发布新版本时，将 [未发布] 项目移动到带日期的新版本部分。

---

## 链接

- [文档](docs/README.md)
- [贡献指南](CONTRIBUTING.md)
- [安全策略](SECURITY.md)
- [GitHub 仓库](https://github.com/NoFxAiOS/nofx)

---

**最后更新:** 2025-11-01
