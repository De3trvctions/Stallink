export type PriceTier = { quantity: number; totalPriceMinor: number };

const currencyPrefix: Record<string, string> = {
  MYR: "RM ",
  SGD: "S$ ",
  CNY: "¥ ",
};

// majorToMinor parses a major-unit amount into integer minor units.
// More than two decimal places is rejected so binary floating point cannot
// turn 1.015 into 101.
export function majorToMinor(amount: string | number): number | null {
  const text = String(amount).trim();
  const match = /^(\d+)(?:\.(\d{1,2}))?$/.exec(text);
  if (!match) return null;
  const minor = Number(match[1]) * 100 + Number((match[2] ?? "").padEnd(2, "0"));
  if (!Number.isSafeInteger(minor)) return null;
  return minor;
}

export function formatMoney(currency: string, minor: number) {
  return `${currencyPrefix[currency] ?? `${currency} `}${(minor / 100).toFixed(2)}`;
}

export function bundleTotal(unitPriceMinor: number, tiers: PriceTier[], quantity: number): number | null {
  if (!Number.isSafeInteger(unitPriceMinor) || unitPriceMinor <= 0 || !Number.isInteger(quantity) || quantity < 1 || quantity > 100_000) {
    return null;
  }
  if (tiers.length > 50) return null;
  const seen = new Set<number>();
  for (const tier of tiers) {
    if (!Number.isInteger(tier.quantity) || tier.quantity <= 1 || tier.quantity > 100_000 || seen.has(tier.quantity)) return null;
    if (!Number.isSafeInteger(tier.totalPriceMinor) || tier.totalPriceMinor <= 0 || tier.totalPriceMinor >= unitPriceMinor * tier.quantity) return null;
    seen.add(tier.quantity);
  }

  const best = new Array<number>(quantity + 1).fill(Number.POSITIVE_INFINITY);
  best[0] = 0;
  for (let count = 1; count <= quantity; count++) {
    const one = best[count - 1] + unitPriceMinor;
    if (Number.isSafeInteger(one)) best[count] = one;
    for (const tier of tiers) {
      if (tier.quantity > count) continue;
      const previous = best[count - tier.quantity];
      const bundled = previous + tier.totalPriceMinor;
      if (Number.isSafeInteger(bundled) && bundled < best[count]) best[count] = bundled;
    }
    if (!Number.isFinite(best[count])) return null;
  }
  return best[quantity];
}

export function lowestUnitRate(unitPriceMinor: number, tiers: PriceTier[]) {
  return tiers.reduce((lowest, tier) => Math.min(lowest, tier.totalPriceMinor / tier.quantity), unitPriceMinor);
}
