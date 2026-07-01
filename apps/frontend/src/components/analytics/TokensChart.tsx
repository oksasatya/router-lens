import { Line, LineChart, XAxis, YAxis } from "recharts";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import type { TokenPoint } from "@/lib/schemas";

const config = {
  input_tokens: { label: "Input", color: "var(--chart-1)" },
  output_tokens: { label: "Output", color: "var(--chart-2)" },
} satisfies ChartConfig;

export function TokensChart({ points }: { readonly points: readonly TokenPoint[] }) {
  return (
    <ChartContainer config={config} className="h-64 w-full">
      <LineChart data={points as TokenPoint[]}>
        <XAxis dataKey="bucket" tickLine={false} axisLine={false} />
        <YAxis tickLine={false} axisLine={false} />
        <ChartTooltip content={<ChartTooltipContent />} />
        <Line type="monotone" dataKey="input_tokens" stroke="var(--color-input_tokens)" dot={false} />
        <Line type="monotone" dataKey="output_tokens" stroke="var(--color-output_tokens)" dot={false} />
      </LineChart>
    </ChartContainer>
  );
}
