# gRPC Query Service Fix - Technical Summary

## Problem Statement

After enabling JSON-RPC server for MetaMask connectivity, gRPC query services were returning errors:
```
unknown service cosmos.evm.vm.v1.Query
```

Testing with `eth_blockNumber` was failing because the EVM module's gRPC query services weren't registered.

---

## Root Cause Analysis

### Issue 1: Missing vm.AppModule Registration
**Location:** [chain/app/app.go](../chain/app/app.go#L439-L452)

The `vm.AppModule` was not being instantiated in the module list, which meant:
- gRPC query services (`Query` service in cosmos/evm/vm/v1) were not registered
- JSON-RPC server couldn't query block height via gRPC
- `eth_blockNumber` and similar queries failed

### Issue 2: "EVM coin info already set" Panic
**Location:** [chain/app/app.go](../chain/app/app.go#L199-L206)

During InitGenesis, the chain panicked with:
```
panic: EVM coin info already set
```

**Root Cause:** 
- `EVMConfigurator.WithEVMCoinInfo()` was called in `app.New()` to set coin info
- Later, `evm.InitGenesis()` tried to set coin info **again** using `sync.Once`
- The `sync.Once` detected duplicate initialization and panicked

### Issue 3: Nil Pointer in precisebank.InitGenesis
**Location:** [chain/app/app.go](../chain/app/app.go#L477-L494)

After removing duplicate coin info setup, a new panic occurred:
```
panic: runtime error: invalid memory address or nil pointer dereference
at precisebank.InitGenesis → ConversionFactor() → GetEVMCoinDecimals()
```

**Root Cause:**
- Module initialization order was: `auth → bank → ... → precisebank → evm → feemarket → erc20`
- When `precisebank.InitGenesis()` ran, it validated conversion factors
- Validation called `GetEVMCoinDecimals()` from EVM keeper
- **BUT** EVM module hadn't initialized yet, so coin info was nil!

---

## Solution Implementation

### Fix 1: Register vm.AppModule
**File:** [chain/app/app.go](../chain/app/app.go#L447)

**Before:**
```go
modules := []module.AppModule{
    auth.NewAppModule(app.appCodec, app.AccountKeeper, nil, nil),
    bank.NewAppModule(app.appCodec, app.BankKeeper, app.AccountKeeper, nil),
    // ... other modules
    feemarket.NewAppModule(app.FeeMarketKeeper),
    erc20.NewAppModule(app.Erc20Keeper, app.AccountKeeper),
    precisebank.NewAppModule(app.PreciseBankKeeper, app.BankKeeper, app.AccountKeeper),
}
```

**After:**
```go
modules := []module.AppModule{
    auth.NewAppModule(app.appCodec, app.AccountKeeper, nil, nil),
    bank.NewAppModule(app.appCodec, app.BankKeeper, app.AccountKeeper, nil),
    // ... other modules
    // EVM modules with gRPC services
    vm.NewAppModule(app.EVMKeeper, app.AccountKeeper, app.BankKeeper, authcodec.NewBech32Codec(AccountAddressPrefix)),
    feemarket.NewAppModule(app.FeeMarketKeeper),
    erc20.NewAppModule(app.Erc20Keeper, app.AccountKeeper),
    precisebank.NewAppModule(app.PreciseBankKeeper, app.BankKeeper, app.AccountKeeper),
}
```

**Key Parameters:**
- `app.EVMKeeper` - Provides EVM state queries
- `app.AccountKeeper` - Account operations
- `app.BankKeeper` - Balance queries
- `authcodec.NewBech32Codec(AccountAddressPrefix)` - Address codec for "mirror" prefix

### Fix 2: Remove Duplicate EVM Coin Info Setup
**File:** [chain/app/app.go](../chain/app/app.go#L199-L206)

**Before:**
```go
// Configure EVM with cosmos coin info
app.EVMConfigurator.WithEVMCoinInfo(evmtypes.EVMCoinInfo{
    Denom:    BondDenom,
    Decimals: 18,
})
```

**After:**
```go
// Note: EVM coin info will be initialized during InitGenesis from bank denom metadata
// Do not configure it here to avoid "EVM coin info already set" panic
```

**Why this works:**
- Bank module creates denom metadata in genesis: `aatom` with 18 decimals
- EVM module's `InitGenesis()` reads this metadata and initializes coin info automatically
- No manual setup needed in `app.New()`

### Fix 3: Correct Module Initialization Order
**File:** [chain/app/app.go](../chain/app/app.go#L477-L494)

**Before:**
```go
app.ModuleManager.SetOrderInitGenesis(
    authtypes.ModuleName,
    banktypes.ModuleName,
    distrtypes.ModuleName,
    stakingtypes.ModuleName,
    consensustypes.ModuleName,
    genutiltypes.ModuleName,
    precisebanktypes.ModuleName,  // WRONG: Runs before EVM!
    evmtypes.ModuleName,
    feemarkettypes.ModuleName,
    erc20types.ModuleName,
)
```

**After:**
```go
// Set init genesis order
// CRITICAL: evm module MUST initialize before precisebank module
// because precisebank.InitGenesis validates using GetEVMCoinDecimals()
// which is only set during evm.InitGenesis
app.ModuleManager.SetOrderInitGenesis(
    authtypes.ModuleName,
    banktypes.ModuleName,
    distrtypes.ModuleName,
    stakingtypes.ModuleName,
    consensustypes.ModuleName,
    genutiltypes.ModuleName,
    // EVM modules - ORDER MATTERS!
    evmtypes.ModuleName,        // FIRST: Initialize EVM coin config
    feemarkettypes.ModuleName,  // SECOND: Fee market uses EVM config
    precisebanktypes.ModuleName, // THIRD: Validates using GetEVMCoinDecimals()
    erc20types.ModuleName,      // FOURTH: ERC20 depends on EVM + precisebank
)
```

**Dependency Chain:**
1. **evm** → Sets coin info from bank metadata
2. **feemarket** → Queries EVM config for gas calculations
3. **precisebank** → Validates conversion factors using `GetEVMCoinDecimals()`
4. **erc20** → Depends on both EVM and precisebank state

---

## Verification

### Test 1: Chain Startup
```bash
cd /home/abdul-sami/work/The-Mirror-Vault/chain
./mirrorvaultd comet unsafe-reset-all
./mirrorvaultd start
```

**Result:** ✅ Chain starts successfully, no panics

### Test 2: JSON-RPC eth_chainId
```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'
```

**Expected Output:**
```json
{"jsonrpc":"2.0","id":1,"result":"0x1e61"}
```

**Result:** ✅ Returns 0x1e61 (7777 decimal)

### Test 3: JSON-RPC eth_blockNumber
```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":2}'
```

**Expected Output:**
```json
{"jsonrpc":"2.0","id":2,"result":"0x1"}
```

**Result:** ✅ Returns current block height in hex

### Test 4: gRPC Query Service
```bash
grpcurl -plaintext localhost:9090 cosmos.evm.vm.v1.Query/Params
```

**Result:** ✅ Returns EVM module params (previously failed with "unknown service")

---

## Key Takeaways

### 1. Module Registration vs Keeper Initialization
- **Keeper creation** happens in `app.New()` - this creates the keeper instance
- **Module registration** happens when adding to `modules` array - this registers gRPC services
- **Both are required** for full functionality

### 2. Genesis Initialization Order Matters
When modules depend on each other's state:
- The **provider** module must initialize first
- The **consumer** module initializes second
- Use comments to document dependencies

Example dependency:
```
precisebank.InitGenesis() 
  → calls keeper.Validate()
    → calls GetEVMCoinDecimals() 
      → requires evm.InitGenesis() to have run first!
```

### 3. Cosmos SDK v0.53 Patterns
With manual wiring (no depinject):
- Module creation is explicit: `vm.NewAppModule(...)`
- Parameter order matters - check `go doc` for signatures
- Use address codecs, not string prefixes: `authcodec.NewBech32Codec()`

### 4. Dual Initialization Patterns
Some keeper configuration can happen in two places:
- **Option A:** Configure in `app.New()` (explicit setup)
- **Option B:** Configure in `InitGenesis()` (from genesis state)

**Choose one!** Using both causes the cosmos/evm's `sync.Once` panic.

**Best Practice:** Prefer `InitGenesis()` for configuration that should be in genesis state (e.g., coin info, chain params).

---

## Related Issues Fixed

This fix also resolved:
1. **Mempool race condition** - Fixed in previous session by adding coin info validation
2. **JSON-RPC server integration** - Now fully operational
3. **MetaMask connectivity** - Can now query chain state

---

## Next Steps

### Immediate (Ready to Execute)
- [x] gRPC query services registered
- [x] Test accounts created (Alice & Bob)
- [x] Private keys exported for MetaMask
- [ ] Test MetaMask connection (see [WALLET_SETUP.md](./WALLET_SETUP.md))
- [ ] Test Keplr connection
- [ ] Verify unified identity (same key in both wallets)

### Future (Awaiting Discussion)
- [ ] VaultGate.sol smart contract deployment
- [ ] x/vault Cosmos module implementation
- [ ] 0x0101 precompile for EVM↔Cosmos calls
- [ ] Frontend UI for dual wallet management

---

## Commit Information

**Branch:** `feature/manual-wiring-migration`

**Files Modified:**
- [chain/app/app.go](../chain/app/app.go)

**Suggested Commit Message:**
```
feat: register EVM gRPC services and fix module init order

- Add vm.NewAppModule to register EVM query services (fixes "unknown service" error)
- Remove duplicate EVMCoinInfo setup (fixes "already set" panic)
- Reorder InitGenesis: evm before precisebank (fixes nil pointer panic)
- Document module initialization dependencies

Fixes JSON-RPC queries: eth_chainId, eth_blockNumber now working
Chain starts cleanly, all gRPC services operational
```

---

## References

- **cosmos/evm documentation:** https://github.com/berachain/cosmos-sdk/tree/evm-stable/x/evm
- **Cosmos SDK v0.53 manual wiring:** https://docs.cosmos.network/v0.53/build/building-modules/
- **JSON-RPC specification:** https://ethereum.org/en/developers/docs/apis/json-rpc/
- **Project docs:** [dev-flow.md](./dev-flow.md), [PROJECT_STATE.md](./PROJECT_STATE.md)
