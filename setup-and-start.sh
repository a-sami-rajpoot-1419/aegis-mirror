#!/bin/bash
# Mirror Vault - Complete Setup and Start Script
# This script will initialize the chain, create accounts, and start everything

set -e

echo "🚀 Mirror Vault - Complete Setup"
echo "=================================="
echo ""

# 1. Clean environment
echo "Step 1: Cleaning environment..."
pkill -9 mirrorvaultd 2>/dev/null || true
pkill -9 ignite 2>/dev/null || true
rm -rf ~/.mirrorvault
sleep 2

# 2. Build and install
echo "Step 2: Building chain binary..."
cd /home/abdul-sami/work/The-Mirror-Vault/chain
go build -o mirrorvaultd ./cmd/mirrorvaultd

CHAIN_DIR="/home/abdul-sami/work/The-Mirror-Vault/chain"
BINARY="$CHAIN_DIR/mirrorvaultd"

echo "✅ Binary built at $BINARY"

# 3. Initialize chain
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

# 5. Add genesis accounts (10000 MVLT each = 10,000,000,000 umvlt)
echo "Step 5: Adding genesis accounts..."
$BINARY genesis add-genesis-account alice 10000000000umvlt --keyring-backend test
$BINARY genesis add-genesis-account bob 10000000000umvlt --keyring-backend test

# Fund the hardcoded EVM test wallet used by contracts/test-backend.js
# ALICE_KEY => EVM address 0x9858EfFD232B4033E47d90003D41EC34EcaEda94
# bech32 (mirror prefix) => mirror1npvwllfr9dqr8erajqqr6s0vxnk2ak5553ldj8
$BINARY genesis add-genesis-account mirror1npvwllfr9dqr8erajqqr6s0vxnk2ak5553ldj8 10000000000000000000000umvlt --keyring-backend test
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
echo "  Each account: 10,000 MVLT (10,000,000,000 umvlt)"
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

# Start the chain
exec $BINARY start \
  --api.enable \
  --log_level info
