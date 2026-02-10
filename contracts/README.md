# Mirror Vault — Solidity contracts

This folder contains the v1 Solidity contract(s) used to demonstrate cross-VM business logic.

## Prereqs
- Node.js + npm

## Install
From this folder:
- `npm install`

## Compile
- `npm run build`

## Deploy (after the chain EVM RPC is running)
- `npm run deploy:local`

Network settings are in `hardhat.config.ts`:
- RPC: `http://127.0.0.1:8545`
- chainId: `7777`

Contract:
- `VaultGate.sol` calls the chain precompile at `0x000...0101`.
