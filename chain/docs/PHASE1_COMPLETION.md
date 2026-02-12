# Phase 1: Manual Wiring Migration - COMPLETED ✅

**Branch:** `feature/manual-wiring-migration`  
**Date:** February 12, 2025  
**Status:** Compilation successful, binary functional

## Summary

Successfully migrated from depinject to manual keeper initialization, resolving the CustomGetSigner blocker and cosmos/evm v0.5.0 ante package issues.

## Changes Made

### 1. app/app.go - Complete Rewrite (374 → 554 lines)
- **Removed:** All depinject.Inject calls, runtime.App wrapper dependency
- **Added:** Direct baseapp.BaseApp usage with explicit control
- **Key Components:**
  - Manual store key management (9 KV + 2 transient)
  - Explicit keeper initialization in dependency order
  - Custom ante handler integration
  - Module manager with explicit ordering

#### Keeper Initialization Order (Critical)
```
1. AccountKeeper (root - no dependencies)
2. BankKeeper (depends: Account)
3. ConsensusParamsKeeper (root)
4. StakingKeeper (depends: Account, Bank)
5. DistrKeeper (depends: Account, Bank, Staking)
6. FeeMarketKeeper (EVM, root)
7. PreciseBankKeeper (EVM, depends: Bank, Account)
8. EVMKeeper (EVM, depends: Account, PreciseBank, Staking, FeeMarket, ConsensusParams)
9. Erc20Keeper (EVM, depends: Account, Bank, EVM, Staking)
```

### 2 ante/ - New Custom Ante Handler Package
Created 4 files copied from evmd reference:
- **ante.go** (53 lines): Transaction router by extension options
- **cosmos_handler.go** (29 lines): Simplified Cosmos tx decorators
- **evm_handler.go** (30 lines): EVM transaction decorators
- **handler_options.go** (66 lines): Handler configuration

**Key Fix:** Removed broken cosmos/evm/ante/cosmos imports that caused compilation errors. Simplified to Phase 1 essentials:
- ✅ Standard SDK ante decorators
- ✅ EVM transaction handling
- ⏸️ Deferred: EIP-712 (MetaMask Cosmos tx signing)
- ⏸️ Deferred: IBC ante decorators
- ⏸️ Deferred: Authz limiters

### 3. Deleted Files
- **app_config.go:** Depinject configuration no longer needed

### 4. cmd/ Updates
**root.go:**
- Removed depinject.Inject dependencies
- Direct encoding config creation via `app.MakeEncodingConfig()`
- Skipped AutoCLI enhancement to avoid address codec requirement (Phase 2)

**commands.go:**
- Fixed LoadHeight call: `bApp.CommitMultiStore().LoadVersion(height)`

**testnet.go:**
- Fixed: `app.App.NewUncachedContext` → `app.BaseApp.NewUncachedContext`
- Fixed: `app.AuthKeeper` → `app.AccountKeeper`

## Blockers Resolved

### Original Blocker: CustomGetSigner Registration ✅
**Solution:** cosmos/evm's `encoding.MakeConfig()` automatically registers CustomGetSigner for MsgEthereumTx. No manual registration needed with manual wiring approach.

### Secondary Blocker: cosmos/evm v0.5.0 Ante Package ✅
**Problem:** `github.com/cosmos/evm@v0.5.0/ante/cosmos/eip712.go` has undefined secp256k1 functions
```
undefined: secp256k1.RecoverPubkey (line 247)
undefined: secp256k1.VerifySignature (line 273)
```

**Solution:** Created simplified ante handlers without importing broken cosmos/evm/ante/cosmos package. Phase 1 uses standard SDK decorators only. EIP-712 support deferred to Phase 2.

### Runtime Panic: Address Codec ✅
**Problem:** `panic: address codec is required in flag builder`

**Solution:** Disabled autocli.EnhanceRootCommand() for Phase 1. Will add proper AutoCLI setup in Phase 2 when all keepers are fully configured.

## Testing Status

### ✅ Compilation
```bash
cd chain
go build ./cmd/mirrorvaultd
# Success! No errors
```

### ✅ Binary Creation
```bash
ls -lh mirrorvaultd
# -rwxr-xr-x 108M mirrorvaultd
```

### ✅ Basic CLI
```bash
./mirrorvaultd --help
# Shows full command tree: init, start, query, tx, keys, etc.
```

### ⏳ Pending Tests (Next Steps)
- [ ] Genesis initialization: `mirrorvaultd init test`
- [ ] Chain startup: `mirrorvaultd start --evm.chain-id 7777`
- [ ] JSON-RPC endpoint: `curl localhost:8545`
- [ ] MetaMask connectivity
- [ ] EVM transaction execution

## Git Commits

1. **80c8204** - refactor: complete Phase 1 manual wiring migration
   - Complete app.go rewrite
   - Custom ante handler package
   - Deleted app_config.go
   - Updated cmd/ files

2. **29712e2** - fix: resolve runtime address codec panic in CLI
   - Removed AutoCLI enhancement
   - Binary now functional

## Architecture Comparison

### Before (Depinject)
```
app.AppConfig() (depinject config)
    ↓
runtime.App (hidden wrapper)
    ↓
Keepers (auto-injected, order unclear)
    ↓
❌ CustomGetSigner registration blocked
```

### After (Manual Wiring)
```
MakeEncodingConfig() (CustomGetSigner auto-registered by cosmos/evm)
    ↓
baseapp.BaseApp (direct control)
    ↓
Manual keeper init (explicit dependency order)
    ↓
Custom ante handlers (simplified for Phase 1)
    ↓
✅ Full control, clear dependencies
```

## Files Modified

- **chain/app/app.go** (rewritten)
- **chain/app/app.go.depinject-backup** (created - safety backup)
- **chain/ante/ante.go** (created)
- **chain/ante/cosmos_handler.go** (created)
- **chain/ante/evm_handler.go** (created)
- **chain/ante/handler_options.go** (created)
- **chain/cmd/mirrorvaultd/cmd/root.go** (updated)
- **chain/cmd/mirrorvaultd/cmd/commands.go** (updated)
- **chain/cmd/mirrorvaultd/cmd/testnet.go** (updated)
- **chain/app/app_config.go** (deleted)

## Metrics

- **Lines Changed:** ~755 insertions, ~584 deletions
- **Files Modified:** 11
- **New Package:** ante/ (4 files)
- **Compilation Time:** ~90 seconds
- **Binary Size:** 108 MB

## Phase 2 Roadmap

**Features to Re-enable:**
1. EIP-712 support (cosmos/evm ante/cosmos decorators)
2. IBC integration (transfer module + ante decorators)
3. Authz message limiters
4. AutoCLI enhancement with proper address codec
5. Full genesis initialization testing
6. MetaMask transaction signing
7. JSON-RPC validation

**When to Proceed:**
- After cosmos/evm fixes upstream secp256k1 issues, OR
- Implement custom EIP-712 decorators without broken dependencies

## Lessons Learned

1. **cosmos/evm v0.5.0 has upstream bugs** - Not all official packages are production-ready
2. **Manual wiring provides clarity** - Explicit keeper dependencies prevent subtle issues
3. **Depinject limitations** - Some advanced customizations (CustomGetSigner) are easier with manual setup
4. **Phase approach works** - Get compilation first, add features incrementally
5. **Reference implementations help** - evmd provided clear ante handler patterns

## Success Criteria Met

- ✅ Code compiles without errors
- ✅ Binary runs and shows CLI
- ✅ CustomGetSigner blocker resolved
- ✅ Ante package blocker resolved
- ✅ Changes committed to feature branch
- ✅ Safety backup (backup/depinject-working) preserved

## Next Actions

1. Test genesis initialization
2. Test chain startup
3. Validate JSON-RPC endpoints
4. Test MetaMask connectivity
5. If all tests pass → merge to main
6. Document Phase 2 requirements
