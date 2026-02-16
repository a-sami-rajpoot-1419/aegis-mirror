package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected interface for the Bank module
// Method signatures must match cosmos-sdk/x/bank/keeper exactly
type BankKeeper interface {
	// SendCoinsFromAccountToModule transfers coins from an account to a module account
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error

	// GetBalance returns the balance of a specific denom for an account
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin

	// SpendableCoins returns all the spendable coins for an account
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}
