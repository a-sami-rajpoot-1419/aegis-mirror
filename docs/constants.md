# Mirror Vault v1 constants

These constants are frozen for v1 to prevent drift.

## Chain identity
- Cosmos `chain-id`: `mirror-vault-localnet`
- Bech32 prefix: `mirror` (accounts: `mirror1...`)
- Coin type (BIP-44): `60`
- Key type: `EthSecp256k1`
- Account type: `EthAccount` (Ethermint-style)

## Native coin
- Base denom: `umvlt`
- Display denom: `MVLT`

## Ports / interfaces
- EVM JSON-RPC: `http://localhost:8545`
- Cosmos REST (LCD): `http://localhost:1317`
- Cosmos gRPC: `http://localhost:9090`
- CometBFT RPC: `http://localhost:26657`

## EVM chain params
- EVM `chainId` (MetaMask numeric): `7777`

## Inter-VM bridge
- **Precompile 0x0101** (Message Storage): `0x0000000000000000000000000000000000000101`
  - Functions: unlock(), storeMessage(string), getMessageCount(address), getLastMessage(address)
- **Precompile 0x0102** (NFT System): `0x0000000000000000000000000000000000000102`
  - Functions: mint(uint256, string), transferFrom(address, address, uint256), ownerOf(uint256), balanceOf(address), tokenURI(uint256)
- Semantics: calling unlock increments `StorageCredit` for the caller

## Vault module
- State per address
  - `storageCredits: uint64`
  - `messageCount: uint64`
  - `lastMessage: string`
- `MsgStoreSecret` (Keplr) and `VaultGate.storeMessage()` (MetaMask) both consume 1 credit per message
- Global state: `messageCount` and `lastMessage` are chain-wide (visible to all accounts)
- Per-address state: `StorageCredit[address]` tracks credits for each account

### Dual Address Indexing
- **Implementation**: `chain/ante/dual_address_decorator.go`
- **Utility Functions**: `chain/utils/address.go`
  - `Bech32ToEthAddress()`: mirror1... → 0x...
  - `EthAddressToBech32()`: 0x... → mirror1...
  - `SDKAddressToBothFormats()`: Get both from SDK address
- **Event Attributes**: All transactions emit:
  - `evm_address`: 0x... format
  - `cosmos_address`: mirror1... format
  - `dual_address`: indicator attribute
- **Purpose**: Explorers and UIs can query transactions by either address format
