package ante

import (
	"fmt"

	anteinterfaces "github.com/cosmos/evm/ante/interfaces"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"mirrorvault/utils"
)

const (
	// Event attribute keys for dual address indexing
	AttributeKeyEVMAddress     = "evm_address"
	AttributeKeyCosmosAddress  = "cosmos_address"
	AttributeKeyDualFormat     = "dual_address"
	AttributeKeySender         = "sender"
	AttributeKeyRecipient      = "recipient"

	// Event types
	EventTypeDualAddress = "dual_address_index"
)

// DualAddressDecorator emits events with both EVM and Cosmos address formats
// This enables blockchain explorers and indexers to query by either format
type DualAddressDecorator struct {
	accountKeeper anteinterfaces.AccountKeeper
	bech32Prefix  string
}

// NewDualAddressDecorator creates a new decorator for dual address indexing
func NewDualAddressDecorator(ak anteinterfaces.AccountKeeper, bech32Prefix string) DualAddressDecorator {
	return DualAddressDecorator{
		accountKeeper: ak,
		bech32Prefix:  bech32Prefix,
	}
}

// AnteHandle emits events with both address formats for transaction indexing
func (dad DualAddressDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// Skip event emission during simulation to save gas
	if simulate {
		return next(ctx, tx, simulate)
	}

	// Extract addresses from transaction
	addresses := utils.ExtractAddressesFromTx(tx)

	// Emit dual address events for each unique address in the transaction
	for idx, addr := range addresses {
		bech32Addr, ethHex, err := utils.SDKAddressToBothFormats(addr, dad.bech32Prefix)
		if err != nil {
			// Log error but don't fail the transaction
			ctx.Logger().Error(
				"failed to convert address to dual format",
				"address", addr.String(),
				"error", err.Error(),
			)
			continue
		}

		// Emit event with both address formats
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				EventTypeDualAddress,
				sdk.NewAttribute(AttributeKeyEVMAddress, ethHex),
				sdk.NewAttribute(AttributeKeyCosmosAddress, bech32Addr),
				sdk.NewAttribute(AttributeKeyDualFormat, fmt.Sprintf("%s (%s)", ethHex, bech32Addr)),
				sdk.NewAttribute("address_index", fmt.Sprintf("%d", idx)),
			),
		)
	}

	// Continue to next decorator
	return next(ctx, tx, simulate)
}

// EmitDualAddressEvent is a helper function to emit dual address events from keepers or modules
// This can be called directly from keeper methods when addresses need to be indexed
func EmitDualAddressEvent(ctx sdk.Context, addr sdk.AccAddress, bech32Prefix string, role string) {
	bech32Addr, ethHex, err := utils.SDKAddressToBothFormats(addr, bech32Prefix)
	if err != nil {
		ctx.Logger().Error(
			"failed to emit dual address event",
			"address", addr.String(),
			"role", role,
			"error", err.Error(),
		)
		return
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			EventTypeDualAddress,
			sdk.NewAttribute(AttributeKeyEVMAddress, ethHex),
			sdk.NewAttribute(AttributeKeyCosmosAddress, bech32Addr),
			sdk.NewAttribute("role", role),
		),
	)
}

// ConvertEthAddressToCosmosEvent emits an event linking an Ethereum transaction to Cosmos address
// This is useful for EVM transactions that also need Cosmos address visibility
func ConvertEthAddressToCosmosEvent(ctx sdk.Context, ethAddr common.Address, bech32Prefix string) {
	bech32Addr, err := utils.EthAddressToBech32(ethAddr.Hex(), bech32Prefix)
	if err != nil {
		ctx.Logger().Error(
			"failed to convert eth address to bech32",
			"eth_address", ethAddr.Hex(),
			"error", err.Error(),
		)
		return
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			EventTypeDualAddress,
			sdk.NewAttribute(AttributeKeyEVMAddress, ethAddr.Hex()),
			sdk.NewAttribute(AttributeKeyCosmosAddress, bech32Addr),
			sdk.NewAttribute("source", "evm_transaction"),
		),
	)
}
