// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title VaultGate - EVM Interface to Mirror Vault Precompile
/// @notice Provides Solidity interface to the stateful precompile at 0x0101
/// @dev Both MetaMask and Keplr can store messages; all write to same on-chain x/vault state
contract VaultGate {
    event Unlocked(address indexed user);
    event MessageStored(address indexed user, string message);

    address public constant MIRROR_VAULT_PRECOMPILE = 0x0000000000000000000000000000000000000101;

    /// @notice Grant storage credit to msg.sender by paying 1 MVLT token
    /// @dev Requires payment of at least 1 MVLT (1e18 wei). Calls precompile unlock(), increments credit in x/vault
    /// @dev Implements requirement: "need of tokens to unlock the message and nft module (1 mvlt)"
    function payToUnlock() external payable {
        // Enforce payment requirement: must send at least 1 MVLT
        // In EVM: 1 MVLT = 1e18 wei
        // In Cosmos: 1 MVLT = 1,000,000 umvlt (micro-mvlt)
        require(msg.value >= 1e18, "Must pay at least 1 MVLT");

        (bool ok, ) = MIRROR_VAULT_PRECOMPILE.call{value: msg.value}(
            abi.encodeWithSignature("payToUnlock()")
        );
        require(ok, "precompile unlock failed");

        emit Unlocked(msg.sender);
    }

    /// @notice Store a message via precompile (MetaMask path)
    /// @dev Requires StorageCredit > 0, consumes 1 credit, updates global state
    /// @param message The message to store on-chain
    function storeMessage(string calldata message) external {
        (bool ok, ) = MIRROR_VAULT_PRECOMPILE.call(
            abi.encodeWithSignature("storeMessage(string)", message)
        );
        require(ok, "precompile storeMessage failed");

        emit MessageStored(msg.sender, message);
    }

    /// @notice Query storage credit for an address (view function)
    /// @dev Calls precompile to read x/vault state
    /// @param user The address to query
    /// @return credit The number of storage credits available
    function getMessageCount(address user) external view returns (uint256 credit) {
        (bool ok, bytes memory data) = MIRROR_VAULT_PRECOMPILE.staticcall(
            abi.encodeWithSignature("getMessageCount(address)", user)
        );
        require(ok, "precompile getMessageCount failed");

        credit = abi.decode(data, (uint256));
    }

    /// @notice Query last message stored by an address (view function)
    /// @dev Calls precompile to read x/vault state
    /// @param user The address to query
    /// @return lastMsg The most recent message stored by that user
    function getLastMessage(address user) external view returns (string memory lastMsg) {
        (bool ok, bytes memory data) = MIRROR_VAULT_PRECOMPILE.staticcall(
            abi.encodeWithSignature("getLastMessage(address)", user)
        );
        require(ok, "precompile getLastMessage failed");

        lastMsg = abi.decode(data, (string));
    }

    /// @notice Get global message count (chain-wide)
    /// @dev Returns total messages stored by all users
    /// @return count Total number of messages on chain
    function getGlobalMessageCount() external view returns (uint256 count) {
        (bool ok, bytes memory data) = MIRROR_VAULT_PRECOMPILE.staticcall(
            abi.encodeWithSignature("getGlobalMessageCount()")
        );
        require(ok, "precompile getGlobalMessageCount failed");

        count = abi.decode(data, (uint256));
    }

    /// @notice Get global last message (chain-wide)
    /// @dev Returns most recent message stored by anyone
    /// @return lastMsg The most recent message on chain
    function getGlobalLastMessage() external view returns (string memory lastMsg) {
        (bool ok, bytes memory data) = MIRROR_VAULT_PRECOMPILE.staticcall(
            abi.encodeWithSignature("getGlobalLastMessage()")
        );
        require(ok, "precompile getGlobalLastMessage failed");

        lastMsg = abi.decode(data, (string));
    }
}
