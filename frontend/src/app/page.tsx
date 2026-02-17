"use client";

import { BrowserProvider, Contract, formatEther, parseEther } from "ethers";
import { useEffect, useMemo, useRef, useState } from "react";
import { MIRROR_NFT_ABI, VAULT_GATE_ABI } from "../lib/abi";
import { tryConvertAnyAddress, evmToMirrorAddress, mirrorToEvmAddress, isMirrorAddress } from "../lib/address";
import { METAMASK_CHAIN_CONFIG, EVM_CHAIN_ID_DEC, EVM_CHAIN_ID_HEX, KEPLR_CHAIN_CONFIG } from "../lib/chainConfig";
import { connectWallet, getInjectedProvider, getReadOnlyProvider, type ActiveWallet } from "../lib/wallet";

type DeployedAddresses = {
  chainId: number;
  deployer: string;
  vaultGate: string;
  mirrorNFT: string;
  updatedAt?: string;
};

type LogType = "info" | "success" | "warning" | "error";
type LogEntry = { time: string; type: LogType; message: string };

function formatUnknownError(err: unknown): string {
  if (err instanceof Error) return err.message;
  if (typeof err === "string") return err;
  if (typeof err === "number" || typeof err === "boolean" || err == null) return String(err);

  const anyErr = err as any;
  if (typeof anyErr?.shortMessage === "string" && anyErr.shortMessage) return anyErr.shortMessage;
  if (typeof anyErr?.message === "string" && anyErr.message) return anyErr.message;

  try {
    const json = JSON.stringify(err);
    if (json && json !== "{}") return json;
  } catch {
    // ignore
  }

  try {
    return String(err);
  } catch {
    return "Unknown error";
  }
}

function formatTime(date: Date): string {
  const pad2 = (n: number) => String(n).padStart(2, "0");
  return `${pad2(date.getHours())}:${pad2(date.getMinutes())}:${pad2(date.getSeconds())}`;
}

async function getFeeOverrides(provider: any): Promise<Record<string, any>> {
  const DEFAULT_PRIORITY = BigInt(1_000_000_000); // 1 gwei
  const DEFAULT_MAX = BigInt(2_000_000_000); // 2 gwei

  try {
    const feeData = await provider.getFeeData?.();
    const overrides: Record<string, any> = {};

    const maxFeePerGas: bigint | null | undefined = feeData?.maxFeePerGas;
    const maxPriorityFeePerGas: bigint | null | undefined = feeData?.maxPriorityFeePerGas;
    const gasPrice: bigint | null | undefined = feeData?.gasPrice;

    if (typeof maxFeePerGas === "bigint" && maxFeePerGas > BigInt(0)) overrides.maxFeePerGas = maxFeePerGas;
    if (typeof maxPriorityFeePerGas === "bigint" && maxPriorityFeePerGas > BigInt(0)) {
      overrides.maxPriorityFeePerGas = maxPriorityFeePerGas;
    }

    // Some providers return only legacy gasPrice.
    if (!overrides.maxFeePerGas && !overrides.maxPriorityFeePerGas) {
      if (typeof gasPrice === "bigint" && gasPrice > BigInt(0)) overrides.gasPrice = gasPrice;
    }

    // If we still have nothing (common on custom EVMs), force a safe EIP-1559 default.
    if (!overrides.gasPrice && !overrides.maxFeePerGas) {
      overrides.maxPriorityFeePerGas = DEFAULT_PRIORITY;
      overrides.maxFeePerGas = DEFAULT_MAX;
    } else if (overrides.maxFeePerGas && !overrides.maxPriorityFeePerGas) {
      overrides.maxPriorityFeePerGas = DEFAULT_PRIORITY;
    } else if (overrides.maxPriorityFeePerGas && !overrides.maxFeePerGas) {
      overrides.maxFeePerGas = overrides.maxPriorityFeePerGas * BigInt(2);
    }

    return overrides;
  } catch {
    return { maxPriorityFeePerGas: DEFAULT_PRIORITY, maxFeePerGas: DEFAULT_MAX };
  }
}

export default function Home() {
  const [deployed, setDeployed] = useState<DeployedAddresses | null>(null);
  const [deployedError, setDeployedError] = useState<string | null>(null);

  const [activeWallet, setActiveWallet] = useState<ActiveWallet | null>(null);
  const [browserProvider, setBrowserProvider] = useState<BrowserProvider | null>(null);
  const [evmAddress, setEvmAddress] = useState<string | null>(null);
  const [mirrorAddress, setMirrorAddress] = useState<string | null>(null);
  const [balanceEth, setBalanceEth] = useState<string>("0");
  const [balanceWei, setBalanceWei] = useState<string>("0");

  const [sendTo, setSendTo] = useState<string>("");
  const [sendAmount, setSendAmount] = useState<string>("");

  const [messageInput, setMessageInput] = useState<string>("");
  const [status, setStatus] = useState<{ type: LogType; text: string } | null>(null);

  const [userMsgCount, setUserMsgCount] = useState<string>("0");
  const [userLastMsg, setUserLastMsg] = useState<string>("");
  const [globalMsgCount, setGlobalMsgCount] = useState<string>("0");
  const [globalLastMsg, setGlobalLastMsg] = useState<string>("");

  const [nftTokenId, setNftTokenId] = useState<string>("");
  const [nftTokenUri, setNftTokenUri] = useState<string>("");
  const [nftQueryId, setNftQueryId] = useState<string>("");
  const [nftOwnerEvm, setNftOwnerEvm] = useState<string>("");
  const [nftOwnerCosmos, setNftOwnerCosmos] = useState<string>("");
  const [nftExists, setNftExists] = useState<boolean>(false);
  const [nftUriRead, setNftUriRead] = useState<string>("");
  const [nftBalance, setNftBalance] = useState<string>("0");
  const [nftOwnedTokenIds, setNftOwnedTokenIds] = useState<string>("");
  const [nftTransferTo, setNftTransferTo] = useState<string>("");
  const [nftTransferId, setNftTransferId] = useState<string>("");

  const [convertInput, setConvertInput] = useState<string>("");
  const conversion = useMemo(() => tryConvertAnyAddress(convertInput), [convertInput]);

  const [autoScroll, setAutoScroll] = useState<boolean>(true);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const logsRef = useRef<HTMLDivElement | null>(null);

  const [chainModalOpen, setChainModalOpen] = useState<boolean>(false);
  const [chainModalWallet, setChainModalWallet] = useState<ActiveWallet | null>(null);

  function addLog(message: string, type: LogType = "info") {
    setLogs((prev) => [...prev, { time: formatTime(new Date()), type, message }]);
  }

  function showStatus(text: string, type: LogType) {
    setStatus({ text, type });
    window.setTimeout(() => setStatus(null), 4500);
  }

  useEffect(() => {
    // Client-only: avoids SSR/client timestamp mismatches.
    setLogs([{ time: formatTime(new Date()), type: "info", message: "Connect a wallet to begin…" }]);
  }, []);

  useEffect(() => {
    if (!autoScroll) return;
    const el = logsRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, [logs, autoScroll]);

  useEffect(() => {
    (async () => {
      try {
        setDeployedError(null);
        const res = await fetch("/api/deployed-addresses", { cache: "no-store" });
        if (!res.ok) {
          const body = await res.json().catch(() => ({}));
          throw new Error(body?.message ?? `Failed to load deployed addresses (${res.status})`);
        }
        const json = (await res.json()) as DeployedAddresses;
        setDeployed(json);
        addLog(`Loaded deployed addresses (chainId ${json.chainId})`, "success");

        // Sanity check: ensure the wrapper contracts are actually deployed on the currently-running chain.
        try {
          const rpc = getReadOnlyProvider();
          const [vaultCode, nftCode] = await Promise.all([
            rpc.getCode(json.vaultGate),
            rpc.getCode(json.mirrorNFT),
          ]);
          if (vaultCode === "0x" || nftCode === "0x") {
            const msg =
              "Wrapper contracts not found on this chain (addresses have no code). Run `./setup-and-start.sh` to deploy them, or re-run `cd contracts && npm run deploy:local` on the current localnet.";
            setDeployedError(msg);
            addLog(msg, "error");
          }
        } catch {
          // ignore sanity-check failures; core UI can still render.
        }
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        setDeployedError(message);
        addLog(`Failed to load deployed addresses: ${message}`, "error");
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function refreshBalance() {
    if (!evmAddress) return;
    const provider = browserProvider ?? getReadOnlyProvider();
    const bal = await provider.getBalance(evmAddress);
    setBalanceWei(bal.toString());
    setBalanceEth(formatEther(bal));
  }

  async function sendMvlt() {
    try {
      if (!evmAddress) throw new Error("Connect a wallet first");
      if (!browserProvider) throw new Error("No wallet provider");

      const toRaw = sendTo.trim();
      if (!toRaw) throw new Error("Recipient is required");

      const amountRaw = sendAmount.trim();
      if (!amountRaw) throw new Error("Amount is required");

      const toEvm = isMirrorAddress(toRaw) ? mirrorToEvmAddress(toRaw) : toRaw;
      const value = parseEther(amountRaw);
      if (value <= BigInt(0)) throw new Error("Amount must be > 0");

      const signer = await getSigner();
      const fee = await getFeeOverrides(browserProvider);

      addLog(`Sending ${amountRaw} MVLT to ${toEvm}…`, "info");
      const tx = await signer.sendTransaction({ to: toEvm, value, gasLimit: BigInt(21000), ...fee });
      addLog(`Tx sent: ${tx.hash}`, "info");
      const receipt = await tx.wait();
      addLog(`Transfer confirmed (block ${receipt.blockNumber})`, "success");
      showStatus("Transfer confirmed", "success");
      setSendTo("");
      setSendAmount("");
      await refreshBalance();
    } catch (err) {
      const message = formatUnknownError(err);
      addLog(`transfer MVLT failed: ${message}`, "error");
      showStatus(message, "error");
    }
  }

  async function connect(wallet: ActiveWallet) {
    try {
      addLog(`Connecting to ${wallet === "metamask" ? "MetaMask" : "Keplr"}…`);

      const { provider, address } = await connectWallet(wallet);
      setActiveWallet(wallet);
      setBrowserProvider(provider);
      setEvmAddress(address);
      setMirrorAddress(evmToMirrorAddress(address));

      const bal = await provider.getBalance(address);
      setBalanceWei(bal.toString());
      setBalanceEth(formatEther(bal));

      addLog(`Connected: ${address}`, "success");
      showStatus(`Connected to ${wallet}`, "success");

      await refreshOnChainState(provider);
      await refreshNftState(provider);
    } catch (err: any) {
      const message = err instanceof Error ? err.message : String(err);
      addLog(`Connection failed: ${message}`, "error");
      showStatus(message, "error");
    }
  }

  async function disconnect() {
    addLog("Disconnected", "info");
    setActiveWallet(null);
    setBrowserProvider(null);
    setEvmAddress(null);
    setMirrorAddress(null);
    setBalanceEth("0");
    setBalanceWei("0");
  }

  function getVaultGate(providerOrSigner: any): Contract {
    if (!deployed?.vaultGate) throw new Error("VaultGate address not loaded");
    return new Contract(deployed.vaultGate, VAULT_GATE_ABI, providerOrSigner);
  }

  function getMirrorNft(providerOrSigner: any): Contract {
    if (!deployed?.mirrorNFT) throw new Error("MirrorNFT address not loaded");
    return new Contract(deployed.mirrorNFT, MIRROR_NFT_ABI, providerOrSigner);
  }

  async function getProviderForRead(): Promise<any> {
    return browserProvider ?? getReadOnlyProvider();
  }

  async function getSigner(): Promise<any> {
    if (!browserProvider) throw new Error("No wallet connected");
    return await browserProvider.getSigner();
  }

  async function refreshOnChainState(provider?: any) {
    try {
      const p = provider ?? (await getProviderForRead());
      const vault = getVaultGate(p);

      const gCount = await vault.getGlobalMessageCount();
      const gLast = await vault.getGlobalLastMessage();
      setGlobalMsgCount(gCount.toString());
      setGlobalLastMsg(gLast);

      if (evmAddress) {
        const uCount = await vault.getMessageCount(evmAddress);
        const uLast = await vault.getLastMessage(evmAddress);
        setUserMsgCount(uCount.toString());
        setUserLastMsg(uLast);
      } else {
        setUserMsgCount("0");
        setUserLastMsg("");
      }

      addLog("Refreshed vault state", "success");
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      addLog(`Failed to refresh vault state: ${message}`, "error");
    }
  }

  async function refreshNftState(provider?: any) {
    try {
      const p = provider ?? (await getProviderForRead());
      const nft = getMirrorNft(p);

      if (evmAddress) {
        const bal = await nft.balanceOf(evmAddress);
        setNftBalance(bal.toString());

        const ids = await nft.tokensOfOwner(evmAddress);
        setNftOwnedTokenIds(Array.isArray(ids) ? ids.map((x: any) => x.toString()).join(", ") : "");
      } else {
        setNftBalance("0");
        setNftOwnedTokenIds("");
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      addLog(`Failed to refresh NFT state: ${message}`, "error");
    }
  }

  async function payToUnlock() {
    try {
      if (!evmAddress) throw new Error("Connect a wallet first");

      // Pre-check balance so we can give a clear message instead of a low-level estimateGas failure.
      const min = parseEther("1");
      const bal = await (browserProvider ?? getReadOnlyProvider()).getBalance(evmAddress);
      if (bal < min) {
        const hint = deployed?.deployer ? ` Try switching to the funded deployer account: ${deployed.deployer}` : "";
        const msg = `Insufficient MVLT balance. Need at least 1 MVLT to unlock.${hint}`;
        addLog(msg, "error");
        showStatus(msg, "error");
        return;
      }

      const signer = await getSigner();
      const vault = getVaultGate(signer);
      const fee = await getFeeOverrides(browserProvider);

      addLog("Sending payToUnlock (1 MVLT)…", "info");
      const value = parseEther("1");

      let receipt: any;
      try {
        const tx = await vault.payToUnlock({ value, ...fee });
        addLog(`Tx sent: ${tx.hash}`, "info");
        receipt = await tx.wait();
      } catch (err: any) {
        // Custom chains sometimes fail eth_estimateGas; fall back to a manual gasLimit send.
        const msg = formatUnknownError(err);
        if (String(err?.code) === "CALL_EXCEPTION" || msg.includes("estimateGas")) {
          addLog(`estimateGas failed; retrying with manual gasLimit… (${msg})`, "warning");
          const txReq = await vault.payToUnlock.populateTransaction({ value, ...fee });
          txReq.gasLimit = BigInt(1_500_000);
          const tx = await signer.sendTransaction(txReq);
          addLog(`Tx sent: ${tx.hash}`, "info");
          receipt = await tx.wait();
        } else {
          throw err;
        }
      }

      addLog(`Confirmed in block ${receipt.blockNumber} (gas ${receipt.gasUsed?.toString?.() ?? "?"})`, "success");
      showStatus("Unlock payment confirmed", "success");
      await refreshBalance();
      await refreshOnChainState();
    } catch (err) {
      const message = formatUnknownError(err);
      addLog(`payToUnlock failed: ${message}`, "error");
      showStatus(message, "error");
    }
  }

  async function storeMessage() {
    const msg = messageInput.trim();
    if (!msg) {
      showStatus("Please enter a message", "warning");
      return;
    }
    if (msg.length > 256) {
      showStatus("Max 256 characters", "warning");
      return;
    }

    if (!evmAddress) {
      showStatus("Connect a wallet first", "warning");
      return;
    }


    try {
      const signer = await getSigner();
      const vault = getVaultGate(signer);
      const fee = await getFeeOverrides(browserProvider);
      addLog(`Storing message (${msg.length} chars)…`, "info");

      let receipt: any;
      try {
        const tx = await vault.storeMessage(msg, { ...fee });
        addLog(`Tx sent: ${tx.hash}`, "info");
        receipt = await tx.wait();
      } catch (err: any) {
        const msgErr = formatUnknownError(err);
        if (String(err?.code) === "CALL_EXCEPTION" || msgErr.includes("estimateGas")) {
          addLog(`estimateGas failed; retrying with manual gasLimit… (${msgErr})`, "warning");
          const txReq = await vault.storeMessage.populateTransaction(msg, { ...fee });
          txReq.gasLimit = BigInt(1_500_000);
          const tx = await signer.sendTransaction(txReq);
          addLog(`Tx sent: ${tx.hash}`, "info");
          receipt = await tx.wait();
        } else {
          throw err;
        }
      }

      addLog(`Message stored (block ${receipt.blockNumber})`, "success");
      showStatus("Message stored", "success");
      setMessageInput("");
      await refreshOnChainState();
      await refreshBalance();
    } catch (err) {
      const message = formatUnknownError(err);
      addLog(`storeMessage failed: ${message}`, "error");
      showStatus(message, "error");
    }
  }

  async function mintNft() {
    try {
      const tokenId = BigInt(nftTokenId || "0");
      if (tokenId <= BigInt(0)) {
        showStatus("Token ID must be > 0", "warning");
        return;
      }
      if (!nftTokenUri.trim()) {
        showStatus("Token URI is required", "warning");
        return;
      }

      const signer = await getSigner();
      const nft = getMirrorNft(signer);
      const fee = await getFeeOverrides(browserProvider);

      addLog(`Minting NFT ${tokenId.toString()}…`, "info");
      const tx = await nft.mint(tokenId, nftTokenUri.trim(), { ...fee });
      addLog(`Tx sent: ${tx.hash}`, "info");
      const receipt = await tx.wait();
      addLog(`Mint confirmed (block ${receipt.blockNumber})`, "success");
      showStatus("NFT minted", "success");
      await refreshNftState();
    } catch (err) {
      const message = formatUnknownError(err);
      addLog(`mint failed: ${message}`, "error");
      showStatus(message, "error");
    }
  }

  async function transferNft() {
    try {
      if (!evmAddress) throw new Error("Connect a wallet first");
      const tokenId = BigInt(nftTransferId || "0");
      if (tokenId <= BigInt(0)) {
        showStatus("Token ID must be > 0", "warning");
        return;
      }
      const toRaw = nftTransferTo.trim();
      if (!toRaw) {
        showStatus("Recipient is required", "warning");
        return;
      }

      const toEvm = isMirrorAddress(toRaw) ? mirrorToEvmAddress(toRaw) : toRaw;

      const signer = await getSigner();
      const nft = getMirrorNft(signer);
      const fee = await getFeeOverrides(browserProvider);
      addLog(`Transferring token ${tokenId.toString()} to ${toEvm}…`, "info");
      const tx = await nft.transferFrom(evmAddress, toEvm, tokenId, { ...fee });
      addLog(`Tx sent: ${tx.hash}`, "info");
      const receipt = await tx.wait();
      addLog(`Transfer confirmed (block ${receipt.blockNumber})`, "success");
      showStatus("NFT transferred", "success");
      await refreshNftState();
    } catch (err) {
      const message = formatUnknownError(err);
      addLog(`transfer failed: ${message}`, "error");
      showStatus(message, "error");
    }
  }

  async function queryNft() {
    try {
      const tokenId = BigInt(nftQueryId || "0");
      if (tokenId <= BigInt(0)) {
        showStatus("Token ID must be > 0", "warning");
        return;
      }

      const p = await getProviderForRead();
      const nft = getMirrorNft(p);
      const [owner, ownerCosmos, exists] = await nft.ownerOf(tokenId);
      setNftOwnerEvm(String(owner));
      setNftOwnerCosmos(String(ownerCosmos));
      setNftExists(Boolean(exists));

      if (exists) {
        const uri = await nft.tokenURI(tokenId);
        setNftUriRead(String(uri));
      } else {
        setNftUriRead("");
      }

      addLog(`Queried token ${tokenId.toString()}`, "success");
    } catch (err) {
      const message = formatUnknownError(err);
      addLog(`query failed: ${message}`, "error");
      showStatus(message, "error");
    }
  }

  async function addChain(wallet: ActiveWallet) {
    try {
      const injected = getInjectedProvider(wallet);
      if (!injected) throw new Error("Provider not available");

      if (wallet === "keplr") {
        try {
          await window.keplr?.experimentalSuggestChain(KEPLR_CHAIN_CONFIG);
          await window.keplr?.enable(KEPLR_CHAIN_CONFIG.chainId);
        } catch {
          // non-fatal
        }
      }

      await injected.request({ method: "wallet_addEthereumChain", params: [METAMASK_CHAIN_CONFIG] });
      addLog("Chain config sent to wallet", "success");
      showStatus("Chain added (approve in wallet)", "success");
    } catch (err) {
      const message = formatUnknownError(err);
      addLog(`add chain failed: ${message}`, "error");
      showStatus(message, "error");
    }
  }

  async function copyLogs() {
    const text = logs.map((l) => `[${l.time}] ${l.message}`).join("\n");
    await navigator.clipboard.writeText(text);
    addLog("Logs copied to clipboard", "success");
  }

  const connected = Boolean(evmAddress);

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <header className="border-b border-zinc-800">
        <div className="mx-auto max-w-6xl px-4 py-6">
          <div className="flex items-center justify-between gap-4">
            <div>
              <div className="text-2xl font-semibold tracking-tight">Mirror Vault Dashboard</div>
              <div className="text-sm text-zinc-400">Cosmos EVM dual-wallet testbed</div>
            </div>
            <div className="flex items-center gap-2">
              <div className="rounded-md border border-zinc-800 bg-zinc-900 px-3 py-2 text-xs text-zinc-300">
                <div>EVM chainId: {EVM_CHAIN_ID_DEC} ({EVM_CHAIN_ID_HEX})</div>
                <div className="truncate">VaultGate: {deployed?.vaultGate ?? "(loading…)"}</div>
                <div className="truncate">MirrorNFT: {deployed?.mirrorNFT ?? "(loading…)"}</div>
              </div>
            </div>
          </div>
          {deployedError ? (
            <div className="mt-3 rounded-md border border-red-900/40 bg-red-950/40 px-3 py-2 text-sm text-red-200">
              {deployedError}
            </div>
          ) : null}
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-6 grid gap-6">
        {status ? (
          <div
            className={`rounded-md border px-3 py-2 text-sm ${
              status.type === "success"
                ? "border-emerald-900/40 bg-emerald-950/30 text-emerald-200"
                : status.type === "warning"
                  ? "border-amber-900/40 bg-amber-950/30 text-amber-200"
                  : status.type === "error"
                    ? "border-red-900/40 bg-red-950/30 text-red-200"
                    : "border-zinc-800 bg-zinc-900 text-zinc-200"
            }`}
          >
            {status.text}
          </div>
        ) : null}

        {/* Wallet connection */}
        <section className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-4">
          <div className="flex items-center justify-between gap-3">
            <div>
              <div className="text-lg font-semibold">Wallet Connection</div>
              <div className="text-sm text-zinc-400">MetaMask + Keplr (EVM provider)</div>
            </div>
            <div className="flex flex-wrap gap-2">
              {!connected ? (
                <>
                  <button
                    onClick={() => connect("metamask")}
                    className="rounded-md bg-amber-500 px-3 py-2 text-sm font-medium text-zinc-950 hover:bg-amber-400"
                  >
                    Connect MetaMask
                  </button>
                  <button
                    onClick={() => connect("keplr")}
                    className="rounded-md bg-sky-500 px-3 py-2 text-sm font-medium text-zinc-950 hover:bg-sky-400"
                  >
                    Connect Keplr
                  </button>
                </>
              ) : (
                <>
                  <button
                    onClick={disconnect}
                    className="rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 hover:bg-zinc-800"
                  >
                    Disconnect
                  </button>
                </>
              )}
              <button
                onClick={() => {
                  setChainModalOpen(true);
                  setChainModalWallet(activeWallet ?? "metamask");
                }}
                className="rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 hover:bg-zinc-800"
                title="Manual chain add/switch help"
              >
                Manual Chain Add
              </button>
            </div>
          </div>

          <div className="mt-4 grid gap-3 md:grid-cols-2">
            <div className="rounded-md border border-zinc-800 bg-zinc-950/30 p-3">
              <div className="text-xs text-zinc-400">EVM Address</div>
              <div className="mt-1 break-all font-mono text-sm">{evmAddress ?? "Not connected"}</div>
            </div>
            <div className="rounded-md border border-zinc-800 bg-zinc-950/30 p-3">
              <div className="text-xs text-zinc-400">Cosmos Address</div>
              <div className="mt-1 break-all font-mono text-sm">{mirrorAddress ?? "Not connected"}</div>
            </div>
            <div className="rounded-md border border-zinc-800 bg-zinc-950/30 p-3">
              <div className="text-xs text-zinc-400">Active Wallet</div>
              <div className="mt-1 text-sm">{activeWallet ? activeWallet : "None"}</div>
            </div>
            <div className="rounded-md border border-zinc-800 bg-zinc-950/30 p-3">
              <div className="text-xs text-zinc-400">Balance (MVLT)</div>
              <div className="mt-1 text-sm">
                {Number(balanceEth).toFixed(6)}
                <div className="mt-1 text-xs text-zinc-400 break-all">wei: {balanceWei}</div>
              </div>
            </div>
          </div>

          <div className="mt-4 rounded-md border border-zinc-800 bg-zinc-950/30 p-3">
            <div className="text-sm font-medium">Send MVLT (in-dapp)</div>
            <div className="mt-2 grid gap-2 md:grid-cols-3">
              <input
                value={sendTo}
                onChange={(e) => setSendTo(e.target.value)}
                placeholder="Recipient (0x… or mirror1…)"
                className="w-full rounded-md border border-zinc-800 bg-zinc-950/60 px-3 py-2 text-sm outline-none focus:border-sky-500 md:col-span-2"
              />
              <input
                value={sendAmount}
                onChange={(e) => setSendAmount(e.target.value)}
                placeholder="Amount (e.g. 1.5)"
                inputMode="decimal"
                className="w-full rounded-md border border-zinc-800 bg-zinc-950/60 px-3 py-2 text-sm outline-none focus:border-sky-500"
              />
            </div>
            <div className="mt-2 flex items-center justify-end">
              <button
                onClick={sendMvlt}
                disabled={!connected || sendTo.trim().length === 0 || sendAmount.trim().length === 0}
                className="rounded-md bg-violet-500 px-3 py-2 text-sm font-medium text-zinc-950 hover:bg-violet-400 disabled:opacity-40"
              >
                Send MVLT
              </button>
            </div>
            <div className="mt-2 text-xs text-zinc-400">
              Uses EVM transfer under the hood; bech32 recipients are auto-converted.
            </div>
          </div>
        </section>

        {/* Vault */}
        <section className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-4">
          <div className="flex items-center justify-between">
            <div>
              <div className="text-lg font-semibold">Vault (Messages)</div>
              <div className="text-sm text-zinc-400">Unlock → store message via precompile</div>
            </div>
            <div className="flex gap-2">
              <button
                onClick={payToUnlock}
                disabled={!connected}
                className="rounded-md bg-emerald-500 px-3 py-2 text-sm font-medium text-zinc-950 hover:bg-emerald-400 disabled:opacity-40"
              >
                Pay 1 MVLT to Unlock
              </button>
              <button
                onClick={() => refreshOnChainState()}
                className="rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 hover:bg-zinc-800"
              >
                Refresh State
              </button>
            </div>
          </div>

          <div className="mt-4 grid gap-4 md:grid-cols-2">
            <div className="rounded-md border border-zinc-800 bg-zinc-950/30 p-3">
              <div className="text-sm font-medium">Write Message</div>
              <textarea
                value={messageInput}
                onChange={(e) => setMessageInput(e.target.value)}
                maxLength={256}
                rows={4}
                placeholder="Enter your message (max 256 characters)…"
                className="mt-2 w-full rounded-md border border-zinc-800 bg-zinc-950/60 px-3 py-2 text-sm outline-none focus:border-sky-500"
              />
              <div className="mt-2 flex items-center justify-between text-xs text-zinc-400">
                <div>{messageInput.length} / 256</div>
                <button
                  onClick={storeMessage}
                  disabled={!connected || messageInput.trim().length === 0}
                  className="rounded-md bg-sky-500 px-3 py-2 text-sm font-medium text-zinc-950 hover:bg-sky-400 disabled:opacity-40"
                >
                  Submit Message
                </button>
              </div>
            </div>

            <div className="rounded-md border border-zinc-800 bg-zinc-950/30 p-3">
              <div className="text-sm font-medium">On-chain State</div>
              <div className="mt-3 grid gap-3">
                <div className="rounded-md border border-zinc-800 bg-zinc-900/40 p-2">
                  <div className="text-xs text-zinc-400">Global Messages</div>
                  <div className="text-lg font-semibold">{globalMsgCount}</div>
                </div>
                <div className="rounded-md border border-zinc-800 bg-zinc-900/40 p-2">
                  <div className="text-xs text-zinc-400">Global Last Message</div>
                  <div className="mt-1 text-sm break-words">{globalLastMsg || "No messages yet"}</div>
                </div>
                <div className="rounded-md border border-zinc-800 bg-zinc-900/40 p-2">
                  <div className="text-xs text-zinc-400">Your Message Count</div>
                  <div className="text-lg font-semibold">{userMsgCount}</div>
                </div>
                <div className="rounded-md border border-zinc-800 bg-zinc-900/40 p-2">
                  <div className="text-xs text-zinc-400">Your Last Message</div>
                  <div className="mt-1 text-sm break-words">{userLastMsg || "-"}</div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* NFT */}
        <section className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-4">
          <div className="flex items-center justify-between">
            <div>
              <div className="text-lg font-semibold">NFT (MirrorNFT)</div>
              <div className="text-sm text-zinc-400">Mint, transfer, and query via precompile-backed wrapper</div>
            </div>
            <button
              onClick={() => refreshNftState()}
              className="rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 hover:bg-zinc-800"
            >
              Refresh NFT State
            </button>
          </div>

          <div className="mt-4 grid gap-4 md:grid-cols-2">
            <div className="rounded-md border border-zinc-800 bg-zinc-950/30 p-3">
              <div className="text-sm font-medium">Mint</div>
              <div className="mt-2 grid gap-2">
                <input
                  value={nftTokenId}
                  onChange={(e) => setNftTokenId(e.target.value)}
                  placeholder="Token ID (e.g. 123)"
                  className="w-full rounded-md border border-zinc-800 bg-zinc-950/60 px-3 py-2 text-sm outline-none focus:border-sky-500"
                />
                <input
                  value={nftTokenUri}
                  onChange={(e) => setNftTokenUri(e.target.value)}
                  placeholder="Token URI (https://… or ipfs://…)"
                  className="w-full rounded-md border border-zinc-800 bg-zinc-950/60 px-3 py-2 text-sm outline-none focus:border-sky-500"
                />
                <button
                  onClick={mintNft}
                  disabled={!connected}
                  className="rounded-md bg-sky-500 px-3 py-2 text-sm font-medium text-zinc-950 hover:bg-sky-400 disabled:opacity-40"
                >
                  Mint
                </button>
              </div>
            </div>

            <div className="rounded-md border border-zinc-800 bg-zinc-950/30 p-3">
              <div className="text-sm font-medium">Your NFTs</div>
              <div className="mt-2 text-sm text-zinc-200">Balance: {nftBalance}</div>
              <div className="mt-2 text-xs text-zinc-400">Token IDs</div>
              <div className="mt-1 break-words font-mono text-xs text-zinc-200">
                {nftOwnedTokenIds || "-"}
              </div>
            </div>

            <div className="rounded-md border border-zinc-800 bg-zinc-950/30 p-3">
              <div className="text-sm font-medium">Transfer</div>
              <div className="mt-2 grid gap-2">
                <input
                  value={nftTransferId}
                  onChange={(e) => setNftTransferId(e.target.value)}
                  placeholder="Token ID"
                  className="w-full rounded-md border border-zinc-800 bg-zinc-950/60 px-3 py-2 text-sm outline-none focus:border-sky-500"
                />
                <input
                  value={nftTransferTo}
                  onChange={(e) => setNftTransferTo(e.target.value)}
                  placeholder="To (0x… or mirror1…)"
                  className="w-full rounded-md border border-zinc-800 bg-zinc-950/60 px-3 py-2 text-sm outline-none focus:border-sky-500"
                />
                <button
                  onClick={transferNft}
                  disabled={!connected}
                  className="rounded-md bg-emerald-500 px-3 py-2 text-sm font-medium text-zinc-950 hover:bg-emerald-400 disabled:opacity-40"
                >
                  Transfer
                </button>
              </div>
            </div>

            <div className="rounded-md border border-zinc-800 bg-zinc-950/30 p-3">
              <div className="text-sm font-medium">Query</div>
              <div className="mt-2 grid gap-2">
                <input
                  value={nftQueryId}
                  onChange={(e) => setNftQueryId(e.target.value)}
                  placeholder="Token ID"
                  className="w-full rounded-md border border-zinc-800 bg-zinc-950/60 px-3 py-2 text-sm outline-none focus:border-sky-500"
                />
                <button
                  onClick={queryNft}
                  className="rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 hover:bg-zinc-800"
                >
                  Query Token
                </button>
                <div className="mt-2 text-xs text-zinc-400">Exists: {nftExists ? "true" : "false"}</div>
                <div className="text-xs text-zinc-400">Owner (EVM):</div>
                <div className="break-all font-mono text-xs">{nftOwnerEvm || "-"}</div>
                <div className="text-xs text-zinc-400">Owner (Cosmos):</div>
                <div className="break-all font-mono text-xs">{nftOwnerCosmos || "-"}</div>
                <div className="text-xs text-zinc-400">URI:</div>
                <div className="break-all font-mono text-xs">{nftUriRead || "-"}</div>
              </div>
            </div>
          </div>
        </section>

        {/* Converter */}
        <section className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-4">
          <div className="text-lg font-semibold">Address Conversion</div>
          <div className="text-sm text-zinc-400">Convert between 0x… and mirror1… deterministically</div>
          <div className="mt-3 grid gap-3 md:grid-cols-2">
            <input
              value={convertInput}
              onChange={(e) => setConvertInput(e.target.value)}
              placeholder="Paste 0x… or mirror1…"
              className="w-full rounded-md border border-zinc-800 bg-zinc-950/60 px-3 py-2 text-sm outline-none focus:border-sky-500"
            />
            <div className="rounded-md border border-zinc-800 bg-zinc-950/30 p-3 text-sm">
              {conversion.error ? (
                <div className="text-red-200">{conversion.error}</div>
              ) : (
                <div className="grid gap-2">
                  <div>
                    <div className="text-xs text-zinc-400">EVM</div>
                    <div className="break-all font-mono text-xs">{conversion.evm}</div>
                  </div>
                  <div>
                    <div className="text-xs text-zinc-400">Cosmos</div>
                    <div className="break-all font-mono text-xs">{conversion.mirror}</div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </section>

        {/* Logs */}
        <section className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-4">
          <div className="flex items-center justify-between">
            <div>
              <div className="text-lg font-semibold">Transaction Logs</div>
              <div className="text-sm text-zinc-400">Local debug stream (wallet + RPC)</div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <button
                onClick={() => setLogs([])}
                className="rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 hover:bg-zinc-800"
              >
                Clear
              </button>
              <button
                onClick={copyLogs}
                className="rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 hover:bg-zinc-800"
              >
                Copy
              </button>
              <label className="flex items-center gap-2 text-sm text-zinc-300">
                <input
                  type="checkbox"
                  checked={autoScroll}
                  onChange={(e) => setAutoScroll(e.target.checked)}
                />
                Auto-scroll
              </label>
            </div>
          </div>

          <div
            ref={logsRef}
            className="mt-3 h-64 overflow-auto rounded-md border border-zinc-800 bg-black p-3 font-mono text-xs"
          >
            {logs.length === 0 ? (
              <div className="text-zinc-500">No logs.</div>
            ) : (
              logs.map((l, idx) => (
                <div key={idx} className="leading-5">
                  <span className="text-zinc-500">[{l.time}]</span>{" "}
                  <span
                    className={
                      l.type === "success"
                        ? "text-emerald-300"
                        : l.type === "warning"
                          ? "text-amber-300"
                          : l.type === "error"
                            ? "text-red-300"
                            : "text-zinc-200"
                    }
                  >
                    {l.message}
                  </span>
                </div>
              ))
            )}
          </div>
        </section>
      </main>

      {/* Chain modal */}
      {chainModalOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
          <div className="w-full max-w-xl rounded-lg border border-zinc-800 bg-zinc-950 p-4">
            <div className="text-lg font-semibold">Chain Configuration Required</div>
            <div className="mt-1 text-sm text-zinc-400">
              Add/switch to Mirror Vault Localnet in your wallet.
            </div>

            <div className="mt-4 rounded-md border border-zinc-800 bg-zinc-900/40 p-3 text-sm">
              <div><span className="text-zinc-400">Network Name:</span> Mirror Vault Localnet</div>
              <div><span className="text-zinc-400">RPC URL:</span> http://localhost:8545</div>
              <div><span className="text-zinc-400">Chain ID:</span> {EVM_CHAIN_ID_DEC} ({EVM_CHAIN_ID_HEX})</div>
              <div><span className="text-zinc-400">Currency:</span> MVLT</div>
            </div>

            <div className="mt-4 flex flex-wrap gap-2">
              <button
                onClick={() => addChain(chainModalWallet ?? "metamask")}
                className="rounded-md bg-sky-500 px-3 py-2 text-sm font-medium text-zinc-950 hover:bg-sky-400"
              >
                Add Chain
              </button>
              <button
                onClick={() => setChainModalOpen(false)}
                className="rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 hover:bg-zinc-800"
              >
                Close
              </button>
            </div>

            <div className="mt-3 text-xs text-zinc-400">
              Tip: On Linux Docker setups, Blockscout may need host-gateway mapping; this UI runs in your browser and can reach localhost directly.
            </div>
          </div>
        </div>
      ) : null}

      <footer className="border-t border-zinc-800">
        <div className="mx-auto max-w-6xl px-4 py-6 text-sm text-zinc-500">
          Mirror Vault UI (local dev) — use browser profiles for multi-pair demos.
        </div>
      </footer>
    </div>
  );
}
