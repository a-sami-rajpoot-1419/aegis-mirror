package cmd

import (
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/spf13/cobra"

	"mirrorvault/app"
)

// NewRootCmd creates a new root command for mirrorvaultd.
func NewRootCmd() *cobra.Command {
	// Create app encoding config with EVM support
	encodingConfig := app.MakeEncodingConfig()

	// AutoCLI options (minimal for Phase 1) - skip autocli enhancement  to avoid address codec requirement
	// autoCliOpts := autocli.AppOptions{}

	// Client context
	clientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(app.DefaultNodeHome).
		WithViper("")

	rootCmd := &cobra.Command{
		Use:           app.Name + "d",
		Short:         "mirrorvault node",
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			clientCtx = clientCtx.WithCmdContext(cmd.Context()).WithViper(app.Name)
			clientCtx, err := client.ReadPersistentCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			clientCtx, err = config.ReadFromClientConfig(clientCtx)
			if err != nil {
				return err
			}

			if err := client.SetCmdClientContextHandler(clientCtx, cmd); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := initAppConfig()
			customCMTConfig := initCometBFTConfig()

			return server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customCMTConfig)
		},
	}

	// Get basic module manager for init commands (simplified for Phase 1)
	moduleBasicManager := app.GetBasicModuleManager()

	initRootCmd(rootCmd, encodingConfig.TxConfig, moduleBasicManager)

	// Skip AutoCLI enhancement for Phase 1 to avoid address codec requirement
	// if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
	// 	panic(err)
	// }

	return rootCmd
}
