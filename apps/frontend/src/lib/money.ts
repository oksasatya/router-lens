const usd = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
  minimumFractionDigits: 2,
  maximumFractionDigits: 6,
});

/** Formats a USD amount (string from the API or number) with 2–6 decimals. */
export function formatUSD(value: string | number): string {
  const n = typeof value === "string" ? Number(value) : value;
  return usd.format(Number.isFinite(n) ? n : 0);
}

/** The unpriced marker — never render an unpriced event as `$0`. */
export const UNPRICED = "—";
