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
- Precompile address: `0x0000000000000000000000000000000000000101`
- Semantics: calling unlock increments `StorageCredit` for the caller

## Vault module
- State per address
  - `storageCredits: uint64`
  - `messageCount: uint64`
  - `lastMessage: string`
- `MsgStoreSecret` consumes 1 credit per message
