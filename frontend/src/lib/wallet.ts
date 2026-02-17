import { BrowserProvider, JsonRpcProvider } from "ethers";
import {
  EVM_CHAIN_ID_DEC,
  EVM_CHAIN_ID_HEX,
  EVM_RPC_HTTP,
  KEPLR_CHAIN_CONFIG,
  METAMASK_CHAIN_CONFIG,
} from "./chainConfig";

export type ActiveWallet = "metamask" | "keplr";

export type Eip1193Provider = {
  request: (args: { method: string; params?: unknown[] | object }) => Promise<unknown>;
  on?: (event: string, listener: (...args: any[]) => void) => void;
  removeListener?: (event: string, listener: (...args: any[]) => void) => void;
};

declare global {
  interface Window {
    ethereum?: Eip1193Provider;
    keplr?: {
      enable: (chainId: string) => Promise<void>;
      experimentalSuggestChain: (config: any) => Promise<void>;
      ethereum?: Eip1193Provider;
    };
  }
}

export function getInjectedProvider(wallet: ActiveWallet): Eip1193Provider | null {
  if (wallet === "metamask") return window.ethereum ?? null;
  return window.keplr?.ethereum ?? null;
}

export async function ensureEvmChain(provider: Eip1193Provider): Promise<void> {
  const chainId = (await provider.request({ method: "eth_chainId" })) as string;
  if (chainId?.toLowerCase() === EVM_CHAIN_ID_HEX) return;

  try {
    await provider.request({
      method: "wallet_switchEthereumChain",
      params: [{ chainId: EVM_CHAIN_ID_HEX }],
    });
  } catch (err: any) {
    // 4902 = unknown chain
    if (err?.code === 4902) {
      await provider.request({
        method: "wallet_addEthereumChain",
        params: [METAMASK_CHAIN_CONFIG],
      });
      // Some wallets add but do not auto-switch.
      try {
        await provider.request({
          method: "wallet_switchEthereumChain",
          params: [{ chainId: EVM_CHAIN_ID_HEX }],
        });
      } catch {
        // ignore
      }
      return;
    }

    // Some providers (or keplr's EVM provider) may not implement switch.
    if (err?.code === -32601) {
      await provider.request({
        method: "wallet_addEthereumChain",
        params: [METAMASK_CHAIN_CONFIG],
      });
      return;
    }

    throw err;
  }
}

export async function suggestKeplrCosmosChain(): Promise<void> {
  if (!window.keplr) throw new Error("Keplr not installed");
  await window.keplr.experimentalSuggestChain(KEPLR_CHAIN_CONFIG);
  await window.keplr.enable(KEPLR_CHAIN_CONFIG.chainId);
}

export async function connectWallet(wallet: ActiveWallet): Promise<{
  provider: BrowserProvider;
  address: string;
}> {
  const injected = getInjectedProvider(wallet);
  if (!injected) {
    throw new Error(wallet === "metamask" ? "MetaMask not detected" : "Keplr EVM provider not detected");
  }

  if (wallet === "keplr") {
    // Suggest cosmos chain config (helps Keplr understand mirror prefix + coin type 60)
    // If user rejects, they can still proceed with EVM-only mode.
    try {
      await suggestKeplrCosmosChain();
    } catch {
      // non-fatal
    }
  }

  // Ensure chain is added/switched before requesting accounts so the first
  // connect click handles network setup end-to-end from the UI.
  await ensureEvmChain(injected);
  await injected.request({ method: "eth_requestAccounts" });

  const browserProvider = new BrowserProvider(injected as any);
  const signer = await browserProvider.getSigner();
  const address = await signer.getAddress();

  const network = await browserProvider.getNetwork();
  if (Number(network.chainId) !== EVM_CHAIN_ID_DEC) {
    throw new Error(`Wrong chainId: expected ${EVM_CHAIN_ID_DEC}, got ${network.chainId}`);
  }

  return { provider: browserProvider, address };
}

export function getReadOnlyProvider(): JsonRpcProvider {
  return new JsonRpcProvider(EVM_RPC_HTTP, EVM_CHAIN_ID_DEC);
}
