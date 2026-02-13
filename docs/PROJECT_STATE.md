# The Mirror Vault — Project State (Decisions + Scope + Build Plan)

Last updated: 2026-02-11

This document is the handoff for continuing development in a fresh Copilot chat.
It records the agreed scope, architecture, frozen constants, and the exact build sequence.

**For detailed implementation status, see [IMPLEMENTATION.md](IMPLEMENTATION.md).**

## 0) Current status

### ✅ Completed (Phase 1: Foundation)
- Repo structure is in place:
  - `chain/` — Ignite-scaffolded Cosmos SDK chain, **fully operational in WSL2**
  - `contracts/` — Hardhat + Solidity scaffolding present
  - `frontend/` — to be scaffolded
  - `docs/` — constants, dev flow, project state, and implementation guide
  - `tools/` — environment setup and safe build scripts
- Chain implementation:
  - **Cosmos SDK v0.53.5** integrated and tested
  - **CometBFT v0.38.19** producing blocks
  - Binary `mirrorvaultd` builds successfully via `tools/chain-build-safe.sh`
  - Genesis initialized with chain-id `mirror-vault-localnet`
  - 3 test accounts (alice, bob, carol) funded with 1B umvlt each
  - **Bank transactions confirmed**: Smoke test passed (alice→bob transfer included at height 2225)
  - REST API and RPC endpoints operational
- Development environment:
  - WSL2 Ubuntu-22.04 with Go 1.25.7, Node.js v23.6.0, Ignite CLI v28.6.1
  - User-local toolchain (no sudo required)
  - Environment script (`tools/env.sh`) standardizes PATH

### ⚠️ In Progress (Phase 1: EVM Integration)
- **Manual wiring migration planning**: COMPLETE ✅
  - Research completed: 3 integration options analyzed
  - Decision: Manual keeper initialization (hybrid approach)
  - Documentation: MANUAL_WIRING_MIGRATION_PLAN.md created
  - Next: Implementation pending approval
- **EVM Integration Status**:
  - Dependencies: cosmos/evm v0.5.0 configured ✅
  - Compilation: All errors fixed ✅
  - Runtime: Blocked by depinject CustomGetSigner issue
  - Solution: Migrate to manual wiring (evmd pattern)

### 🔴 Not Started (Phase 1b: Post-Migration)
- Execute manual wiring migration (5-7 hours estimated)
- Test JSON-RPC endpoints (port 8545)
- Validate MetaMask connectivity
- Test unified identity (Keplr + MetaMask with same key)

### 🔴 Not Started (Phase 2: IBC Integration)
- IBC keeper initialization
- Transfer module integration
- Update Erc20Keeper with TransferKeeper

### 🔴 Not Started (Phase 3: Business Logic)
- x/vault custom module
- Stateful precompile (0x0101)
- Solidity contract deployment and testing

## 1) What we are building (scope)

A sovereign Layer-1 blockchain (“Mirror Vault”) built with:
- Cosmos SDK state machine
- CometBFT consensus
- Cosmos EVM execution (cosmos/evm family)

The differentiator is the “Mirror” identity principle:
- One private key controls a single underlying on-chain account.
- That account is represented simultaneously as:
  - 0x… (EVM hex address)
  - mirror1… (Cosmos bech32 address)

This is not “just deploying smart contracts” — it is building a new chain binary (validators run it) with custom modules and EVM integration.

### v1 success demonstration
1) Unified balance/state:
- Sending coins via Keplr or MetaMask updates the same underlying balance.
- Any account (pair) can send to any other account using either wallet interface.

2) Cross-VM business logic:
- A Solidity contract call (MetaMask) triggers a Go precompile.
- The precompile updates native Cosmos module state.
- A Cosmos message (Keplr) is gated by that state and succeeds only after unlock.

## 2) Frozen constants (do not change in v1)

Single source of truth: docs/constants.md

Key points:
- chain-id: mirror-vault-localnet
- bech32 prefix: mirror
- coin type (BIP-44): 60
- native denom: umvlt (display MVLT)
- EVM chainId: 7777
- EVM JSON-RPC: localhost:8545
- Cosmos REST (LCD): localhost:1317
- Precompile: 0x0000000000000000000000000000000000000101

## 3) Architecture (high-level)

### Components
- Chain node (custom)
  - Cosmos SDK app
  - CometBFT consensus
  - EVM module exposing JSON-RPC
  - Custom module: **x/vault** (stores messages, manages credits)
  - Custom module: **x/nft** (ERC721-compatible NFT storage)
  - Custom bridges: stateful precompiles
    - **0x0101** (Message storage): unlock(), storeMessage(string), getMessageCount(), getLastMessage()
    - **0x0102** (NFT system): mint(), transfer(), ownerOf(), balanceOf(), tokenURI()
  - **Dual address indexing**: ante decorator emits both 0x and mirror1 formats in every transaction event

- Solidity (Hardhat)
  - **VaultGate.sol** exposes message storage precompile (0x0101):
    - `payToUnlock()`: calls precompile to grant credit
    - `storeMessage(string)`: calls precompile to store message
    - `getMessageCount(address)`: view function via precompile
    - `getLastMessage(address)`: view function via precompile
  - **MirrorNFT.sol** exposes NFT precompile (0x0102) - ERC721 compatible:
    - `mint(uint256 tokenId, string uri)`: mint NFT via precompile
    - `transferFrom(address from, address to, uint256 tokenId)`: transfer NFT
    - `ownerOf(uint256 tokenId)`: returns owner with dual addresses
    - `balanceOf(address owner)`: returns NFT count for address
    - `tokenURI(uint256 tokenId)`: returns metadata URI

- Frontend (Next.js)
  - Pro UI, single-page split view
  - Connect MetaMask + Keplr
  - Show both address formats (0x + mirror1) for connected account
  - Send coin actions (EVM + Cosmos)
  - Vault actions: store message from EITHER wallet with fee comparison
  - Global state display: messageCount and lastMessage (same for all users)
  - Logs/debug bar (shows dual addresses in transaction events)

### Business logic (v1)

**Feature 1: Message Storage (Credit-Gated)**
- Storage credit model: counter-based credits per address
  - `payToUnlock()`: +1 credit (MetaMask only for v1)
  - `storeMessage()`: Both wallets can store:
    - **MetaMask**: VaultGate.storeMessage() → precompile 0x0101 → x/vault
    - **Keplr**: MsgStoreSecret → x/vault directly
  - Both paths: require credit > 0, consume 1 credit, update global state
- Global state (visible to ALL accounts):
  - `messageCount`: total messages stored chain-wide
  - `lastMessage`: most recent message stored by anyone
- Per-address state:
  - `StorageCredit[address]`: credits available for that address

**Feature 2: NFT System (Open Minting)**
- ERC721-compatible NFT standard (tokenId + tokenURI)
- Open minting: anyone can mint NFTs (no restrictions)
- Both wallets can mint and transfer:
  - **MetaMask**: MirrorNFT.mint() / transferFrom() → precompile 0x0102 → x/nft
  - **Keplr**: MsgMintNFT / MsgTransferNFT → x/nft directly
- Storage: Cosmos x/nft module (single source of truth)
- Per-NFT state:
  - `NFTs[tokenId]`: owner (mirror1... format), tokenURI, mintedAt
- Per-address state:
  - `ownedNFTs[address]`: array of tokenIds owned
- **Dual address responses**: All queries return owner in BOTH formats (0x + mirror1)

## 4) End-to-end flows (v1)

### A) “Pairing” (identity demonstration)
- User imports the same mnemonic into:
  - MetaMask (EVM)
  - Keplr (Cosmos)
- The UI displays:
  - 0x… address
  - mirror1… address
- The chain configuration ensures these are the same underlying identity.

### B) Coin send/receive (unified balance demonstration)
- Keplr → Keplr: Cosmos MsgSend mirror1… to mirror1…
- MetaMask → MetaMask: EVM native value transfer 0x… to 0x…
- Cross-interface consistency: whichever wallet receives, the other wallet for that same account shows the updated balance.

### C) Vault unlock + store (cross-VM business logic)
1) MetaMask calls VaultGate.payToUnlock()
2) Contract calls precompile at 0x…0101
3) Precompile increments StorageCredit for msg.sender in x/vault
4) Keplr calls MsgStoreSecret(text)
5) x/vault checks credit, consumes 1, writes message, updates count + last

## 5) UI decisions

- Style: “pro/modern” (GitHub-dark inspired).
- Mode: single-user mode (one connected pair at a time).
  - Multi-pair demos are done by switching accounts or using separate browser profiles.
- Wallet testing setup:
  - Need 3 pairs for demo:
    - Pair A mnemonic imported into Keplr A + MetaMask A
    - Pair B mnemonic imported into Keplr B + MetaMask B
    - Pair C mnemonic imported into Keplr C + MetaMask C
  - Total 6 wallet instances, but 3 underlying accounts.
- Genesis prefund: yes (for quick local testing)

## 6) Tooling decisions

### Windows vs WSL
- Contracts + frontend can run on Windows natively.
- Chain scaffolding/build/run will be done inside WSL2 Ubuntu-22.04.

### Docker
- Keep Docker out for v1.
- All chain build/run happens in WSL2 Ubuntu with user-level tooling and `ignite chain serve`.

### Chain base
- Chain foundation is an Ignite scaffold that we customize and extend.
- We integrate `github.com/cosmos/evm` into the scaffold (we are not using Evmos).

## 7) Planned implementation sequence (no repeats)

This is the authoritative order. Each step has a “Done when” check.

1) Scaffold chain project in WSL
- Done when: chain project exists under chain/ and builds.

2) Configure mirror identity (coin type 60 + Eth keys/accounts)
- Done when: same key produces consistent 0x… + mirror1… mapping.

3) Enable EVM JSON-RPC + denom mapping
- Done when: localhost:8545 responds and EVM uses umvlt as the value denom.

4) Genesis prefund A/B/C
- Done when: the 3 generated accounts start with balances and can send.

5) Implement x/vault (message store)
- Done when: MsgStoreSecret stores and query returns count + last.

6) Add StorageCredit gating (consume per message)
- Done when: store fails at 0 credit and succeeds after credit granted.

7) Implement stateful precompile 0x…0101
- Done when: EVM call increments StorageCredit for the caller.

8) Solidity contract (VaultGate.sol)
- Done when: MetaMask tx to payToUnlock causes credit increment.

9) Frontend dashboard
- Done when:
  - connect MetaMask + Keplr works
  - coin send via both paths works
  - unlock then store message works
  - UI shows messageCount + lastMessage

10) Demo script + documentation
- Done when: a new developer can reproduce the full demo.

## 10) Frontend v1 requirements (baked in early to avoid churn)

These are part of v1 scope and should be implemented without changing frozen constants.

### A) Automated wallet provisioning (silent setup)
- Keplr: use `window.keplr.experimentalSuggestChain`.
  - On connect, detect missing chain (e.g., failing `getOfflineSigner(chainId)`), then suggest it.
  - Use RPC `http://localhost:26657` and REST `http://localhost:1317`.
- MetaMask: use `wallet_addEthereumChain` and switch automatically.
  - JSON-RPC: `http://localhost:8545`
  - chainId: 7777 (hex-encoded for MetaMask request)
  - currency symbol: MVLT

### B) Token operations tab (send/receive)
- UI must support sending and receiving tokens for both interfaces:
  - Cosmos send (Keplr/cosmjs): `MsgSend`.
  - EVM send (MetaMask/ethers): native value transfer.

### C) Unified faucet button (local dev tool)
- UI includes a faucet button to fund the connected account.
- Implementation is local-only: Next.js API route shells out to `mirrorvaultd tx bank send` from the local validator to the connected address.
- Funding `mirror1...` is equivalent to funding `0x...` due to unified identity.

### D) Fee comparison engine (gas oracle)
- When preparing a send/unlock/store action, UI shows side-by-side estimated fees:
  - EVM: `eth_estimateGas`.
  - Cosmos: REST `simulate` for the equivalent Cosmos tx.

### E) Global chain state display
- Display global counters from `x/vault` (same for ALL users):
  - Total message count (chain-wide)
  - Last message preview (most recent from any user)
- Balance display: show balance in both MVLT and wei formats
- **Dual address display**: Show both 0x and mirror1 format for connected account
- Transaction logs: Display both address formats in event data

### F) Debug bar + notifications
- Debug bar streams only UI-level activity (wallet connected, tx submitted/confirmed, errors) and relevant hashes.
- Add floating notifications (toast-style) that auto-dismiss.

## 8) How to continue in WSL (next session)

1) Verify distro name:
- In PowerShell: wsl -l -v
- Use Ubuntu-22.04 in commands.

2) Open VS Code Remote:
- Use the VS Code “WSL” remote to open the repo inside Ubuntu.

3) Put the repo in Linux filesystem (recommended)
- From WSL:
  - Create a workspace folder under /home/<user>/
  - Either clone the repo there, or copy from /mnt/c/...

4) Install toolchain in WSL
- Install Go in WSL (separate from Windows Go)
- Install Ignite in WSL (Linux works)

5) Scaffold chain
- Use ignite scaffold chain ... with address prefix mirror

---

## 9) Notes for future agents

- Goal is a working localnet demo: 3 pairs, unified balance, coin sends, and cross-VM unlock → store message.
- Prefer simple/deterministic mechanisms:
  - precompile bridge (not EndBlocker/event scanning)
  - native coin only for v1
- Keep constants stable (docs/constants.md).
