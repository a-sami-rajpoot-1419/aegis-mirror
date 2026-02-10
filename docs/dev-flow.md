# Dev flow (VS Code first, no Docker)

## Why Go is required
The chain is a Cosmos SDK application (a new blockchain binary). Until Go is installed and the chain builds, we can only scaffold non-Go pieces.

## Local components
- Chain node (custom `mirrorvaultd`) — requires Go
- Solidity contracts (Hardhat) — Node/npm
- Frontend dashboard (Next.js) — Node/npm

## Demo approach for 3 pairs
We demonstrate 3 *underlying* accounts (A/B/C). Each is imported into:
- MetaMask (EVM view: `0x...`)
- Keplr (Cosmos view: `mirror1...`)

Wallet apps are replaceable; the seed phrase is the durable identity.

## Recommended multi-user demo setup
Browser extensions share state across tabs. For simultaneous A/B/C:
- Use 3 browser profiles (or 3 different browsers) and import A, B, C separately.
- Open the dashboard in each profile.

Sequential switching in one profile also works for functional proof.
