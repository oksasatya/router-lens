import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { DateRangeFilter, type DateRangePreset } from "@/components/DateRangeFilter";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { projectsQueryOptions } from "@/lib/projects";
import type { EventFilters } from "@/services/eventService";

interface LogFiltersProps {
  readonly filters: EventFilters;
  readonly onChange: (filters: EventFilters) => void;
}

/** All filter state lives in the parent route (single source of truth so
 * changing a filter can reset the cursor stack) — this component is pure UI. */
export function LogFilters({ filters, onChange }: LogFiltersProps) {
  const { t } = useTranslation();
  const projects = useQuery(projectsQueryOptions(1, 100));

  return (
    <div className="flex flex-wrap items-end gap-4">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="log-project">{t("logs.filters.project")}</Label>
        <select
          id="log-project"
          className="h-9 rounded-md border border-input bg-background px-3 text-sm"
          value={filters.projectId ?? ""}
          onChange={(e) => onChange({ ...filters, projectId: e.target.value || undefined })}
        >
          <option value="">{t("logs.filters.allProjects")}</option>
          {projects.data?.items.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="log-provider">{t("logs.filters.provider")}</Label>
        <Input
          id="log-provider"
          value={filters.provider ?? ""}
          onChange={(e) => onChange({ ...filters, provider: e.target.value || undefined })}
          className="h-9 w-32"
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="log-model">{t("logs.filters.model")}</Label>
        <Input
          id="log-model"
          value={filters.model ?? ""}
          onChange={(e) => onChange({ ...filters, model: e.target.value || undefined })}
          className="h-9 w-40"
        />
      </div>
      <label className="flex items-center gap-2 pb-2 text-sm">
        <input
          type="checkbox"
          checked={filters.isError ?? false}
          onChange={(e) => onChange({ ...filters, isError: e.target.checked || undefined })}
        />
        {t("logs.filters.errorsOnly")}
      </label>
      <DateRangeFilter
        value={filters.preset ?? "24h"}
        onChange={(preset: DateRangePreset) => onChange({ ...filters, preset })}
      />
    </div>
  );
}
