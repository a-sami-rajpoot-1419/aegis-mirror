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
      accounts: ["0x418e8edb0fa64960955b0c1d074e2312a9f31a905d28b531874098847eb01bcd"],
    },
  },
};

export default config;
