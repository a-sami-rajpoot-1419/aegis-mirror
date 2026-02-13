package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"mirrorvault/x/nft/types"
)

// Keeper maintains the state for the nft module
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

// MintNFT mints a new NFT
func (k Keeper) MintNFT(ctx sdk.Context, tokenId uint64, owner, tokenURI string) error {
	// Check if tokenId already exists
	if k.Exists(ctx, tokenId) {
		return fmt.Errorf("NFT with tokenId %d already exists", tokenId)
	}

	store := ctx.KVStore(k.storeKey)

	// Create NFT
	nft := types.NewNFT(tokenId, owner, tokenURI, ctx.BlockTime())

	// Store NFT
	nftKey := k.getNFTKey(tokenId)
	nftBytes, err := json.Marshal(nft)
	if err != nil {
		return fmt.Errorf("failed to marshal NFT: %w", err)
	}
	store.Set(nftKey, nftBytes)

	// Add to owner index
	ownerKey := k.getOwnerNFTKey(owner, tokenId)
	store.Set(ownerKey, []byte{1})

	// Increment total supply
	totalSupply := k.GetTotalSupply(ctx)
	supplyBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(supplyBytes, totalSupply+1)
	store.Set(types.TotalSupplyKey, supplyBytes)

	k.Logger(ctx).Info("minted NFT",
		"token_id", tokenId,
		"owner", owner,
		"token_uri", tokenURI,
	)

	return nil
}

// TransferNFT transfers an NFT to a new owner
func (k Keeper) TransferNFT(ctx sdk.Context, tokenId uint64, newOwner string) error {
	nft, err := k.GetNFT(ctx, tokenId)
	if err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	oldOwner := nft.Owner

	// Remove from old owner's index
	oldOwnerKey := k.getOwnerNFTKey(oldOwner, tokenId)
	store.Delete(oldOwnerKey)

	// Update NFT owner
	nft.Owner = newOwner
	nftKey := k.getNFTKey(tokenId)
	nftBytes, err := json.Marshal(nft)
	if err != nil {
		return fmt.Errorf("failed to marshal NFT: %w", err)
	}
	store.Set(nftKey, nftBytes)

	// Add to new owner's index
	newOwnerKey := k.getOwnerNFTKey(newOwner, tokenId)
	store.Set(newOwnerKey, []byte{1})

	k.Logger(ctx).Info("transferred NFT",
		"token_id", tokenId,
		"from", oldOwner,
		"to", newOwner,
	)

	return nil
}

// GetNFT retrieves an NFT by tokenId
func (k Keeper) GetNFT(ctx sdk.Context, tokenId uint64) (*types.NFT, error) {
	store := ctx.KVStore(k.storeKey)
	nftKey := k.getNFTKey(tokenId)
	nftBytes := store.Get(nftKey)

	if nftBytes == nil {
		return nil, fmt.Errorf("NFT with tokenId %d not found", tokenId)
	}

	var nft types.NFT
	if err := json.Unmarshal(nftBytes, &nft); err != nil {
		return nil, fmt.Errorf("failed to unmarshal NFT: %w", err)
	}

	return &nft, nil
}

// Exists checks if an NFT exists
func (k Keeper) Exists(ctx sdk.Context, tokenId uint64) bool {
	store := ctx.KVStore(k.storeKey)
	nftKey := k.getNFTKey(tokenId)
	return store.Has(nftKey)
}

// GetOwner returns the owner of an NFT
func (k Keeper) GetOwner(ctx sdk.Context, tokenId uint64) (string, error) {
	nft, err := k.GetNFT(ctx, tokenId)
	if err != nil {
		return "", err
	}
	return nft.Owner, nil
}

// GetNFTsByOwner returns all NFT tokenIds owned by an address
func (k Keeper) GetNFTsByOwner(ctx sdk.Context, owner string) []uint64 {
	store := ctx.KVStore(k.storeKey)
	prefix := append(types.OwnerNFTsKey, []byte(owner)...)

	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var tokenIds []uint64
	for ; iterator.Valid(); iterator.Next() {
		// Extract tokenId from key (last 8 bytes)
		key := iterator.Key()
		tokenIdBytes := key[len(key)-8:]
		tokenId := binary.BigEndian.Uint64(tokenIdBytes)
		tokenIds = append(tokenIds, tokenId)
	}

	return tokenIds
}

// GetTotalSupply returns the total number of NFTs minted
func (k Keeper) GetTotalSupply(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TotalSupplyKey)
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// Helper functions

func (k Keeper) getNFTKey(tokenId uint64) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, tokenId)
	return append(types.NFTKey, key...)
}

func (k Keeper) getOwnerNFTKey(owner string, tokenId uint64) []byte {
	key := append(types.OwnerNFTsKey, []byte(owner)...)
	tokenIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tokenIdBytes, tokenId)
	return append(key, tokenIdBytes...)
}
