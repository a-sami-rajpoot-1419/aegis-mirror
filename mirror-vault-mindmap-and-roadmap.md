# The Mirror Vault — Mind Map + Build Roadmap

This document is a synthesized “single source of truth” from:
- [highlevel-system-doc.txt](highlevel-system-doc.txt)
- [system requirements and technical specifications.txt](system%20requirements%20and%20technical%20specifications.txt)
- [vscode+docker-setup.txt](vscode%2Bdocker-setup.txt)

## 1) What we are building (Mind Map)

```mermaid
mindmap
  root((The Mirror Vault))
    It's a blockchain
      Sovereign L1
      Cosmos SDK state machine
      CometBFT consensus
      Cosmos EVM execution (cosmos/evm)
    Core innovation: Unified Identity
      One private key
      Two address encodings
        EVM: 0x… (hex)
        Cosmos: mirror1… (bech32)
      Requirements
        BIP-44 coin type 60
        EthSecp256k1 keys
        Ethermint-style accounts (EthAccount)
      Result
        No internal “bridge” between own accounts
        One balance in one bank store
    Interfaces
      EVM interface
        JSON-RPC :8545
        Solidity contracts
        MetaMask + Hardhat/Foundry
      Cosmos interface
        REST :1317
        gRPC
        Keplr + cosmjs
    Business feature: Digital Safety Deposit Box
      Lock side (EVM)
        VaultGate/VaultManager contract
        payToUnlock / unlockStorage
        emits Unlocked(user)
      Bridge (Go precompile)
        stateful precompile at 0x…000101
        writes to Cosmos module store
          increments StorageCredit
          or sets is_unlocked[user]=true
      Vault side (Cosmos module x/vault)
        stores secrets (address -> text)
        MsgStoreSecret
        gate: requires unlock credit
    UX: Mirror Dashboard (Next.js)
      Connect Wallets
        MetaMask chain suggest (wallet_addEthereumChain)
        Keplr chain suggest (experimentalSuggestChain)
      Shows both addresses
        mirror1… and 0x…
        convert tool 0x -> mirror1
      Actions
        EVM: Pay to Unlock
        Cosmos: Save Secret
      Debug bar
        stream events from JSON-RPC + gRPC
        copy tx hash
        highlight errors
    DevOps / delivery
      VS Code Dev Containers
        reproducible toolchain (Go 1.21, Ignite, Node)
      Docker Compose
        chain-node
        api-proxy (nginx)
        frontend

```

## 2) How we will build it (Architecture in one paragraph)

You are not “deploying an app to a chain”; you are **building a new chain binary** (Cosmos SDK app) that runs consensus (CometBFT), exposes **both** Cosmos APIs (gRPC/REST) and Ethereum-style JSON-RPC, and embeds an EVM execution layer. Then you add custom logic at the chain level (x/vault module + stateful EVM precompile) so that a Solidity contract call can flip a permission/credit in native Cosmos state.

## 3) Why this is blockchain creation (not just smart contract deployment)

### A) “Smart contract deployment” (typical EVM dApp)
- You deploy Solidity contracts to an existing chain (Ethereum, Base, etc.).
- You **do not control**:
  - consensus,
  - validator set rules,
  - account/key type,
  - base fee / gas model,
  - native modules like bank/auth,
  - chain IDs/genesis/state storage design.
- Your app’s rules live *inside contracts*, and you inherit the chain’s identity model and transaction pipeline.

### B) “Cosmos appchain creation” (what Mirror Vault is)
- You build a **new executable** (e.g., `mirrorvaultd`) that validators run.
- You define and ship:
  - genesis state,
  - modules (bank/auth/staking + your `x/vault`),
  - account/address configuration,
  - API surface (REST/gRPC + EVM JSON-RPC),
  - and custom runtime hooks (precompiles / keepers / blockers).
- Your core innovation (unified identity) is **not a contract feature**; it’s a **chain-level account and key derivation configuration** (coin type 60 + EthSecp256k1 + EthAccount).

### C) “Pure Cosmos” vs “EVMOS/Ethermint-like” vs “Mirror Vault”
- Pure Cosmos chains typically use secp256k1 + Cosmos coin type (often 118) and have no EVM JSON-RPC.
- EVMOS/Ethermint-like chains add EVM compatibility and often use Ethereum-flavored accounts.
- Mirror Vault is in that family, but your differentiator is:
  - **explicitly guaranteeing** 1:1 mapping between `0x…` and `mirror1…` for the same key,
  - and using a **precompile + native module** workflow as the canonical business logic bridge.

## 4) Spec mismatches to resolve early (to avoid rework)

1) UI theme/colors
- One doc specifies: background `#0D1117`, primary `#58A6FF`.
- Another specifies: primary `#00f2ff`, success `#00ff88`, log bg `#050505`.

2) EVM trigger mechanism
- One place describes: Solidity emits event, and Go sync happens via EndBlocker reading events.
- Another describes: Solidity call executes a **stateful precompile** directly.

Practical note: **precompile is simpler and more deterministic** for “call -> immediate state write”. EndBlocker/event scanning is doable but adds indexing/receipt/log plumbing.

3) Library naming
- Docs mention Ethermint and also `cosmos/evm` (successor). We should pick `cosmos/evm` as the EVM library target, while still using Ethermint-style account types where appropriate.

## 5) Sequenced task list (non-repeating, with gates)

The rule: **we only move to the next task when the “Done when” is true**.

### Phase 0 — Lock the spec (1 short session)
1. Decide UI palette + layout source of truth.
   - Done when: we pick exactly one theme palette and write it into a single UI spec section.
2. Decide bridge mechanism: **Precompile-first** vs **Event/EndBlocker**.
   - Done when: one mechanism selected; the other becomes “out of scope for v1”.

### Phase 1 — Reproducible dev environment (Docker-first)
3. Add Dev Container config.
   - Done when: “Reopen in Container” gives Go 1.21+, Ignite, Node, Solidity tooling.
4. Add docker-compose skeleton for local orchestration.
   - Done when: `chain-node`, `api-proxy`, `frontend` all start and stay healthy.

### Phase 2 — Chain scaffold + identity correctness (the core innovation)
5. Scaffold chain project (Ignite / Cosmos SDK v0.50+).
   - Done when: chain builds a `mirrorvaultd` binary and can start a local node.
6. Configure address prefix + coin type 60.
   - Done when: keys are derived with coin type 60 and addresses use `mirror1…` prefix.
7. Override auth/accounts to EthAccount + EthSecp256k1.
   - Done when: for the same private key, the derived Ethereum address matches the EVM-side address, and Cosmos address is the bech32 encoding of the same underlying bytes.
8. Expose APIs.
   - Done when: REST :1317 and EVM JSON-RPC :8545 respond locally.

### Phase 3 — Native module `x/vault` (Cosmos-side storage)
9. Scaffold `x/vault` module.
   - Done when: module compiles, has keeper, store keys, and wiring in app.go.
10. Implement storage model + MsgStoreSecret.
   - Done when: a signed Cosmos tx can store/retrieve secret by address.
11. Add gating: StorageCredit or is_unlocked check.
   - Done when: storing fails without credit and succeeds with credit.

### Phase 4 — Inter-VM bridge (EVM -> Cosmos state)
12. Implement stateful precompile at `0x…000101`.
   - Done when: calling the precompile increments StorageCredit (or sets unlocked flag) in `x/vault` state.
13. Write Solidity contract (VaultGate/VaultManager) that triggers unlock.
   - Done when: a MetaMask transaction causes the Cosmos-side credit to change for the same user.

### Phase 5 — Frontend (Mirror Dashboard)
14. Wallet connection + chain suggestion.
   - Done when: “Connect Wallets” successfully onboards both MetaMask and Keplr.
15. Address mirror display + convert tool.
   - Done when: UI shows `0x…` and `mirror1…` consistently and conversion is correct.
16. EVM action: Pay/Unlock.
   - Done when: button sends tx via MetaMask and UI logs confirmation.
17. Cosmos action: Save Secret.
   - Done when: text area sends MsgStoreSecret via Keplr and confirms success.
18. Debug bar streaming + copy tx hash + error highlight.
   - Done when: logs show both EVM and Cosmos events and errors are clearly visible.

### Phase 6 — Proof, tests, docs
19. “Success metrics” validation run.
   - Done when:
     - balances are the same in both wallets (same account),
     - 0x/mirror mapping is mathematically consistent,
     - unlock via EVM enables Cosmos secret write.
20. Documentation: docker-compose + devcontainer + API docs.
   - Done when: new developer can run from README and hit :8545 and :1317.
21. Gas fee comparison (optional after core works).
   - Done when: UI can show EVM SSTORE cost vs Cosmos KVStore cost for same payload size (even if approximate at first).

---

## 6) The one-sentence “North Star” (to prevent scope drift)

Build a Cosmos SDK L1 that exposes both EVM and Cosmos interfaces while guaranteeing one-key/two-address identity, and prove it with an EVM unlock that authorizes Cosmos secret storage.
