#!/bin/bash
set -e

echo "🧹 Cleaning old chain data..."
pkill -9 mirrorvaultd 2>/dev/null || true
rm -rf ~/.mirrorvault

CHAIN_DIR="/home/abdul-sami/work/The-Mirror-Vault/chain"
BINARY="$CHAIN_DIR/mirrorvaultd"

echo "🏗️  Initializing chain..."
cd $CHAIN_DIR
$BINARY init mirror-vault-node --chain-id mirror-vault-localnet

echo "👤 Creating test accounts..."
# Use Ethereum BIP44 derivation path (coin type 60) for EVM compatibility
echo "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about" | \
  $BINARY keys add alice --keyring-backend test --recover --coin-type 60 2>&1 | tee /tmp/alice_key.txt

echo "test test test test test test test test test test test junk" | \
  $BINARY keys add bob --keyring-backend test --recover --coin-type 60 2>&1 | tee /tmp/bob_key.txt

ALICE_ADDR=$($BINARY keys show alice -a --keyring-backend test)
BOB_ADDR=$($BINARY keys show bob -a --keyring-backend test)

echo "Alice: $ALICE_ADDR"
echo "Bob: $BOB_ADDR"

echo "💰 Adding genesis accounts..."
# Fund BOTH the Cosmos-derived and Ethereum-derived addresses
# This ensures EVM transactions (which use ETH derivation) can find their balance
$BINARY genesis add-genesis-account $ALICE_ADDR 10000000000000000000000umvlt --keyring-backend test
$BINARY genesis add-genesis-account $BOB_ADDR 10000000000000000000000umvlt --keyring-backend test

# Also fund the Ethereum-derived addresses (these are what EVM transactions will use)
# Alice ETH address: mirror1npvwllfr9dqr8erajqqr6s0vxnk2ak5553ldj8 (from 0x9858EfFD232B4033E47d90003D41EC34EcaEda94)
$BINARY genesis add-genesis-account mirror1npvwllfr9dqr8erajqqr6s0vxnk2ak5553ldj8 10000000000000000000000umvlt --keyring-backend test

echo "✍️  Creating genesis transaction..."
$BINARY genesis gentx alice 1000000000000000000umvlt --chain-id mirror-vault-localnet --keyring-backend test

echo "📦 Collecting genesis transactions..."
$BINARY genesis collect-gentxs

echo "⚙️  Configuring chain parameters..."

# Update staking denom to umvlt
jq '.app_state.staking.params.bond_denom = "umvlt"' ~/.mirrorvault/config/genesis.json > /tmp/genesis.json && mv /tmp/genesis.json ~/.mirrorvault/config/genesis.json

# Update EVM denoms to umvlt
jq '.app_state.evm.params.evm_denom = "umvlt"' ~/.mirrorvault/config/genesis.json > /tmp/genesis.json && mv /tmp/genesis.json ~/.mirrorvault/config/genesis.json
jq '.app_state.evm.params.extended_denom_options.extended_denom = "umvlt"' ~/.mirrorvault/config/genesis.json > /tmp/genesis.json && mv /tmp/genesis.json ~/.mirrorvault/config/genesis.json

# Add denom metadata for umvlt (CRITICAL - EVM module requires this!)
jq '.app_state.bank.denom_metadata += [{
  "description": "The native token of Mirror Vault Chain",
  "denom_units": [
    {
      "denom": "umvlt",
      "exponent": 0,
      "aliases": ["micromvlt"]
    },
    {
      "denom": "mvlt",
      "exponent": 6,
      "aliases": []
    },
    {
      "denom": "MVLT",
      "exponent": 18,
      "aliases": []
    }
  ],
  "base": "umvlt",
  "display": "MVLT",
  "name": "Mirror Vault Token",
  "symbol": "MVLT"
}]' ~/.mirrorvault/config/genesis.json > /tmp/genesis.json && mv /tmp/genesis.json ~/.mirrorvault/config/genesis.json

# Configure minimum gas prices
sed -i 's/minimum-gas-prices = ""/minimum-gas-prices = "0umvlt"/' ~/.mirrorvault/config/app.toml

# Disable base fee for testnet (simplifies gas handling)
jq '.app_state.feemarket.params.no_base_fee = true' ~/.mirrorvault/config/genesis.json > /tmp/genesis.json && mv /tmp/genesis.json ~/.mirrorvault/config/genesis.json
jq '.app_state.feemarket.params.min_gas_price = "0.000000000000000000"' ~/.mirrorvault/config/genesis.json > /tmp/genesis.json && mv /tmp/genesis.json ~/.mirrorvault/config/genesis.json

echo "🚀 Starting blockchain..."
echo "   - EVM JSON-RPC: http://localhost:8545"
echo "   - Cosmos REST: http://localhost:1317"
echo "   - Cosmos gRPC: localhost:9090"
echo "   - CometBFT RPC: http://localhost:26657"
echo ""
echo "📝 Logs: tail -f ~/.mirrorvault/chain.log"
echo ""

cd ~/.mirrorvault
nohup $BINARY start --log_level info > ~/.mirrorvault/chain.log 2>&1 &
CHAIN_PID=$!

echo "Chain started with PID: $CHAIN_PID"
echo "Waiting 10 seconds for initialization..."
sleep 10

if ps -p $CHAIN_PID > /dev/null; then
    echo "✅ Chain is running!"
    echo ""
    echo "Test with:"
    echo "  curl -X POST http://localhost:8545 -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"method\":\"eth_chainId\",\"params\":[],\"id\":1}'"
else
    echo "❌ Chain failed to start. Check logs:"
    echo "   tail -100 ~/.mirrorvault/chain.log"
    exit 1
fi
