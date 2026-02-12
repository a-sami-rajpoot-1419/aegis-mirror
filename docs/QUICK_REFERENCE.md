# Manual Wiring Quick Reference Card

## Essential Facts

| Aspect | Detail |
|--------|--------|
| **Estimated Time** | 5-7 hours implementation |
| **Risk Level** | Low (proven pattern) |
| **Reversibility** | Yes (state machine unchanged) |
| **Dependencies** | cosmos/evm v0.5.0 (already configured) |
| **Files Changed** | app.go (major), root.go (minor), app_config.go (delete) |
| **Line Count** | 374 → ~800 lines (explicit > implicit) |

---

## What You're Getting

### Immediate Benefits
✅ Working EVM integration (MsgEthereumTx accepted)  
✅ JSON-RPC on port 8545 (MetaMask ready)  
✅ Unified identity (same key, dual addresses)  
✅ Full keeper control (explicit dependencies)  
✅ Phase 1 complete → Unblocks Phase 2 & 3  

### Long-term Benefits
✅ Production-grade architecture (Celestia, Sei, dYdX pattern)  
✅ Easy to add modules (x/vault, IBC, etc.)  
✅ Debuggable (no hidden depinject magic)  
✅ Future-proof (can migrate back if cosmos/evm v2 adds depinject)  

---

## Key Changes Summary

### App Struct
```go
// FROM: Hidden wrapper
type App struct {
    *runtime.App  // Black box
    // ...
}

// TO: Explicit control
type App struct {
    *baseapp.BaseApp  // Direct access
    keys    map[string]*storetypes.KVStoreKey
    tkeys   map[string]*storetypes.TransientStoreKey
    ModuleManager *module.Manager
    // ...
}
```

### Keeper Initialization
```go
// FROM: Automatic (depinject)
depinject.Inject(appConfig, &app.AuthKeeper, &app.BankKeeper)

// TO: Explicit order
app.AccountKeeper = authkeeper.NewAccountKeeper(...)
app.BankKeeper = bankkeeper.NewBaseKeeper(...)
app.StakingKeeper = stakingkeeper.NewKeeper(...)
// ... in dependency order
```

### The Critical Fix
```go
// FROM: CustomGetSigner fails in depinject
depinject.Supply(evmtypes.MsgEthereumTxCustomGetSigner)
// Result: Runtime panic

// TO: CustomGetSigner in codec creation
encodingConfig := evmosencoding.MakeConfig(
    basicManager,
    []signingtypes.CustomGetSigner{
        evmtypes.MsgEthereumTxCustomGetSigner,  // ✅ WORKS!
    },
)
```

---

## Execution Checklist

### Pre-Implementation
- [x] Research complete (3 options analyzed)
- [x] Migration plan documented
- [x] Dependency graph created
- [x] Risk assessment done
- [x] Timeline estimated
- [ ] **User approval received** ⏳

### Implementation (5-7 hours)
- [ ] Create feature branch
- [ ] Backup current working state
- [ ] Implement Phase 1: Store Keys (15 min)
- [ ] Implement Phase 2: BaseApp (20 min)
- [ ] Implement Phase 3: Codec + CustomGetSigner (30 min)
- [ ] Implement Phase 4: Keepers (2 hours)
- [ ] Implement Phase 5: ModuleManager (1 hour)
- [ ] Implement Phase 6: Ante Handler (30 min)
- [ ] Clean up: Delete app_config.go, update root.go (20 min)

### Validation (1 hour)
- [ ] Compilation test
- [ ] Genesis init test
- [ ] Chain start test
- [ ] Block production test
- [ ] REST API test (localhost:1317)
- [ ] JSON-RPC test (localhost:8545)
- [ ] Cosmos transaction test
- [ ] EVM query test (eth_chainId, eth_getBalance)
- [ ] MetaMask connection test
- [ ] MetaMask transaction test

### Post-Implementation (30 min)
- [ ] Update README.md
- [ ] Update PROJECT_STATE.md
- [ ] Update IMPLEMENTATION.md
- [ ] Mark Phase 1 complete
- [ ] Merge to main

---

## Keeper Dependency Order (Critical!)

```
Order of initialization (dependencies first):

1. AccountKeeper        (ROOT - no dependencies)
2. BankKeeper          (depends: AccountKeeper)
3. ConsensusParamsKeeper (ROOT - no dependencies)
4. StakingKeeper       (depends: AccountKeeper, BankKeeper)
5. DistrKeeper         (depends: AccountKeeper, BankKeeper, StakingKeeper)

EVM Keepers:
6. FeeMarketKeeper     (ROOT - no dependencies)
7. PreciseBankKeeper   (depends: BankKeeper, AccountKeeper)
8. EVMKeeper          (depends: AccountKeeper, PreciseBankKeeper, 
                                StakingKeeper, FeeMarketKeeper, 
                                ConsensusParamsKeeper)
9. Erc20Keeper        (depends: AccountKeeper, BankKeeper, 
                                EVMKeeper, StakingKeeper)

Special: Erc20Keeper ←→ EVMKeeper (circular reference handled with WithErc20Keeper)
```

---

## Module Manager Order (Consensus Critical!)

### BeginBlockers
```go
distrtypes.ModuleName,
stakingtypes.ModuleName,
feemarkettypes.ModuleName,  // Update EIP-1559 base fee
evmtypes.ModuleName,
```

### EndBlockers
```go
stakingtypes.ModuleName,
evmtypes.ModuleName,  // Process EVM state changes
```

### InitGenesis
```go
authtypes.ModuleName,
banktypes.ModuleName,
distrtypes.ModuleName,
stakingtypes.ModuleName,
consensusparamtypes.ModuleName,
genutiltypes.ModuleName,
// EVM modules
feemarkettypes.ModuleName,
precisebanktypes.ModuleName,
evmtypes.ModuleName,
erc20types.ModuleName,
```

---

## Common Pitfalls (Avoid These!)

### 1. Wrong Keeper Order
❌ **Bad**: Initialize EVMKeeper before PreciseBankKeeper  
✅ **Good**: Follow dependency graph exactly  

### 2. Forgot Store Mounting
❌ **Bad**: Create keys but don't mount stores  
✅ **Good**: Mount all KV, transient, and memory stores  

### 3. Module Permissions Wrong
❌ **Bad**: Give Minter permission without controls  
✅ **Good**: Copy exact permissions from evmd reference  

### 4. Circular Reference Broken
❌ **Bad**: Try to pass Erc20Keeper to EVMKeeper constructor  
✅ **Good**: Pass nil, then call `EVMKeeper.WithErc20Keeper()` after  

### 5. CustomGetSigner Too Late
❌ **Bad**: Register after codec creation  
✅ **Good**: Register during `MakeEncodingConfig()`  

---

## Success Metrics

### After Migration (5-7 hours)
```bash
# 1. Binary compiles
cd chain && go build ./cmd/mirrorvaultd
# Expected: Success, ~100-150 MB binary

# 2. Genesis initializes
mirrorvaultd init test --chain-id mirror-vault-localnet
# Expected: .mirrorvault-mvlt/ directory created

# 3. Chain starts
mirrorvaultd start --evm.chain-id 7777
# Expected: Blocks producing, no panics

# 4. JSON-RPC responds
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
  http://localhost:8545
# Expected: {"jsonrpc":"2.0","id":1,"result":"0x1e61"}

# 5. MetaMask connects
# Add network: http://localhost:8545, Chain ID 7777
# Expected: Network added, balance shows
```

---

## Emergency Rollback

If migration fails critically:

```bash
git checkout backup/depinject-working
cd chain && go build ./cmd/mirrorvaultd
# Back to last known working state
```

**Note**: State is compatible (store keys unchanged), can export/import if needed.

---

## Reference Files

| Document | Purpose |
|----------|---------|
| **MANUAL_WIRING_MIGRATION_PLAN.md** | Complete 800-line technical plan |
| **MIGRATION_SUMMARY.md** | Executive summary (this file) |
| **PROJECT_STATE.md** | Overall project status |
| **IMPLEMENTATION.md** | Phase-by-phase implementation guide |

---

## Support Reference

### EVMD Source (Reference Implementation)
- Location: `/tmp/evm-reference/evmd/app.go`
- Lines: 1,208 (full-featured with IBC, Gov, etc.)
- Our target: ~800 lines (minimal viable + EVM)

### Key Differences from EVMD
- **Missing modules** (not needed for v1): IBC, Mint, Slashing, Gov, Authz, Feegrant, Evidence, Params
- **Custom modules** (Phase 3): x/vault (coming later)
- **Same EVM integration**: vm, feemarket, erc20, precisebank

---

## Decision Time

### Question: Proceed with Manual Wiring Migration?

**If YES**:
- Estimated: 5-7 hours implementation + 1 hour testing
- Risk: Low (proven pattern, reversible)
- Outcome: Phase 1 complete, EVM fully operational
- Next: Reply with approval, I'll begin implementation

**If NO / Need More Info**:
- Review MANUAL_WIRING_MIGRATION_PLAN.md for complete details
- Ask specific questions about any aspect
- Request additional risk analysis or alternatives

---

**Current Status**: ⏳ Planning Complete → Awaiting Implementation Approval

**Ready to execute**: Yes, all planning complete ✅
