/** Renders a millisecond duration compactly: "842ms" under 1s, "8.4s" at/above. */
export function formatDuration(ms: number | null): string {
  if (ms === null) return "—";
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}
