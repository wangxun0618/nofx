# HyperData Terminal sidecar

**Language:** [English](README.md) | [中文](README.zh-CN.md)

The `hyperdata_orderflow` and `hyperdata_positioning` market data sources read
from [HyperData Terminal](https://github.com/Co-Messi/HyperData-Terminal), a
separate open-source project that aggregates order flow, whale positions and
liquidation data from five exchanges.

## Why it runs as a process, not as a library

HyperData Terminal is a Python application (TUI + REST API + paper-trading
engine). There is no importable Go package, so there is nothing to link against.
The backend therefore **consumes its REST API over HTTP**.

That boundary is deliberate, and it is what makes the requirement "the library
must support future updates" actually achievable:

| Concern | How it is handled |
|---|---|
| Upstream ships a new version | Move the pin in `upstream.env`; nothing in the Go tree changes |
| Upstream renames or adds a field | The adapter ignores unknown fields and tolerates loose types, so additions are safe |
| Upstream changes a field's meaning | `sidecar.sh contract` asserts the specific keys each provider reads |
| Upstream breaks the API shape | The provider fails soft; the prompt gains a `data_coverage` block naming it instead of silently dropping the data |
| The sidecar is down | Providers are skipped, trading continues, the gap is reported |
| Upstream major version moves | `VersionWarning` logs it and the block renders a warning line |

Upstream publishes **no git tags** (checked 2026-09-22), so the pin is a commit
SHA. That is the only reproducible option.

## Layout

```
upstream.env   pinned revision, verified major + defaults (the single source of truth)
sidecar.sh     install / update / status / contract / run
Dockerfile     builds the sidecar image from the pin, and refuses to build off-pin
README.md      this document
```

There is deliberately **no separate probe script**. The live compatibility check is
a Go test (`marketdata/providers/hyperdata_upstream_test.go`) so that the contract
has one definition instead of two: it reuses the adapter's own HTTP client, the
same recorded fixtures and the same shape rules. A hand-maintained key list in a
second script drifts away from the fixtures and then passes for the wrong reason.

`docker-compose.hyperdata.yml` (repository root) composes it next to the backend.

## Install

```bash
./deploy/hyperdata/sidecar.sh install
./deploy/hyperdata/sidecar.sh run       # foreground; use a supervisor in production
```

The checkout and its virtualenv live in `$HYPERDATA_DIR` (default
`~/.nofx/hyperdata`), never inside this repository, so an install or upgrade
cannot dirty the working tree.

Point the backend at it in one of two ways:

```bash
# Globally, inherited by every strategy:
HYPERDATA_BASE_URL=http://127.0.0.1:8420
HYPERDATA_API_KEY=<only if the sidecar was started with one>
```

or per strategy, in Strategy Studio → market data sources → *Enable the
HyperData Terminal sidecar*. A strategy-level value wins over the environment,
so both can coexist.

## Verify

```bash
./deploy/hyperdata/sidecar.sh contract
```

This runs two halves:

1. **Recorded contract (offline).** The Go fixture tests in
   `marketdata/providers/hyperdata_contract_test.go` pin the response shapes the
   adapter was built against, including the tolerated drift cases.
2. **Live contract.** `TestHyperDataUpstreamHasNotDrifted` probes the running
   instance and classifies every difference from those fixtures:

   ```
   MISSING       a field the adapter reads is gone. Breaking.
   TYPE CHANGED  the field survived but changed JSON kind. Breaking.
   ADDED         upstream grew a field we ignore. Read the release notes.
   ```

   It is opt-in behind `HYPERDATA_LIVE_URL` so the normal test suite stays
   hermetic, and `sidecar.sh contract` sets that variable for you. When the
   sidecar is not running the live half is skipped with a message rather than
   reported as a failure.

Both matter, and neither substitutes for the other. The fixture tests answer "did
we break our own parsing?" and only fail when the Go code changes. The drift
report answers "what did the upgrade change?" and is the only check that can
notice a real upstream shift.

Know what it cannot see, though, because it compares *shapes*:

- **A field whose meaning changed while its name and type stayed the same.**
  `open_interest` switching from USD to contracts, or a ratio switching from a
  fraction to a percent, is invisible here and would silently rescale the prompt.
  Only the release notes catch that.
- **A rename of a key inside a container the report collapses.** Those keys are
  dynamic by design (venue, symbol, timeframe), so it rarely matters — but a
  fixed wrapper holding a single object is collapsed too, and its name could be
  renamed without a word.

A rename of an ordinary field *is* reported, as `MISSING` on the old name plus
`ADDED` on the new one. The report narrows the review; it does not replace it.

### What keeps the pin honest

`upstream.env` is not documentation — it is read by
`TestUpstreamPinIsStillMeaningful`, which fails the build unless three things
agree:

| Where | What |
|---|---|
| `deploy/hyperdata/upstream.env` | `HYPERDATA_REVISION` (must be a full 40-char SHA) and `HYPERDATA_TESTED_MAJOR` |
| `marketdata/providers/hyperdata_client.go` | `hyperDataTestedMajor` |
| `marketdata/providers/hyperdata_contract_test.go` | the `version` inside `fixtureHealth` |

Because upstream publishes no tags, the revision has to be a commit for the build
to be reproducible; the test rejects anything else. A partial bump — moving the
pin but leaving the constant or the fixture behind — fails loudly instead of
passing quietly.

## Upgrade

```bash
./deploy/hyperdata/sidecar.sh update --to main   # or a tag, or a commit SHA
./deploy/hyperdata/sidecar.sh contract
# when it passes, write the new SHA and major into upstream.env
```

`update` never edits `upstream.env` for you: recording what was actually verified
should be a deliberate act, not a side effect.

If the drift report lists breaking changes, the change is upstream's, not the
adapter's. Options, in order of preference:

1. **Adapt** — add the field to the tolerant type or the mapping, extend the
   fixture in the contract test, and re-run. Remember the fixtures are what the
   drift report compares against, so refreshing them is what records the new
   contract.
2. **Pin back** — leave `HYPERDATA_REVISION` where it was; the previous version
   still works, because nothing else changed.
3. **Disable** — `enable_hyperdata: false` in the strategy. The rest of the
   market-insight layer is unaffected.

## Deploy with Docker

```bash
# The revision is not duplicated in the compose file, so source the pin.
set -a; . deploy/hyperdata/upstream.env; set +a

HYPERDATA_API_KEY=$(openssl rand -hex 24) \
  docker compose -f docker-compose.yml -f docker-compose.hyperdata.yml \
  --profile hyperdata up -d --build
```

Then set the base URL to `http://hyperdata:8420` with that key.

`docker compose` fails immediately if `HYPERDATA_REVISION` or `HYPERDATA_API_KEY`
is unset. That is intended: without the pin the image would be built from
whatever `main` happened to be, and without the key upstream refuses to bind a
non-loopback address, so the container would run and answer nothing.

The API serves wallet-derived positions, liquidation danger zones and live order
flow. Upstream binds to loopback by default and refuses a non-loopback bind
without a key; the overlay sets one and does **not** publish the port, so the
container network is the only way in. Do not add a `ports:` mapping for it — to
inspect it from the host, use:

```bash
docker compose exec hyperdata curl -s localhost:8420/v1/health
```

## Known limits of the data

These are upstream's own caveats, and the prompt states them so the model does
not read more into the numbers than they support:

- **Whale positions are sampled.** Upstream scans a set of addresses it has
  discovered, within a per-cycle budget. It is not a census of the market.
- **Hyperliquid liquidations are inferred** from large trades, and Binance's
  stream is throttled at the source to roughly one liquidation per symbol per
  second. This is not Coinglass-grade coverage.
- **Binance Futures streams are geo-blocked in some regions.** The socket
  connects but delivers nothing; upstream reports the venue as `partial` rather
  than hiding it, and the block tells the model that a single-venue delta is
  weaker evidence.
- **Order flow is per-symbol**, and upstream only has data for the symbols its
  instance tracks. Symbols it does not track are listed as such.
