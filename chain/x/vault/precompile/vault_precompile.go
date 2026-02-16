package precompile

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"

	"mirrorvault/utils"
	vaultkeeper "mirrorvault/x/vault/keeper"
	vaulttypes "mirrorvault/x/vault/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	// VaultGateAddress is the precompile address for vault operations
	VaultGateAddress = "0x0000000000000000000000000000000000000101"
)

var (
	// Function selectors (first 4 bytes of keccak256 of function signature)
	payToUnlockSelector           = []byte{0xbd, 0xe8, 0x39, 0x38} // payToUnlock()
	storeMessageSelector          = []byte{0xd4, 0xe3, 0x6b, 0xa7} // storeMessage(string)
	getMessageCountSelector       = []byte{0xd7, 0x36, 0x3c, 0xe7} // getMessageCount(address)
	getLastMessageSelector        = []byte{0xe0, 0xc0, 0x1b, 0xfe} // getLastMessage(address)
	getGlobalMessageCountSelector = []byte{0xb3, 0x2c, 0x53, 0x91} // getGlobalMessageCount()
	getGlobalLastMessageSelector  = []byte{0x8a, 0xa4, 0x49, 0xd8} // getGlobalLastMessage()
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
		return p.payToUnlock(sdkCtx, evm, contract, valueBig)
	case bytesEqual(selector, storeMessageSelector):
		if readOnly {
			return nil, errors.New("cannot call storeMessage in read-only mode")
		}
		return p.storeMessage(sdkCtx, evm.Origin, args)
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
func (p *VaultGatePrecompile) payToUnlock(ctx sdk.Context, evm *vm.EVM, contract *vm.Contract, value *big.Int) ([]byte, error) {
	beneficiary := evm.Origin
	precompileAddr := contract.Address()

	// Convert beneficiary (EOA / tx origin) to Cosmos bech32
	beneficiaryBech32, err := utils.EthAddressToBech32(beneficiary.Hex(), p.bech32Prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to convert address: %w", err)
	}

	// Validate payment amount (EVM uses 18 decimals for MVLT; 1 MVLT = 1e18 base units)
	// NOTE: The EVM module requires denom metadata for `umvlt` with display exponent 18.
	if value == nil || value.Cmp(big.NewInt(1_000_000_000_000_000_000)) < 0 {
		return nil, fmt.Errorf("Must pay at least 1 MVLT")
	}

	// During EVM execution, Cosmos bank spendables may not reflect EVM value transfers yet.
	// So we move funds using the EVM StateDB: precompile (callee) -> vault module account.
	amountU256, overflow := uint256.FromBig(value)
	if overflow {
		return nil, fmt.Errorf("payment amount overflows uint256")
	}
	if amountU256 == nil {
		return nil, fmt.Errorf("invalid payment amount")
	}

	moduleAcc := authtypes.NewModuleAddress(vaulttypes.ModuleName)
	moduleEthAddr := common.BytesToAddress(moduleAcc)

	precompileBal := evm.StateDB.GetBalance(precompileAddr)
	if precompileBal == nil || precompileBal.Cmp(amountU256) < 0 {
		return nil, fmt.Errorf("insufficient precompile balance for payment")
	}

	evm.StateDB.SubBalance(precompileAddr, amountU256, tracing.BalanceChangeTransfer)
	evm.StateDB.AddBalance(moduleEthAddr, amountU256, tracing.BalanceChangeTransfer)

	// Payment successful - now add the credit to the beneficiary (tx origin)
	if err := p.vaultKeeper.AddCredit(ctx, beneficiaryBech32); err != nil {
		return nil, fmt.Errorf("failed to add credit: %w", err)
	}

	ctx.Logger().Info("credit purchased via payToUnlock",
		"beneficiary_evm", beneficiary.Hex(),
		"beneficiary_cosmos", beneficiaryBech32,
		"payer_evm", precompileAddr.Hex(),
		"payment_wei", valueToString(value),
		"module_evm", moduleEthAddr.Hex(),
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
