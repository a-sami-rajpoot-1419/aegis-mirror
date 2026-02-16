package precompile

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"mirrorvault/utils"
	vaultkeeper "mirrorvault/x/vault/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// VaultGateAddress is the precompile address for vault operations
	VaultGateAddress = "0x0000000000000000000000000000000000000101"
)

var (
	// Function selectors (first 4 bytes of keccak256 of function signature)
	payToUnlockSelector           = []byte{0x3c, 0x6a, 0x24, 0x42} // payToUnlock()
	storeMessageSelector          = []byte{0x72, 0x0f, 0x4e, 0x72} // storeMessage(string)
	getMessageCountSelector       = []byte{0xe6, 0x7c, 0x0e, 0xd3} // getMessageCount(address)
	getLastMessageSelector        = []byte{0xf5, 0x8c, 0x6f, 0x89} // getLastMessage(address)
	getGlobalMessageCountSelector = []byte{0x8d, 0xa5, 0xcb, 0x5b} // getGlobalMessageCount()
	getGlobalLastMessageSelector  = []byte{0xe3, 0xf2, 0x09, 0x17} // getGlobalLastMessage()
)

// VaultGatePrecompile implements the vault operations precompile
type VaultGatePrecompile struct {
	vaultKeeper  vaultkeeper.Keeper
	bech32Prefix string
}

// NewVaultGatePrecompile creates a new VaultGatePrecompile
func NewVaultGatePrecompile(vaultKeeper vaultkeeper.Keeper, bech32Prefix string) *VaultGatePrecompile {
	return &VaultGatePrecompile{
		vaultKeeper:  vaultKeeper,
		bech32Prefix: bech32Prefix,
	}
}

// Address returns the precompile address
func (p *VaultGatePrecompile) Address() common.Address {
	return common.HexToAddress(VaultGateAddress)
}

// RequiredGas returns the gas required to execute the precompiled contract
func (p *VaultGatePrecompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}

	// Base gas cost depending on function
	selector := input[0:4]
	switch {
	case bytesEqual(selector, payToUnlockSelector):
		return 50000 // State write
	case bytesEqual(selector, storeMessageSelector):
		return 100000 // State write + message storage
	case bytesEqual(selector, getMessageCountSelector):
		return 10000 // State read
	case bytesEqual(selector, getLastMessageSelector):
		return 15000 // State read
	case bytesEqual(selector, getGlobalMessageCountSelector):
		return 10000 // State read
	case bytesEqual(selector, getGlobalLastMessageSelector):
		return 15000 // State read
	default:
		return 0
	}
}

// Run executes the precompiled contract
func (p *VaultGatePrecompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) ([]byte, error) {
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
	case bytesEqual(selector, payToUnlockSelector):
		if readOnly {
			return nil, errors.New("cannot call payToUnlock in read-only mode")
		}
		// Convert uint256.Int to big.Int
		valueBig := contract.Value().ToBig()
		return p.payToUnlock(sdkCtx, contract.Caller(), valueBig)
	case bytesEqual(selector, storeMessageSelector):
		if readOnly {
			return nil, errors.New("cannot call storeMessage in read-only mode")
		}
		return p.storeMessage(sdkCtx, contract.Caller(), args)
	case bytesEqual(selector, getMessageCountSelector):
		return p.getMessageCount(sdkCtx, args)
	case bytesEqual(selector, getLastMessageSelector):
		return p.getLastMessage(sdkCtx, args)
	case bytesEqual(selector, getGlobalMessageCountSelector):
		return p.getGlobalMessageCount(sdkCtx)
	case bytesEqual(selector, getGlobalLastMessageSelector):
		return p.getGlobalLastMessage(sdkCtx)
	default:
		return nil, fmt.Errorf("unknown function selector: %x", selector)
	}
}

// payToUnlock adds a storage credit to the caller AFTER validating payment
// Implements requirement: "need of tokens to unlock the message and nft module (1 mirror)"
// Payment must be at least 1 MIRROR (1,000,000 amirror)
func (p *VaultGatePrecompile) payToUnlock(ctx sdk.Context, caller common.Address, value *big.Int) ([]byte, error) {
	// Convert caller address to Cosmos bech32
	cosmosAddr, err := utils.EthAddressToBech32(caller.Hex(), p.bech32Prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to convert address: %w", err)
	}

	// Validate payment amount (value is in wei, which maps to umvlt)
	// 1 MVLT = 1e18 wei in EVM = 1,000,000 umvlt in Cosmos
	// For simplicity: 1 MVLT = 1,000,000 umvlt
	if value == nil || value.Cmp(big.NewInt(1_000_000)) < 0 {
		return nil, fmt.Errorf("insufficient payment: sent %s, required 1000000 umvlt (1 MVLT)",
			valueToString(value))
	}

	// Convert value to SDK coins
	payment := sdk.NewCoins(sdk.NewCoin("umvlt", sdkmath.NewIntFromBigInt(value)))

	// Add credit with payment validation
	if err := p.vaultKeeper.AddCreditWithPayment(ctx, cosmosAddr, payment); err != nil {
		return nil, fmt.Errorf("failed to add credit: %w", err)
	}

	ctx.Logger().Info("credit purchased via payToUnlock",
		"caller_evm", caller.Hex(),
		"caller_cosmos", cosmosAddr,
		"payment", payment.String(),
	)

	// Return success (empty bytes for void function)
	return []byte{}, nil
}

// Helper function to safely convert big.Int to string
func valueToString(value *big.Int) string {
	if value == nil {
		return "0"
	}
	return value.String()
}

// storeMessage stores a message using one credit
func (p *VaultGatePrecompile) storeMessage(ctx sdk.Context, caller common.Address, args []byte) ([]byte, error) {
	// Convert caller address to Cosmos bech32
	cosmosAddr, err := utils.EthAddressToBech32(caller.Hex(), p.bech32Prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to convert address: %w", err)
	}

	// Decode string argument
	stringType, _ := abi.NewType("string", "", nil)
	argsList := abi.Arguments{{Type: stringType}}

	decoded, err := argsList.Unpack(args)
	if err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}

	message := decoded[0].(string)

	// Store message
	if err := p.vaultKeeper.StoreMessage(ctx, cosmosAddr, message); err != nil {
		return nil, err
	}

	return []byte{}, nil
}

// getMessageCount returns the number of messages for an address
func (p *VaultGatePrecompile) getMessageCount(ctx sdk.Context, args []byte) ([]byte, error) {
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

	// Get message count
	count := p.vaultKeeper.GetMessageCount(ctx, cosmosAddr)

	// Encode uint256 return value
	result := make([]byte, 32)
	binary.BigEndian.PutUint64(result[24:], count)
	return result, nil
}

// getLastMessage returns the last message for an address
func (p *VaultGatePrecompile) getLastMessage(ctx sdk.Context, args []byte) ([]byte, error) {
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

	// Get last message
	msg, err := p.vaultKeeper.GetLastMessage(ctx, cosmosAddr)
	if err != nil {
		// Return empty string if no messages
		return encodeString(""), nil
	}

	return encodeString(msg.Content), nil
}

// getGlobalMessageCount returns the total number of messages on chain
func (p *VaultGatePrecompile) getGlobalMessageCount(ctx sdk.Context) ([]byte, error) {
	count := p.vaultKeeper.GetGlobalMessageCount(ctx)

	result := make([]byte, 32)
	binary.BigEndian.PutUint64(result[24:], count)
	return result, nil
}

// getGlobalLastMessage returns the most recent message on chain
func (p *VaultGatePrecompile) getGlobalLastMessage(ctx sdk.Context) ([]byte, error) {
	msg, err := p.vaultKeeper.GetGlobalLastMessage(ctx)
	if err != nil {
		return encodeString(""), nil
	}

	return encodeString(msg.Content), nil
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
