import { fromHex, toBech32 } from "@cosmjs/encoding";

const addr = process.argv[2];
if (!addr) {
  console.error("Usage: node scripts/normalize-mirror-address.mjs <mirror1...|0x...>");
  process.exit(2);
}

const a = addr.trim();

if (/^mirror1[0-9a-z]+$/.test(a)) {
  console.log(a);
  process.exit(0);
}

if (/^0x[0-9a-fA-F]{40}$/.test(a)) {
  const bytes = fromHex(a.slice(2));
  console.log(toBech32("mirror", bytes));
  process.exit(0);
}

console.error(`Unsupported address format: ${a}`);
process.exit(1);
