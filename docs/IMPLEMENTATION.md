# The Mirror Vault — Implementation Guide

**Living Documentation**: This file tracks all implementation details, integration steps, and configuration as the project evolves.

**Last Updated**: 2026-02-11  
**Current Phase**: Cosmos SDK + CometBFT operational; EVM integration pending

---

## Table of Contents
1. [Project Status Overview](#project-status-overview)
2. [Development Environment](#development-environment)
3. [Chain Implementation](#chain-implementation)
4. [Build System](#build-system)
5. [Cosmos SDK Integration](#cosmos-sdk-integration)
6. [CometBFT Consensus Integration](#cometbft-consensus-integration)
7. [Testing & Validation](#testing--validation)
8. [Port Configuration](#port-configuration)
9. [Pending Integrations](#pending-integrations)
10. [Known Issues & Resolutions](#known-issues--resolutions)

---

## Project Status Overview

### ✅ Completed Components

| Component | Version | Status | Notes |
|-----------|---------|--------|-------|
| Cosmos SDK | v0.53.5 | ✅ Integrated | Core state machine operational |
| CometBFT | v0.38.19 | ✅ Integrated | Consensus producing blocks |
| Chain Binary | mirrorvaultd | ✅ Built | Located in `chain/build/` |
| Bank Module | SDK native | ✅ Tested | Transactions confirmed on-chain |
| REST API (LCD) | SDK native | ✅ Operational | Port 13177 (temp) |
| RPC Endpoint | CometBFT | ✅ Operational | Port 26667 (temp) |
| Keyring | SDK native | ✅ Working | Test backend, 3 accounts |

### 🔴 Pending Components

| Component | Status | Blocker |
|-----------|--------|---------|
| Cosmos EVM | Not started | Needs integration |
| JSON-RPC (8545) | Not started | Requires cosmos/evm |
| x/vault Module | Not started | Custom business logic |
| Stateful Precompile | Not started | Requires cosmos/evm + x/vault |
| Unified Identity | Not started | EthAccount/EthSecp256k1 wiring |
| Canonical Ports | Blocked | Port conflict with evmos |

---

## Development Environment

### Platform Requirements
- **Host OS**: Windows 11+ with WSL2
- **WSL2 Distro**: Ubuntu-22.04
- **Reason for WSL2**: Ignite CLI and Cosmos SDK tooling require Unix environment

### Toolchain Installation

All tools are installed in user-local directories (no sudo required).

#### Directory Structure
```
$HOME/
├── go/             # Go workspace
├── .local/bin/     # User binaries (Go, Node, Ignite)
└── .mirrorvault-mvlt/  # Chain home directory
```

#### Installed Tools
- **Go**: v1.25.7 (installed in `$HOME/.local/bin/go`)
- **Node.js**: v23.6.0 LTS (installed in `$HOME/.local/bin/node`)
- **Ignite CLI**: v28.6.1 (Go binary in `$HOME/go/bin/ignite`)

#### Environment Setup Script

**Location**: `tools/env.sh`

**Purpose**: Standardizes PATH and environment variables for consistent builds

**Usage**:
```bash
source tools/env.sh
```

**What it does**:
- Adds `$HOME/.local/bin` to PATH (Go, Node)
- Adds `$HOME/go/bin` to PATH (Ignite, other Go tools)
- Adds `chain/build` to PATH (mirrorvaultd binary)
- Sets `IGNITE_CLI_HEADLESS=1` (prevents interactive prompts)
- **Critical**: Always source this before building or running chain commands

---

## Chain Implementation

### Scaffolding Process

#### Initial Scaffold Command
```bash
cd /home/abdul-sami/work/The-Mirror-Vault
ignite scaffold chain github.com/yourorg/mirrorvault --no-module --address-prefix mirror
mv mirrorvault chain
```

**Decisions Made**:
- `--no-module`: Start with clean slate for custom modules
- `--address-prefix mirror`: Aligns with frozen constant (bech32 prefix)
- Moved scaffold output to `chain/` subdirectory

#### Directory Structure
```
chain/
├── app/                    # Application wiring (Cosmos SDK app)
│   ├── app.go             # Main app initialization & keeper wiring
│   ├── export.go          # Genesis export logic
│   └── sim_test.go        # Simulation tests
├── cmd/
│   └── mirrorvaultd/      # Binary entry point
│       └── cmd/
│           ├── root.go    # CLI root command
│           └── testnet.go # Testnet utility commands
├── proto/                 # Protobuf definitions (future: x/vault)
├── x/                     # Custom modules (empty, future: x/vault)
├── build/                 # Build output directory
│   └── mirrorvaultd      # Compiled binary
├── config.yml            # Ignite chain configuration
├── go.mod                # Go module dependencies
├── go.sum                # Go dependency checksums
├── Makefile              # Build commands
└── README.md             # Chain-specific readme
```

---

## Build System

### Build Strategies

#### 1. Standard Ignite Build (Not Used)
```bash
ignite chain build
```
**Problem**: Caused WSL2 OOM kills and terminal disconnects due to high memory usage

#### 2. Safe Build Script (Active Solution)

**Location**: `tools/chain-build-safe.sh`

**Configuration**:
```bash
GOMAXPROCS=2        # Limit to 2 CPU cores
GOMEMLIMIT=2GiB     # Hard memory limit
GOGC=50             # Aggressive garbage collection
```

**Usage**:
```bash
cd /home/abdul-sami/work/The-Mirror-Vault
source tools/env.sh
bash tools/chain-build-safe.sh
```

**Output**:
- Binary: `chain/build/mirrorvaultd`
- Build log: `/tmp/mirrorvault-chain-build.log`

**Why This Works**:
- Prevents WSL2 memory exhaustion
- Enables compilation on low-memory systems
- Logs are preserved for debugging

#### 3. Git Ignore Configuration

**File**: `chain/.gitignore`

**Added Entry**:
```
build/
```

**Reason**: Binary artifacts should not be committed to version control

---

## Cosmos SDK Integration

### Configuration File: `chain/config.yml`

This file configures Ignite's chain setup and genesis parameters.

#### Key Configuration Sections

##### Validator Configuration
```yaml
validators:
  - name: validator1
    bonded: "100000000umvlt"
    app:
      pruning: "default"
      halt-height: 0
```

##### Genesis Accounts
```yaml
accounts:
  - name: alice
    coins:
      - 1000000000umvlt         # 1 billion base units
  - name: bob
    coins:
      - 1000000000umvlt
  - name: carol
    coins:
      - 1000000000umvlt
```

**Account Details**:
- **Alice**: Primary test account for transactions
- **Bob**: Secondary account for receive testing
- **Carol**: Third account for multi-party scenarios
- **Keyring Backend**: `test` (insecure, local development only)

##### Faucet Configuration
```yaml
faucet:
  name: validator1
  coins:
    - 5000000umvlt            # 5M per faucet request
  coins-max:
    - 100000000umvlt          # 100M lifetime max per address
  port: 4500
```

**Purpose**: Allows easy token distribution during development

##### Genesis Parameters
```yaml
genesis:
  chain_id: "mirror-vault-localnet"
  app_state:
    staking:
      params:
        bond_denom: "umvlt"
    crisis:
      constant_fee:
        denom: "umvlt"
    gov:
      deposit_params:
        min_deposit:
          - denom: "umvlt"
            amount: "10000000"
    mint:
      params:
        mint_denom: "umvlt"
```

**Aligned Constants** (from `docs/constants.md`):
- Chain ID: `mirror-vault-localnet`
- Native denom: `umvlt` (base unit)
- Display denom: `MVLT` (not yet configured for SDK display metadata)

### Module Integration

#### Currently Active Modules
These are standard Cosmos SDK modules included by default:

1. **Auth**: Account management, signatures
2. **Bank**: Token transfers, balances
3. **Staking**: Validator delegation (for consensus)
4. **Gov**: Governance proposals
5. **Crisis**: Invariant checking
6. **Mint**: Token inflation (can be disabled later)
7. **Distribution**: Fee distribution to validators
8. **Slashing**: Validator punishment
9. **Genutil**: Genesis utilities

#### Module Wiring Location
**File**: `chain/app/app.go`

The `app.go` file uses **depinject** (Cosmos SDK v0.50+ pattern) for module wiring:
```go
// Modules are wired via app_config.go and app.yaml (generated by Ignite)
```

**Important**: When adding cosmos/evm or x/vault, they must be registered here.

---

## CometBFT Consensus Integration

### Version & Compatibility
- **CometBFT Version**: v0.38.19
- **Relationship**: Successor to Tendermint Core
- **Integration**: Automatic via Cosmos SDK v0.53.5

### Configuration Files

#### Genesis File
**Location**: `$HOME/.mirrorvault-mvlt/config/genesis.json`

**Key Sections**:
```json
{
  "chain_id": "mirror-vault-localnet",
  "initial_height": "1",
  "consensus_params": {
    "block": {
      "max_bytes": "22020096",
      "max_gas": "-1"
    },
    "evidence": {
      "max_age_num_blocks": "100000",
      "max_age_duration": "172800000000000"
    }
  }
}
```

**Genesis Validation**:
```bash
mirrorvaultd genesis validate --home $HOME/.mirrorvault-mvlt
# Output: File at /home/abdul-sami/.mirrorvault-mvlt/config/genesis.json is a valid genesis file
```

#### Validator Configuration
**Location**: `$HOME/.mirrorvault-mvlt/config/config.toml`

**Relevant Sections** (excerpt):
```toml
[rpc]
laddr = "tcp://127.0.0.1:26657"     # Currently overridden to 26667

[p2p]
laddr = "tcp://0.0.0.0:26656"       # Currently overridden to 26666
```

#### Application Configuration
**Location**: `$HOME/.mirrorvault-mvlt/config/app.toml`

**Relevant Sections** (excerpt):
```toml
[api]
enable = true
address = "tcp://0.0.0.0:1317"      # Currently overridden to 13177

[grpc]
address = "0.0.0.0:9090"            # Currently overridden to 9097
```

### Gentx Process (Genesis Transactions)

Genesis transactions establish the initial validator set.

#### Gentx Generation Command
```bash
cd /home/abdul-sami/work/The-Mirror-Vault
source tools/env.sh

# Clear old gentx
rm -f $HOME/.mirrorvault-mvlt/config/gentx/*.json

# Generate new gentx for alice as validator
mirrorvaultd genesis gentx alice 200000000umvlt \
  --chain-id mirror-vault-localnet \
  --home $HOME/.mirrorvault-mvlt \
  --keyring-backend test \
  --yes

# Collect all gentxs into genesis
mirrorvaultd genesis collect-gentxs --home $HOME/.mirrorvault-mvlt

# Validate final genesis
mirrorvaultd genesis validate --home $HOME/.mirrorvault-mvlt
```

**Why This Was Needed**:
- Initial genesis had wrong chain-id
- After fixing genesis.json manually, gentx signatures were invalid
- Full regeneration was required

---

## Testing & Validation

### Compilation Tests

#### Test Command
```bash
cd /home/abdul-sami/work/The-Mirror-Vault/chain
source ../tools/env.sh
go test ./... -run TestDoesNotExist
```

**Result**: ✅ All packages compile successfully
```
ok      github.com/yourorg/mirrorvault/app
ok      github.com/yourorg/mirrorvault/cmd/mirrorvaultd/cmd
```

### Runtime Smoke Tests

#### 1. Chain Start (Alternate Ports)

**Start Command**:
```bash
cd /home/abdul-sami/work/The-Mirror-Vault
source tools/env.sh

GOMAXPROCS=2 GOMEMLIMIT=2GiB GOGC=50 \
chain/build/mirrorvaultd start \
  --home $HOME/.mirrorvault-mvlt \
  --moniker mirrorvault-localnet \
  --rpc.laddr tcp://127.0.0.1:26667 \
  --api.enable \
  --api.address tcp://127.0.0.1:13177 \
  --grpc.address localhost:9097 \
  --p2p.laddr tcp://0.0.0.0:26666
```

**Verification**:
```bash
# Check CometBFT RPC
curl http://127.0.0.1:26667/status | jq '.result.node_info.network'
# Output: "mirror-vault-localnet"

# Check REST API
curl http://127.0.0.1:13177/cosmos/base/tendermint/v1beta1/node_info | jq '.default_node_info.network'
# Output: "mirror-vault-localnet"
```

#### 2. Account Query Test

**List Keys**:
```bash
mirrorvaultd keys list --home $HOME/.mirrorvault-mvlt --keyring-backend test
```

**Output**:
```
- address: mirror1hj5fveer5cjtn4wd6wstzugjfdxzl0xp96n8v2
  name: alice
  pubkey: '{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"A..."}'
  type: local

- address: mirror1cyyzpxplxdzkeea7kwsydadg6hmtzggvtv7mhf
  name: bob
  type: local

- address: mirror1wqc8gde4gqlq84g4ktc7pj5fkecslzxnnks8d3
  name: carol
  type: local
```

**Query Balance**:
```bash
mirrorvaultd query bank balances mirror1hj5fveer5cjtn4wd6wstzugjfdxzl0xp96n8v2 \
  --node http://127.0.0.1:26667
```

**Result**: ✅ All genesis accounts funded with 1,000,000,000 umvlt

#### 3. Transaction Test (Bank Send)

**Transaction Command**:
```bash
mirrorvaultd tx bank send \
  alice \
  mirror1cyyzpxplxdzkeea7kwsydadg6hmtzggvtv7mhf \
  12345umvlt \
  --chain-id mirror-vault-localnet \
  --home $HOME/.mirrorvault-mvlt \
  --keyring-backend test \
  --node http://127.0.0.1:26667 \
  --yes
```

**Transaction Hash**: `2EA6A9D9C05C2A5B5990F5A7DE2AC3BDE44820C6BC1A4F3E4D8ED9A8F9E0A0C2`

**Query Transaction**:
```bash
mirrorvaultd query tx 2EA6A9... --node http://127.0.0.1:26667
```

**Result**: ✅ Transaction included at height 2225
```json
{
  "code": 0,
  "height": "2225",
  "txhash": "2EA6A9D9...",
  "events": [
    {
      "type": "coin_spent",
      "attributes": [
        {"key": "spender", "value": "mirror1hj5fveer5cjtn4wd6wstzugjfdxzl0xp96n8v2"},
        {"key": "amount", "value": "12345umvlt"}
      ]
    },
    {
      "type": "coin_received",
      "attributes": [
        {"key": "receiver", "value": "mirror1cyyzpxplxdzkeea7kwsydadg6hmtzggvtv7mhf"},
        {"key": "amount", "value": "12345umvlt"}
      ]
    }
  ]
}
```

**Balance Verification After TX**:
```bash
# Alice (sender) balance decreased
mirrorvaultd query bank balances mirror1hj5fveer5cjtn4wd6wstzugjfdxzl0xp96n8v2 --node http://127.0.0.1:26667
# Output: 799987655umvlt (1B - 12345 - gas fees)

# Bob (receiver) balance increased
mirrorvaultd query bank balances mirror1cyyzpxplxdzkeea7kwsydadg6hmtzggvtv7mhf --node http://127.0.0.1:26667
# Output: 1000012345umvlt (1B + 12345)
```

**Conclusion**: ✅ Cosmos SDK + CometBFT fully operational

---

## Port Configuration

### Canonical Ports (Target Configuration)

From `docs/constants.md`:

| Service | Port | Protocol | Status |
|---------|------|----------|--------|
| CometBFT RPC | 26657 | HTTP | ⚠️ Occupied by evmos |
| Cosmos REST (LCD) | 1317 | HTTP | ⚠️ Occupied by evmos |
| EVM JSON-RPC | 8545 | HTTP | 🔴 Not implemented |
| gRPC | 9090 | gRPC | ✅ Available |
| P2P | 26656 | TCP | ✅ Available |
| Faucet | 4500 | HTTP | ✅ Available |

### Current Port Assignment (Temporary)

Due to port conflict with existing evmos chain:

| Service | Current Port | Reason |
|---------|--------------|--------|
| CometBFT RPC | 26667 | Avoids conflict with evmos 26657 |
| Cosmos REST | 13177 | Avoids conflict with evmos 1317 |
| gRPC | 9097 | Avoids conflict with evmos 9090 |
| P2P | 26666 | Avoids conflict with evmos 26656 |

### Port Conflict Details

**Conflicting Process**:
- **Chain**: evmbridge_9000-1
- **Binary**: evmosd v18.1.0
- **Moniker**: evmbridge-node
- **CometBFT**: v0.37.4

**Verification**:
```bash
curl http://127.0.0.1:26657/status | jq '.result.node_info.network'
# Output: "evmbridge_9000-1" (NOT mirror-vault-localnet)
```

**Resolution Plan**:
1. User will stop Docker container running evmos
2. Ports 26657 and 1317 will be freed
3. Mirror Vault will be restarted on canonical ports
4. All documentation will reference canonical ports going forward

---

## Pending Integrations

### 1. Cosmos EVM Module

**Repository**: `github.com/cosmos/evm`  
**Target Version**: v0.5.1  
**Compatibility**: ✅ Verified compatible with Cosmos SDK v0.53.5 and CometBFT v0.38.19

#### Integration Steps (Not Yet Started)

1. **Add Dependency**:
   ```bash
   cd chain
   go get github.com/cosmos/evm@v0.5.1
   ```

2. **Wire EVM Module** in `chain/app/app.go`:
   - Add evm keeper
   - Register evm module
   - Configure JSON-RPC server on port 8545
   - Set EVM chainId to 7777

3. **Configure EVM Parameters**:
   - Base denom mapping: `umvlt` ↔ EVM balance
   - Gas parameters
   - EVM chainId: 7777 (must match `contracts/hardhat.config.ts`)

4. **Enable JSON-RPC Server**:
   - Endpoint: `http://127.0.0.1:8545`
   - Enable eth namespace
   - Enable web3 namespace
   - Enable net namespace

5. **Test MetaMask Connection**:
   - Add custom network in MetaMask
   - Chain ID: 7777
   - RPC URL: http://127.0.0.1:8545
   - Currency symbol: MVLT

#### Expected Deliverables
- [ ] MetaMask can connect to chain
- [ ] Account balances visible in MetaMask
- [ ] Simple contract deployment succeeds (e.g., HelloWorld.sol)
- [ ] Contract interactions work via ethers.js

### 2. Unified Identity Layer

**Goal**: One private key → both `0x...` and `mirror1...` addresses

#### Required Changes

1. **Account Type**:
   - Replace default `BaseAccount` with `EthAccount`
   - Located in cosmos/evm account types

2. **Key Algorithm**:
   - Replace `secp256k1` with `ethsecp256k1`
   - Ensures EVM-compatible signature format

3. **BIP-44 Coin Type**:
   - Set to `60` (Ethereum coin type)
   - Configured in chain genesis/params

4. **Address Derivation**:
   - EVM address: First 20 bytes of Keccak256(pubkey)
   - Cosmos address: Bech32 encoding of same 20 bytes with "mirror" prefix

#### Validation Test
- Import same mnemonic into MetaMask and Keplr
- Both wallets should show equivalent balances
- Sending from either wallet updates same underlying account

### 3. Custom x/vault Module

**Purpose**: Implements storage credit system for cross-VM business logic

#### Module Structure
```
chain/x/vault/
├── keeper/
│   ├── keeper.go          # Core state management
│   ├── credits.go         # Credit increment/decrement
│   └── msg_server.go      # Message handlers
├── types/
│   ├── keys.go            # Store keys
│   ├── messages.go        # MsgStoreSecret definition
│   └── genesis.go         # Genesis state
├── proto/
│   └── vault/
│       └── v1/
│           └── tx.proto   # Protobuf message definitions
└── module.go              # Module interface implementation
```

#### State Schema
```protobuf
message Credit {
  string address = 1;
  uint64 count = 2;
}

message StoredMessage {
  string address = 1;
  uint64 message_count = 2;
  string last_message = 3;
}
```

#### Messages
1. **MsgStoreSecret**:
   - Requires: credit > 0
   - Action: Decrements credit, stores message, increments message_count
   - Returns: Updated message_count

#### Keeper Methods
- `IncrementCredit(address)`: +1 credit (called by precompile)
- `DecrementCredit(address)`: -1 credit (called by MsgStoreSecret handler)
- `GetCredit(address)`: Query credit count
- `StoreMessage(address, message)`: Store message data

### 4. Stateful Precompile (0x0101)

**Address**: `0x0000000000000000000000000000000000000101`

#### Interface (Solidity)
```solidity
interface IVaultPrecompile {
    function unlock() external returns (uint256 newCredit);
}
```

#### Implementation (Go)
**Location**: `chain/x/vault/precompile/vault.go`

**Logic**:
```go
func (p *VaultPrecompile) Run(evm, caller, input) ([]byte, error) {
    // Decode function selector
    if bytes.Equal(input[:4], unlockSelector) {
        // Get Cosmos address from EVM caller
        cosmosAddr := convertEVMtoCosmos(caller)
        
        // Call x/vault keeper to increment credit
        p.vaultKeeper.IncrementCredit(cosmosAddr)
        
        // Query new credit count
        newCredit := p.vaultKeeper.GetCredit(cosmosAddr)
        
        // Return uint256 encoded result
        return encodeUint256(newCredit), nil
    }
    return nil, errors.New("unknown method")
}
```

#### Registration
Precompiles are registered in the EVM module's keeper initialization.

#### Testing Strategy
1. Deploy `VaultGate.sol` via Hardhat
2. Call `payToUnlock()` from MetaMask
3. Query credit via Cosmos CLI: should show +1
4. Attempt `MsgStoreSecret` via Keplr: should succeed
5. Query credit again: should show decremented

---

## Known Issues & Resolutions

### Issue 1: WSL2 OOM During Build
**Symptom**: Terminal disconnect, build failure  
**Cause**: High memory usage during Go compilation  
**Resolution**: Created `tools/chain-build-safe.sh` with memory constraints  
**Status**: ✅ Resolved

### Issue 2: Unused Imports
**Symptom**: Build errors in `app.go`, `export.go`, `sim_test.go`, `testnet.go`  
**Cause**: Ignite scaffold included imports for modules we're not using yet  
**Resolution**: Removed unused imports, ran `go fmt`  
**Status**: ✅ Resolved

### Issue 3: Gentx Signature Verification Failed
**Symptom**: Chain start failed with "gentx signature verification failed"  
**Cause**: Genesis chain-id was modified manually, invalidating signatures  
**Resolution**: Regenerated gentx using `mirrorvaultd genesis gentx alice ...`  
**Status**: ✅ Resolved

### Issue 4: Missing priv_validator_state.json
**Symptom**: Chain start error "file does not exist"  
**Cause**: File wasn't created during genesis init  
**Resolution**: Created minimal JSON:
```json
{
  "height": "0",
  "round": 0,
  "step": 0
}
```
**Status**: ✅ Resolved

### Issue 5: Port Conflict with evmos
**Symptom**: Cannot start Mirror Vault on canonical ports 26657/1317  
**Cause**: Another chain (evmbridge/evmos) is running on those ports  
**Resolution**: Temporarily using alternate ports; evmos will be stopped  
**Status**: ⚠️ In Progress - User stopping Docker container

---

## Next Steps (In Sequence)

### Immediate
1. ✅ Documentation updated (this file)
2. ⏳ User stops evmos Docker container to free ports
3. Restart Mirror Vault on canonical ports (26657, 1317)
4. Re-run smoke tests on canonical endpoints

### Phase 2: EVM Integration
1. Add `github.com/cosmos/evm@v0.5.1` to `chain/go.mod`
2. Wire EVM module in `chain/app/app.go`
3. Configure JSON-RPC server on port 8545
4. Test MetaMask connection and basic contract deployment

### Phase 3: Identity Unification
1. Replace account type with EthAccount
2. Switch to ethsecp256k1 key algorithm
3. Set BIP-44 coin type to 60
4. Validate address pairing with MetaMask + Keplr

### Phase 4: Business Logic
1. Scaffold `x/vault` module
2. Implement credit storage and queries
3. Implement `MsgStoreSecret` handler
4. Build stateful precompile at 0x0101
5. Register precompile with EVM keeper

### Phase 5: End-to-End Testing
1. Deploy `VaultGate.sol` via Hardhat
2. Test unlock flow (Solidity → precompile → credit)
3. Test gated message (MsgStoreSecret requires credit)
4. Validate full cross-VM workflow

### Phase 6: Frontend (Future)
- Scaffold Next.js application
- Integrate MetaMask + Keplr
- Build UI components per design specs

---

## Maintenance Notes

**This document should be updated whenever**:
- A new module is integrated
- Configuration changes are made
- Build process is modified
- Issues are discovered and resolved
- Testing procedures are added or changed

**Last Updated By**: GitHub Copilot  
**Next Update Trigger**: After ports are freed and chain is on canonical endpoints
