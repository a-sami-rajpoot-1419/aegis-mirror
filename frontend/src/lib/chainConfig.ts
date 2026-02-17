export const EVM_CHAIN_ID_DEC = 7777;
export const EVM_CHAIN_ID_HEX = "0x1e61"; // 7777

export const MIRROR_BECH32_PREFIX = "mirror";

export const EVM_RPC_HTTP = "http://localhost:8545";
export const EVM_RPC_WS = "ws://localhost:8546";

export const METAMASK_CHAIN_CONFIG = {
  chainId: EVM_CHAIN_ID_HEX,
  chainName: "Mirror Vault Localnet",
  rpcUrls: [EVM_RPC_HTTP],
  nativeCurrency: {
    name: "Mirror Vault Token",
    symbol: "MVLT",
    decimals: 18,
  },
  blockExplorerUrls: [],
} as const;

// Cosmos-side chain config for Keplr (used for experimentalSuggestChain).
// Note: even if REST endpoints are limited in local builds, Keplr chain suggestion can still work.
export const KEPLR_CHAIN_CONFIG = {
  chainId: "mirror-vault-localnet",
  chainName: "Mirror Vault Localnet",
  rpc: "http://localhost:26657",
  rest: "http://localhost:1317",
  bip44: { coinType: 60 },
  bech32Config: {
    bech32PrefixAccAddr: MIRROR_BECH32_PREFIX,
    bech32PrefixAccPub: `${MIRROR_BECH32_PREFIX}pub`,
    bech32PrefixValAddr: `${MIRROR_BECH32_PREFIX}valoper`,
    bech32PrefixValPub: `${MIRROR_BECH32_PREFIX}valoperpub`,
    bech32PrefixConsAddr: `${MIRROR_BECH32_PREFIX}valcons`,
    bech32PrefixConsPub: `${MIRROR_BECH32_PREFIX}valconspub`,
  },
  currencies: [
    {
      coinDenom: "MVLT",
      coinMinimalDenom: "umvlt",
      coinDecimals: 18,
    },
  ],
  feeCurrencies: [
    {
      coinDenom: "MVLT",
      coinMinimalDenom: "umvlt",
      coinDecimals: 18,
      gasPriceStep: { low: 0, average: 0, high: 0 },
    },
  ],
  stakeCurrency: {
    coinDenom: "MVLT",
    coinMinimalDenom: "umvlt",
    coinDecimals: 18,
  },
  coinType: 60,
  features: ["eth-address-gen", "eth-key-sign"],
} as const;
