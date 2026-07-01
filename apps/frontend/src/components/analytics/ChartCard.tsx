import type { ReactNode } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

interface ChartCardProps {
  readonly title: string;
  readonly isLoading: boolean;
  readonly isEmpty: boolean;
  readonly emptyMessage: string;
  readonly children: ReactNode;
}

/** Shared chrome for every chart on the Analytics screen: title, loading
 * skeleton, and an explicit empty state (never a blank axis frame). */
export function ChartCard({ title, isLoading, isEmpty, emptyMessage, children }: ChartCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-64 w-full" />
        ) : isEmpty ? (
          <p className="flex h-64 items-center justify-center text-sm text-muted-foreground">{emptyMessage}</p>
        ) : (
          children
        )}
      </CardContent>
    </Card>
  );
}
