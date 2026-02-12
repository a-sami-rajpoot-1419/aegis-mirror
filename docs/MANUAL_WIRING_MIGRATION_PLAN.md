# Manual Wiring Migration Plan
## Hybrid Approach: Depinject → Manual Keeper Initialization

**Created**: 2026-02-12  
**Status**: Planning Phase  
**Estimated Effort**: 4-7 hours  

---

## Executive Summary

We are converting from **Ignite's depinject-based wiring** to **manual keeper initialization** (evmd pattern) to enable full Cosmos EVM integration. This is **NOT a downgrade** - it's graduating to "Pro Mode" used by production chains (Celestia, Sei, dYdX, Evmos).

### Why This Migration?

**Root Cause**: cosmos/evm v0.5.0-v1.0.0 was designed for SDK v0.47's manual wiring patterns. The MsgEthereumTx protobuf lacks `cosmos.msg.v1.signer` annotation required by SDK v0.50+'s depinject. No workaround exists.

**The Choice**: Manual wiring is the **only** path to working EVM integration with SDK v0.53.5.

### What We Keep (90% of Ignite Value)

✅ **All existing modules** - x/auth, x/bank, x/staking, x/distribution, x/consensus  
✅ **All protobufs** - Message definitions remain identical  
✅ **Ignite CLI tooling** - For protobuf generation and scaffolding  
✅ **Module logic** - No changes to keeper implementations  
✅ **Project structure** - chain/, contracts/, docs/, tools/  
✅ **CometBFT integration** - Consensus layer unchanged  
✅ **CLI commands** - mirrorvaultd tx/query commands work identically  
✅ **Testing infrastructure** - testutil/ remains functional  

### What Changes (Only app.go)

❌ **depinject.Inject()** → Manual keeper initialization  
❌ **runtime.App wrapper** → Explicit baseapp.BaseApp  
❌ **app_config.go** → Module registration in app.go  
❌ **Automatic wiring** → Explicit dependency order  

---

## Current Structure Analysis

### Files in Scope

```
chain/
├── app/
│   ├── app.go              ⚠️  MAJOR REWRITE (374 lines → ~800 lines)
│   ├── app_config.go       ⚠️  WILL BE REMOVED (depinject config)
│   └── export.go           ✅  KEEP (no changes needed)
├── cmd/
│   └── mirrorvaultd/
│       ├── cmd/
│       │   ├── root.go     ⚠️  MINOR UPDATE (remove depinject)
│       │   ├── config.go   ✅  KEEP (JSON-RPC already configured)
│       │   └── commands.go ✅  KEEP (CLI commands unchanged)
│       └── main.go         ✅  KEEP (entry point unchanged)
├── go.mod                  ✅  KEEP (dependencies correct)
├── go.sum                  ✅  KEEP (checksums valid)
└── testutil/               ✅  KEEP (test utilities unchanged)
```

### Current App Struct (Depinject Pattern)

```go
// chain/app/app.go (current)
type App struct {
    *runtime.App              // ← Depinject wrapper (REPLACED)
    legacyAmino       *codec.LegacyAmino
    appCodec          codec.Codec
    txConfig          client.TxConfig
    interfaceRegistry codectypes.InterfaceRegistry

    // Keepers (injected by depinject)
    AuthKeeper            authkeeper.AccountKeeper
    BankKeeper            bankkeeper.Keeper
    StakingKeeper         *stakingkeeper.Keeper
    DistrKeeper           distrkeeper.Keeper
    ConsensusParamsKeeper consensuskeeper.Keeper

    // EVM keepers (manually initialized post-build)
    FeeMarketKeeper   feemarketkeeper.Keeper
    PreciseBankKeeper precisebankkeeper.Keeper
    EVMKeeper         *evmkeeper.Keeper
    Erc20Keeper       erc20keeper.Keeper

    sm *module.SimulationManager
}
```

**Issue**: `runtime.App` hides store key management and keeper initialization timing. EVM keepers initialized **after** `appBuilder.Build()` which causes:
- Cannot register CustomGetSigner before codec initialization
- No control over module registration order
- Cannot integrate EVM into begin/end blockers

---

## Target Structure (EVMD Pattern)

### Target App Struct (Manual Pattern)

```go
// chain/app/app.go (target)
type App struct {
    *baseapp.BaseApp          // ← Direct BaseApp (EXPLICIT CONTROL)

    legacyAmino       *codec.LegacyAmino
    appCodec          codec.Codec
    txConfig          client.TxConfig
    interfaceRegistry codectypes.InterfaceRegistry

    // Store keys (EXPLICIT - not hidden)
    keys    map[string]*storetypes.KVStoreKey
    tkeys   map[string]*storetypes.TransientStoreKey
    memKeys map[string]*storetypes.MemoryStoreKey

    // Standard SDK keepers (same as before)
    AccountKeeper         authkeeper.AccountKeeper
    BankKeeper            bankkeeper.Keeper
    StakingKeeper         *stakingkeeper.Keeper
    DistrKeeper           distrkeeper.Keeper
    ConsensusParamsKeeper consensuskeeper.Keeper

    // EVM keepers (NOW INTEGRATED PROPERLY)
    FeeMarketKeeper   feemarketkeeper.Keeper
    PreciseBankKeeper precisebankkeeper.Keeper
    EVMKeeper         *evmkeeper.Keeper
    Erc20Keeper       erc20keeper.Keeper

    // Module management (EXPLICIT)
    ModuleManager      *module.Manager
    BasicModuleManager module.BasicManager

    sm          *module.SimulationManager
    configurator module.Configurator
}
```

**Benefits**:
- ✅ Full control over store key creation and mounting
- ✅ Register CustomGetSigner BEFORE codec creation
- ✅ Initialize EVM keepers in proper dependency order
- ✅ Integrate EVM modules into begin/end blockers
- ✅ Control ante handler chain construction
- ✅ Explicit module registration order

---

## Migration Strategy: Component-by-Component

### Phase 1: Store Keys (Manual Management)

**Current** (Hidden in runtime.App):
```go
// Store keys created automatically by depinject
app.App = appBuilder.Build(db, traceStore, baseAppOptions...)
```

**Target** (Explicit Creation):
```go
// Create all store keys explicitly
keys := storetypes.NewKVStoreKeys(
    authtypes.StoreKey,
    banktypes.StoreKey,
    stakingtypes.StoreKey,
    distrtypes.StoreKey,
    consensusparamtypes.StoreKey,
    // EVM keys integrated naturally
    evmtypes.StoreKey,
    feemarkettypes.StoreKey,
    erc20types.StoreKey,
    precisebanktypes.StoreKey,
)

tkeys := storetypes.NewTransientStoreKeys(
    evmtypes.TransientKey,
    feemarkettypes.TransientKey,
)

memKeys := storetypes.NewMemoryStoreKeys(
    // Memory store keys if needed
)

app.keys = keys
app.tkeys = tkeys
app.memKeys = memKeys
```

**What Stays**: Key names unchanged (backward compatibility)  
**What Changes**: Explicit creation timing  
**Migration Time**: 15 minutes

---

### Phase 2: BaseApp Initialization

**Current** (Wrapped):
```go
app.App = appBuilder.Build(db, traceStore, baseAppOptions...)
```

**Target** (Direct):
```go
bApp := baseapp.NewBaseApp(
    Name,
    logger,
    db,
    encodingConfig.TxConfig.TxDecoder(),
    baseAppOptions...,
)

app.BaseApp = bApp
app.SetCommitMultiStoreTracer(traceStore)

// Mount stores
for _, key := range keys {
    bApp.MountStore(key, storetypes.StoreTypeDB)
}
for _, tkey := range tkeys {
    bApp.MountStore(tkey, storetypes.StoreTypeTransient)
}
for _, memkey := range memKeys {
    bApp.MountStore(memkey, storetypes.StoreTypeMemory)
}
```

**What Stays**: BaseApp options (optimistic execution, etc.)  
**What Changes**: Explicit store mounting  
**Migration Time**: 20 minutes

---

### Phase 3: Codec & Interface Registry

**Current** (Depinject-provided):
```go
var appCodec codec.Codec
var legacyAmino *codec.LegacyAmino
var txConfig client.TxConfig
var interfaceRegistry codectypes.InterfaceRegistry

depinject.Inject(appConfig,
    &appCodec,
    &legacyAmino,
    &txConfig,
    &interfaceRegistry,
)
```

**Target** (Manual with CustomGetSigner):
```go
// Create encoding config with custom signers
encodingConfig := evmosencoding.MakeConfig(
    module.NewBasicManager(/* modules */),
    []signingtypes.CustomGetSigner{
        evmtypes.MsgEthereumTxCustomGetSigner, // ← THE KEY FIX
    },
)

app.appCodec = encodingConfig.Codec
app.legacyAmino = encodingConfig.Amino
app.txConfig = encodingConfig.TxConfig
app.interfaceRegistry = encodingConfig.InterfaceRegistry
```

**What Stays**: Codec interfaces (amino, protobuf compatibility)  
**What Changes**: CustomGetSigner registered BEFORE keeper init  
**Migration Time**: 30 minutes  
**Critical**: This is WHERE the depinject blocker is solved

---

### Phase 4: Keeper Initialization (Dependency Order)

**Current** (Partial automatic, partial manual):
```go
// Depinject injects some
depinject.Inject(appConfig,
    &app.AuthKeeper,
    &app.BankKeeper,
    // ...
)

// Manually initialize EVM keepers AFTER build
app.FeeMarketKeeper = feemarketkeeper.NewKeeper(...)
app.PreciseBankKeeper = precisebankkeeper.NewKeeper(...)
app.EVMKeeper = evmkeeper.NewKeeper(...)
app.Erc20Keeper = erc20keeper.NewKeeper(...)
```

**Target** (All manual, explicit order):
```go
// 1. Account Keeper (no dependencies)
app.AccountKeeper = authkeeper.NewAccountKeeper(
    app.appCodec,
    runtime.NewKVStoreService(keys[authtypes.StoreKey]),
    authtypes.ProtoBaseAccount,
    maccPerms,
    authcodec.NewBech32Codec(AccountAddressPrefix),
    AccountAddressPrefix,
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
)

// 2. Bank Keeper (depends on AccountKeeper)
app.BankKeeper = bankkeeper.NewBaseKeeper(
    app.appCodec,
    runtime.NewKVStoreService(keys[banktypes.StoreKey]),
    app.AccountKeeper,
    BlockedAddresses(),
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
    logger,
)

// 3. ConsensusParams Keeper (no dependencies)
app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
    app.appCodec,
    runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]),
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
    runtime.EventService{},
)

// 4. Staking Keeper (depends on AccountKeeper, BankKeeper)
app.StakingKeeper = stakingkeeper.NewKeeper(
    app.appCodec,
    runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
    app.AccountKeeper,
    app.BankKeeper,
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
    authcodec.NewBech32Codec(AccountAddressPrefix+"valoper"),
    authcodec.NewBech32Codec(AccountAddressPrefix+"valcons"),
)

// 5. Distribution Keeper (depends on multiple)
app.DistrKeeper = distrkeeper.NewKeeper(
    app.appCodec,
    runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
    app.AccountKeeper,
    app.BankKeeper,
    app.StakingKeeper,
    authtypes.FeeCollectorName,
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
)

// 6. FeeMarket Keeper (EVM - no dependencies)
app.FeeMarketKeeper = feemarketkeeper.NewKeeper(
    app.appCodec,
    authtypes.NewModuleAddress(govtypes.ModuleName),
    keys[feemarkettypes.StoreKey],
    tkeys[feemarkettypes.TransientKey],
)

// 7. PreciseBank Keeper (EVM - depends on BankKeeper, AccountKeeper)
app.PreciseBankKeeper = precisebankkeeper.NewKeeper(
    app.appCodec,
    keys[precisebanktypes.StoreKey],
    app.BankKeeper,
    app.AccountKeeper,
)

// 8. EVM Keeper (depends on multiple)
app.EVMKeeper = evmkeeper.NewKeeper(
    app.appCodec,
    keys[evmtypes.StoreKey],
    tkeys[evmtypes.TransientKey],
    keys, // All keys for precompile access
    authtypes.NewModuleAddress(govtypes.ModuleName),
    app.AccountKeeper,
    app.PreciseBankKeeper,
    app.StakingKeeper,
    &app.FeeMarketKeeper,
    &app.ConsensusParamsKeeper,
    nil, // Erc20Keeper set after creation
    evmChainID,
    tracer,
)

// 9. ERC20 Keeper (depends on EVMKeeper)
app.Erc20Keeper = erc20keeper.NewKeeper(
    keys[erc20types.StoreKey],
    app.appCodec,
    authtypes.NewModuleAddress(govtypes.ModuleName),
    app.AccountKeeper,
    app.BankKeeper,
    app.EVMKeeper,
    app.StakingKeeper,
    nil, // IBC TransferKeeper - will add in Phase 2 (IBC integration)
)

// Set circular reference
app.EVMKeeper.WithErc20Keeper(&app.Erc20Keeper)
```

**Dependency Graph**:
```
AccountKeeper (root)
├─→ BankKeeper
│   ├─→ PreciseBankKeeper (EVM)
│   └─→ StakingKeeper
│       ├─→ DistrKeeper
│       ├─→ EVMKeeper (EVM)
│       └─→ Erc20Keeper (EVM)
├─→ ConsensusParamsKeeper
│   └─→ EVMKeeper (EVM)
└─→ FeeMarketKeeper (EVM, independent)
```

**What Stays**: Keeper logic and APIs unchanged  
**What Changes**: Initialization timing and order control  
**Migration Time**: 2 hours  
**Critical**: Dependency order must be precise

---

### Phase 5: Module Manager & Routing

**Current** (Modules from depinject):
```go
var appModules map[string]appmodule.AppModule
depinject.Inject(appConfig, &appModules)

// EVM modules created but not registered
_ = vm.NewAppModule(...)
_ = feemarket.NewAppModule(...)
_ = erc20.NewAppModule(...)
_ = precisebank.NewAppModule(...)
```

**Target** (All modules explicit):
```go
// Create all modules
modules := []module.AppModule{
    // Standard SDK modules
    auth.NewAppModule(app.appCodec, app.AccountKeeper, nil, app.GetSubspace(authtypes.ModuleName)),
    bank.NewAppModule(app.appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
    staking.NewAppModule(app.appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
    distr.NewAppModule(app.appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
    consensus.NewAppModule(app.appCodec, app.ConsensusParamsKeeper),
    genutil.NewAppModule(app.AccountKeeper, app.StakingKeeper, app, encodingConfig.TxConfig),
    
    // Cosmos EVM modules (NOW INTEGRATED)
    vm.NewAppModule(app.EVMKeeper, app.AccountKeeper, app.BankKeeper, app.AccountKeeper.AddressCodec()),
    feemarket.NewAppModule(app.FeeMarketKeeper),
    erc20.NewAppModule(app.Erc20Keeper, app.AccountKeeper),
    precisebank.NewAppModule(app.PreciseBankKeeper, app.BankKeeper, app.AccountKeeper),
}

// Module Manager with explicit begin/end blockers
app.ModuleManager = module.NewManager(modules...)

// Set begin/end blocker order
app.ModuleManager.SetOrderBeginBlockers(
    distrtypes.ModuleName,
    stakingtypes.ModuleName,
    // EVM modules
    feemarkettypes.ModuleName, // Update base fee
    evmtypes.ModuleName,       // EVM-specific begin block logic
)

app.ModuleManager.SetOrderEndBlockers(
    stakingtypes.ModuleName,
    evmtypes.ModuleName, // EVM-specific end block logic
)

app.ModuleManager.SetOrderInitGenesis(
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
)

// Register services
app.ModuleManager.RegisterServices(module.NewConfigurator(
    app.appCodec,
    app.MsgServiceRouter(),
    app.GRPCQueryRouter(),
))
```

**What Stays**: Module logic unchanged  
**What Changes**: Explicit module lifecycle control  
**Migration Time**: 1 hour  
**Critical**: Begin/end blocker order affects consensus

---

### Phase 6: Ante Handler & Mempool

**Current** (Default ante handler):
```go
// BaseApp uses default ante handler from depinject
```

**Target** (EVM-aware ante handler):
```go
// Create ante handler chain with EVM support
anteHandler, err := chainante.NewAnteHandler(
    &evmante.AnteHandlerOptions{
        Cdc:                    app.appCodec,
        AccountKeeper:          app.AccountKeeper,
        BankKeeper:             app.BankKeeper,
        ExtensionOptionChecker: nil,
        FeegrantKeeper:         nil,
        SignModeHandler:        encodingConfig.TxConfig.SignModeHandler(),
        SigGasConsumer:         evmante.SigVerificationGasConsumer,
        EVMKeeper:              app.EVMKeeper,
        FeeMarketKeeper:        &app.FeeMarketKeeper,
        MaxTxGasWanted:         0, // No limit
        TxFeeChecker:           nil,
    },
)
if err != nil {
    panic(fmt.Errorf("failed to create ante handler: %w", err))
}

app.SetAnteHandler(anteHandler)
```

**What This Enables**:
- ✅ MsgEthereumTx authentication via ECDSA signature
- ✅ EIP-1559 dynamic fee validation
- ✅ EVM-specific gas metering
- ✅ Unified mempool for both Cosmos and Ethereum transactions

**Migration Time**: 30 minutes

---

### Phase 7: Post Handler (Optional Security)

**Target** (Add post handler for additional security):
```go
// Post handler for additional checks after execution
postHandler, err := posthandler.NewPostHandler(
    posthandler.HandlerOptions{},
)
if err != nil {
    panic(err)
}

app.SetPostHandler(postHandler)
```

**What This Provides**:
- ✅ Post-execution validation
- ✅ Additional security checks
- ✅ Gas refund handling

**Migration Time**: 15 minutes

---

## Removed Components

### 1. app_config.go (Entire File Removed)

**Why**: Depinject configuration no longer needed with manual wiring.

**What it contained**:
- Module configuration protobuf (appv1alpha1.Config)
- Module account permissions
- Begin/end blocker order
- InitGenesis order

**Where it goes**:
- Module permissions → `maccPerms` variable in app.go
- Module order → `ModuleManager.SetOrder*()` calls
- Module config → Direct module instantiation

**Migration**: Copy constants, delete file

---

### 2. Depinject Calls in root.go

**Current** (root.go):
```go
// Client context initialization with depinject
var clientCtx client.Context
depinject.Inject(appConfig, &clientCtx)
```

**Target**:
```go
// Manual client context creation
initClientCtx := client.Context{}.
    WithCodec(encodingConfig.Codec).
    WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
    WithTxConfig(encodingConfig.TxConfig).
    WithLegacyAmino(encodingConfig.Amino).
    WithInput(os.Stdin).
    WithAccountRetriever(authtypes.AccountRetriever{}).
    WithBroadcastMode(flags.BroadcastSync).
    WithHomeDir(DefaultNodeHome).
    WithViper(Name)
```

**Migration Time**: 20 minutes

---

## Security & Best Practices

### 1. Module Account Permissions

**Critical**: Module accounts must have correct permissions for security.

```go
var maccPerms = map[string][]string{
    authtypes.FeeCollectorName:     nil,
    distrtypes.ModuleName:          nil,
    stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
    stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
    
    // EVM module permissions (SECURITY CRITICAL)
    evmtypes.ModuleName:         {authtypes.Minter, authtypes.Burner}, // EVM mints/burns for bridges
    feemarkettypes.ModuleName:   nil,                                  // Fee market doesn't hold funds
    erc20types.ModuleName:       {authtypes.Minter, authtypes.Burner}, // ERC20 conversion mints/burns
    precisebanktypes.ModuleName: {authtypes.Minter, authtypes.Burner}, // Precision adjustment mints/burns
}
```

**Why Critical**:
- Minter permission without proper checks = inflation attack
- Burner permission without validation = fund destruction
- Wrong module authority = governance bypass

---

### 2. Blocked Addresses

**Critical**: Prevent transfers to module accounts.

```go
func BlockedAddresses() map[string]bool {
    blockedAddrs := make(map[string]bool)
    for acc := range GetMaccPerms() {
        addr := authtypes.NewModuleAddress(acc)
        blockedAddrs[addr.String()] = true
    }
    
    // Additional blocked addresses
    blockedAddrs[authtypes.NewModuleAddress(govtypes.ModuleName).String()] = false // Gov CAN receive
    
    return blockedAddrs
}
```

---

### 3. Consensus Parameter Validation

**Critical**: Ensure consensus params don't break chain.

```go
// In InitChainer
app.SetInitChainer(func(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
    var genesisState GenesisState
    if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
        return nil, err
    }
    
    // Validate consensus params BEFORE applying
    if req.ConsensusParams != nil {
        if err := validateConsensusParams(req.ConsensusParams); err != nil {
            return nil, fmt.Errorf("invalid consensus params: %w", err)
        }
    }
    
    return app.ModuleManager.InitGenesis(ctx, app.appCodec, genesisState)
})
```

---

### 4. Store Key Collision Prevention

**Critical**: Ensure no duplicate store keys.

```go
// Validate no key collisions
func validateStoreKeys(keys map[string]*storetypes.KVStoreKey) error {
    seen := make(map[string]bool)
    for name, key := range keys {
        if key == nil {
            return fmt.Errorf("nil store key for %s", name)
        }
        if seen[key.Name()] {
            return fmt.Errorf("duplicate store key: %s", key.Name())
        }
        seen[key.Name()] = true
    }
    return nil
}
```

---

### 5. Circular Dependency Handling

**Pattern**: Some keepers have circular dependencies.

```go
// Example: EVMKeeper needs Erc20Keeper, but Erc20Keeper needs EVMKeeper

// 1. Initialize EVM with nil Erc20Keeper
app.EVMKeeper = evmkeeper.NewKeeper(
    // ... params ...
    nil, // Erc20Keeper placeholder
    // ... params ...
)

// 2. Initialize Erc20 with EVMKeeper
app.Erc20Keeper = erc20keeper.NewKeeper(
    // ... params ...
    app.EVMKeeper,
    // ... params ...
)

// 3. Set circular reference
app.EVMKeeper.WithErc20Keeper(&app.Erc20Keeper)
```

**Why Safe**: Keepers don't use each other during initialization, only during block execution.

---

## Future Compatibility

### 1. Adding New Modules (e.g., x/vault in Phase 3)

**Pattern**:
```go
// 1. Add store key
keys := storetypes.NewKVStoreKeys(
    // ... existing keys ...
    vaulttypes.StoreKey, // ← NEW
)

// 2. Initialize keeper in dependency order
app.VaultKeeper = vaultkeeper.NewKeeper(
    app.appCodec,
    keys[vaulttypes.StoreKey],
    app.AccountKeeper,
    app.BankKeeper,
    app.EVMKeeper, // Can depend on EVM!
)

// 3. Add module to manager
modules := []module.AppModule{
    // ... existing modules ...
    vault.NewAppModule(app.VaultKeeper, app.AccountKeeper),
}

// 4. Add to begin/end blockers if needed
app.ModuleManager.SetOrderEndBlockers(
    // ... existing ...
    vaulttypes.ModuleName,
)

// 5. Add to init genesis order
app.ModuleManager.SetOrderInitGenesis(
    // ... existing ...
    vaulttypes.ModuleName, // Before genutiltypes!
    genutiltypes.ModuleName,
)
```

---

### 2. Adding Stateful Precompiles (Phase 3)

**Pattern**:
```go
// In EVMKeeper initialization
app.EVMKeeper = evmkeeper.NewKeeper(
    // ... params ...
)

// Register stateful precompile AFTER keeper initialization
app.EVMKeeper.WithPrecompiledContracts(
    vm.NewDefaultPrecompiledContracts(), // Default EVM precompiles
    vm.NewStatePrecompile(
        common.HexToAddress("0x0000000000000000000000000000000000000101"),
        vaultPrecompile.NewPrecompile(app.VaultKeeper), // Custom precompile
    ),
)
```

---

### 3. IBC Integration (Phase 2)

**Pattern** (Already structured for future IBC):
```go
// Placeholder in current code
app.Erc20Keeper = erc20keeper.NewKeeper(
    // ... params ...
    nil, // ← TransferKeeper placeholder
)

// When adding IBC in Phase 2:
// 1. Add IBC keeper
app.IBCKeeper = ibckeeper.NewKeeper(...)

// 2. Add Transfer keeper
app.TransferKeeper = transferkeeper.NewKeeper(...)

// 3. Update Erc20 keeper
app.Erc20Keeper = erc20keeper.NewKeeper(
    // ... params ...
    app.TransferKeeper, // ← NOW PROVIDED
)
```

---

## Migration Execution Order

### Critical Path (Must Be Done in This Order)

1. **Backup Current Code** (5 min)
   ```bash
   git checkout -b backup/depinject-working
   git push origin backup/depinject-working
   git checkout -b feature/manual-wiring
   ```

2. **Create maccPerms Variable** (10 min)
   - Extract from app_config.go module account permissions
   - Add to app.go as package-level variable

3. **Create Encoding Config** (30 min)
   - Create `MakeEncodingConfig()` function
   - Register CustomGetSigner ✅ **SOLVES BLOCKER**
   - Test codec creation independently

4. **Rewrite App Struct** (20 min)
   - Replace `*runtime.App` with `*baseapp.BaseApp`
   - Add explicit store key maps
   - Add ModuleManager and BasicModuleManager

5. **Rewrite New() Function** (3 hours)
   - Create store keys
   - Initialize BaseApp
   - Mount stores
   - Create encoding config
   - Initialize keepers in dependency order
   - Create modules
   - Create ModuleManager
   - Set begin/end blocker order
   - Set ante handler
   - Register routes

6. **Update root.go** (20 min)
   - Remove depinject from client context
   - Manual client context creation

7. **Delete app_config.go** (1 min)
   - Remove file
   - Update imports

8. **Test Compilation** (10 min)
   ```bash
   cd chain && go build ./cmd/mirrorvaultd
   ```

9. **Test Genesis Init** (15 min)
   ```bash
   rm -rf ~/.mirrorvault-mvlt
   mirrorvaultd init test --chain-id mirror-vault-localnet
   mirrorvaultd genesis add-genesis-account $(mirrorvaultd keys show alice -a) 1000000000umvlt
   mirrorvaultd genesis gentx alice 1000000umvlt --chain-id mirror-vault-localnet
   mirrorvaultd genesis collect-gentxs
   ```

10. **Test Chain Start** (15 min)
    ```bash
    mirrorvaultd start --evm.chain-id 7777
    # Verify: Blocks producing, no panics
    ```

11. **Test JSON-RPC** (15 min)
    ```bash
    curl -X POST -H "Content-Type: application/json" \
      --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
      http://localhost:8545
    # Expected: {"jsonrpc":"2.0","id":1,"result":"0x1e61"}
    ```

12. **Test MetaMask Connection** (15 min)
    - Add network: RPC http://localhost:8545, Chain ID 7777
    - Import account with known private key
    - Verify balance shows correctly
    - Send test transaction

**Total Estimated Time**: 5-6 hours (conservative estimate)

---

## Testing & Validation Checklist

### Compilation Tests
- [ ] `go build ./cmd/mirrorvaultd` succeeds
- [ ] No import errors
- [ ] No undefined references
- [ ] Binary size reasonable (~100-150 MB)

### Genesis Tests
- [ ] `mirrorvaultd init` creates config
- [ ] `add-genesis-account` adds accounts
- [ ] `gentx` creates validator
- [ ] `collect-gentxs` aggregates transactions
- [ ] Genesis file valid JSON
- [ ] EVM genesis state present

### Runtime Tests
- [ ] Chain starts without panic
- [ ] Blocks produce (check logs)
- [ ] REST API responds (curl localhost:1317)
- [ ] RPC responds (curl localhost:26657)
- [ ] JSON-RPC responds (curl localhost:8545)

### Cosmos Transaction Tests
- [ ] Bank send (alice → bob)
- [ ] Query balance
- [ ] Delegation to validator
- [ ] Query staking info

### EVM Tests
- [ ] `eth_chainId` returns 7777
- [ ] `eth_blockNumber` returns current height
- [ ] `eth_getBalance` for 0x address
- [ ] MetaMask connects successfully
- [ ] MetaMask shows correct chain ID
- [ ] Send transaction via MetaMask (succeeds)

### Integration Tests
- [ ] Same account accessible via Keplr and MetaMask
- [ ] Balance updates visible in both wallets
- [ ] Cosmos tx affects EVM balance
- [ ] EVM tx affects Cosmos balance

---

## Rollback Plan

If migration fails critically:

```bash
# 1. Return to backup branch
git checkout backup/depinject-working

# 2. Rebuild binary
cd chain && go build ./cmd/mirrorvaultd

# 3. Restart chain with old binary
rm -rf ~/.mirrorvault-mvlt
./build/mirrorvaultd init test --chain-id mirror-vault-localnet
# ... re-genesis ...
./build/mirrorvaultd start

# 4. Verify working state
curl localhost:26657/status
```

**State Preservation**: 
- Genesis exports ARE compatible (store keys unchanged)
- If chain was already running production, can export state before migration
- Manual wiring doesn't change state machine, only initialization

---

## Documentation Updates Needed

After successful migration:

1. **Update README.md**:
   - Build instructions unchanged
   - Add note about manual wiring architecture

2. **Update IMPLEMENTATION.md**:
   - Mark Phase 1 (EVM Integration) as complete
   - Update status from "depinject blocked" to "manual wiring operational"

3. **Update PROJECT_STATE.md**:
   - Move EVM integration from "Not Started" to "Completed"
   - Update architecture section

4. **Create ARCHITECTURE.md** (NEW):
   - Document manual wiring pattern
   - Keeper dependency diagram
   - Module initialization order
   - Future module addition guide

5. **Update dev-flow.md**:
   - Add manual wiring development workflow
   - How to add new keepers
   - How to modify module order

---

## Reference: EVMD vs Mirror Vault Mapping

| EVMD Component | Mirror Vault Equivalent | Notes |
|----------------|-------------------------|-------|
| `type EVMD struct` | `type App struct` | Structure identical |
| IBC keepers | Not yet (Phase 2) | Placeholders ready |
| Mint module | Not included | Not needed for v1 |
| Slashing module | Not included | Simple validator set for v1 |
| Gov module | Not included | Can add later |
| Authz module | Not included | Can add later |
| Feegrant module | Not included | Can add later |
| Evidence module | Not included | Can add later |
| Params module | Not included | Using ConsensusParams instead |
| x/vault | Not yet (Phase 3) | Custom module coming |

**Module Count**:
- EVMD: 20 modules (full featured)
- Mirror Vault v1: 9 modules (minimal viable + EVM)

**Why Smaller**:
- Focus on core + EVM for v1
- Governance/slashing not critical for initial testnet
- Can add modules incrementally without breaking changes

---

## Questions & Answers

### Q: Can we go back to depinject if cosmos/evm v2 supports it?
**A**: Yes! Since we're not changing store keys or module logic, if cosmos/evm eventually adds proper depinject support, we can migrate back. The state machine is identical.

### Q: Will this break existing test accounts?
**A**: No. Accounts are stored by address, which doesn't change. Existing keys work identically.

### Q: Can Ignite CLI still be used after migration?
**A**: Yes for:
- Protobuf generation (`ignite generate proto`)
- Running relayers
- Testing tools

No for:
- `ignite scaffold module` (but we can still manually create modules in x/)
- `ignite chain serve` (use `mirrorvaultd start` directly)

### Q: What if we need to debug keeper initialization?
**A**: Manual wiring actually makes this EASIER:
- Add print statements in New()
- See exact initialization order
- No hidden depinject container
- Stack traces show real call path

### Q: Is this pattern stable for production?
**A**: Yes. Major production chains use this:
- Osmosis: Manual wiring since v1
- Celestia: Manual wiring by design
- dYdX v4: Manual wiring for custom sequencer
- Sei: Manual wiring for parallel execution
- Evmos: Manual wiring (this is their reference!)

---

## Conclusion

This migration is:
- ✅ **Necessary**: Only path to working EVM integration
- ✅ **Safe**: Proven pattern used by major chains
- ✅ **Reversible**: Can export/import state
- ✅ **Maintainable**: Explicit > implicit for complex systems
- ✅ **Future-compatible**: Easy to add modules (x/vault, IBC, etc.)

**Estimated Total Effort**: 5-7 hours
- Planning (complete): 2 hours
- Implementation: 3-4 hours
- Testing: 1-1.5 hours
- Documentation: 30 minutes

**Next Step**: Await approval to begin implementation.

---

**Prepared by**: GitHub Copilot  
**Reviewed**: Pending  
**Approved**: Pending  
**Status**: 📋 Planning Complete → ⏳ Awaiting Implementation Approval
