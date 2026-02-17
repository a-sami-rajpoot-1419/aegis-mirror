import fs from "fs";
import path from "path";

export const runtime = "nodejs";

type DeployedAddresses = {
  chainId: number;
  deployer: string;
  vaultGate: string;
  mirrorNFT: string;
  updatedAt?: string;
};

function fromEnv(): DeployedAddresses | null {
  const vaultGate = process.env.NEXT_PUBLIC_VAULT_GATE_ADDRESS;
  const mirrorNFT = process.env.NEXT_PUBLIC_MIRROR_NFT_ADDRESS;
  const chainIdRaw = process.env.NEXT_PUBLIC_EVM_CHAIN_ID;

  if (!vaultGate || !mirrorNFT) return null;

  return {
    chainId: chainIdRaw ? Number(chainIdRaw) : 7777,
    deployer: process.env.NEXT_PUBLIC_DEPLOYER_ADDRESS ?? "",
    vaultGate,
    mirrorNFT,
    updatedAt: new Date().toISOString(),
  };
}

function fromRepoFile(): DeployedAddresses {
  const filePath = path.join(process.cwd(), "..", "contracts", "deployed-addresses.json");
  const raw = fs.readFileSync(filePath, "utf8");
  return JSON.parse(raw) as DeployedAddresses;
}

export async function GET() {
  try {
    const env = fromEnv();
    const payload = env ?? fromRepoFile();

    return Response.json(payload, {
      headers: {
        "cache-control": "no-store",
      },
    });
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return Response.json(
      {
        error: "Failed to load deployed addresses",
        message,
        hint:
          "Deploy wrappers (contracts/scripts/deploy.ts) to generate contracts/deployed-addresses.json, or set NEXT_PUBLIC_VAULT_GATE_ADDRESS and NEXT_PUBLIC_MIRROR_NFT_ADDRESS.",
      },
      { status: 500 },
    );
  }
}
