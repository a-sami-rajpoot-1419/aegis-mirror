import { ethers } from "hardhat";

async function main() {
  const VaultGate = await ethers.getContractFactory("VaultGate");
  const vaultGate = await VaultGate.deploy();
  await vaultGate.waitForDeployment();

  console.log("VaultGate deployed to:", await vaultGate.getAddress());
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
