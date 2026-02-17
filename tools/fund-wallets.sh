#!/bin/bash
set -euo pipefail

ROOT_DIR="/home/abdul-sami/work/The-Mirror-Vault"
FUND_FILE="${FUND_FILE:-$ROOT_DIR/fund-accounts.txt}"

# Funds EVM balances (what MetaMask/Keplr EVM mode shows) by sending native MVLT
# from the Hardhat deployer account to each recipient.
#
# Supports recipients in either format:
# - mirror1... (bech32)
# - 0x... (EVM hex address)

AMOUNT_MVLT="${AMOUNT_MVLT:-1000}"
RECIPIENTS="${RECIPIENTS:-}"
RECIPIENTS_FILE="${RECIPIENTS_FILE:-$FUND_FILE}"

usage() {
  cat <<EOF
Usage:
  bash tools/fund-wallets.sh [mirror1addr|0xaddr ...]

Defaults:
  - Reads addresses from $RECIPIENTS_FILE if no args are given
  - Sends $AMOUNT_MVLT MVLT to each address

Env overrides:
  AMOUNT_MVLT, RECIPIENTS, RECIPIENTS_FILE

Notes:
  - This funds EVM balances via JSON-RPC (no Cosmos bank CLI required)
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

if [ "$#" -gt 0 ]; then
  RECIPIENTS="$(printf "%s," "$@" | sed 's/,$//')"
fi

if [ ! -d "$ROOT_DIR/contracts" ]; then
  echo "❌ contracts/ directory not found at $ROOT_DIR/contracts"
  exit 1
fi

cd "$ROOT_DIR/contracts"
if [ ! -d node_modules ]; then
  npm install
fi

export AMOUNT_MVLT
export RECIPIENTS
export RECIPIENTS_FILE

npm run fund:local
exit 0
