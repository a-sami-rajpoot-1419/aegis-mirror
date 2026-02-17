# Mirror Vault UI — Requirements & Test Checklist (v1)

This checklist is distilled from the project docs + the current on-chain contracts and backend tests.

## Frozen constants (must match chain)
- Cosmos chain-id: `mirror-vault-localnet`
- EVM chainId: `7777` (`0x1e61`)
- Bech32 prefix: `mirror`
- Native denom: `umvlt` (display `MVLT`, 18 decimals)
- JSON-RPC: `http://localhost:8545`
- WS: `ws://localhost:8546`
- CometBFT RPC: `http://localhost:26657`

## Wallet onboarding (UI-visible)
- MetaMask
  - Connect/disconnect
  - `wallet_addEthereumChain` + `wallet_switchEthereumChain`
  - Shows active wallet = MetaMask
- Keplr
  - Connect/disconnect (EVM provider via `window.keplr.ethereum`)
  - `experimentalSuggestChain` (Cosmos config) best-effort
  - Shows active wallet = Keplr

## Identity / address requirements
- UI shows both formats for connected account:
  - EVM: `0x…`
  - Cosmos: `mirror1…` (mathematical conversion, not a lookup)
- Conversion tool
  - Input: 0x or mirror1
  - Output: both formats

## Vault (Message) requirements
- Wrapper: VaultGate (address loaded from `contracts/deployed-addresses.json`)
- Unlock / payment
  - UI action: `payToUnlock()` with exactly `1 MVLT` (1e18 wei)
  - Must confirm tx + show tx hash + gas used
- Message storage
  - UI action: `storeMessage(string)`
  - Max length enforced in UI (256 chars)
- State visibility (view calls)
  - Per-user: `getMessageCount(address)`, `getLastMessage(address)`
  - Global: `getGlobalMessageCount()`, `getGlobalLastMessage()`

Note: there is currently **no view method to query remaining credits**. The UI can only show message counts, and infer unlock success from the tx receipt + ability to store messages.

## NFT requirements
- Wrapper: MirrorNFT (address loaded from `contracts/deployed-addresses.json`)
- Mint
  - UI action: `mint(tokenId, uri)`
- Transfer
  - UI action: `transferFrom(from, to, tokenId)`
  - UI accepts `to` as 0x or mirror1 (mirror1 is converted client-side)
- Queries
  - `ownerOf(tokenId)` returns both owner formats + exists flag
  - `balanceOf(owner)`
  - `tokenURI(tokenId)`
  - `tokensOfOwner(owner)`

## Debug / logs
- UI keeps a live log stream of:
  - wallet connect events
  - chain add/switch events
  - tx send/confirm + receipts
  - read failures and revert reasons
- Controls:
  - clear logs
  - copy logs
  - auto-scroll toggle

## Acceptance criteria (matches backend tests)
- Connect wallet → shows correct chainId + non-zero MVLT balance (prefunded account)
- `payToUnlock()` succeeds with 1 MVLT
- `storeMessage()` increments message count and `getLastMessage()` returns exact string
- NFT mint succeeds, owner matches wallet, transfer succeeds to a new address
