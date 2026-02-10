import { HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";

const config: HardhatUserConfig = {
  solidity: {
    version: "0.8.20",
    settings: {
      optimizer: { enabled: true, runs: 200 },
    },
  },
  networks: {
    mirrorVaultLocal: {
      url: "http://127.0.0.1:8545",
      chainId: 7777,
    },
  },
};

export default config;
