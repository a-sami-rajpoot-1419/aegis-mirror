package ante

import (
	evmante "github.com/cosmos/evm/ante/evm"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
)

// newCosmosAnteHandler creates a simplified ante handler for Cosmos transactions
// Note: This is Phase 1 - simplified without EIP712, IBC, and Authz decorators
// These advanced features will be added in Phase 2 when we integrate IBC
func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		// Get feemarket params for GasWantedDecorator
		feeMarketParams := feemarkettypes.DefaultParams()

		handler := sdk.ChainAnteDecorators(
			ante.NewSetUpContextDecorator(),
			ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
			ante.NewValidateBasicDecorator(),
			ante.NewTxTimeoutHeightDecorator(),
			ante.NewValidateMemoDecorator(options.AccountKeeper),
			ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
			ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker),
			// SetPubKeyDecorator must be called before all signature verification decorators
			ante.NewSetPubKeyDecorator(options.AccountKeeper),
			ante.NewValidateSigCountDecorator(options.AccountKeeper),
			ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
			ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
			ante.NewIncrementSequenceDecorator(options.AccountKeeper),
			evmante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper, &feeMarketParams),
			// Dual address indexing - emit both EVM and Cosmos formats
			NewDualAddressDecorator(options.AccountKeeper, options.Bech32Prefix),
		)

		return handler(ctx, tx, simulate)
	}
}
