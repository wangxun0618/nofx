#!/usr/bin/env bash
#
# Manage the HyperData Terminal sidecar that the hyperdata_* market data
# providers read from.
#
# The Go backend never links against the sidecar — it calls its REST API over
# HTTP. That boundary is the entire reason the two projects can be upgraded
# independently, and this script is the only place that knows how the sidecar is
# built or which revision is current.
#
#   ./deploy/hyperdata/sidecar.sh install          install the pinned revision
#   ./deploy/hyperdata/sidecar.sh update [--to REF]  move to a newer revision
#   ./deploy/hyperdata/sidecar.sh status           what is installed and running
#   ./deploy/hyperdata/sidecar.sh contract         verify compatibility
#   ./deploy/hyperdata/sidecar.sh run              start the REST API
#
# `contract` runs two halves: the recorded fixtures offline, and a live drift
# report against the running instance. The live half is a Go test (see
# marketdata/providers/hyperdata_upstream_test.go) rather than a separate script,
# so there is one contract instead of two that can disagree.
#
# Nothing here touches the repository working tree: the checkout and its
# virtualenv live in HYPERDATA_DIR.

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$HERE/../.." && pwd)"

if [ -f "$HERE/upstream.env" ]; then
  # shellcheck source=/dev/null
  . "$HERE/upstream.env"
fi

HYPERDATA_REPO="${HYPERDATA_REPO:-https://github.com/Co-Messi/HyperData-Terminal.git}"
HYPERDATA_REF="${HYPERDATA_REF:-main}"
HYPERDATA_REVISION="${HYPERDATA_REVISION:-}"
HYPERDATA_TESTED_MAJOR="${HYPERDATA_TESTED_MAJOR:-1}"
HYPERDATA_DIR="${HYPERDATA_DIR:-$HOME/.nofx/hyperdata}"
HYPERDATA_PORT="${HYPERDATA_PORT:-8420}"
HYPERDATA_PYTHON="${HYPERDATA_PYTHON:-python3}"

VENV="$HYPERDATA_DIR/.venv"
REVISION_FILE="$HYPERDATA_DIR/.installed-revision"
MIN_PYTHON="3.12"

log()  { printf '%s\n' "$*"; }
fail() { printf 'error: %s\n' "$*" >&2; exit 1; }

python_version_ok() {
  "$HYPERDATA_PYTHON" - "$MIN_PYTHON" <<'PY'
import sys
minimum = tuple(int(part) for part in sys.argv[1].split("."))
current = sys.version_info[:2]
if current < minimum:
    print(f"python {current[0]}.{current[1]} is too old; upstream needs {sys.argv[1]}+", file=sys.stderr)
    sys.exit(1)
PY
}

require_python() {
  command -v "$HYPERDATA_PYTHON" >/dev/null 2>&1 \
    || fail "$HYPERDATA_PYTHON not found; install Python $MIN_PYTHON+ or set HYPERDATA_PYTHON"
  python_version_ok || fail "Python $MIN_PYTHON+ is required"
}

require_git() {
  command -v git >/dev/null 2>&1 || fail "git not found; it is needed to fetch the upstream revision"
}

# resolve_target prints the revision to check out: an explicit ref when given,
# otherwise the pin from upstream.env.
resolve_target() {
  if [ -n "${1:-}" ]; then
    printf '%s' "$1"
  else
    printf '%s' "$HYPERDATA_REVISION"
  fi
}

checkout_revision() {
  local target="$1"

  if [ ! -d "$HYPERDATA_DIR/.git" ]; then
    log "cloning $HYPERDATA_REPO"
    mkdir -p "$(dirname "$HYPERDATA_DIR")"
    git clone --quiet "$HYPERDATA_REPO" "$HYPERDATA_DIR"
  else
    log "fetching latest from origin"
    git -C "$HYPERDATA_DIR" fetch --quiet --tags origin
  fi

  # A 40-hex target is a commit and has to be fetched explicitly; anything else
  # is a branch or tag that `git fetch origin <name>` already brought down.
  if printf '%s' "$target" | grep -Eq '^[0-9a-f]{40}$'; then
    git -C "$HYPERDATA_DIR" fetch --quiet origin "$target" 2>/dev/null \
      || fail "could not fetch commit $target from origin"
  fi

  log "checking out $target"
  git -C "$HYPERDATA_DIR" checkout --quiet --force "$target" 2>/dev/null \
    || git -C "$HYPERDATA_DIR" checkout --quiet --force "origin/$target" \
    || fail "could not check out $target"

  # A detached checkout at the pinned commit is the point: it makes an accidental
  # `git pull` in the sidecar directory impossible.
  git -C "$HYPERDATA_DIR" submodule update --init --recursive --quiet 2>/dev/null || true
}

install_venv() {
  if [ ! -d "$VENV" ]; then
    log "creating virtualenv at $VENV"
    "$HYPERDATA_PYTHON" -m venv "$VENV"
  fi

  log "installing the sidecar and its dependencies"
  "$VENV/bin/python" -m pip install --quiet --upgrade pip
  "$VENV/bin/python" -m pip install --quiet -e "$HYPERDATA_DIR"
}

record_revision() {
  local revision
  revision="$(git -C "$HYPERDATA_DIR" rev-parse HEAD)"
  printf '%s\n' "$revision" > "$REVISION_FILE"
  log "installed revision $revision"
}

cmd_install() {
  require_python
  require_git

  local target
  target="$(resolve_target "${1:-}")"
  [ -n "$target" ] || fail "no revision configured in upstream.env"

  checkout_revision "$target"
  install_venv
  record_revision

  log ""
  log "Sidecar installed at $HYPERDATA_DIR"
  log "Start it with:  $0 run"
  log "Then verify:    $0 contract"
  log ""
  log "Point the backend at it either globally:"
  log "  HYPERDATA_BASE_URL=http://127.0.0.1:$HYPERDATA_PORT"
  log "or per strategy in Strategy Studio (market data sources → HyperData Terminal)."
}

cmd_update() {
  local ref=""
  while [ $# -gt 0 ]; do
    case "$1" in
      --to) ref="${2:-}"; shift 2 ;;
      *) fail "unknown argument: $1" ;;
    esac
  done

  local before=""
  [ -f "$REVISION_FILE" ] && before="$(cat "$REVISION_FILE")"

  cmd_install "$ref"

  local after
  after="$(cat "$REVISION_FILE")"

  if [ "$before" = "$after" ]; then
    log ""
    log "Revision unchanged ($after); nothing to verify."
  fi

  log ""
  log "Now verify the adapter still matches upstream:"
  log "  $0 contract"
  log ""
  log "When it passes, record what was verified in deploy/hyperdata/upstream.env:"
  log "  HYPERDATA_REVISION=$after"
  log "  HYPERDATA_TESTED_MAJOR=<the major the sidecar reports>"
  log ""
  log "If the major changed, also update hyperDataTestedMajor in"
  log "marketdata/providers/hyperdata_client.go and the \"version\" inside"
  log "fixtureHealth in hyperdata_contract_test.go. A test asserts that all three"
  log "agree, so a partial bump fails the build instead of passing quietly."
}

cmd_status() {
  log "repository:  ${HYPERDATA_REPO:-<unset>}"
  log "checkout:    $HYPERDATA_DIR"
  log "pin:         ${HYPERDATA_REVISION:-<unset>} (verified against upstream major $HYPERDATA_TESTED_MAJOR)"

  if [ ! -d "$HYPERDATA_DIR/.git" ]; then
    log "installed:   no"
    return 0
  fi

  log "installed:   yes ($(git -C "$HYPERDATA_DIR" rev-parse --short HEAD))"
  log "subject:     $(git -C "$HYPERDATA_DIR" log -1 --pretty=%s)"

  if [ -f "$REVISION_FILE" ]; then
    log "recorded:    $(cat "$REVISION_FILE")"
  fi

  if [ -d "$VENV" ]; then
    log "virtualenv:  $VENV"
  else
    log "virtualenv:  missing (run install)"
  fi

  local health
  health="$(curl -fsS --max-time 3 "http://127.0.0.1:$HYPERDATA_PORT/v1/health" 2>/dev/null || true)"
  if [ -z "$health" ]; then
    log "api:         not answering on port $HYPERDATA_PORT"
    return 0
  fi
  log "api:         up on port $HYPERDATA_PORT"

  # Report the major the running process claims, because a mismatch is the
  # earliest possible signal that the sidecar in front of you is not the one the
  # adapter was verified against. It is a warning, not a failure: the adapter
  # tolerates additions and loose types, so a new major is usually still usable.
  local running=""
  if command -v "$HYPERDATA_PYTHON" >/dev/null 2>&1; then
    running="$(printf '%s' "$health" \
      | "$HYPERDATA_PYTHON" -c 'import json,sys; print(json.load(sys.stdin).get("version",""))' 2>/dev/null \
      || true)"
  fi

  if [ -z "$running" ]; then
    log "version:     (not reported)"
    return 0
  fi

  log "version:     $running"
  if [ "${running%%.*}" != "$HYPERDATA_TESTED_MAJOR" ]; then
    log ""
    log "WARNING: the running sidecar reports major ${running%%.*}, but this checkout was"
    log "         verified against major $HYPERDATA_TESTED_MAJOR. Run '$0 contract' before"
    log "         trusting new fields, and update HYPERDATA_REVISION once it passes."
  fi
}

cmd_contract() {
  log "== recorded contract (offline) =="
  if command -v go >/dev/null 2>&1; then
    # The whole package, not a name filter: the live drift report inside it skips
    # itself when HYPERDATA_LIVE_URL is unset, which is exactly the behaviour
    # wanted here.
    ( cd "$REPO_ROOT" && go test ./marketdata/providers/ -count=1 )
  else
    log "go not on PATH; skipping. The fixture tests are the offline half of the"
    log "contract, and go is required to build the backend that reads the sidecar."
  fi

  log ""
  log "== live sidecar contract =="

  if ! curl -fsS --max-time 5 "http://127.0.0.1:$HYPERDATA_PORT/v1/health" >/dev/null 2>&1; then
    log "the sidecar is not answering on port $HYPERDATA_PORT; skipping the live half."
    log "start it with '$0 run' and re-run this command."
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    log "go not on PATH; cannot run the live drift report."
    return 1
  fi

  # The live check is a Go test rather than a separate probe script on purpose:
  # it reuses the same HTTP client, the same recorded fixtures and the same shape
  # rules as the adapter, so there is exactly one contract. A hand-written key
  # list in a second script would drift away from the fixtures and start passing
  # for the wrong reason.
  (
    cd "$REPO_ROOT" \
      && HYPERDATA_LIVE_URL="http://127.0.0.1:$HYPERDATA_PORT" \
         HYPERDATA_LIVE_API_KEY="${HYPERDATA_API_KEY:-}" \
         go test ./marketdata/providers/ -run HyperDataUpstream -v -count=1
  )
}

cmd_run() {
  [ -x "$VENV/bin/python" ] || fail "sidecar is not installed; run '$0 install' first"
  log "starting the REST API on port $HYPERDATA_PORT"
  exec "$VENV/bin/python" "$HYPERDATA_DIR/run_api.py" --port "$HYPERDATA_PORT"
}

usage() {
  # Print the header comment block: everything after the shebang, up to the first
  # line that is not a comment. Deriving it instead of hardcoding a line range
  # means editing the header cannot silently truncate this output.
  awk 'NR > 1 { if ($0 !~ /^#/) exit; sub(/^# ?/, ""); print }' "${BASH_SOURCE[0]}"
}

case "${1:-}" in
  install)  shift; cmd_install "$@" ;;
  update)   shift; cmd_update "$@" ;;
  status)   cmd_status ;;
  contract) cmd_contract ;;
  run)      cmd_run ;;
  ""|-h|--help) usage ;;
  *) usage; exit 1 ;;
esac
