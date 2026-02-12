package cmd

import (
	cmtcfg "github.com/cometbft/cometbft/config"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	cosmosevmserverconfig "github.com/cosmos/evm/server/config"
)

// initCometBFTConfig helps to override default CometBFT Config values.
// return cmtcfg.DefaultConfig if no custom configuration is required for the application.
func initCometBFTConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()

	// these values put a higher strain on node memory
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40

	return cfg
}

// EVMAppConfig extends the default SDK config with EVM-specific settings
type EVMAppConfig struct {
	serverconfig.Config `mapstructure:",squash"`

	EVM     cosmosevmserverconfig.EVMConfig     `mapstructure:"evm"`
	JSONRPC cosmosevmserverconfig.JSONRPCConfig `mapstructure:"json-rpc"`
	TLS     cosmosevmserverconfig.TLSConfig     `mapstructure:"tls"`
}

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	// Optionally allow the chain developer to overwrite the SDK's default
	// server config.
	srvCfg := serverconfig.DefaultConfig()

	// Set minimum gas prices to 0 for development
	// In production, validators should set this to a non-zero value
	srvCfg.MinGasPrices = "0umvlt"

	// EVM configuration
	evmCfg := cosmosevmserverconfig.DefaultEVMConfig()
	evmCfg.EVMChainID = 7777 // Mirror Vault EVM chain ID

	// JSON-RPC configuration
	jsonrpcCfg := cosmosevmserverconfig.DefaultJSONRPCConfig()
	jsonrpcCfg.Enable = true
	jsonrpcCfg.Address = "0.0.0.0:8545"
	jsonrpcCfg.API = []string{"eth", "net", "web3", "txpool", "debug"}
	// Note: EnableUnsafeCORS was removed in v0.5.0 - use reverse proxy for CORS in production // Enable CORS for local development

	// TLS configuration (disabled for local development)
	tlsCfg := cosmosevmserverconfig.DefaultTLSConfig()

	customAppConfig := EVMAppConfig{
		Config:  *srvCfg,
		EVM:     *evmCfg,
		JSONRPC: *jsonrpcCfg,
		TLS:     *tlsCfg,
	}

	// Extend the default template with EVM sections
	customAppTemplate := serverconfig.DefaultConfigTemplate + cosmosevmserverconfig.DefaultEVMConfigTemplate

	return customAppTemplate, customAppConfig
}
