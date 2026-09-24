# Changelog

All notable changes to the NOFX project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**Languages:** [English](CHANGELOG.md) | [中文](CHANGELOG.zh-CN.md)

---

## [Unreleased]

### Added
- Documentation system with multi-language support (EN/CN/RU/UK)
- Complete getting-started guides (Docker, Custom API)
- Architecture documentation with system design details
- User guides with FAQ and troubleshooting
- Community documentation with bounty programs
- **Pluggable market-intelligence layer** (`marketdata`): the `Provider` / `Registry` / `Insight` abstraction — adding a data source means implementing the interface and listing it in `marketdata/providers/registry.go`, with no change to the engine, the trading loop or the prompt builder
- Two fully free, keyless built-in sources: `hyperliquid_flow` (cross-market fund flow: turnover, movers, funding crowding) and `hyperliquid_leverage` (open-interest structure, leverage crowding, mark-vs-oracle dislocation)
- Optional `coinank_liquidation` source: rebuilds **price-bucketed liquidation clusters** from individual liquidation fills, replacing the retired Vergex heatmap; it enables itself once a CoinAnk API key is configured
- New endpoints: `GET /api/market-insights/providers` (list every source) and `GET /api/market-insights` (collect the board)
- Strategy Studio "Market data sources" section: master switch, source cards with free / API-key badges, row count and credential input
- Terminal panels for cross-market fund flow and open-interest structure, rendering exactly the data the AI reads when deciding
- **`directional_signal` source**: votes price momentum, perpetual premium and aggressive order flow into a per-instrument `bias` and a signal-strength `score`, with **every component listed** so the verdict can be audited. Replaces the retired Vergex board without claiming to be authoritative
- **Direction-change timeline**: `marketdata.DirectionTracker` records bias flips (the X→Y transition plus a reason derived from a diff of the component votes), persisted to `data/direction_history.json` so it survives restarts
- **HyperData Terminal sidecar integration**: optional `hyperdata_orderflow` (multi-venue CVD with per-venue attribution, account long/short ratio, basis) and `hyperdata_positioning` (largest tracked positions, distance to liquidation) sources, consumed over HTTP from that independent open-source project's REST API
- **Upstream upgrade tooling** under `deploy/hyperdata/`: a commit pin in `upstream.env`, `sidecar.sh` (install / update / status / contract / run), a `Dockerfile` that refuses to build off-pin, and `docker-compose.hyperdata.yml` for running the sidecar as its own service
- **Live upstream drift report** (`TestHyperDataUpstreamHasNotDrifted`): probes a running sidecar and classifies every difference from the recorded fixtures as `MISSING` / `TYPE CHANGED` (breaking) or `ADDED` (informational), so an upgrade is triaged before it is adopted. It is a Go test rather than a second probe script, so the HTTP contract has one definition instead of two that can disagree
- **The upstream pin is enforced, not documented**: `TestUpstreamPinIsStillMeaningful` fails the build unless the pinned commit, `hyperDataTestedMajor` and the recorded health fixture all describe the same upstream major, so a half-finished bump cannot pass quietly
- **Contract tests for the HTTP source**: fixtures pin the upstream response shapes, with a second group asserting **tolerated drift** (added fields, loosened types, `null`), so an upstream change fails a test instead of silently degrading in a live cycle
- **Data-source coverage block**: sources that contributed nothing are written into the prompt as a `data_coverage` block, so the model cannot read a dead feed as a quiet market
- `marketdata.ServiceBacked`, an optional interface for sources that need another process running but no credential

### Changed
- Reorganized documentation structure into logical categories
- Updated all README files with proper navigation links
- AI inference now talks to eight native providers directly (DeepSeek, OpenAI, Claude, Qwen, Gemini, Grok, Kimi, MiniMax) with your own API keys — no gateway sits in between
- The built-in autopilot strategy now defaults to the native Hyperliquid top-volume universe (`hyper_main`: 24h volume, top 30 instruments) instead of the retired signal board
- Product copy across the landing page, launch flow, strategy studio and terminal no longer mentions pay-per-call model billing
- Market-level data went from four hardcoded chains (context fields typed as concrete sources, a single client on the engine, three `if` blocks in the loop, formatter calls in the prompt) to a single `Registry.Collect` call
- Source failures went from silently dropped to recorded and reported at the end of the prompt, because a silent drop leaves the model unable to tell "nothing happened" from "the source is dead"
- `IndicatorConfig` token estimation now counts the sources actually enabled and budgets orientation prose separately from table rows — the old formula counted rows only and materially under-estimated the direction block

### Removed
- The Claw402 / x402 pay-per-call gateway: model routing, direction-board data, the USDC wallet package, per-call billing records, the launch preflight balance gate and the onboarding wallet flow
- The Vergex signal board, direction-change leaderboard and cost/liquidation heatmap data sources together with their terminal and strategy-studio surfaces (the direction board and its change history are now rebuilt as `directional_signal`; the heatmap is covered by `coinank_liquidation`, `hyperliquid_leverage` and `hyperdata_positioning`; the `signal_managed_exit` mode is **not** restored, because it is trade-layer semantics rather than a data-source capability)
- Three nofxos ranking capabilities: `OIRankingData` with `GetOIRanking`, `netflow.go`, `price.go` and their formatters (the endpoints return 402)
- Dead frontend routes and components: `/data`, `/strategy-market`, the beginner-onboarding wallet page, the beginner guide cards and the onboarding mode selector
- The rest of the nofxos ranking surface, now that nothing calls it: `GetAI500List` / `GetTopRatedCoins` / `GetAvailableCoins`, the AI500 TTL cache and its test, the OI top / OI low ranking types and methods, `GetCoinDataBatch`, `FormatQuantDataForAI` and the unused auth-key accessors. Those endpoints all return 402; only `GetCoinData` is still reachable, so the package stays
- The dead frontend chart stack — roughly 4.5k lines with no importer, reachable only through each other: `AdvancedChart`, `ChartTabs`, `EquityChart`, `ChartWithOrders`, `ChartWithOrdersSimple`, `TradingViewChart`, `TraderDashboardPage`, `PageNotFound`, `BrandHero`, `BrandStats`, the `stores/` directory, `useCounterAnimation`, `ui/input.tsx`, `lib/text.ts`, `utils/indicators.ts`, plus the i18n blocks (`advancedChart`, `chartWithOrders`, `traderDashboard`, `notFound`) that only they read
- Unwired Go helpers: the `safe` package (zero importers), `kernel/prompt_builder.go` (superseded by `engine_prompt.go`, reachable only from its own test), `trader/position_snapshot.go`, `trader/helpers.go` and the `ExportCalculate*` wrappers in `market/data_indicators.go`
- Five npm packages nothing imported: `date-fns`, `@radix-ui/react-slot`, `class-variance-authority`, `lightweight-charts` and `zustand`

### Fixed
- Four UI strings resolved to keys that never existed, so Chinese users saw the raw key or its English fallback: the sign-in button, the chart loading state, the comparison-chart PnL label and the Lighter API-key toast now resolve to `auth.loggingIn`, `chart.loadingChartData`, `chart.leadPnL` and `exchangeCfg.lighterApiKeyImported`
- **Account-level circuit breaker is now real, not a name**: `MaxDailyLoss` / `MaxDrawdown` were declared and never read, `stopUntil` was never assigned (so the pause branch was unreachable) and `dailyPnL` was only ever reset — the reported `daily_pnl` was permanently zero. The loop now measures mark-to-market equity every cycle, pauses trading when it falls past `risk_control.max_daily_loss_pct` from the day's opening equity or `risk_control.max_drawdown_pct` from the running peak, and reports the numbers it acts on. New strategies start with 10% / 20% / 240 min. A pause blocks new positions and never force-closes: exchange-side stops and the profit-giveback monitor keep running
- **Anonymous trade statistics are opt-in**: `EXPERIENCE_IMPROVEMENT` used to default to on, so a fresh install shipped exchange, symbol, order size and leverage to Google Analytics before the operator agreed to anything. It now requires an explicit `true`
- **`signal_managed_exit` is reachable and survives a restart**: `exit_mode` / `signal_score_floor` are exposed in the strategy editor and the API docs, and the entry thesis behind each position is persisted to `data/signal_book_<trader>.json`. Previously the book lived only in memory, so after every restart the loop compared against a fabricated `0.00` entry score and could close the whole book on the first cycle — and the docs still said the mode was "not restored"
- `GET /api/prompt-templates` was called by the web client but has never been registered on this server; the unused helper is gone
- `indicators.enable_quant_data` retried a dead endpoint once per symbol per cycle: an entitlement answer (401/402/403) now latches the source off for the process and states the reason once, in the same note channel candidate-pool degradations use
- `klines.longer_count` was a knob that did nothing — the fetch path always used `primary_count`. It is removed from the schema, the API and the editor rather than left as a trap
- README said "nine exchanges" where ten adapters exist; the entry for the retired `signal_managed_exit` mode in the market-data docs said the mode was not restored

---

## [3.0.0] - 2025-10-30

### Added - Major Architecture Transformation 🚀

**Complete System Redesign - Web-Based Configuration Platform**

This is a **major breaking update** that completely transforms NOFX from a static config-based system to a modern web-based trading platform.

#### Database-Driven Architecture
- SQLite integration replacing static JSON config
- Persistent storage with automatic timestamps
- Foreign key relationships and triggers for data consistency
- Separate tables for AI models, exchanges, traders, and system config

#### Web-Based Configuration Interface
- Complete web-based configuration management (no more JSON editing)
- AI Model setup through web interface (DeepSeek/Qwen API keys)
- Exchange management (Binance/Hyperliquid credentials)
- Dynamic trader creation (combine any AI model with any exchange)
- Real-time control (start/stop traders without system restart)

#### Flexible Architecture
- Separation of concerns (AI models and exchanges independent)
- Mix & match capability (unlimited combinations)
- Scalable design (support for unlimited traders)
- Clean slate approach (no default traders)

#### Enhanced API Layer
- RESTful design with complete CRUD operations
- New endpoints:
  - `GET/PUT /api/models` - AI model configuration
  - `GET/PUT /api/exchanges` - Exchange configuration
  - `POST/DELETE /api/traders` - Trader management
  - `POST /api/traders/:id/start|stop` - Trader control
- Updated documentation for all API endpoints

#### Modernized Codebase
- Type safety with proper separation of configuration types
- Database abstraction with prepared statements
- Comprehensive error handling and validation
- Better code organization (database, API, business logic)

### Changed
- **BREAKING**: Old `config.json` files no longer used
- Configuration must be done through web interface
- Much easier setup and better UX
- No more server restarts for configuration changes

### Why This Matters
- 🎯 **User Experience**: Much easier to configure and manage
- 🔧 **Flexibility**: Create any combination of AI models and exchanges
- 📊 **Scalability**: Support for complex multi-trader setups
- 🔒 **Reliability**: Database ensures data persistence and consistency
- 🚀 **Future-Proof**: Foundation for advanced features

---

## [2.0.2] - 2025-10-29

### Fixed - Critical Bug Fixes: Trade History & Performance Analysis

#### PnL Calculation - Major Error Fixed
- **Fixed**: PnL now calculated as actual USDT amount instead of percentage only
- Previously ignored position size and leverage (e.g., 100 USDT @ 5% = 1000 USDT @ 5%)
- Now: `PnL (USDT) = Position Value × Price Change % × Leverage`
- Impact: Win rate, profit factor, and Sharpe ratio now accurate

#### Position Tracking - Missing Critical Data
- **Fixed**: Open position records now store quantity and leverage
- Previously only stored price and time
- Essential for accurate PnL calculations

#### Position Key Logic - Long/Short Conflict
- **Fixed**: Changed from `symbol` to `symbol_side` format
- Now properly distinguishes between long and short positions
- Example: `BTCUSDT_long` vs `BTCUSDT_short`

#### Sharpe Ratio Calculation - Code Optimization
- **Changed**: Replaced custom Newton's method with `math.Sqrt`
- More reliable, maintainable, and efficient

### Why This Matters
- Historical trade statistics now show real USDT profit/loss
- Performance comparison between different leverage trades is accurate
- AI self-learning mechanism receives correct feedback
- Multi-position tracking (long + short simultaneously) works correctly

---

## [2.0.2] - 2025-10-29

### Fixed - Aster Exchange Precision Error

- Fixed Aster exchange precision error (code -1111)
- Improved price and quantity formatting to match exchange requirements
- Added detailed precision processing logs for debugging
- Enhanced all order functions with proper precision handling

#### Technical Details
- Added `formatFloatWithPrecision` function
- Price and quantity formatted according to exchange specifications
- Trailing zeros removed to optimize API requests

---

## [2.0.1] - 2025-10-29

### Fixed - ComparisonChart Data Processing

- Fixed ComparisonChart data processing logic
- Switched from cycle_number to timestamp grouping
- Resolved chart freezing issue when backend restarts
- Improved chart data display (shows all historical data chronologically)
- Enhanced debugging logs

---

## [2.0.0] - 2025-10-28

### Added - Major Updates

- AI self-learning mechanism (historical feedback, performance analysis)
- Multi-trader competition mode (Qwen vs DeepSeek)
- Binance-style UI (complete interface imitation)
- Performance comparison charts (real-time ROI comparison)
- Risk control optimization (per-coin position limit adjustment)

### Fixed

- Fixed hardcoded initial balance issue
- Fixed multi-trader data sync issue
- Optimized chart data alignment (using cycle_number)

---

## [1.0.0] - 2025-10-27

### Added - Initial Release

- Basic AI trading functionality
- Decision logging system
- Simple Web interface
- Support for Binance Futures
- DeepSeek and Qwen AI model integration

---

## How to Use This Changelog

### For Users
- Check the [Unreleased] section for upcoming features
- Review version sections to understand what changed
- Follow migration guides for breaking changes

### For Contributors
When making changes, add them to the [Unreleased] section under appropriate categories:
- **Added** - New features
- **Changed** - Changes to existing functionality
- **Deprecated** - Features that will be removed
- **Removed** - Features that were removed
- **Fixed** - Bug fixes
- **Security** - Security fixes

When releasing a new version, move [Unreleased] items to a new version section with date.

---

## Links

- [Documentation](docs/README.md)
- [Contributing Guidelines](CONTRIBUTING.md)
- [Security Policy](SECURITY.md)
- [GitHub Repository](https://github.com/NoFxAiOS/nofx)

---

**Last Updated:** 2025-11-01
