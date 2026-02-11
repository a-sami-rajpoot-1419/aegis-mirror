# The Mirror Vault

Hybrid Cosmos + EVM L1 with unified identity mapping (one key → `0x…` + `mirror1…`).

## What’s in this repo
- `chain/` — Cosmos SDK chain (custom blockchain binary). **Requires Go**.
- `contracts/` — Solidity contracts (Hardhat).
- `frontend/` — Next.js dashboard.
- `docs/` — frozen constants + dev flow.

## Frozen v1 constants
See [docs/constants.md](docs/constants.md).

## Status
- Solidity scaffolding is in place.
- Chain scaffolding is in place under `chain/` (WSL2 Ubuntu recommended).

## Next step
- In WSL2 Ubuntu, source `tools/env.sh` to use the repo-local toolchain.
- Start the chain with `cd chain && ignite chain serve`.
