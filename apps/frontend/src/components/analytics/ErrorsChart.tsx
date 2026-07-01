import { Bar, BarChart, XAxis, YAxis } from "recharts";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import type { ErrorPoint } from "@/lib/schemas";

const config = { error_count: { label: "Errors", color: "var(--chart-2)" } } satisfies ChartConfig;

export function ErrorsChart({ points }: { readonly points: readonly ErrorPoint[] }) {
  return (
    <ChartContainer config={config} className="h-64 w-full">
      <BarChart data={points as ErrorPoint[]}>
        <XAxis dataKey="bucket" tickLine={false} axisLine={false} />
        <YAxis tickLine={false} axisLine={false} />
        <ChartTooltip content={<ChartTooltipContent />} />
        <Bar dataKey="error_count" fill="var(--color-error_count)" radius={4} />
      </BarChart>
    </ChartContainer>
  );
}
