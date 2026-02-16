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
cp mirrorvaultd ~/go/bin/
echo "✅ Binary installed"

# 3. Initialize chain
echo "Step 3: Initializing chain..."
mirrorvaultd init mirror-vault --chain-id mirror-vault-localnet --default-denom umvlt --overwrite
echo "✅ Chain initialized"

# 4. Create test accounts (without recovery - generate new)
echo "Step 4: Creating test accounts..."
yes | mirrorvaultd keys add alice --keyring-backend test > /tmp/alice_key.txt 2>&1
yes | mirrorvaultd keys add bob --keyring-backend test > /tmp/bob_key.txt 2>&1
echo "✅ Accounts created"
echo ""
echo "📝 Alice Cosmos address: $(mirrorvaultd keys show alice -a --keyring-backend test)"
echo "📝 Bob Cosmos address: $(mirrorvaultd keys show bob -a --keyring-backend test)"
echo ""

# 5. Add genesis accounts (10000 MVLT each = 10,000,000,000 umvlt)
echo "Step 5: Adding genesis accounts..."
mirrorvaultd genesis add-genesis-account alice 10000000000umvlt --keyring-backend test
mirrorvaultd genesis add-genesis-account bob 10000000000umvlt --keyring-backend test
echo "✅ Genesis accounts added"

# 6. Create validator
echo "Step 6: Creating validator..."
mirrorvaultd genesis gentx alice 1000000umvlt --chain-id mirror-vault-localnet --keyring-backend test
mirrorvaultd genesis collect-gentxs
echo "✅ Validator created"

# 7. Configure EVM & JSON-RPC
echo "Step 7: Configuring EVM..."
# Enable JSON-RPC in app.toml
sed -i.bak 's/enable = false/enable = true/g' ~/.mirrorvault/config/app.toml
sed -i.bak 's/"eth,net,web3"/"eth,net,web3,debug"/g' ~/.mirrorvault/config/app.toml
echo "✅ EVM configured"

# 8. Update genesis for EVM params
echo "Step 8: Updating genesis for EVM compatibility..."
jq '.app_state.evm.params.evm_denom = "umvlt"' ~/.mirrorvault/config/genesis.json > ~/.mirrorvault/config/genesis.json.tmp && mv ~/.mirrorvault/config/genesis.json.tmp ~/.mirrorvault/config/genesis.json
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
echo "  Alice: $(mirrorvaultd keys show alice -a --keyring-backend test)"
echo "  Bob: $(mirrorvaultd keys show bob -a --keyring-backend test)"
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
exec mirrorvaultd start \
  --json-rpc.enable \
  --json-rpc.api eth,web3,net,debug \
  --json-rpc.address 0.0.0.0:8545 \
  --api.enable \
  --log_level info
