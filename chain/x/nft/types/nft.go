package types

import (
	"time"
)

// NFT represents a non-fungible token
type NFT struct {
	TokenId  uint64    `json:"token_id"`  // Unique identifier
	Owner    string    `json:"owner"`     // Cosmos bech32 address
	TokenURI string    `json:"token_uri"` // Metadata URI (IPFS/Arweave)
	MintedAt time.Time `json:"minted_at"` // Mint timestamp
}

// NewNFT creates a new NFT
func NewNFT(tokenId uint64, owner, tokenURI string, mintedAt time.Time) NFT {
	return NFT{
		TokenId:  tokenId,
		Owner:    owner,
		TokenURI: tokenURI,
		MintedAt: mintedAt,
	}
}
