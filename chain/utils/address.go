// Package utils provides utility functions for address conversion and formatting
package utils

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common"
)

// Bech32ToEthAddress converts a Cosmos bech32 address to Ethereum hex format
// Example: mirror1abc...xyz -> 0x1234...abcd
func Bech32ToEthAddress(bech32Addr string) (string, error) {
	// Decode bech32 to get the raw address bytes
	_, addrBytes, err := bech32.DecodeAndConvert(bech32Addr)
	if err != nil {
		return "", err
	}

	// Convert to Ethereum address (0x... format)
	ethAddr := common.BytesToAddress(addrBytes)
	return ethAddr.Hex(), nil
}

// EthAddressToBech32 converts an Ethereum hex address to Cosmos bech32 format
// Example: 0x1234...abcd -> mirror1abc...xyz
func EthAddressToBech32(ethAddr string, prefix string) (string, error) {
	// Remove 0x prefix if present
	ethAddr = strings.TrimPrefix(ethAddr, "0x")

	// Parse as Ethereum address
	addr := common.HexToAddress(ethAddr)

	// Convert to bech32 with specified prefix
	bech32Addr, err := bech32.ConvertAndEncode(prefix, addr.Bytes())
	if err != nil {
		return "", err
	}

	return bech32Addr, nil
}

// SDKAddressToBothFormats converts an SDK address to both Cosmos and EVM formats
// Returns: (bech32, ethHex, error)
func SDKAddressToBothFormats(addr sdk.AccAddress, bech32Prefix string) (string, string, error) {
	// Get bech32 format
	bech32Addr, err := bech32.ConvertAndEncode(bech32Prefix, addr.Bytes())
	if err != nil {
		return "", "", err
	}

	// Get Ethereum hex format
	ethAddr := common.BytesToAddress(addr.Bytes()).Hex()

	return bech32Addr, ethAddr, nil
}

// FormatAddressForEvent creates a dual-format address string for events
// Format: "0x1234...abcd (mirror1abc...xyz)"
func FormatAddressForEvent(addr sdk.AccAddress, bech32Prefix string) string {
	bech32Addr, ethHex, err := SDKAddressToBothFormats(addr, bech32Prefix)
	if err != nil {
		// Fallback to just SDK address if conversion fails
		return addr.String()
	}

	return ethHex + " (" + bech32Addr + ")"
}

// ExtractAddressFromTx extracts all addresses (sender, recipient) from a transaction
// Returns list of SDK addresses found in the transaction
func ExtractAddressesFromTx(tx sdk.Tx) []sdk.AccAddress {
	addresses := make([]sdk.AccAddress, 0)
	seen := make(map[string]bool)

	// Extract from messages
	for _, msg := range tx.GetMsgs() {
		// Get signers (senders)
		signers, err := msg.GetSigners()
		if err == nil {
			for _, signer := range signers {
				signerStr := signer.String()
				if !seen[signerStr] {
					addresses = append(addresses, sdk.AccAddress(signer))
					seen[signerStr] = true
				}
			}
		}
	}

	return addresses
}

// IsBech32Address checks if a string is a valid bech32 address
func IsBech32Address(addr string) bool {
	_, _, err := bech32.DecodeAndConvert(addr)
	return err == nil
}

// IsEthAddress checks if a string is a valid Ethereum hex address
func IsEthAddress(addr string) bool {
	return common.IsHexAddress(addr)
}
