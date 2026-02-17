import fs from "fs";
import path from "path";
import * as bech32 from "bech32";
import { ethers } from "hardhat";

function parseRecipientsFromFile(filePath: string): string[] {
  if (!fs.existsSync(filePath)) return [];
  const text = fs.readFileSync(filePath, "utf8");
  return text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line.length > 0)
    .filter((line) => !line.startsWith("#"));
}

function isHexAddress(value: string): boolean {
  return /^0x[0-9a-fA-F]{40}$/.test(value);
}

function mirrorToHexAddress(mirrorAddr: string): string {
  const decoded = bech32.decode(mirrorAddr);
  if (decoded.prefix !== "mirror") {
    throw new Error(`Unsupported bech32 prefix '${decoded.prefix}' (expected 'mirror')`);
  }
  const bytes = Buffer.from(bech32.fromWords(decoded.words));
  if (bytes.length !== 20) {
    throw new Error(`Unexpected address bytes length ${bytes.length} (expected 20)`);
  }
  return ethers.getAddress("0x" + bytes.toString("hex"));
}

function normalizeRecipient(recipient: string): string {
  const r = recipient.trim();
  if (isHexAddress(r)) return ethers.getAddress(r);
  if (r.startsWith("mirror1")) return mirrorToHexAddress(r);
  throw new Error(`Unsupported recipient format: ${r}`);
}

async function main() {
  const recipientsEnv = process.env.RECIPIENTS;
  const recipientsFile =
    process.env.RECIPIENTS_FILE || path.join(__dirname, "..", "..", "fund-accounts.txt");

  const rawRecipients = (
    recipientsEnv
      ? recipientsEnv
          .split(",")
          .map((s) => s.trim())
          .filter(Boolean)
      : parseRecipientsFromFile(recipientsFile)
  ).filter(Boolean);

  if (rawRecipients.length === 0) {
    throw new Error(
      `No recipients provided. Set RECIPIENTS=mirror1...,0x... or create ${recipientsFile}`
    );
  }

  const seen = new Set<string>();
  const recipients: string[] = [];
  for (const raw of rawRecipients) {
    const hex = normalizeRecipient(raw);
    if (!seen.has(hex)) {
      seen.add(hex);
      recipients.push(hex);
    }
  }

  const amountMvlt = process.env.AMOUNT_MVLT || "1000";
  const value = ethers.parseUnits(amountMvlt, 18);

  const [sender] = await ethers.getSigners();
  const senderAddr = await sender.getAddress();
  const senderBal = await ethers.provider.getBalance(senderAddr);

  console.log(`Sender: ${senderAddr}`);
  console.log(`Sender balance: ${ethers.formatUnits(senderBal, 18)} MVLT`);
  console.log(`Recipients: ${recipients.length}`);
  console.log(`Amount each: ${amountMvlt} MVLT`);

  for (const to of recipients) {
    console.log(`\n→ Funding ${to}`);
    const tx = await sender.sendTransaction({ to, value });
    console.log(`  tx: ${tx.hash}`);
    await tx.wait();
    const bal = await ethers.provider.getBalance(to);
    console.log(`  new balance: ${ethers.formatUnits(bal, 18)} MVLT`);
  }
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
