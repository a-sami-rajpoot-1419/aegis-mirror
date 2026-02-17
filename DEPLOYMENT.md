# Mirror Vault Deployment Info

## Chain Status
✅ **Blockchain Running**
- Chain ID: `mirror-vault-localnet`
- EVM Chain ID: `7777`
- JSON-RPC: http://localhost:8545
- Cosmos REST: http://localhost:1317
- CometBFT RPC: http://localhost:26657

## Start / Restart (Persistent)

### One-time recovery (if you see AppHash mismatch / chain won’t start)
- Run: `FORCE_RESET=1 bash ./setup-and-start.sh`

This wipes `~/.mirrorvault` once to repair inconsistent local state.

### Normal restarts (state persists)
- Stop: `pkill -9 mirrorvaultd`
- Start: `bash ./setup-and-start.sh`

Balances and deployed wrapper contracts persist because `~/.mirrorvault` is reused.

## Automatic Wallet Funding

### Fund wallets on resets (recommended)
- Put your bech32 wallet address(es) in `fund-accounts.txt` (one per line, `mirror1...`).
- Then run a reset start: `FORCE_RESET=1 bash ./setup-and-start.sh`

Those addresses are added to genesis with the same large balance as the test accounts.

### Optional top-up on an existing chain
Run:
- `bash tools/fund-wallets.sh`

This sends native MVLT via EVM (so MetaMask/Keplr EVM mode updates immediately) to every address in `fund-accounts.txt`.

Override amount (MVLT):
- `AMOUNT_MVLT=5000 bash tools/fund-wallets.sh`

## Deployed Contracts

### VaultGate
- **Address:** `0xC5273AbFb36550090095B1EDec019216AD21BE6c`
- **Precompile:** `0x0000000000000000000000000000000000000101`
- **Functions:**
  - `payToUnlock()` - Pay 1 MVLT to add credit
  - `storeMessage(string text)` - Store message using credit
  - `getMessage(address user)` - Retrieve last message
  - `getMessageCount(address user)` - Get total message count

### MirrorNFT
- **Address:** `0x39529fdA4CbB4f8Bfca2858f9BfAeb28B904Adc0`
- **Precompile:** `0x0000000000000000000000000000000000000102`
- **Functions:**
  - `mint(address to, uint256 tokenId, string uri)` - Mint NFT
  - `transferFrom(address from, address to, uint256 tokenId)` - Transfer NFT
  - `ownerOf(uint256 tokenId)` - Get owner (returns both 0x and mirror1 addresses)
  - `exists(uint256 tokenId)` - Check if NFT exists

## Test Accounts

### Alice (Primary Deployment Account)
- **Mnemonic:** `abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about`
- **Private Key:** `0x1ab42cc412b618bdea3a599e3c9bae199ebf030895b039e9db1e30dafb12b727`
- **Ethereum Address:** `0x9858EfFD232B4033E47d90003D41EC34EcaEda94`
- **Cosmos Address:** `mirror1npvwllfr9dqr8erajqqr6s0vxnk2ak5553ldj8`
- **Balance:** ~10,000 MVLT (used some for deployment gas)

### Bob
- **Mnemonic:** `test test test test test test test test test test test junk`
- **Ethereum Address:** TBD (derive from mnemonic with coin-type 60, path m/44'/60'/0'/0/0)

## Critical Address Derivation Fix

**Problem:** Cosmos chains and Ethereum use different address derivation methods from the same private key:
- **Cosmos:** SHA256 → RIPEMD160 of public key
- **Ethereum:** Keccak256 of public key

**Solution:** Fund BOTH derivations in genesis:
1. Cosmos-derived address (for staking/validator): `mirror1gsvdpdxec8hsu57lhxg5xem7refr233zpscthf`
2. Ethereum-derived address (for EVM transactions): `mirror1npvwllfr9dqr8erajqqr6s0vxnk2ak5553ldj8`

This ensures the EVM ante handler (which extracts sender from Ethereum transaction signatures) can find the required balance.

## Next Steps

1. **Backend Testing:**
   - Test payment validation (1 MVLT requirement)
   - Test message storage/retrieval
   - Test NFT minting and transfers
   - Test cross-pair operations (MetaMask ↔ Keplr)

2. **Frontend Development:**
   - Create Next.js project structure
   - Integrate contract addresses and ABIs
   - Implement wallet connections (MetaMask + Keplr)
   - Build UI components per user requirements
   - Test all 12 user requirements

3. **Final Verification:**
   - [ ] Auto-connect works
   - [ ] Add chain module works
   - [ ] Token transfer works (all 4 combinations)
   - [ ] Message adding works
   - [ ] Last message output displays
   - [ ] Total message count displays
   - [ ] Dual addresses display
   - [ ] Total balance displays
   - [ ] NFT minting works
   - [ ] NFT gallery shows unified state
   - [ ] Transaction logs work
   - [ ] Random ID generator works

## Technical Notes

- **Base Fee:** Set to 1000000000 wei (1 gwei) in feemarket params
- **Gas Price:** Hardhat configured to use 2000000000 wei (2 gwei)
- **Denom:** `umvlt` (base unit), `MVLT` (display, 18 decimals)
- **Payment Requirement:** 1 MVLT = 1,000,000 umvlt enforced at 3 layers (Solidity, Precompile, Keeper)
