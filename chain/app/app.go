package app

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cast"

	abci "github.com/cometbft/cometbft/abci/types"

	clienthelpers "cosmossdk.io/client/v2/helpers"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	// Standard SDK modules
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	// Cosmos EVM imports
	chainante "mirrorvault/ante"

	evmosencoding "github.com/cosmos/evm/encoding"
	srvflags "github.com/cosmos/evm/server/flags"
	"github.com/cosmos/evm/x/erc20"
	erc20keeper "github.com/cosmos/evm/x/erc20/keeper"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	"github.com/cosmos/evm/x/feemarket"
	feemarketkeeper "github.com/cosmos/evm/x/feemarket/keeper"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	"github.com/cosmos/evm/x/precisebank"
	precisebankkeeper "github.com/cosmos/evm/x/precisebank/keeper"
	precisebanktypes "github.com/cosmos/evm/x/precisebank/types"
	"github.com/cosmos/evm/x/vm"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"mirrorvault/docs"
)

const (
	// Name is the name of the application.
	Name = "mirrorvault"
	// AccountAddressPrefix is the prefix for accounts addresses.
	AccountAddressPrefix = "mirror"
	// ChainCoinType is the coin type of the chain.
	ChainCoinType = 60
)

// DefaultNodeHome default home directories for the application daemon
var DefaultNodeHome string

var (
	// Module account permissions
	// NOTE: EVM modules have minter/burner permissions for bridges and conversions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		stakingtypes.BondedPoolName:    {authtypes.Burner, stakingtypes.ModuleName},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, stakingtypes.ModuleName},
		// Cosmos EVM modules
		evmtypes.ModuleName:         {authtypes.Minter, authtypes.Burner}, // EVM mints/burns for bridges
		feemarkettypes.ModuleName:   nil,                                  // Fee market doesn't hold funds
		erc20types.ModuleName:       {authtypes.Minter, authtypes.Burner}, // ERC20 conversion mints/burns
		precisebanktypes.ModuleName: {authtypes.Minter, authtypes.Burner}, // Precision adjustment mints/burns
	}

	// Blocked account addresses
	blockAccAddrs = []string{
		authtypes.FeeCollectorName,
		distrtypes.ModuleName,
		stakingtypes.BondedPoolName,
		stakingtypes.NotBondedPoolName,
		// Note: We intentionally don't block gov module (when added Phase 2)
	}
)

// App implements servertypes.Application
var _ servertypes.Application = (*App)(nil)

// App extends the baseapp.BaseApp with custom keepers and module management
type App struct {
	*baseapp.BaseApp

	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	// Store keys
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// Standard SDK keepers
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	ConsensusParamsKeeper consensuskeeper.Keeper

	// Cosmos EVM keepers
	FeeMarketKeeper   feemarketkeeper.Keeper
	PreciseBankKeeper precisebankkeeper.Keeper
	EVMKeeper         *evmkeeper.Keeper
	Erc20Keeper       erc20keeper.Keeper

	// Module management
	ModuleManager      *module.Manager
	BasicModuleManager module.BasicManager

	// Simulation manager
	sm *module.SimulationManager

	// Module configurator
	configurator module.Configurator
}

func init() {
	var err error
	clienthelpers.EnvPrefix = Name
	DefaultNodeHome, err = clienthelpers.GetNodeHomeDirectory("." + Name)
	if err != nil {
		panic(err)
	}
}

// MakeEncodingConfig creates the encoding configuration with EVM support.
func MakeEncodingConfig() evmosencoding.Config {
	// Use default chain ID for encoding setup (actual chain ID set later in baseapp)
	// The cosmos/evm MakeConfig automatically registers CustomGetSigner for MsgEthereumTx
	encodingConfig := evmosencoding.MakeConfig(7777)

	// Register all module interfaces with the encoding config
	// This is required for genesis commands (add-genesis-account, gentx, etc.)
	moduleBasicManager := GetBasicModuleManager()
	moduleBasicManager.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	return encodingConfig
}

// New returns a reference to an initialized App with manual keeper initialization.
func New(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *App {
	// Create encoding config with EVM support
	encodingConfig := MakeEncodingConfig()

	app := &App{}
	app.legacyAmino = encodingConfig.Amino
	app.appCodec = encodingConfig.Codec
	app.txConfig = encodingConfig.TxConfig
	app.interfaceRegistry = encodingConfig.InterfaceRegistry

	// Get EVM Chain ID from app options
	evmChainID := cast.ToUint64(appOpts.Get(srvflags.EVMChainID))
	if evmChainID == 0 {
		evmChainID = 7777 // default EVM chain ID for mirror-vault
	}

	// Configure EVM coin info (required for precisebank module initialization)
	// This must be done before creating any keepers that depend on EVM coin denomination
	// For 18 decimals: Denom and ExtendedDenom must be the same
	evmConfigurator := evmtypes.NewEVMConfigurator().
		WithEVMCoinInfo(evmtypes.EvmCoinInfo{
			Denom:         "aatom", // 1e-18 of base denom (18 decimals)
			ExtendedDenom: "aatom", // Must match Denom for 18 decimals
			DisplayDenom:  "atom",  // human-readable denomination
			Decimals:      18,      // EVM uses 18 decimals
		})
	if err := evmConfigurator.Configure(); err != nil {
		panic(fmt.Errorf("failed to configure EVM coin info: %w", err))
	}

	// Get tracer from app options
	tracer := cast.ToString(appOpts.Get(srvflags.EVMTracer))

	// Initialize BaseApp
	bApp := baseapp.NewBaseApp(
		Name,
		logger,
		db,
		encodingConfig.TxConfig.TxDecoder(),
		baseAppOptions...,
	)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion("v0.1.0") // App version
	bApp.SetInterfaceRegistry(app.interfaceRegistry)
	bApp.SetTxEncoder(encodingConfig.TxConfig.TxEncoder())

	app.BaseApp = bApp

	// Create store keys
	app.keys = storetypes.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		stakingtypes.StoreKey,
		distrtypes.StoreKey,
		consensustypes.StoreKey,
		// EVM store keys
		evmtypes.StoreKey,
		feemarkettypes.StoreKey,
		erc20types.StoreKey,
		precisebanktypes.StoreKey,
	)

	app.tkeys = storetypes.NewTransientStoreKeys(
		evmtypes.TransientKey,
		feemarkettypes.TransientKey,
	)

	app.memKeys = storetypes.NewMemoryStoreKeys()

	// Mount stores
	for _, key := range app.keys {
		bApp.MountStore(key, storetypes.StoreTypeDB)
	}
	for _, tkey := range app.tkeys {
		bApp.MountStore(tkey, storetypes.StoreTypeTransient)
	}
	for _, memkey := range app.memKeys {
		bApp.MountStore(memkey, storetypes.StoreTypeMemory)
	}

	// Initialize keepers in dependency order
	// Phase 1: Root keepers (no dependencies)

	// Account Keeper
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		app.appCodec,
		runtime.NewKVStoreService(app.keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		authcodec.NewBech32Codec(AccountAddressPrefix),
		AccountAddressPrefix,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Bank Keeper
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		app.appCodec,
		runtime.NewKVStoreService(app.keys[banktypes.StoreKey]),
		app.AccountKeeper,
		BlockedAddresses(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		logger,
	)

	// Consensus Params Keeper
	app.ConsensusParamsKeeper = consensuskeeper.NewKeeper(
		app.appCodec,
		runtime.NewKVStoreService(app.keys[consensustypes.StoreKey]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.EventService{},
	)

	// Set consensus params keeper in baseapp (required for chain startup)
	bApp.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

	// Phase 2: Keepers that depend on Account/Bank

	// Staking Keeper
	app.StakingKeeper = stakingkeeper.NewKeeper(
		app.appCodec,
		runtime.NewKVStoreService(app.keys[stakingtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		authcodec.NewBech32Codec(AccountAddressPrefix+"valoper"),
		authcodec.NewBech32Codec(AccountAddressPrefix+"valcons"),
	)

	// Distribution Keeper
	app.DistrKeeper = distrkeeper.NewKeeper(
		app.appCodec,
		runtime.NewKVStoreService(app.keys[distrtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Phase 3: EVM Keepers

	// FeeMarket Keeper (independent)
	app.FeeMarketKeeper = feemarketkeeper.NewKeeper(
		app.appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		app.keys[feemarkettypes.StoreKey],
		app.tkeys[feemarkettypes.TransientKey],
	)

	// PreciseBank Keeper (depends on Bank, Account)
	app.PreciseBankKeeper = precisebankkeeper.NewKeeper(
		app.appCodec,
		app.keys[precisebanktypes.StoreKey],
		app.BankKeeper,
		app.AccountKeeper,
	)

	// EVM Keeper (depends on many keepers)
	app.EVMKeeper = evmkeeper.NewKeeper(
		app.appCodec,
		app.keys[evmtypes.StoreKey],
		app.tkeys[evmtypes.TransientKey],
		app.keys, // Pass all keys for precompile access
		authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper,
		app.PreciseBankKeeper,
		app.StakingKeeper,
		&app.FeeMarketKeeper,
		&app.ConsensusParamsKeeper,
		nil, // Erc20Keeper will be set after initialization (circular dependency)
		evmChainID,
		tracer,
	)

	// ERC20 Keeper (depends on EVM, Bank, Account, Staking)
	app.Erc20Keeper = erc20keeper.NewKeeper(
		app.keys[erc20types.StoreKey],
		app.appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
		app.EVMKeeper,
		app.StakingKeeper,
		nil, // TransferKeeper - will be added when we integrate IBC (Phase 2)
	)

	// Note: EVMKeeper circular reference with Erc20 Keeper may be set during keeper construction
	// If WithErc20Keeper method exists, uncomment: app.EVMKeeper.WithErc20Keeper(&app.Erc20Keeper)

	// Create modules
	modules := []module.AppModule{
		auth.NewAppModule(app.appCodec, app.AccountKeeper, nil, nil),
		bank.NewAppModule(app.appCodec, app.BankKeeper, app.AccountKeeper, nil),
		staking.NewAppModule(app.appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, nil),
		distr.NewAppModule(app.appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, nil),
		consensus.NewAppModule(app.appCodec, app.ConsensusParamsKeeper),
		genutil.NewAppModule(app.AccountKeeper, app.StakingKeeper, app.BaseApp, encodingConfig.TxConfig),
		feemarket.NewAppModule(app.FeeMarketKeeper),
		erc20.NewAppModule(app.Erc20Keeper, app.AccountKeeper),
		precisebank.NewAppModule(app.PreciseBankKeeper, app.BankKeeper, app.AccountKeeper),
	}

	// Create basic module manager
	app.BasicModuleManager = module.NewBasicManager(
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		distr.AppModuleBasic{},
		genutil.AppModuleBasic{GenTxValidator: genutiltypes.DefaultMessageValidator},
		consensus.AppModuleBasic{},
		// EVM modules
		vm.AppModuleBasic{},
		feemarket.AppModuleBasic{},
		erc20.AppModuleBasic{},
		precisebank.AppModuleBasic{},
	)

	// Create module manager
	app.ModuleManager = module.NewManager(modules...)

	// Set begin/end blocker order
	app.ModuleManager.SetOrderBeginBlockers(
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		// EVM modules
		feemarkettypes.ModuleName, // Update EIP-1559 base fee
		evmtypes.ModuleName,       // EVM begin block logic
	)

	app.ModuleManager.SetOrderEndBlockers(
		stakingtypes.ModuleName,
		feemarkettypes.ModuleName, // Update EIP-1559 base fee
		evmtypes.ModuleName,       // EVM end block logic
	)

	// Set init genesis order
	app.ModuleManager.SetOrderInitGenesis(
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		consensustypes.ModuleName,
		genutiltypes.ModuleName,
		// EVM modules
		feemarkettypes.ModuleName,
		precisebanktypes.ModuleName,
		evmtypes.ModuleName,
		erc20types.ModuleName,
	)

	// Register services
	app.configurator = module.NewConfigurator(
		app.appCodec,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
	)

	// Register interfaces before registering services
	app.BasicModuleManager.RegisterInterfaces(app.interfaceRegistry)

	app.ModuleManager.RegisterServices(app.configurator)

	// Set ante handler with EVM support (custom router from evmd pattern)
	anteHandler := chainante.NewAnteHandler(chainante.HandlerOptions{
		Cdc:                    app.appCodec,
		AccountKeeper:          app.AccountKeeper,
		BankKeeper:             app.BankKeeper,
		ExtensionOptionChecker: nil,
		FeegrantKeeper:         nil,
		SignModeHandler:        encodingConfig.TxConfig.SignModeHandler(),
		SigGasConsumer:         nil, // Use default
		EvmKeeper:              app.EVMKeeper,
		FeeMarketKeeper:        &app.FeeMarketKeeper,
		MaxTxGasWanted:         0, // No limit
		TxFeeChecker:           nil,
	})
	app.SetAnteHandler(anteHandler)

	// Wire up InitChainer, BeginBlocker, and EndBlocker (required for manual wiring)
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)

	// Load latest version
	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			panic(fmt.Errorf("failed to load latest version: %w", err))
		}
	}

	return app
}

// InitChainer handles the initial chain state from genesis.
func (app *App) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState map[string]json.RawMessage
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		return nil, err
	}

	return app.ModuleManager.InitGenesis(ctx, app.appCodec, genesisState)
}

// LegacyAmino returns App's amino codec.
func (app *App) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns App's app codec.
func (app *App) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns App's InterfaceRegistry.
func (app *App) InterfaceRegistry() codectypes.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns App's TxConfig
func (app *App) TxConfig() client.TxConfig {
	return app.txConfig
}

// GetKey returns the KVStoreKey for the provided store key.
func (app *App) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
func (app *App) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemoryStoreKey for the provided store key.
func (app *App) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// RegisterNodeService implements servertypes.Application by registering the node service.
func (app *App) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	// The baseapp already provides node service registration
	// This is a no-op placeholder to satisfy the interface
}

// RegisterTendermintService implements servertypes.Application by registering CometBFT service.
func (app *App) RegisterTendermintService(clientCtx client.Context) {
	// The baseapp already provides Tendermint service registration
	// This is a no-op placeholder to satisfy the interface
}

// RegisterTxService implements servertypes.Application by registering transaction service.
func (app *App) RegisterTxService(clientCtx client.Context) {
	// The baseapp already provides tx service registration
	// This is a no-op placeholder to satisfy the interface
}

// SimulationManager implements the SimulationApp interface
func (app *App) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided API server.
func (app *App) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	// register app's OpenAPI routes
	docs.RegisterOpenAPIService(Name, apiSvr.Router)
}

// GetMaccPerms returns a copy of the module account permissions.
func GetMaccPerms() map[string][]string {
	dup := make(map[string][]string)
	for k, v := range maccPerms {
		dup[k] = v
	}
	return dup
}

// GetBasicModuleManager returns the basic module manager for CLI initialization
func GetBasicModuleManager() module.BasicManager {
	return module.NewBasicManager(
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		distr.AppModuleBasic{},
		genutil.AppModuleBasic{GenTxValidator: genutiltypes.DefaultMessageValidator},
		consensus.AppModuleBasic{},
		// EVM modules
		vm.AppModuleBasic{},
		feemarket.AppModuleBasic{},
		erc20.AppModuleBasic{},
		precisebank.AppModuleBasic{},
	)
}
func BlockedAddresses() map[string]bool {
	result := make(map[string]bool)
	for _, addr := range blockAccAddrs {
		result[addr] = true
	}
	return result
}

// DefaultGenesis returns a default genesis from the registered modules
func (app *App) DefaultGenesis() map[string]json.RawMessage {
	return app.BasicModuleManager.DefaultGenesis(app.appCodec)
}

// BeginBlocker application updates every begin block
func (app *App) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return app.ModuleManager.BeginBlock(ctx)
}

// EndBlocker application updates every end block
func (app *App) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.ModuleManager.EndBlock(ctx)
}

// GetStoreKeys returns all the stored store keys
func (app *App) GetStoreKeys() []storetypes.StoreKey {
	keys := make([]storetypes.StoreKey, 0, len(app.keys))
	for _, key := range app.keys {
		keys = append(keys, key)
	}
	return keys
}
