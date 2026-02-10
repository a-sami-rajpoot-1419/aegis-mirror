# The Mirror Vault — Project State (Decisions + Scope + Build Plan)

Last updated: 2026-02-10

This document is the handoff for continuing development in a fresh Copilot chat.
It records the agreed scope, architecture, frozen constants, and the exact build sequence.

## 0) Current status

- Repo structure is in place:
  - chain/ (empty placeholder; chain not scaffolded yet)
  - contracts/ (Hardhat + Solidity scaffolding present)
  - frontend/ (placeholder)
  - docs/ (constants + dev flow + this state doc)
- Windows-native Cosmos tooling hit a blocker:
  - Ignite CLI does not compile natively on Windows due to Unix-only process APIs.
  - Decision: build/run the chain inside WSL2 Ubuntu.
- WSL2 distro available:
  - Ubuntu-22.04 (WSL2) is installed and running.
  - Note: the distro name is Ubuntu-22.04 (not Ubuntu).

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
  - Custom module: x/vault
  - Custom bridge: stateful precompile

- Solidity (Hardhat)
  - VaultGate.sol calls precompile address and emits event

- Frontend (Next.js)
  - Pro UI, single-page split view
  - Connect MetaMask + Keplr
  - Show both address formats
  - Send coin actions (EVM + Cosmos)
  - Vault actions (pay/unlock + store message)
  - Logs/debug bar

### Business logic (v1)
- Storage credit model: counter-based credits per address
  - payToUnlock(): +1 credit
  - MsgStoreSecret: requires credit > 0, consumes 1 credit, stores message
- Stored message data:
  - messageCount: total messages stored
  - lastMessage: most recent message

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
- Keep Docker out for now.
- Constraint: do not create/publish custom images.
- Note: generic Cosmos/Tendermint images cannot run the custom chain without building the chain binary.

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
