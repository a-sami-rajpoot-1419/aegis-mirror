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
      accounts: ["0x1ab42cc412b618bdea3a599e3c9bae199ebf030895b039e9db1e30dafb12b727"],
      gasPrice: 2000000000, // 2 gwei - above the 1 gwei base fee
    },
  },
};

export default config;
