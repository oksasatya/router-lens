import { Line, LineChart, XAxis, YAxis } from "recharts";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import type { LatencyPoint } from "@/lib/schemas";

const config = {
  avg_latency_ms: { label: "Avg (ms)", color: "var(--chart-4)" },
  p95_latency_ms: { label: "P95 (ms)", color: "var(--chart-5)" },
} satisfies ChartConfig;

export function LatencyChart({ points }: { readonly points: readonly LatencyPoint[] }) {
  return (
    <ChartContainer config={config} className="h-64 w-full">
      <LineChart data={points as LatencyPoint[]}>
        <XAxis dataKey="bucket" tickLine={false} axisLine={false} />
        <YAxis tickLine={false} axisLine={false} />
        <ChartTooltip content={<ChartTooltipContent />} />
        <Line type="monotone" dataKey="avg_latency_ms" stroke="var(--color-avg_latency_ms)" dot={false} connectNulls />
        <Line type="monotone" dataKey="p95_latency_ms" stroke="var(--color-p95_latency_ms)" dot={false} connectNulls />
      </LineChart>
    </ChartContainer>
  );
}
