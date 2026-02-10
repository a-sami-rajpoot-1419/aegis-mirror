// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @notice Minimal v1 gate contract.
/// The chain implements a stateful precompile at 0x000...0101.
/// Calling payToUnlock() triggers the precompile, which increments StorageCredit
/// for msg.sender in the native Cosmos x/vault module.
contract VaultGate {
    event Unlocked(address indexed user);

    address public constant MIRROR_VAULT_PRECOMPILE = 0x0000000000000000000000000000000000000101;

    function payToUnlock() external {
        // Call the precompile. We don’t assume an ABI yet; success is enough.
        (bool ok, ) = MIRROR_VAULT_PRECOMPILE.call("");
        require(ok, "precompile call failed");

        emit Unlocked(msg.sender);
    }
}
