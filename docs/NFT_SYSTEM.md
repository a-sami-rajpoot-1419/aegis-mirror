# Mirror Vault NFT System - Implementation Guide

**Last Updated**: 2025-02-12  
**Purpose**: Detailed specification for ERC721-compatible NFT system with dual-wallet support

---

## Overview

The Mirror Vault NFT system provides ERC721-compatible NFTs that work seamlessly across both MetaMask (EVM) and Keplr (Cosmos) wallets. All NFT state is stored in the Cosmos x/nft module, with a precompile at `0x0102` bridging EVM calls to Cosmos state.

**Key Features**:
- ✅ ERC721 standard compliance
- ✅ Open minting (anyone can mint)
- ✅ Dual-wallet support (MetaMask + Keplr)
- ✅ Dual address in all responses (0x + mirror1)
- ✅ Full transfer support from both wallets
- ✅ Single source of truth (Cosmos x/nft module)

---

## Architecture Layers

### Layer 1: Storage (Cosmos x/nft Module)

**Location**: `chain/x/nft/`

**State Structure**:
```go
type NFT struct {
    TokenId   uint64    // Unique identifier
    Owner     string    // Cosmos address (mirror1... format)
    TokenURI  string    // Metadata location (IPFS/Arweave)
    MintedAt  time.Time // Mint timestamp
}

// Storage maps
nfts:       map[tokenId]NFT           // Primary NFT storage
ownerNFTs:  map[cosmosAddr][]tokenId  // Ownership index
tokenCount: uint64                    // Total minted counter
```

**Keeper Functions**:
```go
// Minting
MintNFT(ctx, tokenId, owner, uri) error
Exists(ctx, tokenId) bool

// Ownership
GetOwner(ctx, tokenId) string
GetNFTsByOwner(ctx, owner) []uint64
TransferNFT(ctx, tokenId, newOwner) error

// Queries
GetNFT(ctx, tokenId) (NFT, error)
GetTokenURI(ctx, tokenId) (string, error)
GetTotalSupply(ctx) uint64
```

**Messages** (Cosmos TX types):
```protobuf
message MsgMintNFT {
  string sender = 1;
  uint64 token_id = 2;
  string token_uri = 3;
}

message MsgTransferNFT {
  string sender = 1;
  string recipient = 2;
  uint64 token_id = 3;
}
```

---

### Layer 2: Precompile Bridge (0x0102)

**Location**: `chain/x/nft/precompile/nft.go`

**Purpose**: Translate EVM calls to Cosmos x/nft operations

**Function Mapping**:
```
EVM Call                     →  Cosmos Operation
─────────────────────────────────────────────────
mint(tokenId, uri)           →  nftKeeper.MintNFT()
transferFrom(from, to, id)   →  nftKeeper.TransferNFT()
ownerOf(tokenId)             →  nftKeeper.GetOwner() + address conversion
balanceOf(owner)             →  len(nftKeeper.GetNFTsByOwner())
tokenURI(tokenId)            →  nftKeeper.GetTokenURI()
tokensOfOwner(owner)         →  nftKeeper.GetNFTsByOwner()
```

**Address Conversion**:
```go
// EVM to Cosmos
func convertEthToCosmos(evmAddr common.Address) string {
    // Uses chain/utils/address.go
    return utils.EthAddressToBech32(evmAddr, "mirror")
}

// Cosmos to EVM
func convertCosmosToEth(cosmosAddr string) common.Address {
    // Uses chain/utils/address.go
    return utils.Bech32ToEthAddress(cosmosAddr)
}
```

**Implementation Pattern**:
```go
type NFTPrecompile struct {
    nftKeeper NFTKeeper
    bech32Prefix string
}

func (p *NFTPrecompile) Run(evm *vm.EVM, contract *vm.Contract, input []byte) ([]byte, error) {
    // Parse function selector (first 4 bytes)
    method := input[:4]
    
    switch method {
    case mintSelector:
        return p.mint(evm, input[4:])
    case transferFromSelector:
        return p.transferFrom(evm, input[4:])
    case ownerOfSelector:
        return p.ownerOf(input[4:])
    // ... other functions
    }
}
```

---

### Layer 3: Solidity Interface (MirrorNFT.sol)

**Location**: `contracts/contracts/MirrorNFT.sol` ✅ Created

**Key Functions**:

#### State-Changing
```solidity
function mint(uint256 tokenId, string calldata uri) external {
    // Calls precompile 0x0102
    // Emits: NFTMinted(tokenId, owner, ownerCosmos, uri)
}

function transferFrom(address from, address to, uint256 tokenId) external {
    // Requires: caller == from
    // Calls precompile 0x0102
    // Emits: NFTTransferred(tokenId, from, to, fromCosmos, toCosmos)
}
```

#### View Functions (Dual Address Support)
```solidity
function ownerOf(uint256 tokenId) external view returns (
    address owner,          // EVM format (0x...)
    string memory ownerCosmos, // Cosmos format (mirror1...)
    bool exists
)

function getNFT(uint256 tokenId) external view returns (
    address owner,
    string memory ownerCosmos,
    string memory uri,
    bool exists
)
```

---

## Data Flow Examples

### Example 1: Mint NFT via MetaMask

**User Action**: Alice calls `MirrorNFT.mint(tokenId=1, uri="ipfs://Qm...")`

**Step-by-Step**:
```
1. MetaMask        → eth_sendTransaction
2. EVM             → Execute MirrorNFT.mint()
3. MirrorNFT.sol   → CALL 0x0102.mint(1, "ipfs://...")
4. Precompile      → Convert caller 0xABC... to mirror1xyz...
5. Precompile      → Validate tokenId=1 not exists
6. x/nft Keeper    → Store NFT(id=1, owner=mirror1xyz, uri="ipfs://...")
7. Event Emitted   → NFTMinted {
                       tokenId: 1,
                       owner: "0xABC...",
                       ownerCosmos: "mirror1xyz...",
                       tokenURI: "ipfs://..."
                     }
8. MetaMask        → Transaction confirmed
9. Keplr (same acc) → NFT appears automatically (same owner)
```

**Query Result** (from MetaMask):
```javascript
await mirrorNFT.ownerOf(1)
// Returns: ["0xABC123...", "mirror1xyz789...", true]
```

**Query Result** (from Keplr gRPC):
```bash
mirrorvaultd query nft nft 1
# Returns:
# owner: mirror1xyz789...
# owner_evm: 0xABC123...
# token_uri: ipfs://...
```

---

### Example 2: Transfer NFT Cross-Pair (Alice → Bob)

**User Action**: Alice (MetaMask) transfers tokenId=1 to Bob

**Step-by-Step**:
```
1. MetaMask (Alice)  → transferFrom(0xABC, 0xDEF, 1)
2. MirrorNFT.sol     → Validate caller == from
3. MirrorNFT.sol     → CALL 0x0102.transferFrom(0xABC, 0xDEF, 1)
4. Precompile        → Convert 0xABC → mirror1xyz (Alice)
5. Precompile        → Convert 0xDEF → mirror1abc (Bob)
6. Precompile        → Validate current owner == mirror1xyz
7. x/nft Keeper      → UpdateOwner(tokenId=1, newOwner=mirror1abc)
8. Event Emitted     → NFTTransferred {
                         tokenId: 1,
                         from: "0xABC...",
                         to: "0xDEF...",
                         fromCosmos: "mirror1xyz...",
                         toCosmos: "mirror1abc..."
                       }
9. Bob's MetaMask    → NFT appears (0xDEF owns it)
10. Bob's Keplr      → NFT appears (mirror1abc owns it)
```

---

### Example 3: Mint NFT via Keplr

**User Action**: Carol submits `MsgMintNFT(tokenId=2, uri="ar://...")`

**Step-by-Step**:
```
1. Keplr (Carol)     → Sign MsgMintNFT transaction
2. Cosmos TX         → Broadcast to chain
3. x/nft Handler     → Validate tokenId=2 not exists
4. x/nft Handler     → Validate sender signature
5. x/nft Keeper      → Store NFT(id=2, owner=mirror1def, uri="ar://...")
6. Event Emitted     → coin_spent, coin_received (gas)
                     → nft_minted {
                         token_id: 2,
                         owner_cosmos: "mirror1def...",
                         owner_evm: "0xCAR012...",
                         token_uri: "ar://..."
                       }
7. Keplr (Carol)     → Transaction confirmed
8. MetaMask (Carol)  → NFT appears automatically (same owner)
```

**Query Result** (from Keplr):
```bash
mirrorvaultd query nft owner mirror1def
# Returns:
# nfts:
#   - token_id: 2
#     owner: mirror1def...
#     owner_evm: 0xCAR012...
#     token_uri: ar://...
```

**Query Result** (from MetaMask):
```javascript
await mirrorNFT.balanceOf("0xCAR012...")
// Returns: 1

await mirrorNFT.tokensOfOwner("0xCAR012...")
// Returns: [2]
```

---

## Event Schema

### EVM Events (Emitted by MirrorNFT.sol)

```solidity
event NFTMinted(
    uint256 indexed tokenId,
    address indexed owner,        // EVM format
    string ownerCosmos,           // Cosmos format
    string tokenURI
);

event NFTTransferred(
    uint256 indexed tokenId,
    address indexed from,         // EVM format
    address indexed to,           // EVM format
    string fromCosmos,            // Cosmos format
    string toCosmos               // Cosmos format
);
```

### Cosmos Events (Emitted by x/nft module)

```
EventType: nft_minted
Attributes:
  - token_id: "1"
  - owner_cosmos: "mirror1xyz..."
  - owner_evm: "0xABC..."
  - token_uri: "ipfs://..."
  - dual_address: "indexed"

EventType: nft_transferred
Attributes:
  - token_id: "1"
  - from_cosmos: "mirror1xyz..."
  - from_evm: "0xABC..."
  - to_cosmos: "mirror1abc..."
  - to_evm: "0xDEF..."
  - dual_address: "indexed"
```

---

## UI Integration

### Displaying NFTs (Frontend)

**Query NFTs for Connected Account**:
```javascript
// MetaMask connected (0xABC...)
const balance = await mirrorNFT.balanceOf(connectedAddress);
const tokenIds = await mirrorNFT.tokensOfOwner(connectedAddress);

// For each tokenId, get details with dual addresses
for (const tokenId of tokenIds) {
  const [owner, ownerCosmos, uri] = await mirrorNFT.ownerOf(tokenId);
  
  // Display in UI:
  // Token #1
  // Owner: 0xABC... (mirror1xyz...)
  // Metadata: ipfs://...
}
```

**Display Format**:
```
┌─────────────────────────────────────┐
│  My NFTs                           │
├─────────────────────────────────────┤
│  Token #1                          │
│  Owner: 0xABC... (mirror1xyz...)   │
│  URI: ipfs://Qm...                 │
│  [Transfer] [View Metadata]        │
├─────────────────────────────────────┤
│  Token #5                          │
│  Owner: 0xABC... (mirror1xyz...)   │
│  URI: ar://abc...                  │
│  [Transfer] [View Metadata]        │
└─────────────────────────────────────┘
```

### NFT Gallery View

**Show Both Address Formats**:
- Primary: Display address format of connected wallet
- Secondary: Show alternate format in parentheses
- Tooltip: "Same NFT visible in both MetaMask and Keplr"

---

## Security Considerations

### Validation Rules

**Minting**:
- ✅ TokenId must be unique (not yet minted)
- ✅ TokenURI can be any string (no format validation in v1)
- ✅ No minting restrictions (open minting)
- ✅ Minter becomes owner

**Transfer**:
- ✅ Caller must be current owner (no approval system in v1)
- ✅ Recipient can be zero address (burn)
- ✅ TokenId must exist
- ✅ Both addresses converted to Cosmos format for internal logic

### Attack Vectors & Mitigations

**TokenId Collision**:
- Risk: Two users try to mint same tokenId
- Mitigation: Atomic check-and-set in keeper (transaction isolation)

**Unauthorized Transfer**:
- Risk: User transfers NFT they don't own
- Mitigation: Ownership validation in precompile before state change

**Address Conversion Attacks**:
- Risk: Malformed addresses causing invalid conversions
- Mitigation: Validation in utils/address.go, return error on invalid input

---

## Testing Checklist

### Unit Tests (Go)

- [ ] x/nft keeper: MintNFT with valid data
- [ ] x/nft keeper: MintNFT fails for duplicate tokenId
- [ ] x/nft keeper: TransferNFT updates ownership
- [ ] x/nft keeper: GetNFTsByOwner returns correct list
- [ ] Precompile: mint() converts addresses correctly
- [ ] Precompile: ownerOf() returns dual addresses
- [ ] Precompile: transferFrom() validates ownership

### Integration Tests (Solidity + Chain)

- [ ] Deploy MirrorNFT.sol via Hardhat
- [ ] Mint NFT via MetaMask → query via Keplr (same NFT)
- [ ] Mint NFT via Keplr → query via MetaMask (same NFT)
- [ ] Transfer via MetaMask → verify ownership in Keplr
- [ ] Transfer via Keplr → verify ownership in MetaMask
- [ ] Query ownerOf() returns both address formats
- [ ] Events contain dual addresses

### End-to-End Tests (UI + Wallets)

- [ ] Connect MetaMask → mint NFT → see in UI
- [ ] Connect Keplr (same key) → see same NFT
- [ ] Transfer to different account → recipient sees in both wallets
- [ ] Query NFT gallery shows all owned NFTs
- [ ] Metadata display works (IPFS/Arweave fetch)

---

## Implementation Order

1. **x/nft module** (3-4 hours)
   - Keeper functions
   - Message handlers (MsgMintNFT, MsgTransferNFT)
   - Genesis state
   - Query services

2. **0x0102 precompile** (2-3 hours)
   - Function dispatcher
   - mint() implementation
   - transferFrom() implementation
   - View functions (ownerOf, balanceOf, etc.)
   - Address conversion integration

3. **MirrorNFT.sol** ✅ Already created
   - Deploy via Hardhat
   - Test all functions

4. **Frontend integration** (4-5 hours)
   - NFT gallery component
   - Mint UI
   - Transfer UI
   - Dual address display

---

## References

- ERC721 Standard: https://eips.ethereum.org/EIPS/eip-721
- Cosmos NFT Module: https://github.com/cosmos/cosmos-sdk/tree/main/x/nft
- Precompile Pattern: Based on cosmos/evm precompile architecture
- Dual Address Indexing: See `docs/DUAL_ADDRESS_INDEXING.md`
