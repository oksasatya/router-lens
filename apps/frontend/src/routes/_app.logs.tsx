import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { CursorPagination } from "@/components/CursorPagination";
import { DataTable, type DataTableColumn } from "@/components/DataTable";
import { LogFilters } from "@/components/logs/LogFilters";
import { LogStatusBadge } from "@/components/logs/LogStatusBadge";
import { Button } from "@/components/ui/button";
import { eventsQueryOptions } from "@/lib/events";
import { formatDuration } from "@/lib/duration";
import { formatTimestamp } from "@/lib/date";
import { UNPRICED, formatUSD } from "@/lib/money";
import { formatTokens } from "@/lib/token";
import { eventsExportUrl, type EventFilters } from "@/services/eventService";
import type { Event } from "@/lib/schemas";

export const Route = createFileRoute("/_app/logs")({
  component: LogsRoute,
});

function LogsRoute() {
  const { t } = useTranslation();
  const [filters, setFilters] = useState<EventFilters>({ preset: "24h" });
  // The cursor stack: [] = first page. Pushing a cursor navigates forward;
  // popping navigates back. The CURRENT page's cursor is the top of the
  // stack's PREVIOUS entry — see `currentCursor` below.
  const [cursorStack, setCursorStack] = useState<string[]>([]);
  const currentCursor = cursorStack.at(-1) ?? "";

  const query = useQuery(eventsQueryOptions(filters, currentCursor));

  function handleFiltersChange(next: EventFilters) {
    setFilters(next);
    setCursorStack([]); // any filter change resets to the first page
  }

  const columns: DataTableColumn<Event>[] = [
    { key: "time", header: t("logs.columns.time"), cell: (e) => formatTimestamp(e.request_started_at) },
    { key: "provider", header: t("logs.columns.provider"), cell: (e) => e.provider },
    { key: "model", header: t("logs.columns.model"), cell: (e) => e.model },
    {
      key: "tokens",
      header: t("logs.columns.tokens"),
      cell: (e) => `${formatTokens(e.input_tokens)} / ${formatTokens(e.output_tokens)}`,
    },
    { key: "cost", header: t("logs.columns.cost"), cell: (e) => (e.cost_usd ? formatUSD(e.cost_usd) : UNPRICED) },
    { key: "latency", header: t("logs.columns.latency"), cell: (e) => formatDuration(e.latency_ms) },
    { key: "status", header: t("logs.columns.status"), cell: (e) => <LogStatusBadge event={e} /> },
  ];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-heading font-semibold">{t("nav.logs")}</h1>
        <Button render={<a href={eventsExportUrl(filters)} />} nativeButton={false} variant="outline" size="sm">
          {t("logs.export")}
        </Button>
      </div>
      <LogFilters filters={filters} onChange={handleFiltersChange} />
      <DataTable
        columns={columns}
        rows={query.data?.items ?? []}
        rowKey={(e) => e.id}
        isLoading={query.isLoading}
        emptyMessage={t("logs.empty")}
      />
      <CursorPagination
        hasMore={Boolean(query.data?.next_cursor)}
        hasPrevious={cursorStack.length > 0}
        isLoading={query.isFetching}
        onNext={() => {
          const next = query.data?.next_cursor;
          if (next) setCursorStack((stack) => [...stack, next]);
        }}
        onPrevious={() => setCursorStack((stack) => stack.slice(0, -1))}
      />
    </div>
  );
}
