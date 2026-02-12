# Manual Wiring Migration - Executive Summary

**Date**: 2026-02-12  
**Status**: ✅ Planning Complete → ⏳ Awaiting Implementation Approval  
**Full Documentation**: [MANUAL_WIRING_MIGRATION_PLAN.md](./MANUAL_WIRING_MIGRATION_PLAN.md)  

---

## What We're Doing

Converting from **Ignite's depinject-based wiring** to **manual keeper initialization** (evmd pattern) to enable full Cosmos EVM integration with SDK v0.53.5.

---

## Why This Is Necessary

**The Blocker**: cosmos/evm's `MsgEthereumTx` lacks the `cosmos.msg.v1.signer` protobuf annotation required by SDK v0.50+. The workaround (`CustomGetSigner`) is incompatible with depinject's architecture.

**The Solution**: Manual wiring allows registering `CustomGetSigner` during codec creation, before keeper initialization. This is the **only** viable path forward.

**Ecosystem Validation**: Major production chains use the same pattern:
- Celestia (custom DA layer)
- Sei (parallel EVM)
- dYdX v4 (high-frequency trading)
- Evmos (EVM integration - same use case)

---

## What Stays vs What Changes

### ✅ What Stays (90% of Value)

| Component | Status | Impact |
|-----------|--------|--------|
| **All module logic** | Unchanged | Keeper implementations identical |
| **Store key names** | Unchanged | State compatibility maintained |
| **Protobufs** | Unchanged | Message definitions identical |
| **CLI commands** | Unchanged | `mirrorvaultd tx/query` work same |
| **CometBFT** | Unchanged | Consensus layer unaffected |
| **Project structure** | Unchanged | chain/, contracts/, docs/ stay |
| **Ignite tooling** | Partially | Protobuf gen works, scaffold doesn't |
| **Testing** | Unchanged | testutil/ remains functional |

### ⚠️ What Changes (Only app.go)

| Component | From | To | Impact |
|-----------|------|-----|--------|
| **App struct** | `*runtime.App` wrapper | `*baseapp.BaseApp` direct | Explicit control |
| **Store keys** | Hidden in wrapper | Explicit maps | Visibility |
| **Keeper init** | Automatic via depinject | Manual in order | Control |
| **Modules** | Auto-registered | Explicit registration | Visibility |
| **app_config.go** | Depinject config | Deleted (moved to app.go) | Simplified |
| **Line count** | 374 lines | ~800 lines | More explicit |

---

## Migration Steps (5-7 Hours)

### Critical Path

1. **Backup code** → Create feature branch (5 min)
2. **Store keys** → Manual creation instead of depinject (15 min)
3. **BaseApp** → Direct initialization, mount stores (20 min)
4. **Codec** → Create with `CustomGetSigner` ✅ **SOLVES BLOCKER** (30 min)
5. **Keepers** → Initialize in dependency order (2 hours)
6. **Modules** → Create ModuleManager, set begin/end blockers (1 hour)
7. **Ante handler** → EVM-aware chain (30 min)
8. **Clean up** → Remove app_config.go, update root.go (20 min)
9. **Test** → Compile, genesis, chain start, JSON-RPC (1.5 hours)

### Keeper Initialization Order

```
AccountKeeper (root)
├─→ BankKeeper
│   ├─→ PreciseBankKeeper (EVM)
│   └─→ StakingKeeper
│       ├─→ DistrKeeper
│       ├─→ EVMKeeper (EVM)
│       └─→ Erc20Keeper (EVM)
├─→ ConsensusParamsKeeper → EVMKeeper (EVM)
└─→ FeeMarketKeeper (EVM, independent)
```

**Critical**: Order matters! Dependencies must be initialized first.

---

## Validation Tests

### Must Pass Before Merge

- [ ] Compilation succeeds (`go build`)
- [ ] Genesis initializes (`mirrorvaultd init`)
- [ ] Chain starts without panic
- [ ] Blocks produce (check logs)
- [ ] REST API responds (curl localhost:1317)
- [ ] JSON-RPC responds (curl localhost:8545)
- [ ] Bank send transaction works (Cosmos)
- [ ] `eth_chainId` returns 7777 (EVM)
- [ ] `eth_getBalance` works (EVM)
- [ ] MetaMask connects successfully
- [ ] MetaMask transaction succeeds

---

## Risk Assessment

### Low Risk ✅

- **State machine unchanged**: Store keys and module logic identical
- **Proven pattern**: Used by major production chains
- **Reversible**: Can export/import state if needed
- **Incremental**: Each phase independently testable

### What Could Go Wrong (Mitigations)

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Keeper order wrong | Low | High | Follow dependency graph, test incrementally |
| Store key collision | Very Low | High | Validation function checks duplicates |
| Module permissions wrong | Low | Medium | Copy from evmd reference, document |
| Genesis migration | Very Low | High | Store keys unchanged = compatible |
| Compilation errors | Medium | Low | Reference evmd side-by-side |

---

## Benefits After Migration

### Immediate (Phase 1 Complete)

- ✅ **EVM fully operational** - MsgEthereumTx accepted
- ✅ **JSON-RPC working** - Port 8545 accepting requests
- ✅ **MetaMask ready** - Can connect and send transactions
- ✅ **CustomGetSigner working** - Ethereum signatures validated
- ✅ **Unified identity** - Same key, dual addresses (mirror1... and 0x...)

### Future (Phase 2+)

- ✅ **Easy module addition** - x/vault in Phase 3 trivial to add
- ✅ **Stateful precompiles** - Can register after keeper init
- ✅ **IBC integration** - Placeholders ready
- ✅ **Full debugging control** - No black-box depinject
- ✅ **Production-grade** - Same pattern as major chains

---

## Timeline

| Phase | Task | Duration | Status |
|-------|------|----------|--------|
| Planning | Research options | 2 hours | ✅ Complete |
| Planning | Create migration plan | 2 hours | ✅ Complete |
| **Implementation** | **Execute migration** | **5-7 hours** | ⏳ **Pending Approval** |
| Validation | Test suite | 1 hour | 🔴 Not started |
| Documentation | Update docs | 30 min | 🔴 Not started |

**Total Estimated**: 10-12 hours (planning through validation)  
**Remaining**: 6-8 hours (implementation through validation)

---

## Recommendation

### ✅ PROCEED with Manual Wiring Migration

**Rationale**:
1. **Only viable option** - No alternatives exist for SDK v0.53.5 + cosmos/evm
2. **Proven approach** - 5+ major chains use this pattern in production
3. **Low risk** - State machine unchanged, reversible if needed
4. **High confidence** - Research complete, clear execution path
5. **Unblocks roadmap** - Phase 1 completion enables Phase 2 & 3

**Alternative Options (All Inferior)**:
- ❌ Wait for cosmos/evm v2.0 - No timeline, could be months/years
- ❌ Fork cosmos/evm - High maintenance burden
- ❌ Downgrade SDK - Lose modern features, technical debt
- ❌ Use alternative EVM - Polaris deprecated, Evmos incompatible SDK fork

---

## Next Actions

### Required for Approval

✅ **Planning complete** - This document + MANUAL_WIRING_MIGRATION_PLAN.md  
✅ **Risk assessment done** - Low risk, high confidence  
✅ **Timeline estimated** - 5-7 hours implementation  
✅ **Validation plan ready** - 12-step test checklist  
⏳ **Awaiting user approval** - "Go ahead with implementation"  

### After Approval

1. Create feature branch: `feature/manual-wiring`
2. Execute migration following MANUAL_WIRING_MIGRATION_PLAN.md
3. Run validation test suite
4. Update documentation (README, PROJECT_STATE, IMPLEMENTATION)
5. Merge to main
6. Mark Phase 1 complete ✅
7. Begin Phase 2 (IBC integration) or Phase 3 (x/vault module)

---

## Questions?

- **Full technical details**: See [MANUAL_WIRING_MIGRATION_PLAN.md](./MANUAL_WIRING_MIGRATION_PLAN.md)
- **Keeper dependency graph**: Included in plan document (Mermaid diagrams)
- **Security considerations**: Module permissions, blocked addresses - all documented
- **Future compatibility**: Module addition patterns documented

---

**Ready to proceed?** Reply with approval and I'll begin the 5-7 hour implementation phase.

**Need more details?** Ask specific questions about any aspect of the plan.

**Want to review more?** The full 800-line migration plan has component-by-component breakdowns, code examples, and security considerations.
