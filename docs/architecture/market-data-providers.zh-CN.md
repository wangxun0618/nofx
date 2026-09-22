# 市场数据源（可插拔层）

**语言:** [English](market-data-providers.md) | [中文](market-data-providers.zh-CN.md)

面向策略引擎的市场情报层。它把「AI 在做决策前能看到哪些市场级数据」从硬编码流程
变成一组可插拔的数据源，新增数据源不需要改动引擎、交易循环或提示词构建器。

---

## 为什么要这一层

改造之前，市场级数据是**四条硬编码链路**：

| 环节 | 旧写法 |
|------|--------|
| 上下文结构 | `kernel.Context` 里三个字段，类型直接绑死 `nofxos.*` |
| 引擎 | `StrategyEngine` 持有单一 `nofxos` client 字段 |
| 交易循环 | 三段独立的 `if EnableXxxRanking { ... }` |
| 提示词 | `kernel` 直接调 `nofxos.FormatXxxForAI` |

结果是加一个数据源要改四个地方，而且 `usesHyperliquidNativeUniverse()` 会把
nofxos 整体跳过——默认策略因此根本拿不到市场级数据。

现在数据流是：

```
用户请求 → 匹配 skill → skill 内部调用 tool → 返回结果
```

具体到本层：

```
交易循环 → Registry.Collect(ctx, req, cfg)
              ├── directional_signal      ┐
              ├── hyperliquid_flow        │
              ├── hyperliquid_leverage    ├─ 并发抓取，各自 12s 超时
              ├── hyperdata_orderflow     │  （后两者需本地侧车）
              ├── hyperdata_positioning   │
              └── coinank_liquidation     ┘
           → []*marketdata.Insight
              ├── Markdown  → 提示词
              └── Payload   → HTTP API / 前端面板
           → 若本轮有源失败，追加一个 data_coverage 块说明缺了什么
```

---

## 核心抽象

三个类型，都在 `marketdata` 包：

### `Insight` — 一个渲染好的数据块

```go
type Insight struct {
    Provider   string    `json:"provider"`      // 稳定标识，如 "hyperliquid_flow"
    Title      string    `json:"title"`         // 给 UI 看的短标题
    Markdown   string    `json:"markdown"`      // 英文提示词块
    MarkdownZh string    `json:"markdown_zh"`   // 中文提示词块，空则回退英文
    Payload    any       `json:"payload"`       // 结构化数据，喂给 API/UI
    FetchedAt  time.Time `json:"fetched_at"`
}
```

`Insight` 刻意做到「格式无关」：同一份数据同时供 LLM 提示词和前端面板使用，
所以**终端里看到的和 AI 看到的一定是同一份数据**，不会漂移。

`PromptBlock(language)` 负责选语言，`zh` 且中文块非空时用中文，否则回退英文。

### `Provider` — 一个数据源

```go
type Provider interface {
    Name() string
    Description() string
    RequiresAPIKey() bool
    Enabled(cfg store.IndicatorConfig) bool
    Fetch(ctx context.Context, req Request, cfg store.IndicatorConfig) (*Insight, error)
}
```

约定：

- **必须并发安全。** 注册表从多个 goroutine 调 `Fetch`，长驻进程跨交易日复用同一实例。
- **`Fetch` 返回 `(nil, nil)` 表示「已启用但本轮没数据」**，不算错误，但会在覆盖度块里
  被列为「可达但无数据」——见下文「失败可见性」。
- `RequiresAPIKey()` 只影响 UI 徽章与提示，不参与 `Enabled` 判断；`Enabled` 才决定是否真的抓。

#### 可选能力：`ServiceBacked`

有些数据源需要**另一个进程在运行**，但它不需要任何凭证——`RequiresAPIKey` 表达不了
这种情况。这类源可以实现一个可选接口：

```go
type ServiceBacked interface {
    RequiredService() string   // 面向用户的进程名，UI 用它解释为什么该源默认关闭
}
```

API 会把它作为 `requires_service` 返回，前端据此显示「需要本地服务」徽章而非「免费」。
这是可选接口而不是 `Provider` 的必选方法，所以已有 provider 不受影响。

### `Registry` — 数据源集合

```go
func NewRegistry(providers ...Provider) *Registry
func (r *Registry) Register(p Provider)                  // 重名忽略并告警
func (r *Registry) All() []Provider                      // 全量，UI 列清单用
func (r *Registry) Active(cfg store.IndicatorConfig) []Provider
func (r *Registry) Collect(ctx, req, cfg) []*Insight
```

`Collect` 的行为是这一层的关键：

- 并发抓取所有 active provider
- 每个 provider 单独 `context.WithTimeout`，默认 **12s**（`defaultFetchTimeout`）
- 失败只记日志并跳过，**任何数据源坏掉都不会阻塞交易**
- 结果按**注册顺序**返回，保证提示词在轮次之间稳定（否则 AI 每轮看到的数据顺序都变）
- 未设置 `req.Limit` 时补默认值 **10**（`defaultRowLimit`）
- 若有源未贡献，**末尾追加一个覆盖度块**（见下）

### 失败可见性

这是本层刻意补上的一个洞。软降级本身是对的——一个源挂掉不该让交易停摆——但如果
**静默**丢弃，模型就无法区分两件事：

| 情况 | 同一份提示词 |
|------|--------------|
| 「全市场都没有拥挤」 | 只有资金流块，其余为空 |
| 「资金流那个源挂了」 | 只有资金流块，其余为空 |

两者的正确解读完全相反。所以 `Collect` 会记录 `SourceFailure{Provider, Reason}`，
并在数据块之后追加一个 `data_coverage` insight：

```markdown
## Data source coverage

2 of 5 selected market-context sources contributed this cycle. The ones below
did not, so their subjects are simply absent from the blocks above rather than
confirmed quiet.

- hyperdata_orderflow — hyperdata health: Get "http://127.0.0.1:8420/v1/health": connection refused
- coinank_liquidation — reachable but reported nothing this cycle

Weight the remaining blocks accordingly; do not read a gap as a neutral reading.
```

细节：

- 覆盖度块**永远排在最后**，先给数据再讲缺口
- 上游错误字符串会被压成单行并截断到 240 字符（`truncateReason`），
  否则一个多行错误会把 markdown 列表顶坏
- **全部源都正常时不生成这个块**，避免噪音
- 它本身也是一个 `Insight`，所以 API 和终端面板同样能看到；provider 名固定为
  `data_coverage`（`marketdata.CoverageProvider`）

---

## 选择规则：allow-list 语义

`SourceSetting(cfg, name)` 的规则只有两条：

1. `cfg.MarketInsightSources` **为空** → 选中所有 `Enabled` 的数据源
2. 非空 → 视为显式白名单

第 1 条是刻意的设计：**新注册的数据源开箱即用**。如果空列表等于「什么都不要」，
那么每次上线新数据源都要用户去改一遍配置，否则它永远拿不到数据。

`Registry.Active` 还要求 `cfg.EnableMarketInsights == true`——这是总开关，
关掉后连 `Enabled` 都不问，直接返回 `nil`。

---

## 内置数据源

### `directional_signal` — 方向信号（原 Vergex 方向榜的平替）

- 免费，无需 key，默认启用
- 数据源：Hyperliquid `metaAndAssetCtxs`（价格动量、溢价）+ 可选的 HyperData 订单流
- 覆盖约 120 个品种（`directionUniverseSize`）

**设计立场比算法本身重要。** 原 Vergex 返回一个不透明的 `bias` + `score`，并且在
系统提示词里被声明为「权威方向，K 线不得否决」。这一版反过来：**所有分量都摊开给
模型看**，bias 只是「哪一侧票多」，score 只是「这些票的平均强度」。

三个分量，各自投票 bullish / bearish / neutral：

| 分量 | 依据 | 投票规则 |
|------|------|----------|
| `momentum` | 24h 涨跌幅 | 对全市场做 z-score，`|z| ≥ 0.5σ` 才投票 |
| `premium` | markPx 对 oraclePx 的溢价 | 绝对阈值 `±0.02%`（溢价本身量级就小，用相对阈值没意义） |
| `flow` | HyperData 累计 CVD | 绝对阈值 `±$25K`，否则视为噪音弃权 |

**为什么资金费率不参与投票：** 它的符号是**双解**的。资金费率高可以读成「多头拥挤
（反向指标）」，也可以读成「多头愿意付费（顺势确认）」，两者都成立。把这种歧义硬塞
进投票等于制造虚假确定性，所以它只作为 `hyperliquid_flow` 的**拥挤度列**出现，
不进方向票。

投票与分数：

- `bullish` / `bearish` / `neutral` 计数由三个分量的票数统计得出
- **平票判 neutral**：两分量 1:1 是真正未决，强行选边就是制造确信
- `score` = 各分量标准化贡献的均值，裁剪到 `±3`。动量和溢价贡献其 z-score；
  订单流贡献**符号**——它只抓少数几个品种，没有横截面可标准化。这个不对称是刻意的，
  提示词里也写明了。

每个品种的状态与**方向变更时间线**一起渲染：

```markdown
### Your instruments

| Instrument | Bias | Score | Component evidence |
| :--- | :--- | ---: | :--- |
| BTC | bullish | +1.42 | momentum +1.90σ (+3.20%/24h) (bullish); premium +0.052% (bullish); flow $1.2M (bullish) |

### Recent direction changes

- BTC bullish → neutral, 3h ago — premium bullish→neutral; momentum bullish→neutral
```

Payload 形状：

```json
{
  "top_bullish": [{ "symbol": "PURR", "bias": "bullish", "score": 2.41,
                    "bullish": 3, "bearish": 0, "neutral": 0,
                    "components": [{ "name": "momentum", "vote": "bullish", "detail": "+2.80σ (+11.40%/24h)" }] }],
  "top_bearish": [],
  "instruments": [],
  "recent_changes": [{ "symbol": "BTC", "from_bias": "bullish", "to_bias": "neutral",
                       "from_score": 1.42, "to_score": 0.31,
                       "reason": "premium bullish→neutral; momentum bullish→neutral",
                       "changed_at": "2026-09-22T09:31:00Z" }],
  "universe_size": 234,
  "components": ["momentum", "premium", "flow"]
}
```

#### 方向变更时间线

这是原 Vergex 三项能力里**唯一纯粹净损失**的一项——原版直接把
`direction-change/history` 的 JSON 塞进提示词，而下线后没有任何免费数据源提供它。

实现是自建的：`marketdata.DirectionTracker` 保存每个品种的上一次状态与一个有界的
变更日志，**落盘到 `data/direction_history.json`**（可用环境变量
`MARKET_DIRECTION_HISTORY_PATH` 覆盖）：

- **跨重启存活。** 时间线的价值就在于重启后仍能说「2 小时前翻转」，否则频繁
  重启的进程永远报告空历史
- **首次观测不算变更。** 否则全新安装后的第一个周期，每个品种都会报「翻转」
- **原因是算出来的，不是编的。** 变更原因来自**分量票的 diff**：
  `momentum bullish→bearish`；若没有单个分量翻转（只是聚合越过了阈值），
  就如实说 `no component flipped; score moved ...`，而不是硬编一个理由
- **写入原子化。** 先写 `.tmp` 再 rename，崩溃不会留下半截文件
- **所有持久化失败都退化为纯内存**：目录不可写、文件损坏都只告警一次，绝不阻塞交易
- 日志上限 200 条（`directionHistoryMaxChanges`），跟踪上限 400 个品种

### `hyperliquid_flow` — 跨市场资金流

- 免费，无需 key
- 数据源：`POST https://api.hyperliquid.xyz/info` 的 `metaAndAssetCtxs`
- 覆盖约 120 个成交量最大的品种（`flowUniverseSize`）

五张榜：

| 榜单 | 含义 |
|------|------|
| `most_traded` | 24h 名义成交额前 N |
| `most_crowded_long` | 正向资金费率最极端 → 多头拥挤 |
| `most_crowded_short` | 负向资金费率最极端 → 空头拥挤 |
| `gainers` / `losers` | 24h 涨跌幅两端 |

Payload 形状：

```json
{
  "most_traded": [{
    "symbol": "BTC", "dex": "",
    "volume_24h": 1234567890,
    "change_24h_pct": -1.23,
    "open_interest_usd": 987654321,
    "funding_rate_1h": 0.0000125,
    "funding_annual_pct": 10.95
  }],
  "most_crowded_long": [], "most_crowded_short": [],
  "gainers": [], "losers": [],
  "universe_size": 234
}
```

资金费率的年化是 `hourly * 24 * 365 * 100`，因为上游给的是**小时**费率，
直接乘 3 次会被误读成日费率。

**基线校准（重要）：** Hyperliquid 的资金费率公式含固定基础项（0.01%/8h），
年化后约 **+11%**。因此 BTC / ETH 这类主流品种会长期稳定打印 `+11.0%`，
这**不是**多头拥挤，而是基准锚。两个 provider 的提示词里都显式写明了这一点
（`baselineFundingNote`），并提示模型只解读**偏离**基准的部分——这是整份数据里
最容易被误读的一个数字。实测中 PURR `+278%`、`xyz:MINIMAX` `+224%` 这类才是真信号。

### `hyperliquid_leverage` — 持仓结构 / 杠杆拥挤度

- 免费，无需 key
- 与 flow 共用同一个上游端点
- 覆盖约 120 个品种（`leverageUniverseSize`）

榜单：`largest_open_interest`、`long_crowded`、`short_crowded`、`candidates`
（`candidates` 只包含当前候选池与持仓中的品种，最多 12 行）。

**重要：这是代理指标，不是真实的强平热力图。** 代码注释里明确写了这一点——
Hyperliquid 的免费 feed 只发布聚合持仓量与资金费率，不发布按价格分桶的强平订单簿。
本 provider 用 **OI + 资金费率 + markPx 对 oraclePx 的偏离** 刻画「杠杆拥挤」，
用来替代性回答「哪里仓位重、哪一侧被拉得过紧」，但**不能**等同于逐价位的强平簇。

需要真正的价格分桶强平簇，请启用下面的 `coinank_liquidation`。

Payload 形状：

```json
{
  "largest_open_interest": [{
    "symbol": "ETH", "dex": "",
    "open_interest_usd": 4567890123,
    "volume_24h": 2345678901,
    "max_leverage": 25,
    "funding_annual_pct": -4.38,
    "premium_pct": -0.012
  }],
  "long_crowded": [], "short_crowded": [], "candidates": [],
  "universe_size": 234
}
```

### `hyperdata_orderflow` / `hyperdata_positioning` — HyperData Terminal 侧车（可选）

这两个源读的是 [HyperData Terminal](https://github.com/Co-Messi/HyperData-Terminal)
的 REST API。它是一个**独立的开源 Python 项目**（TUI + REST + 纸交易引擎），
不是可 import 的 Go 库，因此本层通过 **HTTP** 消费它。

| 项 | 值 |
|----|----|
| 开关 | `cfg.EnableHyperData`（默认 **false**） |
| 地址 | `cfg.HyperDataBaseURL`，空则回退环境变量 `HYPERDATA_BASE_URL`，再回退 `http://127.0.0.1:8420` |
| 凭证 | `cfg.HyperDataAPIKey`，空则回退 `HYPERDATA_API_KEY`；两者皆空表示侧车未设密钥 |
| 需要 key | 否（侧车是本地服务，`RequiresAPIKey()` 返回 false） |
| 徽章 | 实现 `ServiceBacked`，UI 显示「需要本地服务」 |

**为什么默认关闭：** 它是一个需要另行启动的独立进程。默认打开只会让每个周期都记一条
连接失败日志并白等超时。关闭时这两个源会被跳过，其余数据源照常运行。

`hyperdata_orderflow` 提供：

- **累计成交量差（CVD）**，含**逐交易所归因**（`hyperliquid` / `binance`）
- 逐窗口净额（优先取最短可用窗口）、每秒成交笔数、聚合信号
- 账户多空比、永续对现货的基差

**诚实的覆盖度披露**是这个块的重点。它会把各交易所的订阅状态写进提示词，例如
`binance (partial, stale, 0/2 sockets, 2 symbols dark)`，并明确告诉模型
**单交易所的 CVD 是更弱的证据**。上游对 Binance Futures 的地理限制是诚实报告的，
本层不把它悄悄合并成一个「双所」数字。

`hyperdata_positioning` 提供最大的被追踪仓位与最接近强平的仓位。提示词开头写死了
一句限定：

> **Sample, not a census**：这些是该实例在每周期扫描预算内**已发现**的钱包所持的仓位。
> 把它读作「大额持仓的样本」，**绝不是**全市场总持仓或完整的强平价位。

上游数据本身的已知折扣（同样写进了提示词）：

- Hyperliquid 的强平是**从大额成交推断**的，不是交易所直出
- Binance 的强平流在源头被限流到约 1 笔/品种/秒
- 部分地区的 Binance Futures 流被地理封锁（socket 连上但不推数据）

#### 上游升级

侧车是独立进程，这正是它**能独立升级**的原因：Go 侧只讲 HTTP，永远不链接它。
适配层在**运行期**对上游变更的三条应对规则：

1. **未知字段忽略**（`encoding/json` 默认行为）→ 上游**新增**字段永远安全
2. **类型宽松字段用容错类型**（`flexStrings` / `flexText` / `any`）→ 上游把
   `signal` 从字符串改成对象也不会失败，会降级成可读文本
3. **版本比对**：读到 `version` 与 `hyperDataTestedMajor`（当前为 1）主版本不一致时，
   在日志和提示词里各写一条警告，**但不失败**——因为规则 1、2 已经兜住了大多数变更

这三条决定的是**一个交易周期内**会发生什么，它们不会告诉你「这次升级改了什么」，而那是另一半：

- **版本钉。** `deploy/hyperdata/upstream.env` 记录适配层验证过的确切 40 位 commit
  和实测主版本。它被 `TestUpstreamPinIsStillMeaningful` 读取，所以只改「版本三件套」
  （env 文件、`hyperDataTestedMajor`、health fixture 的 `version`）中一部分时，
  构建会失败，而不会悄悄通过。
- **漂移报告。** `TestHyperDataUpstreamHasNotDrifted` 把录制的 fixture 与**运行中**的
  侧车对比，把每处差异分类为 `MISSING` / `TYPE CHANGED`（破坏性）或 `ADDED`（信息性）。
  它由 `HYPERDATA_LIVE_URL` 环境变量控制、默认跳过，以保证常规测试套件不依赖外部进程。
  这一项回答的才是「这次升级究竟改了什么」——契约测试只能回答「我们自己的解析有没有坏」。
  由于它比对的是**形状**，限制也很具体：名字与类型都没变、只有**含义**变了的字段（单位从
  USD 改成张数）不会被发现；被折叠容器内的键改名同样不会被发现。普通字段改名会被报成
  `MISSING` + `ADDED`。

漂移检查刻意做成**一个 Go 测试，而不是另写一个探测脚本**：它复用同一个客户端、同一批
fixture、同一套折叠规则，因此只有**一份**契约，而不是两份可能互相矛盾的东西。

运行手册、容器部署与版本推进流程见
[`deploy/hyperdata/README.zh-CN.md`](../../deploy/hyperdata/README.zh-CN.md)。

### `coinank_liquidation` — 价格分桶强平簇（可选）

- **需要 API Key**（`cfg.CoinankAPIKey`），没配 key 时 `Enabled` 返回 `false`
- 上游：`https://api.coinank.com`
- 用 `client.LiquidationOrders(...)` 拿**逐笔**强平记录（含 `price` 字段），
  在本地按价格分成 **12 个桶**（`liquidationBucketCount`）
- 每次最多处理 **3 个**品种（`maxLiquidationSymbols`），优先当前持仓

这是对原 Vergex「持仓成本 / 强平热力图」的 1:1 替代：Vergex 卖的正是这个
「按价位聚合的强平簇」，而逐笔数据 + 本地分桶能算出同样的东西，且不绑定付费。

Payload 形状：

```json
{
  "symbols": [{
    "symbol": "BTC",
    "bins": [{
      "bucket_start_price": 61000,
      "bucket_end_price": 62000,
      "long_liq_usd": 1200000,
      "short_liq_usd": 340000
    }]
  }]
}
```

---

## 如何新增一个数据源

**唯一改动点。** 三步：

### 1. 新建 `marketdata/providers/<你的源>.go`

```go
package providers

type MyProvider struct{}

func NewMyProvider() *MyProvider { return &MyProvider{} }

func (p *MyProvider) Name() string        { return "my_source" }
func (p *MyProvider) Description() string  { return "What this source offers" }
func (p *MyProvider) RequiresAPIKey() bool { return false }

func (p *MyProvider) Enabled(cfg store.IndicatorConfig) bool {
    // 需要 key 的源在这里判断：return cfg.MyAPIKey != ""
    return true
}

func (p *MyProvider) Fetch(ctx context.Context, req marketdata.Request, cfg store.IndicatorConfig) (*marketdata.Insight, error) {
    limit := req.NormalizeLimit(10)
    // ... 抓数据 ...
    return &marketdata.Insight{
        Provider:   p.Name(),
        Title:      "My source",
        Markdown:   "...",   // 英文提示词块
        MarkdownZh: "...",   // 中文提示词块（可省，省了回退英文）
        Payload:    payload, // 给 API / UI
        FetchedAt:  time.Now(),
    }, nil
}
```

### 2. 注册到 `marketdata/providers/registry.go`

```go
func defaultProviders() []marketdata.Provider {
    return []marketdata.Provider{
        NewDirectionalSignalProvider(),
        NewHyperliquidFlowProvider(),
        NewHyperliquidLeverageProvider(),
        NewHyperDataOrderflowProvider(),
        NewHyperDataPositioningProvider(),
        NewCoinankLiquidationProvider(),
        NewMyProvider(),   // ← 加这一行
    }
}
```

数组顺序**就是提示词里的块顺序**，所以有讲究：`directional_signal` 排在最前，
因为它是唯一给出结论的源；后面的描述性数据是模型用来核对结论的。

### 3. 需要的话，加一个配置字段

如果新源需要 API Key 或专属开关，在 `store.IndicatorConfig` 里加字段即可。
**不需要**加 `EnableMySource` 布尔开关——用 `MarketInsightSources` 白名单表达
「启用哪些源」就够了，白名单为空时自动全开。

例外是需要**外部进程**的源：那种情况下 `Enabled` 应该读一个独立开关
（如 `EnableHyperData`），因为「白名单为空 = 全开」的语义会让一个未启动的服务
在每个周期都被尝试连接。

### 4. 如果新源走 HTTP，补一份契约测试

这是硬要求，不是建议。`marketdata/providers/hyperdata_contract_test.go` 是范例：

- 用 `httptest` 把上游响应形状钉成 fixture，字段名逐字来自上游源码
- 另有一组**「允许的漂移」**用例：新增字段、类型放宽、`null`——断言这些不会让解析失败
- 表格渲染用 `assertTablesAligned` 断言列数一致。提示词里的表格是**按位置**读的，
  一行少一列会把右侧所有值静默错位（这个 bug 在开发本层时真实出现过一次）

上游改了字段，改 fixture 就会看到哪些断言失败——这比在实盘周期里发现要早得多。

### 不需要改的地方

- `kernel/engine.go` — `CollectInsights` 只认 `Registry`
- `kernel/engine_prompt.go` / `kernel/formatter.go` — 只遍历 `ctx.Insights`
- `trader/auto_trader_loop.go` — 只有一次 `CollectInsights` 调用
- `api/handler_market_insights.go` — 从同一个注册表列清单，自动带上新源

**前端也已经自动适配**：`/api/market-insights/providers` 会返回新源，
策略工作室的数据源卡片列表会直接多出一张卡。

---

## 前后端衔接

### 后端接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/market-insights/providers` | 列出**全部**数据源（含未启用的），含 `name` / `description` / `requires_key` / `requires_service` |
| `GET` | `/api/market-insights?lang=&sources=&hyperdata=&hyperdata_url=` | 抓取数据。`sources` 为逗号分隔白名单，省略则跑所有 enabled 源；`hyperdata=1` 显式请求侧车源 |

`hyperdata` 参数存在的原因是预览接口用的是**合成配置**而不是某个策略的配置，
所以侧车源（默认关闭）需要显式请求。服务访问密钥**不接受**通过查询串传递。

### 配置字段

`store.IndicatorConfig`：

```go
EnableMarketInsights bool     `json:"enable_market_insights"`
MarketInsightSources []string `json:"market_insight_sources,omitempty"`
MarketInsightLimit   int      `json:"market_insight_limit,omitempty"`
CoinankAPIKey        string   `json:"coinank_api_key,omitempty"`

// 可选侧车（HyperData Terminal）
EnableHyperData  bool   `json:"enable_hyperdata"`
HyperDataBaseURL string `json:"hyperdata_base_url,omitempty"`
HyperDataAPIKey  string `json:"hyperdata_api_key,omitempty"`
```

默认策略（`GetDefaultStrategyConfig`）：总开关 `true`，行数 `10`，
`MarketInsightSources` 留空 → 全部免费源开箱启用；`EnableHyperData` 为 `false`。

地址与密钥的解析顺序是**策略字段 → 环境变量 → 内置默认值**。环境变量存在是为了
让一次部署只需全局配一次，而策略字段仍然优先，保留了按策略路由的能力。

### Token 估算

`store.StrategyConfig` 的 token 估算按数据源数量动态计算：
空白名单时按 **3 个源**（方向信号 + 资金流 + 持仓结构）起算，
`EnableHyperData` 打开再 **+2**，配了 CoinAnk key 再 **+1**。

每个源的预算拆成两部分，因为**说明性文字不随行数增长**：

```
(numSources × 700 字符固定开销 + numSources × limit × 90 字符) / 4
```

早期版本只按 `limit × 80` 估算，会显著低估方向信号块——它主要是一段引导文字，
表格只占一小部分。

### 前端

| 位置 | 作用 |
|------|------|
| `web/src/pages/StrategyStudioPage.tsx` | 「市场数据源」配置区：总开关、数据源卡片（带「免费 / 需要 API Key / 需要本地服务」徽章）、榜单行数下拉，以及侧车开关 + 地址 + 密钥 |
| `web/src/components/terminal/MarketInsightsPanel.tsx` | 终端面板，按 provider 渲染方向信号 / 资金流 / 持仓结构 / 订单流 / 追踪持仓 / 数据覆盖度六种表 |
| `web/src/lib/api/data.ts` | `getMarketInsightSources()` / `getMarketInsights()` |

面板用 SWR key 区分是否请求侧车（`'market-insights'` 与 `'market-insights:sidecar'`），
同一类面板**共享一次请求**。轮询间隔 `120s`（上游本身缓存 5 分钟，
轮询更快只会重复渲染同样的数字）。

---

## 已下线的能力与替代关系

| 原能力 | 现状 |
|--------|------|
| Vergex 方向信号榜（`bias` + z-score 多空投票） | **`directional_signal`**：透明投票重建，但**不**声明为权威、**不**绑定付费 |
| Vergex 方向变更历史（X→Y 时间线 + 原因） | **`directional_signal`** 的内建时间线 + `DirectionTracker` 落盘 |
| Vergex 持仓成本 / 强平热力图 | `hyperliquid_leverage`（免费代理指标）+ `coinank_liquidation`（可选，真价位分桶）+ `hyperdata_positioning`（可选，逐仓距离） |
| Vergex 逐币明细（当前方向 / 历史 / 热力图 JSON） | `hyperdata_orderflow`（CVD 逐所归因）+ `hyperdata_positioning`（逐仓明细） |
| nofxos OI / NetFlow / Price 三个排行接口 | 已删除（端点返回 402，内置 key 已废弃） |
| `signal_managed_exit` 出场模式 | **未恢复**。它与数据源无关，属交易层语义；当前 `validateProtectionPrices` 强制每仓同时带止损与止盈 |

### 与原 Vergex 的两处关键差异

同一份「方向」结论，新旧两版的使用方式不同，这是刻意为之：

| | 原 Vergex | 现在的 `directional_signal` |
|---|---|---|
| 结论来源 | 外部付费服务，算法不可见 | 本地从公开数据计算，**分量逐一列出** |
| 提示词地位 | 系统提示词声明为「权威方向」 | 无优先序声明，与其余数据块平等 |
| 与 K 线的关系 | 明写「K 线不得否决、反转或提前离场」 | 由 AI 自行权衡 |
| 成本 | 绑定 claw402 钱包，按次计费 | 免费 |

也就是说：**能力恢复了，权威性没有恢复**。如果希望 AI 重拾「跟随信号、不因 K 线
提前离场」的行为，那是提示词层的改动，不应该夹在数据层里做。

原 `provider/nofxos` 中被删除的内容：`netflow.go`、`price.go`、`OIRankingData`
及三个格式化函数。保留的部分是 `fetchOIRanking` 与 `GetOITopSymbols` 系列——
它们服务于**候选池构建**，与市场情报层无关。

---

## 相关文件

```
marketdata/
├── insight.go                 # Insight / Request / Provider / SourceSetting / ServiceBacked
├── registry.go                # Registry：并发收集、超时、软降级、覆盖度块
├── direction_history.go       # DirectionTracker：方向和变更时间线，落盘 + 原子写
└── providers/
    ├── registry.go            # defaultProviders() ← 唯一接线点
    ├── format.go              # 年化、USD 缩写、符号显示、排序
    ├── directional_signal.go  # 方向信号 + 时间线接入
    ├── hyperliquid_flow.go
    ├── hyperliquid_leverage.go
    ├── hyperdata_client.go    # HyperData REST 客户端 + 容错类型 + 版本比对
    ├── hyperdata_orderflow.go
    ├── hyperdata_positioning.go
    ├── hyperdata_contract_test.go  # 录制的 fixture + 容错漂移断言
    ├── hyperdata_upstream_test.go  # 活体漂移报告 + upstream.env 版本钉断言
    └── coinank_liquidation.go

deploy/hyperdata/              # 侧车的安装 / 升级 / 契约校验
├── upstream.env               # 钉住的 revision + 实测主版本（唯一真源）
├── sidecar.sh                 # install | update | status | contract | run
├── Dockerfile                 # 按 pin 构建，且拒绝构建偏离 pin 的代码
├── README.md                  # 升级流程与失败处理
└── README.zh-CN.md

docker-compose.hyperdata.yml   # 附加式 compose 文件，profile 为 "hyperdata"

api/
├── handler_market_insights.go   # 两个 HTTP 接口
└── server.go                    # 路由注册

kernel/
├── engine.go               # CollectInsights / MarketInsightProviders
├── engine_prompt.go        # 主提示词路径（渲染 ctx.Insights）
└── formatter.go            # 备用路径（同）
```

---

[← 返回架构文档](README.zh-CN.md)
