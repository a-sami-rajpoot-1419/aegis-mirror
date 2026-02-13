package types

const (
	// ModuleName defines the module name
	ModuleName = "nft"

	// StoreKey defines the primary store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName
)

// Store prefixes
var (
	// NFTKey prefix for storing NFTs
	// Key: tokenId (uint64) -> Value: NFT
	NFTKey = []byte{0x01}

	// OwnerNFTsKey prefix for indexing NFTs by owner
	// Key: bech32 address (mirror1...) + tokenId -> Value: empty (existence check)
	OwnerNFTsKey = []byte{0x02}

	// TotalSupplyKey stores the total number of NFTs minted
	TotalSupplyKey = []byte{0x03}
)
