# The Mirror Vault

Hybrid Cosmos + EVM L1 with unified identity mapping (one key → `0x…` + `mirror1…`).

## What’s in this repo
- `chain/` — Cosmos SDK chain (custom blockchain binary). **Requires Go**.
- `contracts/` — Solidity contracts (Hardhat).
- `frontend/` — Next.js dashboard (to be scaffolded).
- `docs/` — frozen constants, architecture, and implementation guide.
- `tools/` — environment setup and build scripts.

## Documentation

- **[Constants](docs/constants.md)** — Frozen v1 configuration (chain-id, ports, denoms)
- **[Implementation Guide](docs/IMPLEMENTATION.md)** — Complete integration details, configurations, and build process
- **[Project State](docs/PROJECT_STATE.md)** — Architecture, scope, and business logic design
- **[Dev Flow](docs/dev-flow.md)** — WSL2 toolchain and workflow

## Current Status

✅ **Operational**: Chain producing blocks, bank transactions confirmed  
⏳ **In Progress**: Manual wiring migration (Phase 1 - EVM Integration)  
🔴 **Pending**: JSON-RPC endpoints, MetaMask connectivity, x/vault module

**See [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md) for detailed status and next steps.**

## Tech Stack Versions

**Core:**
- Cosmos SDK: v0.53.5
- CometBFT: v0.38.21
- Cosmos EVM: v0.5.0
- Go: 1.25.7 (runtime), 1.24.1+ (required)
- Go-Ethereum: cosmos/go-ethereum v1.16.2-cosmos-1 (Cosmos fork)

**SDK Modules:**
- cosmossdk.io/core: v0.11.3
- cosmossdk.io/store: v1.1.2
- cosmossdk.io/depinject: v1.2.1

**Database & Storage:**
- cosmos-db: v1.1.3
- keyring: cosmos/keyring v1.2.0 (Cosmos fork)

## Quick Start

### Prerequisites
- WSL2 Ubuntu 22.04 (Windows) or native Linux
- Internet connection for toolchain download

### Setup
```bash
# Clone repo
cd /home/abdul-sami/work/The-Mirror-Vault

# Load environment (adds Go, Node, Ignite to PATH)
source tools/env.sh

# Build chain binary
bash tools/chain-build-safe.sh

# Start chain (temporary ports until evmos conflict resolved)
chain/build/mirrorvaultd start \
  --home $HOME/.mirrorvault-mvlt \
  --rpc.laddr tcp://127.0.0.1:26667 \
  --api.enable \
  --api.address tcp://127.0.0.1:13177
```

### Verify Chain
```bash
# Check RPC
curl http://127.0.0.1:26667/status | jq '.result.node_info.network'
# Should output: "mirror-vault-localnet"

# Check REST API
curl http://127.0.0.1:13177/cosmos/base/tendermint/v1beta1/node_info | jq '.default_node_info.network'
# Should output: "mirror-vault-localnet"
```

## Development Workflow

1. **Always source environment first**: `source tools/env.sh`
2. **Build changes**: `bash tools/chain-build-safe.sh`
3. **Refer to implementation docs**: [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md)

## Next Milestones

1. Free ports 26657/1317 (stop evmos container)
2. Integrate Cosmos EVM + JSON-RPC (port 8545)
3. Implement x/vault module + precompile (0x0101)
4. Deploy VaultGate.sol via Hardhat
5. Test cross-VM workflow (EVM ↔ Cosmos)
