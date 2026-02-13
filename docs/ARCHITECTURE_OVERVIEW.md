# The Mirror Vault — Architecture Overview

**Last Updated**: 2025-02-12  
**Purpose**: High-level architectural decisions and design principles

---

## Core Principles

### 1. Unified Identity (The "Mirror" Concept)

**Two Features, Same Pattern**: Message Storage + NFT System

**One Private Key = One Account**
- Same cryptographic key produces two address representations:
  - **0x...** (EVM/Ethereum format) for MetaMask
  - **mirror1...** (Cosmos Bech32 format) for Keplr
- Same balance, same state, same identity
- No "pairing" or "linking" required — mathematically identical

**Implementation**:
- BIP-44 Coin Type: **60** (Ethereum standard)
- Key Type: **EthSecp256k1** (Ethereum curve)
- Address derivation: Same public key hash → encoded as hex (0x) or bech32 (mirror1)

---

### 2. Dual Address Indexing

**All Transactions Emit Both Formats**

**Why**: Enable seamless cross-wallet visibility and querying

**Implementation**:
- **Location**: `chain/ante/dual_address_decorator.go`
- **Trigger**: Ante handler intercepts every transaction (Cosmos and EVM)
- **Event Emission**: Each transaction emits:
  ```
  dual_address_index {
    evm_address: "0x..."
    cosmos_address: "mirror1..."
  }
  ```

**Benefits**:
- Block explorers can index by either format
- Users can search transactions using whichever address format they have
- UI can display both formats for transparency
- No address conversion needed at query time

**Utility Functions** (`chain/utils/address.go`):
- `Bech32ToEthAddress()` — mirror1... → 0x...
- `EthAddressToBech32()` — 0x... → mirror1...
- `SDKAddressToBothFormats()` — Get both from SDK address

---

### 3. Global State Architecture

**Message Storage is Chain-Wide, Not Per-User Conversation**

**Design Decision**:
- `messageCount`: Total messages stored by ALL users (global counter)
- `lastMessage`: Most recent message from any user (global state)
- `StorageCredit[address]`: Per-address credit balance (gating mechanism)

**Why Global**:
- Demonstrates unified blockchain state
- All accounts (regardless of wallet) see the same chain state
- No siloed data between wallets
- Simpler v1 implementation (no user-to-user messaging yet)

**Future Enhancement** (Post-v1):
- Per-user message history
- User-to-user encrypted messaging
- Message threads

---

### 4. Dual-Wallet Message Storage

**Both MetaMask and Keplr Can Store Messages**

**Design Decision**: User choice via fee comparison

#### MetaMask Path (EVM → Cosmos)
```
User → VaultGate.storeMessage(text) → Precompile 0x0101 → x/vault keeper
```
1. User calls `VaultGate.sol` contract from MetaMask
2. Contract makes low-level call to precompile at `0x0101`
3. Precompile converts EVM address → Cosmos address
4. Precompile checks `StorageCredit[address] > 0`
5. Precompile calls `x/vault.StoreMessage()`
6. Keeper decrements credit, stores message, updates global state

#### Keplr Path (Cosmos Native)
```
User → MsgStoreSecret → x/vault keeper
```
1. User submits `MsgStoreSecret` transaction via Keplr
2. x/vault message handler validates sender
3. Handler checks `StorageCredit[address] > 0`
4. Handler stores message, updates global state

**Result**: Same on-chain state, different execution paths

**Fee Comparison** (UI Feature):
- Show gas estimate for both paths
- User decides which wallet to use based on cost
- Typically: Cosmos native is cheaper

---

### 5. NFT System Architecture

**ERC721-Compatible NFTs with Dual-Wallet Support**

**Design Decision**: Open minting, full transfer support, dual address in all responses

#### Storage & Standard
- **Standard**: ERC721 (tokenId + tokenURI)
- **Storage**: Cosmos x/nft module (single source of truth)
- **Minting**: Open to anyone (no restrictions)
- **Transfer**: Full support from both MetaMask and Keplr
- **Collection**: Single global collection (v1)

#### MetaMask Path (EVM → Cosmos)
```
User → MirrorNFT.mint(tokenId, uri) → Precompile 0x0102 → x/nft keeper
```
1. User calls `MirrorNFT.sol` contract from MetaMask
2. Contract calls precompile at `0x0102`
3. Precompile converts caller 0x → mirror1
4. Precompile validates tokenId not yet minted
5. Precompile stores NFT in x/nft module
6. Event emitted with BOTH address formats

**Query Response**:
```javascript
// MetaMask calls: MirrorNFT.ownerOf(tokenId)
{
  owner: "0xABC123...",           // EVM format
  ownerCosmos: "mirror1xyz789...", // Cosmos format
  tokenURI: "ipfs://Qm...",
  exists: true
}
```

#### Keplr Path (Cosmos Native)
```
User → MsgMintNFT(tokenId, uri) → x/nft keeper
```
1. User submits `MsgMintNFT` transaction via Keplr
2. x/nft message handler validates tokenId unique
3. Handler stores NFT with owner as mirror1... format
4. Event emitted with BOTH address formats

**Query Response**:
```javascript
// Keplr queries: QueryNFT(tokenId)
{
  owner: "mirror1xyz789...",      // Cosmos format
  ownerEvm: "0xABC123...",        // EVM format
  tokenURI: "ipfs://Qm...",
  exists: true
}
```

#### Transfer Flow (Cross-Pair)
**MetaMask Transfer**:
```
Alice (0xABC) → MirrorNFT.transferFrom(from=0xABC, to=0xDEF, tokenId=1)
  → Precompile 0x0102
  → Converts addresses to mirror1 format
  → x/nft.UpdateOwner(tokenId=1, owner=mirror1def)
  → Event with dual addresses (from + to)
```

**Keplr Transfer**:
```
Alice (mirror1xyz) → MsgTransferNFT(tokenId=1, recipient=mirror1def)
  → x/nft.UpdateOwner(tokenId=1, owner=mirror1def)
  → Event with dual addresses
```

**Result**: Bob receives NFT visible in BOTH his MetaMask (0xDEF) and Keplr (mirror1def)

---

### 6. Precompile Architecture (0x0101 + 0x0102)

**Stateful Precompile = EVM ↔ Cosmos Bridge**

**Two Precompiles for Two Features**

#### Precompile 0x0101 (Message Storage)

**Address**: `0x0000000000000000000000000000000000000101`

**Purpose**: Allow EVM transactions to read/write Cosmos x/vault state

**Interface** (4 functions):
```solidity
interface IVaultPrecompile {
    // State-changing functions
    function unlock() external returns (bool);
    function storeMessage(string calldata message) external returns (bool);
    
    // View functions (read-only)
    function getMessageCount(address user) external view returns (uint256);
    function getLastMessage(address user) external view returns (string memory);
}
```

**Implementation**: Go code in `chain/x/vault/precompile/vault.go`

**Key Design**:
- **unlock()**: Only grants credits (no message storage)
- **storeMessage()**: Stores message AND consumes credit
- View functions enable MetaMask to query Cosmos state directly

#### Precompile 0x0102 (NFT System)

**Address**: `0x0000000000000000000000000000000000000102`

**Purpose**: Allow EVM transactions to mint/transfer/query NFTs in Cosmos x/nft

**Interface** (6 functions):
```solidity
interface IERC721Precompile {
    // State-changing functions
    function mint(uint256 tokenId, string calldata uri) external returns (bool);
    function transferFrom(address from, address to, uint256 tokenId) external returns (bool);
    
    // View functions (with dual address support)
    function ownerOf(uint256 tokenId) external view returns (address owner, string memory ownerCosmos);
    function balanceOf(address owner) external view returns (uint256 balance);
    function tokenURI(uint256 tokenId) external view returns (string memory uri);
    function tokensOfOwner(address owner) external view returns (uint256[] memory tokenIds);
}
```

**Implementation**: Go code in `chain/x/nft/precompile/nft.go`

**Key Design**:
- **mint()**: Open minting, stores in x/nft with owner converted to mirror1
- **transferFrom()**: Updates ownership in x/nft
- **ownerOf()**: Returns BOTH address formats (dual address support)
- All view functions query x/nft Cosmos state

**Registration**: Both precompiles registered in `chain/app/app.go` during EVM keeper initialization

---

### 6. Credit Gating System

**Purpose**: Demonstrate cross-VM state dependency

**Flow**:
1. User must call `payToUnlock()` (MetaMask only for v1)
2. Grants +1 `StorageCredit` for that address
3. User can then store message via EITHER wallet
4. Each message consumes 1 credit

**Why MetaMask-Only for Unlock**:
- Preserves cross-VM demonstration (EVM action enables Cosmos action)
- Shows precompile functionality
- Can be expanded in v2 to allow Cosmos-side credit purchases

---

## Data Flow Diagrams

### Message Storage (MetaMask Path)
```
┌──────────┐         ┌─────────────────┐         ┌──────────────┐         ┌────────────┐
│ MetaMask │ ──────> │ VaultGate.sol   │ ──────> │ Precompile   │ ──────> │  x/vault   │
│  (User)  │  call   │ storeMessage()  │  call   │   0x0101     │  write  │  keeper    │
└──────────┘         └─────────────────┘         └──────────────┘         └────────────┘
                                                         │
                                                         ▼
                                                  Check credit > 0
                                                  Decrement credit
                                                  Store message
                                                  Update global state
```

### Message Storage (Keplr Path)
```
┌──────────┐         ┌──────────────────┐         ┌────────────┐
│  Keplr   │ ──────> │ MsgStoreSecret   │ ──────> │  x/vault   │
│  (User)  │  sign   │   (Cosmos msg)   │  handle │  keeper    │
└──────────┘         └──────────────────┘         └────────────┘
                                                         │
                                                         ▼
                                                  Check credit > 0
                                                  Decrement credit
                                                  Store message
                                                  Update global state
```

### NFT Minting (MetaMask Path)
```
┌──────────┐         ┌─────────────────┐         ┌──────────────┐         ┌────────────┐
│ MetaMask │ ──────> │ MirrorNFT.sol   │ ──────> │ Precompile   │ ──────> │   x/nft    │
│  (User)  │  call   │ mint(id, uri)   │  call   │   0x0102     │  write  │  keeper    │
└──────────┘         └─────────────────┘         └──────────────┘         └────────────┘
                                                         │
                                                         ▼
                                                  Convert 0x → mirror1
                                                  Validate tokenId unique
                                                  Store NFT(id, owner, uri)
                                                  Emit event (dual addresses)
```

### NFT Minting (Keplr Path)
```
┌──────────┐         ┌──────────────────┐         ┌────────────┐
│  Keplr   │ ──────> │  MsgMintNFT      │ ──────> │   x/nft    │
│  (User)  │  sign   │ (id, uri)        │  handle │  keeper    │
└──────────┘         └──────────────────┘         └────────────┘
                                                         │
                                                         ▼
                                                  Validate tokenId unique
                                                  Store NFT(id, owner, uri)
                                                  Emit event (dual addresses)
```

### NFT Query (With Dual Addresses)
```
┌──────────┐         ┌─────────────────┐         ┌──────────────┐         ┌────────────┐
│ MetaMask │ ──────> │ MirrorNFT.sol   │ ──────> │ Precompile   │ ──────> │   x/nft    │
│  Query   │  view   │ ownerOf(id)     │  static │   0x0102     │  read   │  keeper    │
└──────────┘         └─────────────────┘         └──────────────┘         └────────────┘
                                                         │
                                                         ▼
                                                  Query NFT owner (mirror1...)
                                                  Convert mirror1 → 0x
                                                  Return: {owner: 0x, ownerCosmos: mirror1}
```

### Dual Address Indexing
```
┌────────────────┐
│  Transaction   │
│   (Any Type)   │
└───────┬────────┘
        │
        ▼
┌───────────────────────┐
│  Ante Handler Chain   │
│  (Before Execution)   │
└───────┬───────────────┘
        │
        ▼
┌──────────────────────────┐
│ DualAddressDecorator     │
│ - Extract addresses      │
│ - Convert formats        │
│ - Emit events            │
└───────┬──────────────────┘
        │
        ▼
┌────────────────────────────────┐
│  Event Emitted:                │
│  dual_address_index {          │
│    evm_address: "0x..."        │
│    cosmos_address: "mirror1..."│
│  }                             │
└────────────────────────────────┘
```

---

## Key Files

### Implementation Files
- `chain/app/app.go` — Keeper wiring, module registration
- `chain/ante/dual_address_decorator.go` — Dual indexing decorator
- `chain/ante/handler_options.go` — Ante handler config
- `chain/utils/address.go` — Address conversion utilities

### To Be Implemented (Message Storage)
- `chain/x/vault/keeper/` — Credit + message storage logic
- `chain/x/vault/types/` — Protobuf messages (MsgStoreSecret)
- `chain/x/vault/precompile/` — 0x0101 precompile implementation
- `contracts/contracts/VaultGate.sol` — Solidity interface ✅ Created

### To Be Implemented (NFT System)
- `chain/x/nft/keeper/` — NFT storage + ownership logic
- `chain/x/nft/types/` — Protobuf messages (MsgMintNFT, MsgTransferNFT)
- `chain/x/nft/precompile/` — 0x0102 precompile implementation
- `contracts/contracts/MirrorNFT.sol` — ERC721 interface ✅ Created

### Documentation
- `docs/PROJECT_STATE.md` — High-level scope and business logic
- `docs/IMPLEMENTATION.md` — Technical implementation details
- `docs/constants.md` — Frozen configuration values
- `docs/DUAL_ADDRESS_INDEXING.md` — Dual indexing implementation guide

---

## Design Rationale

### Why Option B (Precompile Storage) Over Option A (Contract Storage)?

**Option A**: Store messages in VaultGate.sol mapping (EVM storage)
- ❌ Keplr can't access EVM contract storage
- ❌ Two separate data stores (Cosmos vs EVM)
- ❌ No unified state

**Option B**: Precompile writes to x/vault (Cosmos storage) ✅
- ✅ Single source of truth
- ✅ Both wallets access same data
- ✅ Cosmos queries work for MetaMask transactions
- ✅ True cross-VM state sharing

---

## Testing Strategy

### v1 Validation Checklist

**Identity & Indexing**:
- [ ] Generate account, verify both 0x and mirror1 formats
- [ ] Import same key to MetaMask and Keplr
- [ ] Verify balance shows identically in both wallets
- [ ] Verify transaction events contain both address formats

**Coin Transfers**:
- [ ] Send from MetaMask → receive in Keplr (same balance)
- [ ] Send from Keplr → receive in MetaMask (same balance)
- [ ] Query transaction by 0x address
- [ ] Query transaction by mirror1 address

**Message Storage (Cross-VM Workflow)**:
- [ ] Unlock via MetaMask → verify credit increased
- [ ] Store message via MetaMask → verify stored in x/vault
- [ ] Store message via Keplr → verify stored in x/vault
- [ ] Query global messageCount from both wallets
- [ ] Query lastMessage from both wallets (same result)
- [ ] Verify credit consumption (starts at 1, becomes 0 after store)

**NFT System (Dual-Wallet Minting)**:
- [ ] Mint NFT via MetaMask → verify visible in Keplr
- [ ] Mint NFT via Keplr → verify visible in MetaMask
- [ ] Query ownerOf(tokenId) via MetaMask → returns both addresses
- [ ] Query NFT via Keplr gRPC → returns both addresses
- [ ] Verify tokenURI metadata accessible from both wallets
- [ ] Check balanceOf() matches in both wallet queries

**NFT Transfers (Cross-Pair)**:
- [ ] Alice (MetaMask) mints tokenId=1
- [ ] Alice transfers to Bob via MetaMask → Bob sees in Keplr
- [ ] Bob transfers to Carol via Keplr → Carol sees in MetaMask
- [ ] Query ownership chain via both RPC endpoints
- [ ] Verify events contain dual addresses for sender + recipient

**Fee Comparison**:
- [ ] Estimate gas for MetaMask storeMessage()
- [ ] Simulate MsgStoreSecret cost via REST
- [ ] Verify UI displays both estimates
- [ ] Estimate gas for MetaMask mint()
- [ ] Simulate MsgMintNFT cost via REST
- [ ] Compare EVM vs Cosmos gas costs

---

## Future Enhancements (Post-v1)

**Message Storage Enhancements**:
1. Per-User Message History
   - Store array of messages per address
   - Query: `getMessages(address)` returns all messages from that user

2. User-to-User Messaging
   - `sendMessage(recipient, text)`
   - Encrypted messaging support

3. Credit Marketplace
   - Buy credits via Cosmos (native coin payment)
   - Transfer credits between addresses

**NFT System Enhancements**:
1. NFT Collections
   - Multiple named collections (not just single global)
   - Collection metadata (name, symbol, description)

2. Advanced NFT Features
   - Approve/setApprovalForAll (ERC721 full compatibility)
   - Royalties support (EIP-2981)
   - On-chain metadata (SVG generation)

3. NFT Marketplace
   - List NFTs for sale (EVM or Cosmos currency)
   - Auction support
   - Bid/offer system

**Cross-Feature Integration**:
4. Message-NFT Linking
   - NFTs with embedded messages
   - Message receipts as NFTs
   - NFT-gated message access

**Infrastructure**:
5. IBC Integration
   - Cross-chain message storage
   - Cross-chain NFT transfers
   - Unified identity across IBC-connected chains

6. Enhanced Precompiles
   - More complex state queries
   - Batch operations (mint multiple NFTs)
   - Gas optimization

---

## References

- Cosmos EVM Docs: https://github.com/cosmos/evm
- Cosmos SDK Docs: https://docs.cosmos.network
- EVM Precompiles: https://www.evm.codes/precompiled
- Dual Indexing Implementation: `docs/DUAL_ADDRESS_INDEXING.md`
