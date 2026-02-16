import { ethers } from "hardhat";
import fs from "fs";
import path from "path";

async function main() {
  console.log("Deploying Mirror Vault contracts...\n");

  const [deployer] = await ethers.getSigners();
  const network = await ethers.provider.getNetwork();

  // Deploy VaultGate
  console.log("1. Deploying VaultGate...");
  const VaultGate = await ethers.getContractFactory("VaultGate");
  const vaultGate = await VaultGate.deploy();
  await vaultGate.waitForDeployment();
  const vaultGateAddress = await vaultGate.getAddress();
  console.log("   VaultGate deployed to:", vaultGateAddress);

  // Deploy MirrorNFT
  console.log("\n2. Deploying MirrorNFT...");
  const MirrorNFT = await ethers.getContractFactory("MirrorNFT");
  const mirrorNFT = await MirrorNFT.deploy();
  await mirrorNFT.waitForDeployment();
  const mirrorNFTAddress = await mirrorNFT.getAddress();
  console.log("   MirrorNFT deployed to:", mirrorNFTAddress);

  // Print summary
  console.log("\n" + "=".repeat(60));
  console.log("DEPLOYMENT SUMMARY");
  console.log("=".repeat(60));
  console.log("VaultGate Contract:  ", vaultGateAddress);
  console.log("  - Precompile:      ", "0x0000000000000000000000000000000000000101");
  console.log("  - Functions:       ", "payToUnlock(), storeMessage()");
  console.log("");
  console.log("MirrorNFT Contract:  ", mirrorNFTAddress);
  console.log("  - Precompile:      ", "0x0000000000000000000000000000000000000102");
  console.log("  - Functions:       ", "mint(), transferFrom(), ownerOf()");
  console.log("=".repeat(60));
  console.log("\nSave these addresses for frontend integration!");

  const outPath = path.join(__dirname, "..", "deployed-addresses.json");
  const payload = {
    chainId: Number(network.chainId),
    deployer: deployer.address,
    vaultGate: vaultGateAddress,
    mirrorNFT: mirrorNFTAddress,
    updatedAt: new Date().toISOString(),
  };
  fs.writeFileSync(outPath, JSON.stringify(payload, null, 2));
  console.log(`\n📝 Wrote ${outPath}`);
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
