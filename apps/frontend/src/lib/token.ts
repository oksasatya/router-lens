const compact = new Intl.NumberFormat("en-US", { notation: "compact", maximumFractionDigits: 1 });
const full = new Intl.NumberFormat("en-US");

/** Token counts: grouped (12,000) under 100k, compact (1.2M) above. */
export function formatTokens(n: number): string {
  return n >= 100_000 ? compact.format(n) : full.format(n);
}
