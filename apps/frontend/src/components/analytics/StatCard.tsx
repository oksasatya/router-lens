import type { ReactNode } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

interface StatCardProps {
  readonly label: string;
  readonly value: ReactNode;
  readonly hint?: string;
  readonly tone?: "default" | "danger";
}

/** A single overview metric. Numbers use tabular figures (`tabular-nums`) so
 * columns of stat cards don't jitter as digits update on refetch. */
export function StatCard({ label, value, hint, tone = "default" }: StatCardProps) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className={cn("text-2xl font-heading font-semibold tabular-nums", tone === "danger" && "text-destructive")}>
          {value}
        </div>
        {hint && <p className="mt-1 text-xs text-muted-foreground">{hint}</p>}
      </CardContent>
    </Card>
  );
}
