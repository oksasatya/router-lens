import { Area, AreaChart, XAxis, YAxis } from "recharts";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import type { CostPoint } from "@/lib/schemas";

const config = { cost_usd: { label: "Cost (USD)", color: "var(--chart-3)" } } satisfies ChartConfig;

/** Nullable (unpriced) buckets render as a gap, not a dip to 0 — cost_usd is
 * pre-parsed to a JS number here ONLY for the chart's Y axis; the raw string
 * from the schema is never mutated, this is a local rendering concern. */
export function CostChart({ points }: { readonly points: readonly CostPoint[] }) {
  const data = points.map((p) => ({ bucket: p.bucket, cost_usd: p.cost_usd === null ? null : Number(p.cost_usd) }));
  return (
    <ChartContainer config={config} className="h-64 w-full">
      <AreaChart data={data}>
        <XAxis dataKey="bucket" tickLine={false} axisLine={false} />
        <YAxis tickLine={false} axisLine={false} />
        <ChartTooltip content={<ChartTooltipContent />} />
        <Area type="monotone" dataKey="cost_usd" stroke="var(--color-cost_usd)" fill="var(--color-cost_usd)" fillOpacity={0.2} connectNulls={false} />
      </AreaChart>
    </ChartContainer>
  );
}
