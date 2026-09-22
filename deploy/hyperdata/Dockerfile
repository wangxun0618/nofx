# HyperData Terminal sidecar.
#
# Built from upstream at a pinned revision rather than vendored, so this image
# can be rebuilt or moved forward without touching the nofx source tree. The pin
# is verified after checkout: a build that silently used a different commit would
# make the compatibility contract meaningless.
#
# Build (normally driven by docker-compose.hyperdata.yml):
#   docker build \
#     --build-arg HYPERDATA_REPO=https://github.com/Co-Messi/HyperData-Terminal.git \
#     --build-arg HYPERDATA_REF=main \
#     --build-arg HYPERDATA_REVISION=<sha> \
#     -t nofx-hyperdata deploy/hyperdata

FROM python:3.12-slim

ARG HYPERDATA_REPO="https://github.com/Co-Messi/HyperData-Terminal.git"
ARG HYPERDATA_REF="main"
ARG HYPERDATA_REVISION=""

ENV PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

# A container cannot use upstream's loopback default, and the port mapping or the
# compose network would both be unreachable without this. Upstream refuses a
# non-loopback bind unless HYPERDATA_API_KEY is set, so the image is only
# startable with that variable supplied — which is deliberate, and the reason
# docker-compose.hyperdata.yml uses the `${VAR:?}` form to fail loudly.
ENV HYPERDATA_API_HOST=0.0.0.0

# git is required to fetch the pinned revision; curl is used by the healthcheck.
RUN apt-get update \
 && apt-get install -y --no-install-recommends git curl \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Fetch the exact revision. `git fetch <sha>` works on GitHub; the branch fetch
# is a fallback for hosts that refuse arbitrary-SHA fetches.
#
# The fallback is why the check below exists: without it a failed SHA fetch would
# leave the build silently running a branch head, and every compatibility claim
# made about this image would be about different code than the pin names.
RUN set -eux; \
    git init --quiet .; \
    git remote add origin "$HYPERDATA_REPO"; \
    git fetch --quiet --depth 1 origin "$HYPERDATA_REVISION" \
      || git fetch --quiet --depth 1 origin "$HYPERDATA_REF"; \
    git checkout --quiet FETCH_HEAD; \
    git rev-parse HEAD > /app/.upstream-revision; \
    if [ -n "$HYPERDATA_REVISION" ] && [ "$(cat /app/.upstream-revision)" != "$HYPERDATA_REVISION" ]; then \
      echo "refusing to build: requested $HYPERDATA_REVISION but checked out $(cat /app/.upstream-revision)" >&2; \
      exit 1; \
    fi; \
    cat /app/.upstream-revision

# Upstream's pyproject.toml declares no [build-system] table, so pip falls back
# to the legacy setuptools backend and needs setuptools and wheel present.
# Relying on the base image to happen to ship them is not something a build
# should depend on — Python 3.12 stopped bundling setuptools by default — so they
# are installed explicitly instead.
RUN pip install --upgrade pip setuptools wheel \
 && pip install -e .

# Record which API contract this image was verified against, next to the revision
# so an operator can compare it with hyperDataTestedMajor without opening the
# repository.
RUN printf 'verified-against-major=1\n' > /app/.upstream-contract

EXPOSE 8420

# The REST API, not the TUI: a container has no useful terminal, and the backend
# consumes HTTP.
#
# Note that upstream refuses a non-loopback bind unless HYPERDATA_API_KEY is set,
# so compose supplies one. It serves position and order-flow intelligence, which
# upstream explicitly warns against exposing — do not publish this port.
CMD ["sh", "-c", "exec python run_api.py --port ${HYPERDATA_PORT:-8420}"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=45s --retries=3 \
  CMD curl -fsS "http://127.0.0.1:${HYPERDATA_PORT:-8420}/v1/live" >/dev/null || exit 1
