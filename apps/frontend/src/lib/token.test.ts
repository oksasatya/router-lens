import { describe, expect, it } from "vitest";
import { formatTokens } from "./token";

describe("formatTokens", () => {
  it("groups thousands below 100k", () => {
    expect(formatTokens(12000)).toBe("12,000");
  });
  it("compacts millions", () => {
    expect(formatTokens(1_200_000)).toBe("1.2M");
  });
});
