#!/bin/bash
# Mirror Vault - Complete Setup and Start Script
# This script will initialize the chain, create accounts, and start everything

set -e

echo "🚀 Mirror Vault - Complete Setup"
echo "=================================="
echo ""

ROOT_DIR="/home/abdul-sami/work/The-Mirror-Vault"
FUND_FILE="$ROOT_DIR/fund-accounts.txt"

# Funding behavior:
# - On fresh init / FORCE_RESET: any addresses in fund-accounts.txt (and EXTRA_GENESIS_ACCOUNTS)
#   are added to genesis with the same large balance as the test accounts.
# - On normal restart (persistent state): balances are preserved and we do not auto-top-up.
#   If you want a top-up anyway, set TOP_UP_ON_START=1.
TOP_UP_ON_START="${TOP_UP_ON_START:-}"

STATE_HOME="$HOME/.mirrorvault"
GENESIS_FILE="$STATE_HOME/config/genesis.json"
APP_FILE="$STATE_HOME/config/app.toml"
STATE_EXISTS=0
if [ -f "$GENESIS_FILE" ] && [ -f "$APP_FILE" ]; then
  STATE_EXISTS=1
fi

# 1. Clean environment
echo "Step 1: Cleaning environment..."
pkill -9 mirrorvaultd 2>/dev/null || true
pkill -9 ignite 2>/dev/null || true

# Persistence behavior:
# - Default: if prior state exists, reuse it (balances/contracts persist)
# - To force a clean reset: FORCE_RESET=1 bash ./setup-and-start.sh
if [ -n "${FORCE_RESET:-}" ]; then
  echo "⚠️  FORCE_RESET is set; wiping $STATE_HOME"
  rm -rf "$STATE_HOME"
  STATE_EXISTS=0
  sleep 2
else
  if [ "$STATE_EXISTS" -eq 1 ]; then
    echo "✅ Reusing existing chain state at $STATE_HOME (persistent balances/contracts)"
    sleep 1
  else
    echo "ℹ️  No existing state found; initializing a fresh localnet"
    sleep 1
  fi
fi

# 2. Build and install
echo "Step 2: Building chain binary..."
cd /home/abdul-sami/work/The-Mirror-Vault/chain
go build -o mirrorvaultd ./cmd/mirrorvaultd

CHAIN_DIR="/home/abdul-sami/work/The-Mirror-Vault/chain"
BINARY="$CHAIN_DIR/mirrorvaultd"

echo "✅ Binary built at $BINARY"

if [ "$STATE_EXISTS" -eq 1 ]; then
  echo "Step 3: Starting existing chain (no re-init)..."

  # Start the chain in background so we can deploy wrapper contracts, then wait.
  $BINARY start \
    --api.enable \
    --log_level info \
    > /tmp/mirrorvaultd.log 2>&1 &

  NODE_PID=$!
  echo "✅ mirrorvaultd started (pid $NODE_PID), logs: /tmp/mirrorvaultd.log"

  cleanup() {
    echo "\n🛑 Stopping mirrorvaultd (pid $NODE_PID)..."
    kill "$NODE_PID" 2>/dev/null || true
  }
  trap cleanup INT TERM HUP EXIT

  echo "⏳ Waiting for JSON-RPC http://localhost:8545 ..."
  for i in $(seq 1 40); do
    if ! kill -0 "$NODE_PID" 2>/dev/null; then
      echo "❌ mirrorvaultd exited while starting. Showing last log lines:"
      tail -n 80 /tmp/mirrorvaultd.log | cat || true
      if grep -q "state.AppHash does not match" /tmp/mirrorvaultd.log 2>/dev/null; then
        echo ""
        echo "✅ Detected AppHash mismatch (CometBFT/app DB out of sync)."
        echo "   One-time fix: FORCE_RESET=1 bash ./setup-and-start.sh"
        echo "   After that, normal restarts (pkill + ./setup-and-start.sh) will preserve balances."
      fi
      exit 1
    fi
    if curl -sS --max-time 1 -H 'content-type: application/json' \
      --data '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' \
      http://localhost:8545 >/dev/null 2>&1; then
      echo "✅ JSON-RPC is up"
      break
    fi
    sleep 0.5
  done

  echo "⏳ Waiting for first block..."
  for i in $(seq 1 60); do
    if ! kill -0 "$NODE_PID" 2>/dev/null; then
      echo "❌ mirrorvaultd exited while waiting for first block. Showing last log lines:"
      tail -n 80 /tmp/mirrorvaultd.log | cat || true
      if grep -q "state.AppHash does not match" /tmp/mirrorvaultd.log 2>/dev/null; then
        echo ""
        echo "✅ Detected AppHash mismatch (CometBFT/app DB out of sync)."
        echo "   One-time fix: FORCE_RESET=1 bash ./setup-and-start.sh"
        echo "   After that, normal restarts (pkill + ./setup-and-start.sh) will preserve balances."
      fi
      exit 1
    fi
    BN=$(curl -sS --max-time 1 -H 'content-type: application/json' \
      --data '{"jsonrpc":"2.0","id":2,"method":"eth_blockNumber","params":[]}' \
      http://localhost:8545 2>/dev/null | sed -n 's/.*"result":"\(0x[0-9a-fA-F]*\)".*/\1/p')
    if [ -n "$BN" ] && [ "$BN" != "0x0" ]; then
      echo "✅ Block production started (blockNumber=$BN)"
      break
    fi
    sleep 0.5
  done

  # Deploy wrappers only if missing (no code at recorded addresses)
  NEED_DEPLOY=1
  if [ -f /home/abdul-sami/work/The-Mirror-Vault/contracts/deployed-addresses.json ]; then
    VAULT=$(jq -r '.vaultGate // empty' /home/abdul-sami/work/The-Mirror-Vault/contracts/deployed-addresses.json 2>/dev/null)
    NFT=$(jq -r '.mirrorNFT // empty' /home/abdul-sami/work/The-Mirror-Vault/contracts/deployed-addresses.json 2>/dev/null)
    if [ -n "$VAULT" ] && [ -n "$NFT" ]; then
      VC=$(curl -sS --max-time 2 -H 'content-type: application/json' \
        --data "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"eth_getCode\",\"params\":[\"$VAULT\",\"latest\"]}" \
        http://localhost:8545 2>/dev/null | sed -n 's/.*"result":"\(0x[0-9a-fA-F]*\)".*/\1/p')
      NC=$(curl -sS --max-time 2 -H 'content-type: application/json' \
        --data "{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"eth_getCode\",\"params\":[\"$NFT\",\"latest\"]}" \
        http://localhost:8545 2>/dev/null | sed -n 's/.*"result":"\(0x[0-9a-fA-F]*\)".*/\1/p')
      if [ -n "$VC" ] && [ "$VC" != "0x" ] && [ -n "$NC" ] && [ "$NC" != "0x" ]; then
        NEED_DEPLOY=0
        echo "✅ Wrapper contracts already deployed (code present)"
      fi
    fi
  fi

  if [ "$NEED_DEPLOY" -eq 1 ]; then
    echo "📦 Deploying wrapper contracts (VaultGate + MirrorNFT)..."
    cd /home/abdul-sami/work/The-Mirror-Vault/contracts
    if [ ! -d node_modules ]; then
      npm install --silent
    fi

    set +e
    DEPLOY_OK=0
    for i in $(seq 1 10); do
      npm run deploy:local
      if [ $? -eq 0 ]; then
        DEPLOY_OK=1
        break
      fi
      echo "⚠️  Deploy failed (attempt $i/10). Retrying in 1s..."
      sleep 1
    done
    set -e

    if [ "$DEPLOY_OK" -eq 1 ]; then
      echo "✅ Deployed addresses written to contracts/deployed-addresses.json"
    else
      echo "❌ Contract deploy failed after retries. Chain is still running."
      echo "   You can retry manually: (cd contracts && npm run deploy:local)"
    fi
  fi

  if [ -n "$TOP_UP_ON_START" ]; then
    echo "💸 TOP_UP_ON_START is set; funding addresses from $FUND_FILE (and EXTRA_GENESIS_ACCOUNTS) via bank send..."
    POST_FUND_AMT="${POST_FUND_AMT:-1000000000000000000000umvlt}"

    declare -A _seen_fund_addrs
    FUND_ADDRS=()

    if [ -f "$FUND_FILE" ]; then
      while IFS= read -r line; do
        addr="$(echo "$line" | xargs)"
        [ -z "$addr" ] && continue
        echo "$addr" | grep -qE '^#' && continue
        if [ -z "${_seen_fund_addrs[$addr]:-}" ]; then
          _seen_fund_addrs[$addr]=1
          FUND_ADDRS+=("$addr")
        fi
      done < "$FUND_FILE"
    fi

    if [ -n "${EXTRA_GENESIS_ACCOUNTS:-}" ]; then
      IFS=',' read -r -a EXTRA_ADDRS <<< "$EXTRA_GENESIS_ACCOUNTS"
      for addr0 in "${EXTRA_ADDRS[@]}"; do
        addr="$(echo "$addr0" | xargs)"
        [ -z "$addr" ] && continue
        if [ -z "${_seen_fund_addrs[$addr]:-}" ]; then
          _seen_fund_addrs[$addr]=1
          FUND_ADDRS+=("$addr")
        fi
      done
    fi

    if [ "${#FUND_ADDRS[@]}" -eq 0 ]; then
      echo "ℹ️  No fund addresses found. Add bech32 addresses to $FUND_FILE or set EXTRA_GENESIS_ACCOUNTS."
    else
      set +e
      for addr in "${FUND_ADDRS[@]}"; do
        echo "  - funding $addr with $POST_FUND_AMT"
        $BINARY tx bank send alice "$addr" "$POST_FUND_AMT" \
          --chain-id mirror-vault-localnet \
          --keyring-backend test \
          --node http://localhost:26657 \
          --gas auto \
          --gas-adjustment 1.3 \
          --gas-prices 0umvlt \
          -y -b block >/dev/null 2>&1
        if [ $? -ne 0 ]; then
          echo "    ⚠️  failed to fund $addr (check address format; must be mirror1...)"
        fi
      done
      set -e
      echo "✅ Top-up finished"
    fi
  fi

  echo "🚀 Chain is running. Press Ctrl+C to stop."
  wait "$NODE_PID"
  exit 0
fi

# 3. Initialize chain (fresh localnet only)
echo "Step 3: Initializing chain..."
$BINARY init mirror-vault --chain-id mirror-vault-localnet --default-denom umvlt --overwrite
echo "✅ Chain initialized"

# 4. Create test accounts (without recovery - generate new)
echo "Step 4: Creating test accounts..."
# Use Ethereum BIP44 derivation path (coin type 60) for EVM compatibility
yes | $BINARY keys add alice --keyring-backend test --coin-type 60 > /tmp/alice_key.txt 2>&1
yes | $BINARY keys add bob --keyring-backend test --coin-type 60 > /tmp/bob_key.txt 2>&1
echo "✅ Accounts created"
echo ""
echo "📝 Alice Cosmos address: $($BINARY keys show alice -a --keyring-backend test)"
echo "📝 Bob Cosmos address: $($BINARY keys show bob -a --keyring-backend test)"
echo ""

# 5. Add genesis accounts
echo "Step 5: Adding genesis accounts..."

# NOTE: EVM uses 18-decimal base units; fund with 10,000 * 1e18 = 1e22 umvlt.
FUND_AMT="10000000000000000000000umvlt"

$BINARY genesis add-genesis-account alice "$FUND_AMT" --keyring-backend test
$BINARY genesis add-genesis-account bob "$FUND_AMT" --keyring-backend test

# Fund the hardcoded EVM test wallet used by contracts/test-backend.js
# ALICE_KEY => EVM address 0x9858EfFD232B4033E47d90003D41EC34EcaEda94
# bech32 (mirror prefix) => mirror1npvwllfr9dqr8erajqqr6s0vxnk2ak5553ldj8
$BINARY genesis add-genesis-account mirror1npvwllfr9dqr8erajqqr6s0vxnk2ak5553ldj8 "$FUND_AMT" --keyring-backend test

# Fund the Hardhat/MetaMask default dev account used in contracts/hardhat.config.ts
# Private key lives in contracts/hardhat.config.ts; its EVM address is 0xf39f...2266
# mirror bech32 derived from raw 20-byte address: mirror17w0adeg64ky0daxwd2ugyuneellmjgnx7uk5xa
$BINARY genesis add-genesis-account mirror17w0adeg64ky0daxwd2ugyuneellmjgnx7uk5xa "$FUND_AMT" --keyring-backend test

# Optionally fund additional accounts (e.g. addresses you connect from MetaMask/Keplr)
# Usage:
#   EXTRA_GENESIS_ACCOUNTS="mirror1...,mirror1..." bash ./setup-and-start.sh
declare -A _seen_genesis_addrs
GENESIS_FUND_ADDRS=()

if [ -f "$FUND_FILE" ]; then
  echo "➕ Funding addresses from $FUND_FILE (genesis)..."
  while IFS= read -r line; do
    addr="$(echo "$line" | xargs)"
    [ -z "$addr" ] && continue
    echo "$addr" | grep -qE '^#' && continue
    if [ -z "${_seen_genesis_addrs[$addr]:-}" ]; then
      _seen_genesis_addrs[$addr]=1
      GENESIS_FUND_ADDRS+=("$addr")
    fi
  done < "$FUND_FILE"
fi

if [ -n "${EXTRA_GENESIS_ACCOUNTS:-}" ]; then
  echo "➕ Funding EXTRA_GENESIS_ACCOUNTS (genesis)..."
  IFS=',' read -r -a EXTRA_ADDRS <<< "$EXTRA_GENESIS_ACCOUNTS"
  for addr0 in "${EXTRA_ADDRS[@]}"; do
    addr="$(echo "$addr0" | xargs)"
    [ -z "$addr" ] && continue
    if [ -z "${_seen_genesis_addrs[$addr]:-}" ]; then
      _seen_genesis_addrs[$addr]=1
      GENESIS_FUND_ADDRS+=("$addr")
    fi
  done
fi

if [ "${#GENESIS_FUND_ADDRS[@]}" -gt 0 ]; then
  set +e
  for addr in "${GENESIS_FUND_ADDRS[@]}"; do
    echo "  - $addr"
    $BINARY genesis add-genesis-account "$addr" "$FUND_AMT" --keyring-backend test >/dev/null 2>&1 || \
      echo "  ⚠️  Could not fund $addr (skipping; must be mirror1...)"
  done
  set -e
fi
echo "✅ Genesis accounts added"

# 6. Create validator
echo "Step 6: Creating validator..."
$BINARY genesis gentx alice 1000000umvlt --chain-id mirror-vault-localnet --keyring-backend test
$BINARY genesis collect-gentxs
echo "✅ Validator created"

# 7. Configure EVM & JSON-RPC
echo "Step 7: Configuring EVM..."
# Enable JSON-RPC in app.toml
sed -i.bak 's/enable = false/enable = true/g' ~/.mirrorvault/config/app.toml
sed -i.bak 's/"eth,net,web3"/"eth,net,web3,debug"/g' ~/.mirrorvault/config/app.toml
# Ensure WS is reachable from Docker (bind to 0.0.0.0:8546)
sed -i.bak 's/ws-address = "127\.0\.0\.1:8546"/ws-address = "0.0.0.0:8546"/g' ~/.mirrorvault/config/app.toml
# Permit common dev origins (Blockscout, browsers)
sed -i.bak 's/ws-origins = \["127\.0\.0\.1", "localhost"\]/ws-origins = ["*", "127.0.0.1", "localhost", "host.docker.internal"]/g' ~/.mirrorvault/config/app.toml
# Minimum gas prices must be configured (required by Cosmos SDK server start)
sed -i.bak 's/minimum-gas-prices = ""/minimum-gas-prices = "0umvlt"/g' ~/.mirrorvault/config/app.toml
echo "✅ EVM configured"

# 8. Update genesis for EVM params
echo "Step 8: Updating genesis for EVM compatibility..."
jq '.app_state.evm.params.evm_denom = "umvlt"' ~/.mirrorvault/config/genesis.json > ~/.mirrorvault/config/genesis.json.tmp && mv ~/.mirrorvault/config/genesis.json.tmp ~/.mirrorvault/config/genesis.json
jq '.app_state.evm.params.extended_denom_options.extended_denom = "umvlt"' ~/.mirrorvault/config/genesis.json > ~/.mirrorvault/config/genesis.json.tmp && mv ~/.mirrorvault/config/genesis.json.tmp ~/.mirrorvault/config/genesis.json
jq '.app_state.staking.params.bond_denom = "umvlt"' ~/.mirrorvault/config/genesis.json > ~/.mirrorvault/config/genesis.json.tmp && mv ~/.mirrorvault/config/genesis.json.tmp ~/.mirrorvault/config/genesis.json

# Add denom metadata for umvlt (required by EVM module at genesis)
jq '.app_state.bank.denom_metadata += [{
  "description": "The native token of Mirror Vault Chain",
  "denom_units": [
    {"denom": "umvlt", "exponent": 0, "aliases": ["micromvlt"]},
    {"denom": "mvlt", "exponent": 6, "aliases": []},
    {"denom": "MVLT", "exponent": 18, "aliases": []}
  ],
  "base": "umvlt",
  "display": "MVLT",
  "name": "Mirror Vault Token",
  "symbol": "MVLT"
}]' ~/.mirrorvault/config/genesis.json > ~/.mirrorvault/config/genesis.json.tmp && mv ~/.mirrorvault/config/genesis.json.tmp ~/.mirrorvault/config/genesis.json

jq '.app_state.feemarket.params.no_base_fee = true' ~/.mirrorvault/config/genesis.json > ~/.mirrorvault/config/genesis.json.tmp && mv ~/.mirrorvault/config/genesis.json.tmp ~/.mirrorvault/config/genesis.json
jq '.app_state.feemarket.params.min_gas_price = "0.000000000000000000"' ~/.mirrorvault/config/genesis.json > ~/.mirrorvault/config/genesis.json.tmp && mv ~/.mirrorvault/config/genesis.json.tmp ~/.mirrorvault/config/genesis.json
# Force base fee to 0 for local dev so wallets that propose 0 fees don't fail.
jq '.app_state.feemarket.params.base_fee = "0.000000000000000000"' ~/.mirrorvault/config/genesis.json > ~/.mirrorvault/config/genesis.json.tmp && mv ~/.mirrorvault/config/genesis.json.tmp ~/.mirrorvault/config/genesis.json
jq '.consensus.params.block.max_gas = "30000000"' ~/.mirrorvault/config/genesis.json > ~/.mirrorvault/config/genesis.json.tmp && mv ~/.mirrorvault/config/genesis.json.tmp ~/.mirrorvault/config/genesis.json
echo "✅ Genesis updated"

echo ""
echo "=================================="
echo "✅ Setup complete!"
echo "=================================="
echo ""
echo "🌐 Chain Configuration:"
echo "  Chain ID: mirror-vault-localnet"
echo "  Denom: umvlt (base), MVLT (display)"
echo "  EVM Chain ID: 7777"
echo ""
echo "👤 Test Accounts:"
echo "  Alice: $($BINARY keys show alice -a --keyring-backend test)"
echo "  Bob: $($BINARY keys show bob -a --keyring-backend test)"
echo ""
echo "💰 Account Balances:"
echo "  Each account: 10,000 MVLT (10,000,000,000,000,000,000,000 umvlt)"
echo ""
echo "🚀 Starting chain..."
echo "   Endpoints will be at:"
echo "   - EVM JSON-RPC: http://localhost:8545"
echo "   - Cosmos REST: http://localhost:1317"
echo "   - Cosmos gRPC: http://localhost:9090"
echo "   - CometBFT RPC: http://localhost:26657"
echo ""
echo "Starting in 3 seconds..."
sleep 3

# Start the chain in background so we can deploy wrapper contracts, then wait.
$BINARY start \
  --api.enable \
  --log_level info \
  > /tmp/mirrorvaultd.log 2>&1 &

NODE_PID=$!
echo "✅ mirrorvaultd started (pid $NODE_PID), logs: /tmp/mirrorvaultd.log"

cleanup() {
  echo "\n🛑 Stopping mirrorvaultd (pid $NODE_PID)..."
  kill "$NODE_PID" 2>/dev/null || true
}
# Stop the node when the script is interrupted/exits.
trap cleanup INT TERM HUP EXIT

echo "⏳ Waiting for JSON-RPC http://localhost:8545 ..."
for i in $(seq 1 40); do
  if ! kill -0 "$NODE_PID" 2>/dev/null; then
    echo "❌ mirrorvaultd exited while starting. Showing last log lines:"
    tail -n 80 /tmp/mirrorvaultd.log | cat || true
    if grep -q "state.AppHash does not match" /tmp/mirrorvaultd.log 2>/dev/null; then
      echo ""
      echo "✅ Detected AppHash mismatch (CometBFT/app DB out of sync)."
      echo "   One-time fix: FORCE_RESET=1 bash ./setup-and-start.sh"
      echo "   After that, normal restarts (pkill + ./setup-and-start.sh) will preserve balances."
    fi
    exit 1
  fi
  if curl -sS --max-time 1 -H 'content-type: application/json' \
    --data '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' \
    http://localhost:8545 >/dev/null 2>&1; then
    echo "✅ JSON-RPC is up"
    break
  fi
  sleep 0.5
done

echo "⏳ Waiting for first block..."
for i in $(seq 1 60); do
  if ! kill -0 "$NODE_PID" 2>/dev/null; then
    echo "❌ mirrorvaultd exited while waiting for first block. Showing last log lines:"
    tail -n 80 /tmp/mirrorvaultd.log | cat || true
    if grep -q "state.AppHash does not match" /tmp/mirrorvaultd.log 2>/dev/null; then
      echo ""
      echo "✅ Detected AppHash mismatch (CometBFT/app DB out of sync)."
      echo "   One-time fix: FORCE_RESET=1 bash ./setup-and-start.sh"
      echo "   After that, normal restarts (pkill + ./setup-and-start.sh) will preserve balances."
    fi
    exit 1
  fi
  BN=$(curl -sS --max-time 1 -H 'content-type: application/json' \
    --data '{"jsonrpc":"2.0","id":2,"method":"eth_blockNumber","params":[]}' \
    http://localhost:8545 2>/dev/null | sed -n 's/.*"result":"\(0x[0-9a-fA-F]*\)".*/\1/p')
  if [ -n "$BN" ] && [ "$BN" != "0x0" ]; then
    echo "✅ Block production started (blockNumber=$BN)"
    break
  fi
  sleep 0.5
done

echo "📦 Deploying wrapper contracts (VaultGate + MirrorNFT)..."
cd /home/abdul-sami/work/The-Mirror-Vault/contracts

if [ ! -d node_modules ]; then
  npm install --silent
fi

set +e
DEPLOY_OK=0
for i in $(seq 1 10); do
  npm run deploy:local
  if [ $? -eq 0 ]; then
    DEPLOY_OK=1
    break
  fi
  echo "⚠️  Deploy failed (attempt $i/10). Retrying in 1s..."
  sleep 1
done
set -e

if [ "$DEPLOY_OK" -eq 1 ]; then
  echo "✅ Deployed addresses written to contracts/deployed-addresses.json"
else
  echo "❌ Contract deploy failed after retries. Chain is still running."
  echo "   You can retry manually: (cd contracts && npm run deploy:local)"
fi

echo "🚀 Chain is running. Press Ctrl+C to stop."
wait "$NODE_PID"
