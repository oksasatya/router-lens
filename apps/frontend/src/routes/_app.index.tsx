import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { DataTable, type DataTableColumn } from "@/components/DataTable";
import { LogStatusBadge } from "@/components/logs/LogStatusBadge";
import { StatCard } from "@/components/analytics/StatCard";
import { Button } from "@/components/ui/button";
import { overviewQueryOptions } from "@/lib/analytics";
import { eventsQueryOptions } from "@/lib/events";
import { projectsQueryOptions } from "@/lib/projects";
import { UNPRICED, formatUSD } from "@/lib/money";
import { formatTokens } from "@/lib/token";
import { formatTimestamp } from "@/lib/date";
import type { Event, Project } from "@/lib/schemas";

export const Route = createFileRoute("/_app/")({
  component: DashboardRoute,
});

/** Same null-semantics rule as Analytics: zero requests renders "—", not "Unpriced". */
function costOrUnpriced(usd: string | null, totalRequests: number): string {
  if (totalRequests === 0) return "—";
  return usd ? formatUSD(usd) : UNPRICED;
}

function DashboardRoute() {
  const { t } = useTranslation();
  const overview = useQuery(overviewQueryOptions({ preset: "24h" }));
  const events = useQuery(eventsQueryOptions({ preset: "24h", limit: 5 }, ""));
  const projects = useQuery(projectsQueryOptions(1, 5));

  const o = overview.data;

  const eventColumns: DataTableColumn<Event>[] = [
    { key: "time", header: t("logs.columns.time"), cell: (e) => formatTimestamp(e.request_started_at) },
    { key: "provider", header: t("logs.columns.provider"), cell: (e) => e.provider },
    { key: "model", header: t("logs.columns.model"), cell: (e) => e.model },
    { key: "status", header: t("logs.columns.status"), cell: (e) => <LogStatusBadge event={e} /> },
  ];
  const projectColumns: DataTableColumn<Project>[] = [
    {
      key: "name",
      header: t("projects.fields.name"),
      cell: (p) => (
        <Link to="/projects/$projectId" params={{ projectId: p.id }} className="font-medium hover:underline">
          {p.name}
        </Link>
      ),
    },
  ];

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-heading font-semibold">{t("nav.dashboard")}</h1>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard label={t("dashboard.statRequests")} value={o ? formatTokens(o.total_requests) : "—"} />
        <StatCard label={t("dashboard.statCost")} value={o ? costOrUnpriced(o.total_cost_usd, o.total_requests) : "—"} />
        <StatCard
          label={t("dashboard.statErrorRate")}
          value={o ? `${(o.error_rate * 100).toFixed(1)}%` : "—"}
          tone={o && o.error_rate > 0 ? "danger" : "default"}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-heading font-semibold">{t("dashboard.recentEvents")}</h2>
            <Button render={<Link to="/logs" />} nativeButton={false} variant="link" size="sm">
              {t("dashboard.viewAllLogs")}
            </Button>
          </div>
          <DataTable
            columns={eventColumns}
            rows={events.data?.items ?? []}
            rowKey={(e) => e.id}
            isLoading={events.isLoading}
            emptyMessage={t("dashboard.noEvents")}
          />
        </div>

        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-heading font-semibold">{t("dashboard.recentProjects")}</h2>
            <Button render={<Link to="/projects" search={{ page: 1 }} />} nativeButton={false} variant="link" size="sm">
              {t("dashboard.viewAllProjects")}
            </Button>
          </div>
          {!projects.isLoading && projects.data?.items.length === 0 ? (
            <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed border-border py-12 text-center">
              <p className="text-sm text-muted-foreground">{t("dashboard.noProjectsTitle")}</p>
              <Button render={<Link to="/projects" search={{ page: 1 }} />} nativeButton={false} size="sm">
                {t("dashboard.noProjectsCta")}
              </Button>
            </div>
          ) : (
            <DataTable
              columns={projectColumns}
              rows={projects.data?.items ?? []}
              rowKey={(p) => p.id}
              isLoading={projects.isLoading}
              emptyMessage={t("dashboard.noProjectsTitle")}
            />
          )}
        </div>
      </div>
    </div>
  );
}
