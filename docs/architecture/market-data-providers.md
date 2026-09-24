# Market Data Providers (Pluggable Layer)

**Language:** [English](market-data-providers.md) | [中文](market-data-providers.zh-CN.md)

The market-intelligence layer that feeds the strategy engine. It turns "what
market-wide data can the AI see before deciding" from a hardcoded pipeline into a
set of pluggable sources — adding a source requires no change to the engine, the
trading loop or the prompt builder.

---

## Why this layer exists

Before the rework, market-level data was **four hardcoded chains**:

| Layer | Old shape |
|-------|-----------|
| Context struct | Three fields on `kernel.Context`, typed directly as `nofxos.*` |
| Engine | `StrategyEngine` held a single `nofxos` client field |
| Trading loop | Three separate `if EnableXxxRanking { ... }` blocks |
| Prompt | `kernel` called `nofxos.FormatXxxForAI` directly |

Adding one source meant editing four places — and `usesHyperliquidNativeUniverse()`
skipped nofxos wholesale, so the default strategy never received market-level data
at all.

The data flow now:

```
trading loop → Registry.Collect(ctx, req, cfg)
                 ├── directional_signal      ┐
                 ├── hyperliquid_flow        │
                 ├── hyperliquid_leverage    ├─ concurrent, 12s timeout each
                 ├── hyperdata_orderflow     │  (last three: sidecar / key)
                 ├── hyperdata_positioning   │
                 └── coinank_liquidation     ┘
              → []*marketdata.Insight
                 ├── Markdown  → prompt
                 └── Payload   → HTTP API / web UI
              → plus a data_coverage block when a source contributed nothing
```

---

## Core abstractions

Three types, all in package `marketdata`:

### `Insight` — one rendered block

```go
type Insight struct {
    Provider   string    `json:"provider"`      // stable id, e.g. "hyperliquid_flow"
    Title      string    `json:"title"`         // short heading for the UI
    Markdown   string    `json:"markdown"`      // English prompt block
    MarkdownZh string    `json:"markdown_zh"`   // Chinese block; falls back to English
    Payload    any       `json:"payload"`       // structured form for API/UI
    FetchedAt  time.Time `json:"fetched_at"`
}
```

`Insight` is deliberately format-agnostic: one payload feeds both the LLM prompt
and the terminal panel, so **the terminal can never drift from what the AI sees**.

`PromptBlock(language)` picks the language — `zh` with a non-empty Chinese block
wins, otherwise English.

### `Provider` — one data source

```go
type Provider interface {
    Name() string
    Description() string
    RequiresAPIKey() bool
    Enabled(cfg store.IndicatorConfig) bool
    Fetch(ctx context.Context, req Request, cfg store.IndicatorConfig) (*Insight, error)
}
```

Contract:

- **Must be safe for concurrent use.** The registry calls `Fetch` from several
  goroutines and long-running processes reuse one instance across cycles.
- **Returning `(nil, nil)` means "enabled but no data this cycle"** — not an error,
  but it is listed in the coverage block as "reachable but reported nothing".
- `RequiresAPIKey()` only drives the UI badge and prompting; it does not gate the
  fetch. `Enabled` does.

#### Optional capability: `ServiceBacked`

Some sources need **another process to be running** but require no credential at
all, and `RequiresAPIKey` cannot express that. Such a source may implement an
optional interface:

```go
type ServiceBacked interface {
    RequiredService() string   // user-facing process name, shown in the UI
}
```

The API surfaces it as `requires_service` and the UI badges it "Needs a local
service" instead of "Free". It is an optional interface rather than a method on
`Provider`, so no existing provider is affected.

### `Registry` — the provider set

```go
func NewRegistry(providers ...Provider) *Registry
func (r *Registry) Register(p Provider)                  // duplicates ignored + warned
func (r *Registry) All() []Provider                      // everything, for the UI
func (r *Registry) Active(cfg store.IndicatorConfig) []Provider
func (r *Registry) Collect(ctx, req, cfg) []*Insight
```

`Collect` is where the guarantees live:

- fetches every active provider concurrently
- gives each one its own `context.WithTimeout`, **12s** by default (`defaultFetchTimeout`)
- logs and skips failures — **a broken source can never block trading**
- returns results in **registration order**, keeping the prompt stable between
  cycles (otherwise the AI sees the blocks reshuffled every cycle)
- fills `req.Limit` with **10** when unset (`defaultRowLimit`)
- **appends a coverage block** when any source contributed nothing (below)

### Failure visibility

This is a hole the layer deliberately closes. Failing soft is correct — one dead
source must not stop trading — but dropping it *silently* leaves the model unable
to tell two opposite situations apart:

| Situation | The same prompt looks like |
|---|---|
| "nothing is crowded anywhere" | the flow block, everything else empty |
| "the flow source is down" | the flow block, everything else empty |

`Collect` therefore records `SourceFailure{Provider, Reason}` and appends a
`data_coverage` insight after the data blocks:

```markdown
## Data source coverage

2 of 5 selected market-context sources contributed this cycle. The ones below
did not, so their subjects are simply absent from the blocks above rather than
confirmed quiet.

- hyperdata_orderflow — hyperdata health: Get "http://127.0.0.1:8420/v1/health": connection refused
- coinank_liquidation — reachable but reported nothing this cycle

Weight the remaining blocks accordingly; do not read a gap as a neutral reading.
```

Details that matter:

- the block **always comes last** — what arrived first, what is missing after
- upstream error strings are flattened to one line and cut at 240 characters
  (`truncateReason`), because a multi-line error would break the markdown list
- **nothing is emitted when every source delivered**, so there is no constant noise
- it is an `Insight` like any other, so the API and the terminal see it too; its
  provider name is fixed at `data_coverage` (`marketdata.CoverageProvider`)

---

## Selection: allow-list semantics

`SourceSetting(cfg, name)` has exactly two rules:

1. `cfg.MarketInsightSources` **empty** → select every `Enabled` provider
2. non-empty → treat as an explicit allow-list

Rule 1 is deliberate: **a newly registered source works out of the box.** If an
empty list meant "nothing", every new source would need a config edit before it
could ever produce data.

`Registry.Active` additionally requires `cfg.EnableMarketInsights == true` — the
master switch. When off it returns `nil` without even consulting `Enabled`.

---

## Built-in sources

### `directional_signal` — direction bias (replaces the Vergex board)

- Free, no key, on by default
- Inputs: Hyperliquid `metaAndAssetCtxs` (momentum, premium) plus optional
  HyperData order flow
- Covers roughly 120 instruments (`directionUniverseSize`)

**The design stance matters more than the arithmetic.** The retired Vergex board
returned an opaque `bias` + `score` and the system prompt declared it
"authoritative direction, which candles may not veto". This one inverts that:
**every component is exposed**, `bias` is only "which side has more votes", and
`score` is only the average strength of those votes.

Three components, each voting bullish / bearish / neutral:

| Component | Evidence | Vote rule |
|---|---|---|
| `momentum` | 24h price change | z-scored against the universe; votes only at `|z| ≥ 0.5σ` |
| `premium` | mark price over oracle | absolute `±0.02%` (premium is small by construction; a relative threshold would be meaningless) |
| `flow` | HyperData cumulative CVD | absolute `±$25K`, below which tape noise abstains |

**Why funding does not vote:** its sign is genuinely ambiguous. High funding reads
both as "longs are crowded" (contrarian) and as "longs are paying up" (trend
confirmation), and both are defensible. Forcing that into a vote would manufacture
conviction the input does not support, so funding appears only as a crowding
column in `hyperliquid_flow`.

Scoring:

- the bullish/bearish/neutral counts are tallies of the component votes
- **ties resolve to neutral**: a 1:1 split is genuinely undecided, and picking a
  side there would invent conviction
- `score` is the mean of the standardised contributions, clipped to `±3`.
  Momentum and premium contribute their z-score; order flow contributes its
  **sign**, because it covers only a handful of instruments and has no
  cross-section to standardise against. That asymmetry is intentional and is
  restated in the prompt.

Each instrument's state is rendered alongside the **direction timeline**:

```markdown
### Your instruments

| Instrument | Bias | Score | Component evidence |
| :--- | :--- | ---: | :--- |
| BTC | bullish | +1.42 | momentum +1.90σ (+3.20%/24h) (bullish); premium +0.052% (bullish); flow $1.2M (bullish) |

### Recent direction changes

- BTC bullish → neutral, 3h ago — premium bullish→neutral; momentum bullish→neutral
```

Payload shape:

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

#### The direction timeline

Of the three Vergex capabilities this was the **only pure net loss**: the old
build piped the `direction-change/history` JSON straight into the prompt, and no
free source provides it.

The replacement is built here. `marketdata.DirectionTracker` holds each
instrument's previous state and a bounded change log, **persisted to
`data/direction_history.json`** (override with `MARKET_DIRECTION_HISTORY_PATH`):

- **Survives restarts.** A timeline's whole value is still saying "flipped two
  hours ago" after a redeploy; a process that restarts often would otherwise
  always report an empty history
- **A first observation is never a change.** Otherwise every instrument reports a
  flip on the first cycle after a fresh install
- **The reason is computed, not invented.** It comes from a diff of the component
  votes (`momentum bullish→bearish`). When no single component flipped and only
  the aggregate crossed the threshold, it says exactly that
  (`no component flipped; score moved ...`)
- **Writes are atomic** — a sibling `.tmp` then an `os.Rename`, so a crash cannot
  leave a half-encoded file
- **Every persistence failure degrades to memory**: an unwritable directory or a
  corrupt file warns once and never blocks a cycle
- the log is bounded at 200 entries (`directionHistoryMaxChanges`) and 400 tracked
  instruments (`directionHistoryMaxStates`)

### `hyperliquid_flow` — cross-market capital flow

- Free, no key
- Upstream: `metaAndAssetCtxs` on `POST https://api.hyperliquid.xyz/info`
- Covers the ~120 highest-turnover instruments (`flowUniverseSize`)

Five boards:

| Board | Meaning |
|-------|---------|
| `most_traded` | Top N by 24h notional turnover |
| `most_crowded_long` | Most extreme positive funding → longs crowded |
| `most_crowded_short` | Most extreme negative funding → shorts crowded |
| `gainers` / `losers` | Both ends of the 24h change distribution |

Payload shape:

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

Funding is annualized as `hourly * 24 * 365 * 100` because the upstream figure is
an **hourly** rate — multiplying by 3 would silently misreport it as daily.

**Baseline calibration (important):** Hyperliquid's funding formula includes a
fixed base component (0.01% per 8h), which annualizes to roughly **+11%**. Majors
like BTC and ETH therefore print a steady `+11.0%` — that is the base rate, not
long crowding. Both providers state this explicitly in their prompt text
(`baselineFundingNote`) and instruct the model to read only the **deviation** from
it. This is the single most misleading number in the feed: a live check showed
PURR at `+278%` and `xyz:MINIMAX` at `+224%`, which are the real signals.

### `hyperliquid_leverage` — open-interest structure and crowding

- Free, no key
- Shares the same upstream endpoint as `flow`
- Covers ~120 instruments (`leverageUniverseSize`)

Boards: `largest_open_interest`, `long_crowded`, `short_crowded`, `candidates`
(the last contains only current candidates and open positions, capped at 12 rows).

**Important: this is a proxy metric, not a real liquidation heatmap.** The code
comments state it plainly — the free Hyperliquid feed publishes aggregate open
interest and funding, not a price-bucketed liquidation order book. This provider
uses **OI + funding + the markPx deviation from oraclePx** to characterize
leveraged crowding. It answers "where is size concentrated and which side is
stretched", but it is **not** equivalent to per-price liquidation clusters.

For genuine price-bucketed clusters, enable `coinank_liquidation` below.

Payload shape:

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

### `hyperdata_orderflow` / `hyperdata_positioning` — HyperData sidecar (opt-in)

These two read the REST API of
[HyperData Terminal](https://github.com/Co-Messi/HyperData-Terminal), an
**independent open-source Python project** (TUI + REST + paper-trading engine).
There is no importable Go package, so this layer consumes it over **HTTP**.

| | |
|---|---|
| Switch | `cfg.EnableHyperData` (default **false**) |
| URL | `cfg.HyperDataBaseURL`, else `HYPERDATA_BASE_URL`, else `http://127.0.0.1:8420` |
| Credential | `cfg.HyperDataAPIKey`, else `HYPERDATA_API_KEY`; both empty means the sidecar has no key |
| Key required | No — it is a local service, so `RequiresAPIKey()` returns false |
| Badge | Implements `ServiceBacked`; the UI shows "Needs a local service" |

**Why off by default:** it is a separate process someone has to start. Enabling it
by default would log a connection failure and burn the fetch timeout every single
cycle. While off, both sources are skipped and everything else runs normally.

`hyperdata_orderflow` provides:

- **cumulative volume delta** with **per-venue attribution** (`hyperliquid` /
  `binance`)
- per-window net volume (shortest available window), trades per second, aggregate
  signal
- account long/short ratio, and perpetual basis against spot

**Honest coverage disclosure is the point of this block.** It writes each venue's
subscription state into the prompt (`binance (partial, stale, 0/2 sockets, 2
symbols dark)`) and tells the model plainly that **a single-venue delta is weaker
evidence**. Upstream reports the Binance Futures geo-restriction honestly; this
layer does not quietly merge one venue's tape into a two-venue number.

`hyperdata_positioning` provides the largest tracked positions and the ones
closest to liquidation. Its prompt block opens with a hard qualifier:

> **Sample, not a census**: these are positions held by wallets this instance has
> *discovered* within its per-cycle scan budget. Read it as examples of large
> positioning, **never** as the market's total open interest or a complete set of
> liquidation levels.

Known discounts in the upstream data, all stated in the prompt:

- Hyperliquid liquidations are **inferred from large trades**, not exchange-native
- Binance's liquidation stream is throttled at the source to roughly one per
  symbol per second
- Binance Futures streams are geo-blocked in some regions (the socket connects but
  delivers nothing)

#### Upgrading the sidecar

The sidecar being a separate process is exactly what lets it **upgrade
independently**: the Go side speaks HTTP and never links against it. Three rules
make upstream change survivable at runtime:

1. **Unknown fields are ignored** (`encoding/json` default) → upstream *additions*
   are always safe
2. **Loose-typed fields use tolerant types** (`flexStrings` / `flexText` / `any`)
   → upstream turning `signal` from a string into an object degrades to readable
   text instead of failing the call
3. **Version comparison**: when the reported `version` disagrees with
   `hyperDataTestedMajor` (currently 1) the adapter warns in the log and in the
   prompt, but **does not fail** — rules 1 and 2 already cover most changes

Those rules decide what happens *during a trading cycle*. They do not tell you
what an upgrade changed before you adopt it, which is the other half:

- **The pin.** `deploy/hyperdata/upstream.env` names the exact 40-character commit
  the adapter was validated against, plus the tested major. It is read by
  `TestUpstreamPinIsStillMeaningful`, so a bump that changes only half of the
  version trio (the env file, `hyperDataTestedMajor`, the health fixture's
  `version`) fails the build instead of passing quietly.
- **The drift report.** `TestHyperDataUpstreamHasNotDrifted` compares the recorded
  fixtures against a *running* sidecar and classifies every difference as
  `MISSING` / `TYPE CHANGED` (breaking) or `ADDED` (informational). It is opt-in
  behind `HYPERDATA_LIVE_URL` so the normal suite stays hermetic. This is the part
  that answers "what did the upgrade actually change" — the contract test can only
  answer "did we break our own parsing". Because it compares *shapes*, its limits
  are specific: a field whose **meaning** changed while its name and type stayed
  the same (a unit switching from USD to contracts) passes unnoticed, and so does
  a rename of a key inside a collapsed container. A rename of an ordinary field is
  reported as `MISSING` plus `ADDED`.

The drift check is a **Go test, not a separate probe script**, on purpose: it
reuses the same client, the same fixtures and the same collapse rules as the
adapter, so there is one contract instead of two that can disagree.

Runbook, container deployment and the bump procedure:
[`deploy/hyperdata/README.md`](../../deploy/hyperdata/README.md).

### `coinank_liquidation` — price-bucketed liquidation clusters (opt-in)

- **Requires an API key** (`cfg.CoinankAPIKey`); without one `Enabled` returns `false`
- Upstream: `https://api.coinank.com`
- Pulls **individual** liquidation fills via `client.LiquidationOrders(...)` (each
  carries a `price`), then buckets them locally into **12 price bands**
  (`liquidationBucketCount`)
- Processes at most **3** symbols per cycle (`maxLiquidationSymbols`), open
  positions first

This is the 1:1 replacement for the old Vergex "position cost / liquidation
heatmap": Vergex sold exactly this — price-aggregated liquidation clusters — and
per-fill data plus local bucketing computes the same thing without a paid
subscription.

Payload shape:

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

## Adding a new source

**One change point.** Three steps:

### 1. Create `marketdata/providers/<yours>.go`

```go
package providers

type MyProvider struct{}

func NewMyProvider() *MyProvider { return &MyProvider{} }

func (p *MyProvider) Name() string         { return "my_source" }
func (p *MyProvider) Description() string  { return "What this source offers" }
func (p *MyProvider) RequiresAPIKey() bool { return false }

func (p *MyProvider) Enabled(cfg store.IndicatorConfig) bool {
    // key-gated sources check here: return cfg.MyAPIKey != ""
    return true
}

func (p *MyProvider) Fetch(ctx context.Context, req marketdata.Request, cfg store.IndicatorConfig) (*marketdata.Insight, error) {
    limit := req.NormalizeLimit(10)
    // ... fetch ...
    return &marketdata.Insight{
        Provider:   p.Name(),
        Title:      "My source",
        Markdown:   "...",   // English prompt block
        MarkdownZh: "...",   // optional; falls back to English
        Payload:    payload, // for API / UI
        FetchedAt:  time.Now(),
    }, nil
}
```

### 2. Register it in `marketdata/providers/registry.go`

```go
func defaultProviders() []marketdata.Provider {
    return []marketdata.Provider{
        NewDirectionalSignalProvider(),
        NewHyperliquidFlowProvider(),
        NewHyperliquidLeverageProvider(),
        NewHyperDataOrderflowProvider(),
        NewHyperDataPositioningProvider(),
        NewCoinankLiquidationProvider(),
        NewMyProvider(),   // ← add this line
    }
}
```

The array order **is the block order in the prompt**, which is why it is not
arbitrary: `directional_signal` leads because it is the only source that states a
verdict, and the descriptive sources that follow are what a reader uses to check it.

### 3. Add a config field only if needed

If the source needs an API key or a dedicated toggle, add the field to
`store.IndicatorConfig`. **Do not** add an `EnableMySource` boolean — the
`MarketInsightSources` allow-list already expresses "which sources are on", and an
empty list enables everything.

The exception is a source that needs **an external process**: give it its own
switch (like `EnableHyperData`), because "empty allow-list means everything" would
otherwise make an unstarted service get contacted every cycle.

### 4. If the source speaks HTTP, add a contract test

This is a requirement, not a suggestion. `marketdata/providers/hyperdata_contract_test.go`
is the reference:

- `httptest` fixtures pin the response shapes, with field names taken verbatim from
  the upstream source
- a second group asserts **tolerated drift**: added fields, loosened types, `null`
- rendered tables go through `assertTablesAligned`, which checks column counts.
  Prompt tables are read **positionally**, so a row with a missing cell silently
  reassigns every value to its right — a bug that occurred once while this layer
  was being written

When upstream changes a field, changing the fixture shows exactly which assertions
break, which is far earlier than a live trading cycle would.

### What you do not touch

- `kernel/engine.go` — `CollectInsights` only knows `Registry`
- `kernel/engine_prompt.go` / `kernel/formatter.go` — iterate `ctx.Insights`
- `trader/auto_trader_loop.go` — a single `CollectInsights` call
- `api/handler_market_insights.go` — lists the registry, so new sources appear automatically

**The frontend adapts automatically too**: `/api/market-insights/providers`
returns the new source and Strategy Studio grows one more source card.

---

## Frontend / backend contract

### Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/market-insights/providers` | Lists **all** sources (enabled or not) with `name` / `description` / `requires_key` / `requires_service` |
| `GET` | `/api/market-insights?lang=&sources=&hyperdata=&hyperdata_url=` | Collects data; `sources` is a comma-separated allow-list, omitted means every enabled source; `hyperdata=1` explicitly requests the sidecar sources |

The `hyperdata` parameter exists because the preview endpoint builds a **synthetic
config** rather than reading a strategy's, so the sidecar sources (off by default)
have to be requested explicitly. The service credential is deliberately **not**
accepted over the query string.

### Configuration fields

`store.IndicatorConfig`:

```go
EnableMarketInsights bool     `json:"enable_market_insights"`
MarketInsightSources []string `json:"market_insight_sources,omitempty"`
MarketInsightLimit   int      `json:"market_insight_limit,omitempty"`
CoinankAPIKey        string   `json:"coinank_api_key,omitempty"`

// Optional sidecar (HyperData Terminal)
EnableHyperData  bool   `json:"enable_hyperdata"`
HyperDataBaseURL string `json:"hyperdata_base_url,omitempty"`
HyperDataAPIKey  string `json:"hyperdata_api_key,omitempty"`
```

Default strategy (`GetDefaultStrategyConfig`): master switch `true`, limit `10`,
`MarketInsightSources` left empty → every free source runs out of the box;
`EnableHyperData` stays `false`.

URL and credential resolve in the order **strategy field → environment variable →
built-in default**. The environment fallback exists so one deployment configures
the sidecar once, while the strategy field still wins and keeps per-strategy
routing possible.

### Token estimation

The strategy token estimate is computed from the source count: an empty allow-list
starts at **3 sources** (direction, flow, open-interest structure), plus **2** when
`EnableHyperData` is on, plus **1** when a CoinAnk key is set.

Each source's budget is split in two, because **orientation prose does not scale
with the row count**:

```
(numSources × 700 fixed chars + numSources × limit × 90 chars) / 4
```

An earlier version budgeted only `limit × 80`, which materially under-counted the
direction block — that block is mostly orientation text around a small table.

### Frontend

| Location | Role |
|----------|------|
| `web/src/pages/StrategyStudioPage.tsx` | "Market sources" config area: master switch, source cards badged free / API-key / needs-a-local-service, row-count dropdown, plus the sidecar switch, URL and key |
| `web/src/components/terminal/MarketInsightsPanel.tsx` | Terminal panels — one table per provider: direction, flow, leverage, order flow, positioning, coverage |
| `web/src/lib/api/data.ts` | `getMarketInsightSources()` / `getMarketInsights()` |

Panels use distinct SWR keys for sidecar and non-sidecar requests
(`'market-insights'` vs `'market-insights:sidecar'`), so panels of the same kind
share **one** request. Polling is `120s` because the upstream board is cached for
five minutes — polling faster would only re-render identical numbers.

---

## Retired capabilities and their replacements

| Old capability | Status |
|----------------|--------|
| Vergex direction board (`bias` + z-score vote) | **`directional_signal`** — a transparent reconstruction that is neither declared authoritative nor tied to a paid gateway |
| Vergex direction-change history (X→Y timeline + reason) | The **`directional_signal`** timeline backed by `DirectionTracker` |
| Vergex position cost / liquidation heatmap | `hyperliquid_leverage` (free proxy) + `coinank_liquidation` (opt-in, true price buckets) + `hyperdata_positioning` (opt-in, per-position distance) |
| Vergex per-symbol detail (direction / history / heatmap JSON) | `hyperdata_orderflow` (CVD with per-venue attribution) + `hyperdata_positioning` (per-position detail) |
| nofxos OI / NetFlow / Price ranking endpoints | Removed (endpoints return 402; the bundled key is dead) |
| `signal_managed_exit` | **Implemented** (trade-layer semantics, unrelated to any data source). `risk_control.exit_mode` takes `fixed` (default), `signal` or `both`; under `signal` the direction read recorded at entry closes the position when it flips or decays below `signal_score_floor`, and `validateProtectionPrices` requires a stop but no longer a target. Entry theses persist to `data/signal_book_<trader>.json`, so they survive a restart |

### Two deliberate differences from the old board

Same subject, different use. This is intentional:

| | Old Vergex | `directional_signal` |
|---|---|---|
| Verdict origin | External paid service, algorithm invisible | Computed locally from public data, **every component listed** |
| Prompt standing | System prompt declared it authoritative | No priority claim; equal to the other blocks |
| Relationship to candles | Explicitly "candles may not veto, reverse or exit early" | Left to the model to weigh |
| Cost | Bound to a claw402 wallet, billed per call | Free |

In other words: **the capability is restored, the authority is not.** Restoring
"follow the signal and do not exit early on candles" is a prompt-layer change and
should not be smuggled into the data layer.

Removed from `provider/nofxos`: `netflow.go`, `price.go`, `OIRankingData` and its
three formatters. What remains is `fetchOIRanking` and the `GetOITopSymbols`
family — those serve **candidate-pool construction** and are unrelated to this layer.

---

## File map

```
marketdata/
├── insight.go                 # Insight / Request / Provider / SourceSetting / ServiceBacked
├── registry.go                # Registry: concurrent collect, timeout, soft degradation, coverage
├── direction_history.go       # DirectionTracker: bias + change timeline, atomic file persistence
└── providers/
    ├── registry.go            # defaultProviders() ← the only wiring point
    ├── format.go              # annualization, USD compaction, symbol display, ranking
    ├── directional_signal.go  # direction verdict + timeline wiring
    ├── hyperliquid_flow.go
    ├── hyperliquid_leverage.go
    ├── hyperdata_client.go    # HyperData REST client + tolerant types + version check
    ├── hyperdata_orderflow.go
    ├── hyperdata_positioning.go
    ├── hyperdata_contract_test.go  # recorded fixtures + tolerated-drift assertions
    ├── hyperdata_upstream_test.go  # live drift report + upstream.env pin assertions
    └── coinank_liquidation.go

deploy/hyperdata/              # sidecar install / upgrade / contract verification
├── upstream.env               # the pinned revision + tested major (single source of truth)
├── sidecar.sh                 # install | update | status | contract | run
├── Dockerfile                 # builds the pin, and refuses to build off-pin
├── README.md                  # upgrade procedure and failure handling
└── README.zh-CN.md

docker-compose.hyperdata.yml   # additive compose file, profile "hyperdata"

api/
├── handler_market_insights.go   # the two HTTP endpoints
└── server.go                    # route registration

kernel/
├── engine.go               # CollectInsights / MarketInsightProviders
├── engine_prompt.go        # primary prompt path (renders ctx.Insights)
└── formatter.go            # fallback path (same)
```

---

[← Back to architecture docs](README.md)
