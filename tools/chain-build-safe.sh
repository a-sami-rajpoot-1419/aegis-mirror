#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="/home/abdul-sami/work/The-Mirror-Vault"
CHAIN_DIR="$REPO_DIR/chain"
LOG_FILE="${1:-/tmp/mirrorvault-chain-build.log}"

cd "$REPO_DIR"
# shellcheck disable=SC1091
source tools/env.sh

mkdir -p "$CHAIN_DIR/build"

{
  echo "[chain-build-safe] started: $(date -Is)"
  echo "[chain-build-safe] go: $(go version)"
  echo "[chain-build-safe] cwd: $CHAIN_DIR"

  cd "$CHAIN_DIR"

  # Reduce peak memory/CPU usage (prevents WSL OOM/disconnect).
  export GOMAXPROCS="${GOMAXPROCS:-2}"
  export GOFLAGS="${GOFLAGS:--p=1}"
  export GOMEMLIMIT="${GOMEMLIMIT:-2GiB}"
  export GOGC="${GOGC:-50}"

  echo "[chain-build-safe] env: GOMAXPROCS=$GOMAXPROCS GOFLAGS=$GOFLAGS GOMEMLIMIT=$GOMEMLIMIT GOGC=$GOGC"

  echo "[chain-build-safe] go mod download"
  go mod download

  echo "[chain-build-safe] build mirrorvaultd"
  go build -trimpath -o ./build/mirrorvaultd ./cmd/mirrorvaultd

  echo "[chain-build-safe] done: $(date -Is)"
  echo "[chain-build-safe] binary: $CHAIN_DIR/build/mirrorvaultd"
} >"$LOG_FILE" 2>&1
