package ante

import (
	evmante "github.com/cosmos/evm/ante/evm"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// newMonoEVMAnteHandler creates the sdk.AnteHandler implementation for the EVM transactions.
func newMonoEVMAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		// IMPORTANT: Use keeper params (genesis/runtime), not defaults.
		// Defaults caused wallets sending maxFeePerGas=0 to fail even when genesis disabled base fee.
		evmParams := options.EvmKeeper.GetParams(ctx)
		feeMarketParams := options.FeeMarketKeeper.GetParams(ctx)

		handler := sdk.ChainAnteDecorators(
			evmante.NewEVMMonoDecorator(
				options.AccountKeeper,
				options.FeeMarketKeeper,
				options.EvmKeeper,
				options.MaxTxGasWanted,
				&evmParams,
				&feeMarketParams,
			),
			// Dual address indexing - emit both EVM and Cosmos formats
			NewDualAddressDecorator(options.AccountKeeper, options.Bech32Prefix),
		)

		return handler(ctx, tx, simulate)
	}
}
