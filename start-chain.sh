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
cp mirrorvaultd ~/go/bin/

# Initialize chain
echo "🔧 Initializing chain..."
mirrorvaultd init mirror-vault --chain-id mirror-vault-localnet --default-denom umvlt --overwrite

# Add test accounts with large balances
echo "👤 Creating test accounts..."
echo "alarm client shove cycle squirrel essence" | mirrorvaultd keys add alice --recover --keyring-backend test
echo "word6 word7 word8 word9 word10 word11 word12 word13 word14 word15 word16 word17 word18 word19 word20 word21 word22 word23 word24" | mirrorvaultd keys add bob --recover --keyring-backend test 2>/dev/null || mirrorvaultd keys add bob --keyring-backend test

# Add genesis accounts with 10000 MVLT each (10000000000 umvlt)
echo "💰 Adding genesis accounts..."
mirrorvaultd genesis add-genesis-account alice 10000000000umvlt --keyring-backend test
mirrorvaultd genesis add-genesis-account bob 10000000000umvlt --keyring-backend test

# Create validator
echo "⚡ Creating validator..."
mirrorvaultd genesis gentx alice 1000000umvlt --chain-id mirror-vault-localnet --keyring-backend test
mirrorvaultd genesis collect-gentxs

# Update app.toml for EVM
echo "⚙️  Configuring EVM..."
sed -i 's/enable = false/enable = true/g' ~/.mirrorvault/config/app.toml
sed -i 's/api = "eth,net,web3"/api = "eth,net,web3,debug"/g' ~/.mirrorvault/config/app.toml

echo "✅ Setup complete! Starting chain..."
echo ""
exec mirrorvaultd start --json-rpc.enable --json-rpc.api eth,web3,net,debug --api.enable --log_level info
