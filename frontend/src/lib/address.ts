import { fromBech32, toBech32 } from "@cosmjs/encoding";
import { getAddress, isAddress } from "ethers";
import { MIRROR_BECH32_PREFIX } from "./chainConfig";

export function is0xAddress(input: string): boolean {
  try {
    return isAddress(input);
  } catch {
    return false;
  }
}

export function normalize0xAddress(input: string): string {
  return getAddress(input);
}

export function evmToMirrorAddress(evmAddress: string, prefix = MIRROR_BECH32_PREFIX): string {
  const normalized = normalize0xAddress(evmAddress);
  const bytes = Buffer.from(normalized.slice(2), "hex");
  return toBech32(prefix, bytes);
}

export function isMirrorAddress(input: string, prefix = MIRROR_BECH32_PREFIX): boolean {
  try {
    const decoded = fromBech32(input);
    return decoded.prefix === prefix && decoded.data.length === 20;
  } catch {
    return false;
  }
}

export function mirrorToEvmAddress(mirrorAddress: string, prefix = MIRROR_BECH32_PREFIX): string {
  const decoded = fromBech32(mirrorAddress);
  if (decoded.prefix !== prefix) {
    throw new Error(`Unexpected bech32 prefix: ${decoded.prefix}`);
  }
  if (decoded.data.length !== 20) {
    throw new Error("Invalid address bytes length");
  }
  return getAddress(`0x${Buffer.from(decoded.data).toString("hex")}`);
}

export function tryConvertAnyAddress(input: string): { evm?: string; mirror?: string; error?: string } {
  const trimmed = input.trim();
  if (!trimmed) return { error: "Empty input" };

  if (is0xAddress(trimmed)) {
    const evm = normalize0xAddress(trimmed);
    return { evm, mirror: evmToMirrorAddress(evm) };
  }

  if (isMirrorAddress(trimmed)) {
    const evm = mirrorToEvmAddress(trimmed);
    return { evm, mirror: trimmed };
  }

  return { error: "Not a valid 0x… or mirror1… address" };
}
