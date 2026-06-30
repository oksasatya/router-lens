import { describe, expect, it } from "vitest";
import { formatRelative, formatTimestamp } from "./date";

describe("date formatters", () => {
  it("formatTimestamp yields a stable yyyy-MM-dd HH:mm:ss shape (timezone-agnostic check)", () => {
    expect(formatTimestamp("2026-06-29T10:00:00Z")).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/);
  });
  it("formatRelative produces a suffixed phrase", () => {
    expect(formatRelative("2020-01-01T00:00:00Z")).toMatch(/ago$/);
  });
});
