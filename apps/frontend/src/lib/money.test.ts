import { describe, expect, it } from "vitest";
import { formatUSD, UNPRICED } from "./money";

describe("formatUSD", () => {
  it("keeps small fractional costs (2–6 decimals)", () => {
    expect(formatUSD("0.054")).toBe("$0.054");
  });
  it("pads whole amounts to 2 decimals", () => {
    expect(formatUSD(12)).toBe("$12.00");
  });
  it("falls back to $0.00 on garbage input", () => {
    expect(formatUSD("not-a-number")).toBe("$0.00");
  });
  it("unpriced is an em dash, never $0", () => {
    expect(UNPRICED).toBe("—");
  });
});
