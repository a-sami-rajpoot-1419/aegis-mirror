package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"mirrorvault/x/vault/types"
)

// Keeper maintains the state for the vault module
type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey
}

// NewKeeper creates a new Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: storeKey,
	}
}

// Logger returns a module-specific logger
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// AddCredit adds storage credits to an account
func (k Keeper) AddCredit(ctx sdk.Context, address string) error {
	store := ctx.KVStore(k.storeKey)
	key := append(types.UserCreditsKey, []byte(address)...)

	currentCredits := k.GetCredits(ctx, address)
	newCredits := currentCredits + 1

	creditBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(creditBytes, newCredits)
	store.Set(key, creditBytes)

	k.Logger(ctx).Info("added credit", "address", address, "new_credits", newCredits)
	return nil
}

// GetCredits returns the number of storage credits for an account
func (k Keeper) GetCredits(ctx sdk.Context, address string) uint64 {
	store := ctx.KVStore(k.storeKey)
	key := append(types.UserCreditsKey, []byte(address)...)

	bz := store.Get(key)
	if bz == nil {
		return 0
	}

	return binary.BigEndian.Uint64(bz)
}

// UseCredit decrements storage credits for an account
func (k Keeper) UseCredit(ctx sdk.Context, address string) error {
	credits := k.GetCredits(ctx, address)
	if credits == 0 {
		return fmt.Errorf("no credits available for %s", address)
	}

	store := ctx.KVStore(k.storeKey)
	key := append(types.UserCreditsKey, []byte(address)...)

	newCredits := credits - 1
	creditBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(creditBytes, newCredits)
	store.Set(key, creditBytes)

	k.Logger(ctx).Info("used credit", "address", address, "remaining_credits", newCredits)
	return nil
}

// StoreMessage stores a message for an account (requires 1 credit)
func (k Keeper) StoreMessage(ctx sdk.Context, address, content string) error {
	// Check and use credit
	if err := k.UseCredit(ctx, address); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)

	// Get user's message count
	countKey := append(types.UserMessageCountKey, []byte(address)...)
	countBz := store.Get(countKey)
	var messageIndex uint64 = 0
	if countBz != nil {
		messageIndex = binary.BigEndian.Uint64(countBz)
	}

	// Create message
	msg := types.NewMessage(address, content, ctx.BlockTime(), messageIndex)

	// Store message
	msgKey := append(types.UserMessagesKey, []byte(address)...)
	indexBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(indexBytes, messageIndex)
	msgKey = append(msgKey, indexBytes...)

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	store.Set(msgKey, msgBytes)

	// Update user message count
	newCount := messageIndex + 1
	countBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(countBytes, newCount)
	store.Set(countKey, countBytes)

	// Update global message count
	globalCount := k.GetGlobalMessageCount(ctx)
	globalCountBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(globalCountBytes, globalCount+1)
	store.Set(types.GlobalMessageCountKey, globalCountBytes)

	// Update global last message
	store.Set(types.GlobalLastMessageKey, msgBytes)

	k.Logger(ctx).Info("stored message",
		"address", address,
		"index", messageIndex,
		"global_count", globalCount+1,
	)

	return nil
}

// GetMessageCount returns the number of messages stored by an account
func (k Keeper) GetMessageCount(ctx sdk.Context, address string) uint64 {
	store := ctx.KVStore(k.storeKey)
	key := append(types.UserMessageCountKey, []byte(address)...)

	bz := store.Get(key)
	if bz == nil {
		return 0
	}

	return binary.BigEndian.Uint64(bz)
}

// GetLastMessage returns the most recent message from an account
func (k Keeper) GetLastMessage(ctx sdk.Context, address string) (*types.Message, error) {
	count := k.GetMessageCount(ctx, address)
	if count == 0 {
		return nil, fmt.Errorf("no messages for address %s", address)
	}

	store := ctx.KVStore(k.storeKey)
	msgKey := append(types.UserMessagesKey, []byte(address)...)

	lastIndex := count - 1
	indexBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(indexBytes, lastIndex)
	msgKey = append(msgKey, indexBytes...)

	msgBytes := store.Get(msgKey)
	if msgBytes == nil {
		return nil, fmt.Errorf("message not found")
	}

	var msg types.Message
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}

// GetGlobalMessageCount returns the total number of messages on chain
func (k Keeper) GetGlobalMessageCount(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GlobalMessageCountKey)
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// GetGlobalLastMessage returns the most recent message on chain
func (k Keeper) GetGlobalLastMessage(ctx sdk.Context) (*types.Message, error) {
	store := ctx.KVStore(k.storeKey)
	msgBytes := store.Get(types.GlobalLastMessageKey)
	if msgBytes == nil {
		return nil, fmt.Errorf("no messages stored yet")
	}

	var msg types.Message
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}
