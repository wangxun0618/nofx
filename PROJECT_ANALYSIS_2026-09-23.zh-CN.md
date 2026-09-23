# NOFX 项目现状分析

> 分析时间：2026-09-23 · 工作区 `/Users/ailiao/nofx`
> 说明：本次为只读体检 + 全量构建/测试验证。本沙箱内 `git` 不可用，未做版本与提交校验。

---

## 0. 结论摘要

| 维度 | 状态 |
| --- | --- |
| Go 编译 | ✅ `go build ./...` = 0 |
| Go 静态检查 | ✅ `go vet ./...` = 0（含测试一起编译） |
| Go 测试 | ✅ `go test ./...` 全绿，含 10 家交易所适配器 |
| 前端类型 | ✅ `tsc --noEmit` = 0 |
| 前端构建 | ✅ `vite build` 通过 |
| 前端 lint | ⚠️ 沙箱内 eslint 启动即报 IPC 错误，无法取数；既有 prettier 漂移未清 |
| 运行状态 | ✅ 后端 8080、前端 3000 均返回 200 |
| 默认零配置可用性 | ✅ 3 个免密钥数据源真实产出 |
| 架构分层 | ✅ 清晰：接入 / 决策 / 执行 / 基础设施 |
| 最高优先风险 | ⚠️ 账户级回撤熔断**未实现**，但配置与界面看起来像有 |

**一句话**：工程健康度很好，三轮"去付费依赖"改造已经收口。真正的待办不是"再加功能"，而是两个**承诺与实现不一致**的点——账户级熔断、默认开启的交易明细遥测。

---

## 1. 项目定位与规模

NOFX 是一个开源的 AI 交易终端：**策略本身就是语言模型**。每个 trader 跑一个持续循环（读市场结构 → 决策 → 下单 → 记录推理），Go 运行时在模型之外用代码钳制每一笔订单。许可 AGPL-3.0，文档覆盖 EN/CN/JA/KO/RU/UK/VI 七种语言。

| 项 | 规模 |
| --- | --- |
| Go 源文件 | 275 个（约 195 生产 + 约 80 测试） |
| Go 生产代码 | 约 58,500 行 |
| 前端源文件 | 134 个（13 个测试文件） |
| 前端代码 | 约 31,400 行 |
| 文档 | 82 份 Markdown |
| 交易所适配器 | 10 家 |
| npm 运行时依赖 | 15 个 |
| 数据库表 | 16 张（SQLite / Postgres 双后端） |

交易所适配器规模（含测试）：

| 适配器 | 行数 | 适配器 | 行数 |
| --- | --- | --- | --- |
| lighter | 4,365 | binance | 3,516 |
| hyperliquid | 3,564 | kucoin | 2,362 |
| okx | 2,268 | aster | 2,141 |
| bitget | 1,693 | bybit | 1,626 |
| gate | 1,579 | indodax | 1,267 |

---

## 2. 架构与主链路

### 分层

| 层 | 目录 | 职责 |
| --- | --- | --- |
| 接入层 | `api/` `web/` | Gin 路由（`api/route_registry.go` 注册表 + schema 校验）、React SPA |
| 决策层 | `kernel/` `marketdata/` | 提示词构造（`engine_prompt.go`）、决策解析与校验（`engine_position.go`）、选币（`local_coin_source.go`）、可插拔数据源 Registry |
| 执行层 | `trader/` | AutoTrader 主循环、网格、风控、节流 + 10 家交易所适配器 |
| 基础设施 | `store/` `mcp/` `crypto/` `provider/` `telegram/` `telemetry/` | GORM 持久化、多模型客户端、凭证加密、行情源、TG 机器人、遥测 |

### 主链路（已验证闭环，无断点）

```
main.go:106  manager.NewTraderManager
main.go:109  LoadTradersFromStore
  └ manager/trader_manager.go:629  addTraderFromStore
      └ :786  若 IsRunning 则 go trader.Run()
trader/auto_trader.go:443  Run
  └ :551  runCycle
    └ auto_trader_loop.go:62  buildTradingContext   （:422 定义）
    └ :105  kernel.GetFullDecisionWithStrategy
        └ kernel/engine_analysis.go:46
        └ :120  mcpClient.CallWithMessages（调用户自己的模型）
        └ :129  解析 + validateDecisions（硬限制在此拦截）
    └ auto_trader_loop.go:211  signalExitDecisions        ← signal 出场模式
    └ :218  filterDecisionsToStrategyUniverse
    └ :278  tradeThrottleReason                            ← 冷却/重复拦截
    └ :285  executeDecisionWithRecord
        └ auto_trader_orders.go:27 → :47  executeOpenLongWithRecord
            └ :135  trader.OpenLong
            └ :156  SetStopLoss（失败即紧急平仓）
```

---

## 3. 数据源能力矩阵

总开关 `enable_market_insights`（新策略默认 true）；`market_insight_sources` 为空时表示"选中所有自身已启用的源"。

| 数据源 | 默认启用条件 | 需要密钥 | 需要本地进程 | 默认零配置下产出 |
| --- | --- | --- | --- | --- |
| `directional_signal` | 恒 true | 否 | 否 | ✅ 是 |
| `hyperliquid_flow` | 恒 true | 否 | 否 | ✅ 是 |
| `hyperliquid_leverage` | 恒 true | 否 | 否 | ✅ 是 |
| `coinank_liquidation` | `coinank_api_key` 非空 | 是 | 否 | ❌ 否 |
| `hyperdata_orderflow` | `enable_hyperdata` | 否 | 是（sidecar） | ❌ 否 |
| `hyperdata_positioning` | `enable_hyperdata` | 否 | 是 | ❌ 否 |

**这意味着：用户什么 key 都不配、不装 sidecar，也能拿到方向信号 + 跨市场资金流 + 持仓杠杆结构三块真实数据。** 这是三轮改造最核心的成果。

### 本地重建的选币能力

| 原能力 | 现状 | 实现位置 |
| --- | --- | --- |
| `oi_top` / `oi_low` | ✅ 本地重建（ΔOI 快照榜） | `kernel/local_coin_source.go:307`，快照落盘 `data/oi_snapshot.json` |
| `ai500` | ✅ 以 screener 替代（自有榜，非复刻） | `kernel/local_coin_source.go:415` |
| 方向信号 + 方向变更历史 | ✅ 重建 | `marketdata/providers/directional_signal.go` + `marketdata/direction_history.go` |
| `signal_managed_exit` | ✅ 新增（交易层语义） | `trader/auto_trader_signal_exit.go` |

---

## 4. 风险与缺口

### P0-1｜账户级回撤熔断是空壳（配置与实现不一致）

这是本次体检发现的最重要问题。三处证据：

| 位置 | 现象 |
| --- | --- |
| `trader/auto_trader.go:148-149` | `MaxDailyLoss` / `MaxDrawdown` 两个字段**只有定义，全仓库无任何读取**，注释自称 `(hint)` |
| `trader/auto_trader.go:182` + `auto_trader_loop.go:43` | `stopUntil` 只被**检查**和输出，**全仓库无任何赋值** → 暂停分支永远不成立，是死代码 |
| `trader/auto_trader.go:178` + `auto_trader_loop.go:56` | `dailyPnL` 只被**重置为 0**（每 24h）和写出（`auto_trader_decision.go:193`），**从无累加** → 上报的 `daily_pnl` 恒为 0 |

同时对比：

- **网格交易有真实现**：`auto_trader_grid.go:133-205` 的 `checkMaxDrawdown` 与 `dailyPnL` 累加、日亏比例计算都是完整的。
- **提示词层面有软约束**：`kernel/schema.go:271-277` 把 `MaxDailyLoss = -10%` 写成给模型的规则（"Stop trading when daily loss reaches -10%"），但这依赖模型自觉遵守。
- **持仓级有硬保护**：`auto_trader_risk.go:18-30,123-134`（价移 >+5% 且回吐 ≥40% 强制平仓）。

**结论**：合约自动交易的**账户级**日亏/回撤熔断在代码层不存在，只有"提示模型"和"网格专属"两种。但配置结构体和界面都提供了该字段，会让人误以为开了就有保护。

**建议**：二选一——① 实现（在 `runCycle` 逐笔累计已实现 PnL，超阈值写 `stopUntil` 并置账户级暂停）；② 明确降级为"仅提示"，在 UI 上标注"由 AI 自主判断，代码不强制"。

### P0-2｜交易明细默认上报到 Google Analytics

| 位置 | 内容 |
| --- | --- |
| `config/config.go:78` | `ExperienceImprovement: true` —— **默认开启** |
| `telemetry/experience.go:126-136` | 上报 `exchange`、`symbol`、`amount_usd`、`leverage`、`user_id`、`trader_id` |
| `telemetry/experience.go:199-227` | 上报模型名、输入/输出 token 数 |
| `telemetry/experience.go:15-16` | 目标是 `google-analytics.com/mp/collect` |
| `trader/auto_trader_decision.go:357` | 实际调用点 |

**结论**：你的交易所、交易币种、下单金额、杠杆倍数会默认外发。字段里没有 API key 或私钥，但不能说无隐私影响——交易金额与杠杆直接反映资金规模。有隐私政策文档（`docs/i18n/*/PRIVACY POLICY.md`），可用 `EXPERIENCE_IMPROVEMENT=false` 关闭。

**建议**：改为默认关闭（opt-in），或至少在首次启动界面显式告知一次。

### P1-1｜前端调用不存在的后端端点

`web/src/lib/api/config.ts:29` 请求 `GET /api/prompt-templates`，全仓库无任何路由或 handler 注册。调用会 404。

### P1-2｜文档与代码脱节

- `docs/architecture/STRATEGY_MODULE.md` 仍在描述 `GetTopRatedCoins` / `GetOITopPositions`（已删除），并引用一个**不存在**的 `provider/data_provider.go`。
- `README.md:26` 写 "any of **nine** exchanges"，实际代码有 **10** 家适配器（表格漏了 Indodax）。

### P1-3｜nofxos 残留端点已失效

`provider/nofxos` 包仍在（只剩 `GetCoinData`），但它走的 `/api/coin/{SYM}` 端点实测返回 **402**。由于 `enable_quant_data` 默认 false（`store/strategy.go:1072`）、且 Hyperliquid 原生宇宙会跳过该调用（`kernel/engine.go:884-887`），**默认路径不受影响**——但一旦有人打开这个开关就会失败。

### P1-4｜HyperData sidecar 未启动时的延迟成本

启用 `enable_hyperdata` 但 sidecar 没跑时，两个 provider 各自最坏耗 8s（HTTP 超时，`hyperdata_client.go:78`）。不会中断周期、不会静默（会写入 `data_coverage` 提示块），但会拖慢每个决策周期。默认关闭，所以只影响主动开启者。

### P2｜清理余量（低风险）

- 主链路外的死代码：`manager/trader_manager.go:92 StartAll`（仅测试调）、`:119 AutoStartRunningTraders`（无调用方）、`kernel/engine_analysis.go:39 GetFullDecision`（生产只用带 strategy 的版本）。
- 后端孤儿端点（前端不调用）：`/api/health`、`/api/traders/:id/sync-balance`、`/api/strategies/public`、`/api/strategies/estimate-tokens`、`/api/strategies/preview-prompt`、`/api/strategies/test-run`、`/api/orders`、`/api/orders/:id/fills`、`/api/open-orders`、`/api/trades`。其中部分可能是有意保留给外部/调试用，**不建议无差别删**。
- 重复实现：符号归一化 5 份、HTTP client 封装 4 份、前端 `getShortName` 4 份。
- `docs/` 下 82 份文档中，规划类与实现类混放，缺乏"已实现/已废弃"标记。

---

## 5. 已收口的历史改造

| 改造 | 状态 |
| --- | --- |
| 移除 Claw402 / x402 付费网关（含 USDC 钱包、按次计费） | ✅ 无残留 |
| 移除 Vergex 信号榜 | ✅ 方向榜与变更历史已重建为 `directional_signal` |
| 移除已废弃的 nofxos 选币接口 | ✅ 三档改本地实现，失败显式降级 |
| 引入可插拔 `marketdata` 层 | ✅ Provider / Registry / Insight 三层抽象，新增数据源无需改引擎 |
| 集成 HyperData-Terminal | ✅ HTTP sidecar + 契约测试 + 活体漂移报告 + 被强制的版本钉 |
| 死代码清理 | ✅ 删 29 文件 / 约 7,000 行 / 5 个 npm 依赖 / 62 个死 i18n key |
| 补齐 4 个缺失 i18n key | ✅ 中文界面不再回显原始 key |

改造成果的一个可验证标志：`marketdata/registry.go` 的 `Collect` 是引擎接触市场的**唯一**入口，失败会被记入 `SourceFailure` 并追加 `data_coverage` 块。这一条把"死数据源"和"市场安静"区分开了，是之前架构做不到的。

---

## 6. 建议的下一步（按优先级）

1. **决定账户级熔断的去向**（半天至一天）。要么实现真实熔断，要么把配置降级并在 UI 说明。这是当前唯一"看起来有保护、实际没有"的地方。
2. **决定遥测默认值**（1 小时）。改 opt-in 或在启动时告知。
3. **修 `prompt-templates` 断层**（10 分钟）。要么补 handler，要么删前端调用。
4. **文档一致性批处理**（1–2 小时）。STRATEGY_MODULE.md 重写、README 交易所数量、docs 加状态标记。
5. **单独跑一次 `prettier --write`**（10 分钟）。既有的 263 处格式化漂移与本轮改动无关，不要混在功能提交里。
6. **P2 清理**（可选）。死符号可删；孤儿端点建议逐个确认用途后再定。

---

## 附：验证命令

```bash
# Go
export PATH="/Users/ailiao/.workbuddy/binaries/go/versions/go1.25.11/bin:$PATH"
go build ./...          # 0
go vet ./...            # 0
go test ./...           # 全绿

# 前端
cd web
npx tsc --noEmit        # 0
npx vite build          # 通过
```
