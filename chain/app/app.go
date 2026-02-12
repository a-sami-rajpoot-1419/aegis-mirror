package app

import (
	"io"

	"github.com/spf13/cast"

	clienthelpers "cosmossdk.io/client/v2/helpers"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	// Cosmos EVM imports
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

	srvflags "github.com/cosmos/evm/server/flags"

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
	_ runtime.AppI            = (*App)(nil)
	_ servertypes.Application = (*App)(nil)
)

// App extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type App struct {
	*runtime.App
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	// keepers
	AuthKeeper            authkeeper.AccountKeeper
	BankKeeper            bankkeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	ConsensusParamsKeeper consensuskeeper.Keeper

	// Cosmos EVM keepers
	FeeMarketKeeper   feemarketkeeper.Keeper
	PreciseBankKeeper precisebankkeeper.Keeper
	EVMKeeper         *evmkeeper.Keeper
	Erc20Keeper       erc20keeper.Keeper

	// simulation manager
	sm *module.SimulationManager
}

func init() {
	var err error
	clienthelpers.EnvPrefix = Name
	DefaultNodeHome, err = clienthelpers.GetNodeHomeDirectory("." + Name)
	if err != nil {
		panic(err)
	}
}

// AppConfig returns the default app config.
func AppConfig() depinject.Config {
	return depinject.Configs(
		appConfig,
		depinject.Supply(
			// Supply EVM custom signers globally - needed for MsgEthereumTx
			evmtypes.MsgEthereumTxCustomGetSigner,
			// supply custom module basics
			map[string]module.AppModuleBasic{
				genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
			},
		),
	)
}

// New returns a reference to an initialized App.
func New(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *App {
	var (
		app        = &App{}
		appBuilder *runtime.AppBuilder

		// merge the AppConfig and other configuration in one config
		appConfig = depinject.Configs(
			AppConfig(),
			depinject.Supply(
				appOpts, // supply app options
				logger,  // supply logger
				// here alternative options can be supplied to the DI container.
				// those options can be used f.e to override the default behavior of some modules.
				// for instance supplying a custom address codec for not using bech32 addresses.
				// read the depinject documentation and depinject module wiring for more information
				// on available options and how to use them.
			),
		)
	)

	var appModules map[string]appmodule.AppModule
	if err := depinject.Inject(appConfig,
		&appBuilder,
		&appModules,
		&app.appCodec,
		&app.legacyAmino,
		&app.txConfig,
		&app.interfaceRegistry,
		&app.AuthKeeper,
		&app.BankKeeper,
		&app.StakingKeeper,
		&app.DistrKeeper,
		&app.ConsensusParamsKeeper,
	); err != nil {
		panic(err)
	}

	// Get EVM Chain ID from app options
	evmChainID := cast.ToUint64(appOpts.Get(srvflags.EVMChainID))
	if evmChainID == 0 {
		evmChainID = 7777 // default EVM chain ID for mirror-vault
	}

	// Register EVM store keys manually (cosmos/evm doesn't support depinject yet)
	// Note: cosmos/evm modules use standard KV and transient keys, no object keys
	evmStoreKeys := storetypes.NewKVStoreKeys(
		evmtypes.StoreKey,
		feemarkettypes.StoreKey,
		erc20types.StoreKey,
		precisebanktypes.StoreKey,
	)
	evmTransientKeys := storetypes.NewTransientStoreKeys(
		evmtypes.TransientKey,
		feemarkettypes.TransientKey,
	)

	// add to default baseapp options
	// enable optimistic execution
	baseAppOptions = append(baseAppOptions, baseapp.SetOptimisticExecution())

	// build app
	app.App = appBuilder.Build(db, traceStore, baseAppOptions...)

	// Now initialize EVM keepers after app is built
	// These keepers cannot be initialized via depinject (cosmos/evm doesn't support it yet)

	// Mount EVM store keys to the multistore
	for _, key := range evmStoreKeys {
		app.App.MountStore(key, storetypes.StoreTypeDB)
	}
	for _, key := range evmTransientKeys {
		app.App.MountStore(key, storetypes.StoreTypeTransient)
	}

	// FeeMarket keeper - manages EIP-1559 base fee
	app.FeeMarketKeeper = feemarketkeeper.NewKeeper(
		app.appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		evmStoreKeys[feemarkettypes.StoreKey],
		evmTransientKeys[feemarkettypes.TransientKey],
	)

	// PreciseBank keeper - enables 18-decimal precision for EVM compatibility
	app.PreciseBankKeeper = precisebankkeeper.NewKeeper(
		app.appCodec,
		evmStoreKeys[precisebanktypes.StoreKey],
		app.BankKeeper,
		app.AuthKeeper,
	)

	// Get tracer from app options
	tracer := cast.ToString(appOpts.Get(srvflags.EVMTracer))

	// Collect all store keys for EVM keeper
	// Get keys map from CommitMultiStore
	allKeys := make(map[string]*storetypes.KVStoreKey)
	for key, value := range evmStoreKeys {
		allKeys[key] = value
	}

	// EVM keeper - core execution engine
	app.EVMKeeper = evmkeeper.NewKeeper(
		app.appCodec,
		evmStoreKeys[evmtypes.StoreKey],
		evmTransientKeys[evmtypes.TransientKey],
		allKeys,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AuthKeeper,
		app.PreciseBankKeeper,
		app.StakingKeeper,
		&app.FeeMarketKeeper,
		&app.ConsensusParamsKeeper,
		nil, // Erc20Keeper will be set after initialization
		evmChainID,
		tracer,
	)

	// ERC20 keeper - handles native<->ERC20 conversion
	// NOTE: We need IBC TransferKeeper for erc20 module, but we don't have IBC integrated yet.
	// For now, pass nil - this means IBC-related erc20 features won't work until Phase 2.
	app.Erc20Keeper = erc20keeper.NewKeeper(
		evmStoreKeys[erc20types.StoreKey],
		app.appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AuthKeeper,
		app.BankKeeper,
		app.EVMKeeper,
		app.StakingKeeper,
		nil, // TransferKeeper - will be added when we integrate IBC
	)

	// Create EVM modules for later wiring
	// Note: These modules cannot be registered with depinject ModuleManager
	// They will be handled through custom routing in the BaseApp
	_ = vm.NewAppModule(app.EVMKeeper, app.AuthKeeper, app.BankKeeper, app.AuthKeeper.AddressCodec())
	_ = feemarket.NewAppModule(app.FeeMarketKeeper)
	_ = erc20.NewAppModule(app.Erc20Keeper, app.AuthKeeper)
	_ = precisebank.NewAppModule(app.PreciseBankKeeper, app.BankKeeper, app.AuthKeeper)

	// TODO: Wire EVM modules into msg/query routing
	// For now, keepers are initialized and will be available for AnteHandler and mempool

	/****  Module Options ****/

	// create the simulation manager and define the order of the modules for deterministic simulations
	app.sm = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, make(map[string]module.AppModuleSimulation))
	app.sm.RegisterStoreDecoders()

	// A custom InitChainer can be set if extra pre-init-genesis logic is required.
	// By default, when using app wiring enabled module, this is not required.
	// For instance, the upgrade module will set automatically the module version map in its init genesis thanks to app wiring.
	// However, when registering a module manually (i.e. that does not support app wiring), the module version map
	// must be set manually as follow. The upgrade module will de-duplicate the module version map.
	//
	// app.SetInitChainer(func(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	// 	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.ModuleManager.GetVersionMap())
	// 	return app.App.InitChainer(ctx, req)
	// })

	if err := app.Load(loadLatest); err != nil {
		panic(err)
	}

	return app
}

// LegacyAmino returns App's amino codec.
func (app *App) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns App's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
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
	kvStoreKey, ok := app.UnsafeFindStoreKey(storeKey).(*storetypes.KVStoreKey)
	if !ok {
		return nil
	}
	return kvStoreKey
}

// SimulationManager implements the SimulationApp interface
func (app *App) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *App) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	app.App.RegisterAPIRoutes(apiSvr, apiConfig)
	// register swagger API in app.go so that other applications can override easily
	if err := server.RegisterSwaggerAPI(apiSvr.ClientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
		panic(err)
	}

	// register app's OpenAPI routes.
	docs.RegisterOpenAPIService(Name, apiSvr.Router)
}

// GetMaccPerms returns a copy of the module account permissions
//
// NOTE: This is solely to be used for testing purposes.
func GetMaccPerms() map[string][]string {
	dup := make(map[string][]string)
	for _, perms := range moduleAccPerms {
		dup[perms.GetAccount()] = perms.GetPermissions()
	}

	return dup
}

// BlockedAddresses returns all the app's blocked account addresses.
func BlockedAddresses() map[string]bool {
	result := make(map[string]bool)

	if len(blockAccAddrs) > 0 {
		for _, addr := range blockAccAddrs {
			result[addr] = true
		}
	} else {
		for addr := range GetMaccPerms() {
			result[addr] = true
		}
	}

	return result
}
