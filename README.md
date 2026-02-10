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
- Chain scaffolding is blocked until Go is installed.

## Next step
Install Go (1.21+) and then we scaffold the chain under `chain/`.
