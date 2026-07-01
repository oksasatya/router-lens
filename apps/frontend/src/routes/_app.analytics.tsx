import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { CostChart } from "@/components/analytics/CostChart";
import { ErrorsChart } from "@/components/analytics/ErrorsChart";
import { LatencyChart } from "@/components/analytics/LatencyChart";
import { StatCard } from "@/components/analytics/StatCard";
import { ChartCard } from "@/components/analytics/ChartCard";
import { TokensChart } from "@/components/analytics/TokensChart";
import { DataTable, type DataTableColumn } from "@/components/DataTable";
import { DateRangeFilter, type DateRangePreset } from "@/components/DateRangeFilter";
import {
  costSeriesQueryOptions,
  errorSeriesQueryOptions,
  latencySeriesQueryOptions,
  modelsQueryOptions,
  overviewQueryOptions,
  providersQueryOptions,
  tokensSeriesQueryOptions,
} from "@/lib/analytics";
import { formatDuration } from "@/lib/duration";
import { UNPRICED, formatUSD } from "@/lib/money";
import { formatTokens } from "@/lib/token";
import type { ModelStat, ProviderStat } from "@/lib/schemas";

export const Route = createFileRoute("/_app/analytics")({
  component: AnalyticsRoute,
});

/** Renders a nullable cost string as USD, or UNPRICED — never coerces to $0 (decision 10/11). */
function costOrUnpriced(usd: string | null, totalRequests: number): string {
  if (totalRequests === 0) return "—";
  return usd ? formatUSD(usd) : UNPRICED;
}

function AnalyticsRoute() {
  const { t } = useTranslation();
  const [preset, setPreset] = useState<DateRangePreset>("24h");
  const filters = { preset };
  const seriesFilters = { preset, interval: "day" as const };

  const overview = useQuery(overviewQueryOptions(filters));
  const tokens = useQuery(tokensSeriesQueryOptions(seriesFilters));
  const cost = useQuery(costSeriesQueryOptions(seriesFilters));
  const latency = useQuery(latencySeriesQueryOptions(seriesFilters));
  const errors = useQuery(errorSeriesQueryOptions(seriesFilters));
  const providers = useQuery(providersQueryOptions(filters));
  const models = useQuery(modelsQueryOptions(filters));

  const o = overview.data;

  const providerColumns: DataTableColumn<ProviderStat>[] = [
    { key: "provider", header: t("analytics.columns.provider"), cell: (s) => s.provider },
    { key: "requests", header: t("analytics.columns.requests"), cell: (s) => formatTokens(s.request_count) },
    { key: "tokens", header: t("analytics.columns.tokens"), cell: (s) => `${formatTokens(s.input_tokens)} / ${formatTokens(s.output_tokens)}` },
    { key: "cost", header: t("analytics.columns.cost"), cell: (s) => costOrUnpriced(s.cost_usd, s.request_count) },
  ];
  const modelColumns: DataTableColumn<ModelStat>[] = [
    { key: "model", header: t("analytics.columns.model"), cell: (s) => `${s.provider} / ${s.model}` },
    { key: "requests", header: t("analytics.columns.requests"), cell: (s) => formatTokens(s.request_count) },
    { key: "tokens", header: t("analytics.columns.tokens"), cell: (s) => `${formatTokens(s.input_tokens)} / ${formatTokens(s.output_tokens)}` },
    { key: "cost", header: t("analytics.columns.cost"), cell: (s) => costOrUnpriced(s.cost_usd, s.request_count) },
  ];

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-heading font-semibold">{t("nav.analytics")}</h1>
        <DateRangeFilter value={preset} onChange={setPreset} />
      </div>

      <div className="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-6">
        <StatCard label={t("analytics.stats.requests")} value={o ? formatTokens(o.total_requests) : "—"} />
        <StatCard label={t("analytics.stats.tokens")} value={o ? `${formatTokens(o.total_input_tokens)} / ${formatTokens(o.total_output_tokens)}` : "—"} />
        <StatCard label={t("analytics.stats.cost")} value={o ? costOrUnpriced(o.total_cost_usd, o.total_requests) : "—"} hint={o && o.unpriced_count > 0 ? t("analytics.stats.unpricedHint", { count: o.unpriced_count }) : undefined} />
        <StatCard label={t("analytics.stats.latency")} value={o ? `${formatDuration(o.avg_latency_ms)} / ${formatDuration(o.p95_latency_ms)}` : "—"} hint={t("analytics.stats.latencyHint")} />
        <StatCard label={t("analytics.stats.errorRate")} value={o ? `${(o.error_rate * 100).toFixed(1)}%` : "—"} tone={o && o.error_rate > 0 ? "danger" : "default"} />
        <StatCard label={t("analytics.stats.mostUsed")} value={o?.most_used_model || "—"} hint={o?.most_used_provider} />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ChartCard title={t("analytics.charts.tokens")} isLoading={tokens.isLoading} isEmpty={!tokens.data?.length} emptyMessage={t("analytics.charts.empty")}>
          <TokensChart points={tokens.data ?? []} />
        </ChartCard>
        <ChartCard title={t("analytics.charts.cost")} isLoading={cost.isLoading} isEmpty={!cost.data?.length} emptyMessage={t("analytics.charts.empty")}>
          <CostChart points={cost.data ?? []} />
        </ChartCard>
        <ChartCard title={t("analytics.charts.latency")} isLoading={latency.isLoading} isEmpty={!latency.data?.length} emptyMessage={t("analytics.charts.empty")}>
          <LatencyChart points={latency.data ?? []} />
        </ChartCard>
        <ChartCard title={t("analytics.charts.errors")} isLoading={errors.isLoading} isEmpty={!errors.data?.length} emptyMessage={t("analytics.charts.empty")}>
          <ErrorsChart points={errors.data ?? []} />
        </ChartCard>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <div className="flex flex-col gap-2">
          <h2 className="text-lg font-heading font-semibold">{t("analytics.providers")}</h2>
          <DataTable columns={providerColumns} rows={providers.data ?? []} rowKey={(s) => s.provider} isLoading={providers.isLoading} emptyMessage={t("analytics.charts.empty")} />
        </div>
        <div className="flex flex-col gap-2">
          <h2 className="text-lg font-heading font-semibold">{t("analytics.models")}</h2>
          <DataTable columns={modelColumns} rows={models.data ?? []} rowKey={(s) => `${s.provider}/${s.model}`} isLoading={models.isLoading} emptyMessage={t("analytics.charts.empty")} />
        </div>
      </div>
    </div>
  );
}
