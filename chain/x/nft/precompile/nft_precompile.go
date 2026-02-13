package precompile

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"mirrorvault/utils"
	nftkeeper "mirrorvault/x/nft/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// MirrorNFTAddress is the precompile address for NFT operations
	MirrorNFTAddress = "0x0000000000000000000000000000000000000102"
)

var (
	// Function selectors (first 4 bytes of keccak256 of function signature)
	mintSelector          = []byte{0xa0, 0x71, 0x2d, 0x68} // mint(uint256,string)
	transferFromSelector  = []byte{0x23, 0xb8, 0x72, 0xdd} // transferFrom(address,address,uint256)
	ownerOfSelector       = []byte{0x63, 0x52, 0x21, 0x1e} // ownerOf(uint256)
	balanceOfSelector     = []byte{0x70, 0xa0, 0x82, 0x31} // balanceOf(address)
	tokenURISelector      = []byte{0xc8, 0x7b, 0x56, 0xdd} // tokenURI(uint256)
	tokensOfOwnerSelector = []byte{0x8b, 0x2c, 0x7f, 0x55} // tokensOfOwner(address)
)

// MirrorNFTPrecompile implements the NFT operations precompile
type MirrorNFTPrecompile struct {
	nftKeeper    nftkeeper.Keeper
	bech32Prefix string
}

// NewMirrorNFTPrecompile creates a new MirrorNFTPrecompile
func NewMirrorNFTPrecompile(nftKeeper nftkeeper.Keeper, bech32Prefix string) *MirrorNFTPrecompile {
	return &MirrorNFTPrecompile{
		nftKeeper:    nftKeeper,
		bech32Prefix: bech32Prefix,
	}
}

// Address returns the precompile address
func (p *MirrorNFTPrecompile) Address() common.Address {
	return common.HexToAddress(MirrorNFTAddress)
}

// RequiredGas returns the gas required to execute the precompiled contract
func (p *MirrorNFTPrecompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}

	selector := input[0:4]
	switch {
	case bytesEqual(selector, mintSelector):
		return 100000 // State write + indexing
	case bytesEqual(selector, transferFromSelector):
		return 80000 // State write + index updates
	case bytesEqual(selector, ownerOfSelector):
		return 15000 // State read
	case bytesEqual(selector, balanceOfSelector):
		return 20000 // State read + iteration
	case bytesEqual(selector, tokenURISelector):
		return 10000 // State read
	case bytesEqual(selector, tokensOfOwnerSelector):
		return 30000 // State read + iteration
	default:
		return 0
	}
}

// Run executes the precompiled contract
func (p *MirrorNFTPrecompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) ([]byte, error) {
	input := contract.Input
	if len(input) < 4 {
		return nil, errors.New("input too short")
	}

	// Get SDK context from EVM
	ctx, ok := evm.StateDB.(interface{ GetContext() sdk.Context })
	if !ok {
		return nil, errors.New("failed to get SDK context")
	}
	sdkCtx := ctx.GetContext()

	// Parse function selector
	selector := input[0:4]
	args := input[4:]

	switch {
	case bytesEqual(selector, mintSelector):
		if readOnly {
			return nil, errors.New("cannot call mint in read-only mode")
		}
		return p.mint(sdkCtx, contract.Caller(), args)
	case bytesEqual(selector, transferFromSelector):
		if readOnly {
			return nil, errors.New("cannot call transferFrom in read-only mode")
		}
		return p.transferFrom(sdkCtx, contract.Caller(), args)
	case bytesEqual(selector, ownerOfSelector):
		return p.ownerOf(sdkCtx, args)
	case bytesEqual(selector, balanceOfSelector):
		return p.balanceOf(sdkCtx, args)
	case bytesEqual(selector, tokenURISelector):
		return p.tokenURI(sdkCtx, args)
	case bytesEqual(selector, tokensOfOwnerSelector):
		return p.tokensOfOwner(sdkCtx, args)
	default:
		return nil, fmt.Errorf("unknown function selector: %x", selector)
	}
}

// mint mints a new NFT
func (p *MirrorNFTPrecompile) mint(ctx sdk.Context, caller common.Address, args []byte) ([]byte, error) {
	// Convert caller address to Cosmos bech32
	cosmosAddr, err := utils.EthAddressToBech32(caller.Hex(), p.bech32Prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to convert address: %w", err)
	}

	// Decode arguments (uint256 tokenId, string uri)
	uint256Type, _ := abi.NewType("uint256", "", nil)
	stringType, _ := abi.NewType("string", "", nil)
	argsList := abi.Arguments{
		{Type: uint256Type},
		{Type: stringType},
	}

	decoded, err := argsList.Unpack(args)
	if err != nil {
		return nil, fmt.Errorf("failed to decode arguments: %w", err)
	}

	tokenId := decoded[0].(*big.Int).Uint64()
	uri := decoded[1].(string)

	// Mint NFT
	if err := p.nftKeeper.MintNFT(ctx, tokenId, cosmosAddr, uri); err != nil {
		return nil, err
	}

	return []byte{}, nil
}

// transferFrom transfers an NFT
func (p *MirrorNFTPrecompile) transferFrom(ctx sdk.Context, caller common.Address, args []byte) ([]byte, error) {
	// Decode arguments (address from, address to, uint256 tokenId)
	addressType, _ := abi.NewType("address", "", nil)
	uint256Type, _ := abi.NewType("uint256", "", nil)
	argsList := abi.Arguments{
		{Type: addressType}, // from
		{Type: addressType}, // to
		{Type: uint256Type}, // tokenId
	}

	decoded, err := argsList.Unpack(args)
	if err != nil {
		return nil, fmt.Errorf("failed to decode arguments: %w", err)
	}

	from := decoded[0].(common.Address)
	to := decoded[1].(common.Address)
	tokenId := decoded[2].(*big.Int).Uint64()

	// Validate caller is owner
	callerCosmos, err := utils.EthAddressToBech32(caller.Hex(), p.bech32Prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to convert caller address: %w", err)
	}

	fromCosmos, err := utils.EthAddressToBech32(from.Hex(), p.bech32Prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to convert from address: %w", err)
	}

	if callerCosmos != fromCosmos {
		return nil, errors.New("caller is not owner")
	}

	// Check current owner
	currentOwner, err := p.nftKeeper.GetOwner(ctx, tokenId)
	if err != nil {
		return nil, err
	}

	if currentOwner != fromCosmos {
		return nil, errors.New("from address is not owner")
	}

	// Convert recipient to Cosmos
	toCosmos, err := utils.EthAddressToBech32(to.Hex(), p.bech32Prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to address: %w", err)
	}

	// Transfer NFT
	if err := p.nftKeeper.TransferNFT(ctx, tokenId, toCosmos); err != nil {
		return nil, err
	}

	return []byte{}, nil
}

// ownerOf returns the owner of an NFT with DUAL ADDRESS FORMAT
func (p *MirrorNFTPrecompile) ownerOf(ctx sdk.Context, args []byte) ([]byte, error) {
	// Decode tokenId argument
	uint256Type, _ := abi.NewType("uint256", "", nil)
	argsList := abi.Arguments{{Type: uint256Type}}

	decoded, err := argsList.Unpack(args)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tokenId: %w", err)
	}

	tokenId := decoded[0].(*big.Int).Uint64()

	// Get owner
	owner, err := p.nftKeeper.GetOwner(ctx, tokenId)
	if err != nil {
		// Return (address(0), "", false) if not found
		return encodeOwnerResult(common.Address{}, "", false), nil
	}

	// Convert owner to EVM address
	ownerEVM, err := utils.Bech32ToEthAddress(owner)
	if err != nil {
		return nil, fmt.Errorf("failed to convert owner address: %w", err)
	}

	// Return (address owner, string ownerCosmos, bool exists)
	return encodeOwnerResult(common.HexToAddress(ownerEVM), owner, true), nil
}

// balanceOf returns the number of NFTs owned by an address
func (p *MirrorNFTPrecompile) balanceOf(ctx sdk.Context, args []byte) ([]byte, error) {
	// Decode address argument
	addressType, _ := abi.NewType("address", "", nil)
	argsList := abi.Arguments{{Type: addressType}}

	decoded, err := argsList.Unpack(args)
	if err != nil {
		return nil, fmt.Errorf("failed to decode address: %w", err)
	}

	addr := decoded[0].(common.Address)
	cosmosAddr, err := utils.EthAddressToBech32(addr.Hex(), p.bech32Prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to convert address: %w", err)
	}

	// Get NFTs owned by address
	tokenIds := p.nftKeeper.GetNFTsByOwner(ctx, cosmosAddr)

	// Encode uint256 return value
	result := make([]byte, 32)
	binary.BigEndian.PutUint64(result[24:], uint64(len(tokenIds)))
	return result, nil
}

// tokenURI returns the metadata URI for an NFT
func (p *MirrorNFTPrecompile) tokenURI(ctx sdk.Context, args []byte) ([]byte, error) {
	// Decode tokenId argument
	uint256Type, _ := abi.NewType("uint256", "", nil)
	argsList := abi.Arguments{{Type: uint256Type}}

	decoded, err := argsList.Unpack(args)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tokenId: %w", err)
	}

	tokenId := decoded[0].(*big.Int).Uint64()

	// Get NFT
	nft, err := p.nftKeeper.GetNFT(ctx, tokenId)
	if err != nil {
		return encodeString(""), nil
	}

	return encodeString(nft.TokenURI), nil
}

// tokensOfOwner returns all NFT tokenIds owned by an address
func (p *MirrorNFTPrecompile) tokensOfOwner(ctx sdk.Context, args []byte) ([]byte, error) {
	// Decode address argument
	addressType, _ := abi.NewType("address", "", nil)
	argsList := abi.Arguments{{Type: addressType}}

	decoded, err := argsList.Unpack(args)
	if err != nil {
		return nil, fmt.Errorf("failed to decode address: %w", err)
	}

	addr := decoded[0].(common.Address)
	cosmosAddr, err := utils.EthAddressToBech32(addr.Hex(), p.bech32Prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to convert address: %w", err)
	}

	// Get NFTs owned by address
	tokenIds := p.nftKeeper.GetNFTsByOwner(ctx, cosmosAddr)

	// Convert to []*big.Int
	bigInts := make([]*big.Int, len(tokenIds))
	for i, id := range tokenIds {
		bigInts[i] = new(big.Int).SetUint64(id)
	}

	// Encode uint256[] return value
	uint256Type, _ := abi.NewType("uint256[]", "", nil)
	result, err := abi.Arguments{{Type: uint256Type}}.Pack(bigInts)
	if err != nil {
		return nil, fmt.Errorf("failed to encode result: %w", err)
	}

	return result, nil
}

// Helper functions

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func encodeString(s string) []byte {
	stringType, _ := abi.NewType("string", "", nil)
	result, _ := abi.Arguments{{Type: stringType}}.Pack(s)
	return result
}

// encodeOwnerResult encodes the ownerOf return value (address, string, bool)
func encodeOwnerResult(owner common.Address, ownerCosmos string, exists bool) []byte {
	addressType, _ := abi.NewType("address", "", nil)
	stringType, _ := abi.NewType("string", "", nil)
	boolType, _ := abi.NewType("bool", "", nil)

	result, _ := abi.Arguments{
		{Type: addressType},
		{Type: stringType},
		{Type: boolType},
	}.Pack(owner, ownerCosmos, exists)

	return result
}
