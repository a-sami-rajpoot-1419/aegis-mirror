# Wallet Setup Guide - Mirror Vault

## System Status
✅ **JSON-RPC Server:** Running on port 8545  
✅ **Chain ID (EVM):** 7777 (0x1e61)  
✅ **Chain ID (Cosmos):** mirror-1  
✅ **gRPC Queries:** Operational  
✅ **Test Accounts:** Created (Alice & Bob)

---

## Test Accounts

### Account 1: Alice
**Mnemonic:**
```
fragile maid song peace uniform fee december era velvet alpha recycle fat blade swarm coin beach weather adjust into hill word reduce shell slender
```

**Cosmos Address:** `mirror1uskahcc3ljj5uuw7w2pcyu9vhmt3k2gnmvxqrf`  
**Private Key (Hex):** `71923d8ad4c092191a4896e53aaefccb992da68a3b28e6a5abc3ad02b621e764`

**To compute EVM address:**
- Import private key into MetaMask → will derive 0x... address automatically
- Both addresses share same private key (unified identity!)

---

### Account 2: Bob  
**Mnemonic:**
```
fox group canoe burden gossip excuse wheat test erase truly narrow mother thing wash urban luggage such nuclear kingdom slow over half elbow inch
```

**Cosmos Address:** `mirror1vwn78eq0rnxxafugdrk0kzc338k42r4677vh5f`  
**Private Key (Hex):** `2b213e9efacdc8f8982f04f8f2f9df6c15a2f4ad954e2df776d04fd6b510d43c`

---

## MetaMask Setup (EVM Side)

### 1. Add Mirror Vault Network to MetaMask

**Network Configuration:**
- **Network Name:** Mirror Vault Localnet
- **RPC URL:** `http://localhost:8545`
- **Chain ID:** 7777
- **Currency Symbol:** ATOM
- **Block Explorer:** (None - local testnet)

**Steps:**
1. Open MetaMask → Settings → Networks → Add Network
2. Enter the configuration above
3. Save

### 2. Import Test Accounts

**Method 1: Import via Private Key (Recommended)**
1. MetaMask → Account Icon → Import Account
2. Select "Private Key"
3. Paste Alice's private key: `71923d8ad4c092191a4896e53aaefccb992da68a3b28e6a5abc3ad02b621e764`
4. Repeat for Bob: `2b213e9efacdc8f8982f04f8f2f9df6c15a2f4ad954e2df776d04fd6b510d43c`

**Method 2: Import via Mnemonic**
1. MetaMask → Account Icon → Import using Secret Recovery Phrase
2. Enter Alice's or Bob's 24-word mnemonic
3. MetaMask will derive the same address as Keplr!

---

## Keplr Setup (Cosmos Side)

### 1. Add Mirror Vault Chain to Keplr

Since this is a local testnet, you need to add it manually using Keplr's experimental features.

**Chain Configuration (for Keplr):**
```json
{
  "chainId": "mirror-1",
  "chainName": "Mirror Vault Localnet",
  "rpc": "http://localhost:26657",
  "rest": "http://localhost:1317",
  "bip44": {"coinType": 60},
  "bech32Config": {
    "bech32PrefixAccAddr": "mirror",
    "bech32PrefixAccPub": "mirrorpub",
    "bech32PrefixValAddr": "mirrorvaloper",
    "bech32PrefixValPub": "mirrorvaloperpub",
    "bech32PrefixConsAddr": "mirrorvalcons",
    "bech32PrefixConsPub": "mirrorvalconspub"
  },
  "currencies": [
    {
      "coinDenom": "ATOM",
      "coinMinimalDenom": "umirror",
      "coinDecimals": 6,
      "coinGeckoId": "cosmos"
    }
  ],
  "feeCurrencies": [
    {
      "coinDenom": "ATOM",
      "coinMinimalDenom": "umirror",
      "coinDecimals": 6,
      "coinGeckoId": "cosmos",
      "gasPriceStep": {
        "low": 0.01,
        "average": 0.025,
        "high": 0.04
      }
    }
  ],
  "stakeCurrency": {
    "coinDenom": "ATOM",
    "coinMinimalDenom": "umirror",
    "coinDecimals": 6,
    "coinGeckoId": "cosmos"
  },
  "features": ["eth-address-gen", "eth-key-sign"]
}
```

**Steps:**
1. Install Keplr browser extension
2. Use Keplr's developer mode to add custom chain
3. OR: Use the chain configuration above with `window.keplr.experimentalSuggestChain()`

### 2. Import Test Accounts into Keplr

**Method 1: Import via Mnemonic (Recommended)**
1. Keplr → Add Account → Import existing account
2. Enter Alice's 24-word mnemonic
3. **CRITICAL:** Select account HD path with coin type **60** (Ethereum)
4. This ensures same private key as MetaMask!

**Method 2: Import via Private Key**
1. Keplr → Add Account → Import private key
2. Paste private key in hex format
3. Confirm coin type 60

---

## Unified Identity Verification

### The "Mirror" Principle
One private key → Two interfaces:
- **MetaMask:** Shows 0x... (hex-encoded address)
- **Keplr:** Shows mirror1... (bech32-encoded address)
- **Balance:** SHARED between both wallets!

### How to Verify

**Test 1: Send from CLI, check MetaMask**
```bash
cd /home/abdul-sami/work/The-Mirror-Vault/chain

# Send tokens to Alice's Cosmos address
./mirrorvaultd tx bank send validator mirror1uskahcc3ljj5uuw7w2pcyu9vhmt3k2gnmvxqrf 1000000umirror \
  --chain-id mirror-1 \
  --keyring-backend test \
  --yes

# Wait 5 seconds, then check MetaMask
# Alice's MetaMask balance should increase!
```

**Test 2: Send from MetaMask, check CLI**
```bash
# In MetaMask: Send 0.001 ATOM to Bob's address
# Then check via CLI:
./mirrorvaultd query bank balances mirror1vwn78eq0rnxxafugdrk0kzc338k42r4677vh5f
```

**Test 3: Address Conversion**
```bash
# Get Alice's EVM address from her Cosmos address
./mirrorvaultd keys parse mirror1uskahcc3ljj5uuw7w2pcyu9vhmt3k2gnmvxqrf

# Should show both representations!
```

---

## Testing Checklist

- [ ] Chain running (check: `ps aux | grep mirrorvaultd`)
- [ ] JSON-RPC responding (test: `curl -X POST http://localhost:8545 -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'`)
- [ ] MetaMask connected to Mirror Vault network
- [ ] Keplr connected to mirror-1 chain
- [ ] Alice account imported in both wallets
- [ ] Bob account imported in both wallets
- [ ] Balance visible in MetaMask (after funding)
- [ ] Balance visible in Keplr (same amount!)
- [ ] Transfer from Keplr → balance updates in MetaMask
- [ ] Transfer from MetaMask → balance updates in CLI query

---

## Troubleshooting

### MetaMask shows 0 balance
- Check chain is running: `ps aux | grep mirrorvaultd`
- Check JSON-RPC: `curl http://localhost:8545 -X POST ...`
- Verify network configuration (chain ID must be 7777)
- Ensure account was funded (use `mirrorvaultd tx bank send`)

### Keplr can't connect
- Check if RPC is accessible: `curl http://localhost:26657/status`
- Verify coin type 60 was used during import
- Try reimporting with correct mnemonic

### Addresses don't match
- **This is expected!** Same key, different encoding:
  - EVM uses hex: `0x...` (40 chars)
  - Cosmos uses bech32: `mirror1...` (43+ chars)
- Use `mirrorvaultd keys parse <address>` to convert between formats

### Chain won't start
- Reset with: `./mirrorvaultd comet unsafe-reset-all`
- Check logs: `tail -100 /tmp/chain_test2.log`
- Verify genesis is valid: `./mirrorvaultd genesis validate`

---

## Next Steps (NOT YET IMPLEMENTED)

1. **gRPC Service Registration** - Additional query endpoints
2. **Smart Contract Deployment** - VaultGate.sol
3. **x/vault Module** - Cosmos-side storage
4. **0x0101 Precompile** - EVM↔Cosmos bridge
5. **Frontend UI** - Dual wallet dashboard

---

## Quick Start Command

**Start chain in background:**
```bash
cd /home/abdul-sami/work/The-Mirror-Vault/chain
./mirrorvaultd start > /tmp/chain.log 2>&1 &

# Check status
curl -s -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'

# Should return: {"jsonrpc":"2.0","id":1,"result":"0x1e61"}
```

**Stop chain:**
```bash
pkill mirrorvaultd
```

---

## Security Notice

⚠️ **TESTING ONLY**: All private keys and mnemonics in this document are for local testing only. **NEVER use these accounts on mainnet or with real funds!**

The keyring backend is set to `test` mode which stores keys unencrypted for development purposes.
