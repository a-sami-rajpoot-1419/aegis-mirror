#!/bin/bash
set -e

echo "🚀 Starting Mirror Vault Chain Setup"
echo "======================================"

# Clean state
echo "📁 Cleaning old state..."
rm -rf ~/.mirrorvault
pkill -9 mirrorvaultd 2>/dev/null || true

# Copy binary
echo "📦 Installing binary..."
cd /home/abdul-sami/work/The-Mirror-Vault/chain

CHAIN_DIR="/home/abdul-sami/work/The-Mirror-Vault/chain"
BINARY="$CHAIN_DIR/mirrorvaultd"

# Initialize chain
echo "🔧 Initializing chain..."
$BINARY init mirror-vault --chain-id mirror-vault-localnet --default-denom umvlt --overwrite

# Add test accounts with large balances
echo "👤 Creating test accounts..."
echo "alarm client shove cycle squirrel essence" | $BINARY keys add alice --recover --keyring-backend test
echo "word6 word7 word8 word9 word10 word11 word12 word13 word14 word15 word16 word17 word18 word19 word20 word21 word22 word23 word24" | $BINARY keys add bob --recover --keyring-backend test 2>/dev/null || $BINARY keys add bob --keyring-backend test

# Add genesis accounts with 10000 MVLT each (10000000000 umvlt)
echo "💰 Adding genesis accounts..."

# NOTE: EVM uses 18-decimal base units; fund with 10,000 * 1e18 = 1e22 umvlt.
FUND_AMT="10000000000000000000000umvlt"

$BINARY genesis add-genesis-account alice "$FUND_AMT" --keyring-backend test
$BINARY genesis add-genesis-account bob "$FUND_AMT" --keyring-backend test

# Fund the Hardhat/MetaMask default dev account used in contracts/hardhat.config.ts
$BINARY genesis add-genesis-account mirror17w0adeg64ky0daxwd2ugyuneellmjgnx7uk5xa "$FUND_AMT" --keyring-backend test

# Optionally fund additional accounts (e.g. addresses you connect from MetaMask/Keplr)
# Usage:
#   EXTRA_GENESIS_ACCOUNTS="mirror1...,mirror1..." bash ./start-chain.sh
if [ -n "${EXTRA_GENESIS_ACCOUNTS:-}" ]; then
	echo "➕ Funding EXTRA_GENESIS_ACCOUNTS..."
	IFS=',' read -r -a EXTRA_ADDRS <<< "$EXTRA_GENESIS_ACCOUNTS"
	for addr in "${EXTRA_ADDRS[@]}"; do
		addr="$(echo "$addr" | xargs)"
		[ -z "$addr" ] && continue
		echo "  - $addr"
		$BINARY genesis add-genesis-account "$addr" "$FUND_AMT" --keyring-backend test >/dev/null 2>&1 || \
			echo "  ⚠️  Could not fund $addr (skipping)"
	done
fi

# Create validator
echo "⚡ Creating validator..."
$BINARY genesis gentx alice 1000000umvlt --chain-id mirror-vault-localnet --keyring-backend test
$BINARY genesis collect-gentxs

# Update app.toml for EVM
echo "⚙️  Configuring EVM..."
sed -i 's/enable = false/enable = true/g' ~/.mirrorvault/config/app.toml
sed -i 's/api = "eth,net,web3"/api = "eth,net,web3,debug"/g' ~/.mirrorvault/config/app.toml
sed -i 's/minimum-gas-prices = ""/minimum-gas-prices = "0umvlt"/g' ~/.mirrorvault/config/app.toml

# Ensure WS is reachable from Docker (bind to 0.0.0.0:8546)
sed -i 's/ws-address = "127\.0\.0\.1:8546"/ws-address = "0.0.0.0:8546"/g' ~/.mirrorvault/config/app.toml
sed -i 's/ws-origins = \["127\.0\.0\.1", "localhost"\]/ws-origins = ["*", "127.0.0.1", "localhost", "host.docker.internal"]/g' ~/.mirrorvault/config/app.toml

# Set feemarket base fee to 0 for local dev.
jq '.app_state.feemarket.params.no_base_fee = true | .app_state.feemarket.params.min_gas_price = "0.000000000000000000" | .app_state.feemarket.params.base_fee = "0.000000000000000000"' \
	~/.mirrorvault/config/genesis.json > ~/.mirrorvault/config/genesis.json.tmp && mv ~/.mirrorvault/config/genesis.json.tmp ~/.mirrorvault/config/genesis.json

echo "✅ Setup complete! Starting chain..."
echo ""
exec $BINARY start --api.enable --log_level info
