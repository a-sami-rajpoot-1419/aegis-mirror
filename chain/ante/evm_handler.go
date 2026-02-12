package ante

import (
	evmante "github.com/cosmos/evm/ante/evm"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	vmtypes "github.com/cosmos/evm/x/vm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// newMonoEVMAnteHandler creates the sdk.AnteHandler implementation for the EVM transactions.
func newMonoEVMAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		// Get default params
		evmParams := vmtypes.DefaultParams()
		feeMarketParams := feemarkettypes.DefaultParams()
		
		handler := sdk.ChainAnteDecorators(
			evmante.NewEVMMonoDecorator(
				options.AccountKeeper,
				options.FeeMarketKeeper,
				options.EvmKeeper,
				options.MaxTxGasWanted,
				&evmParams,
				&feeMarketParams,
			),
		)
		
		return handler(ctx, tx, simulate)
	}
}
