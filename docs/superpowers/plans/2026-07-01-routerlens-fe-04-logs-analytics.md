# RouterLens FE Plan 04 — Request Logs + Analytics Screens

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `/logs` placeholder with a real, filterable, keyset-paginated request-log table + CSV export, and add a new `/analytics` screen (overview stat cards + tokens/cost/latency/errors time-series charts + provider/model distribution tables) — the last two dashboard screens, consuming the backend endpoints shipped in Plans 05–06.

**Architecture:** Mirrors FE-03's layering exactly (`services/*Service.ts` → `lib/*.ts` `queryOptions` → route component), extending the existing `event` bounded context on the frontend. Two new small shared components (`DateRangeFilter`, `CursorPagination`) join the existing `DataTable`/`ConfirmDialog` set — both are deliberately minimal (a 3-button preset toggle, not a calendar date-range picker; a cursor-stack pager, not a page-number pager) because the backend itself only supports bounded presets/cursors, not arbitrary offset browsing. Charts use the shadcn `Chart` wrapper (Recharts under the hood) reading the **existing** `--chart-1`..`--chart-5` CSS variables already defined in `index.css` (Slate Aurora palette) — no new color tokens needed.

**Tech Stack:** Vite + React 19 + TanStack Router/Query, zod v4, axios, shadcn/ui (Base UI primitives) + Tailwind v4, **Recharts** (new dependency, added via the shadcn `chart` component). Bun as the package manager/runner.

## Global Constraints

- **Reuse over new abstraction (HARD):** `DataTable`, `ConfirmDialog`, `Field`, `FormError`, the `api` axios client, `formatUSD`/`formatTokens`/`formatTimestamp`/`formatRelative`, and the `paginated()` zod helper are ALL REUSED as-is. Do not fork or reimplement any of them. The only new shared components are `DateRangeFilter` and `CursorPagination` (Task 1) — nothing else needs a new abstraction.
- **Keyset, not offset, for logs:** `GET /api/v1/events` returns `{ items, next_cursor }` (never `{ items, pagination }`) — do NOT reuse `paginated()` or `OffsetPagination` for the logs list. A dedicated `eventCursorPageSchema` + `CursorPagination` component exist precisely because this shape differs from every other list in the app.
- **Date range is a bounded preset, not a calendar (ponytail — deliberate simplification):** the backend's `datetime.ParseRange` only accepts `preset=24h|7d|30d` or explicit `from`/`to`, defaults to 24h, hard-caps at 90 days (CLAUDE.md decision 6). `DateRangeFilter` exposes exactly the three presets as a segmented toggle — no calendar/custom-range UI in this plan. Add one later only if a real user asks for it.
- **Unpriced stays visually distinct, never `$0` (decision 10, already encoded in `lib/money.ts`'s `UNPRICED` constant):** every cost cell/stat that can be `null` renders `UNPRICED` ("—"), never coerces to `formatUSD(0)`.
- **CSV export is a plain link, not a fetch:** `GET /events/export.csv` streams a file; trigger it with a real `<a href="...">` (browser handles the download, cookies ride along same-origin) — never `fetch()` + blob-URL gymnastics for this.
- **Chart colors come from the existing theme:** `--chart-1` through `--chart-5` are already defined (light + dark) in `apps/frontend/src/index.css` — reference them via the shadcn `ChartConfig` `color: "var(--chart-N)"` pattern. Do not add new hex/oklch values for series colors.
- **i18n:** every new user-facing string goes into BOTH `en.json` and `id.json` under new `logs.*` / `analytics.*` sections, following the existing nesting convention (see `common.*`/`auth.*` for the pattern).

### Sonar guardrails — write compliant from the first commit

```
TypeScript / React:
- typescript:S6759 — React props readonly (every component prop interface uses `readonly` fields).
- typescript:S7764 — `globalThis` not `window` (with `?.` for SSR-safety, even though this is a pure SPA).
- typescript:S3358 — no nested ternaries (extract to a small function or if/else).
- typescript:S3923 — no identical-branch ternaries.
- typescript:S4624 — no nested template literals.
- typescript:S6582 — prefer optional chaining `?.`.
- typescript:S7755 — `arr.at(-1)` over `arr[arr.length - 1]` (used by the cursor stack "peek current" logic).
- typescript:S6819 — real elements over ARIA roles (`<button>` not `<div role="button">`).
- typescript:S1874 — no deprecated APIs (`z.email(...)`, not `z.string().email()` — already the house style).
- typescript:S6606 — prefer `??=`.
- typescript:S6479 — stable list keys (never array index) — use `row.id` / `point.bucket` / `stat.provider` etc.

When fixing one instance, check sibling files for the same anti-pattern and fix-forward.
Review the diff against this list BEFORE marking compliant.
```

### Skill brief for implementer subagents (every task)

> Invoke `frontend-design` first — per this project's standing rule this auto-chains `ui-ux-pro-max` + `frontend-senior` + the Tailwind skill set + `color-expert`, plus `vercel:shadcn` (this project uses shadcn/ui). This is **operational dashboard work** (data tables + charts, not marketing) — apply the core discipline (accessibility, responsive, semantic tokens) without inventing new visual identity; the Slate Aurora system (colors, type, radius) is already locked in `index.css` and must be reused, not redesigned. Finish each task's UI work with `react-doctor` before considering it done (lint/a11y/bundle check).

### Performance discipline (frontend-senior, stated once — applies to Task 3)

Recharts is a real bundle-size cost (~50-90KB gzipped) that only the Analytics screen needs. TanStack Router's file-based routing already code-splits per route by default (`@tanstack/router-plugin` generates one chunk per route file) — **do not** import anything from `recharts` or `@/components/ui/chart` from any file outside `routes/_app.analytics.tsx` and its own `components/analytics/*` subtree, so the Logs screen and the rest of the app never pull the chart bundle. Verify after Task 3 with `bun run build` — the `analytics` route's JS chunk should be a separate file from the main bundle in the Vite build output.

### Algorithmic complexity

Cursor pagination ("Previous") is a client-side stack push/pop — O(1) per navigation, O(page count visited) memory, never re-fetches already-seen pages from a growing offset. `CursorPagination`'s "peek current cursor" is `arr.at(-1)` — O(1), not a re-scan. No client-side sorting/filtering of the logs list (all filtering is server-side query params) — no O(n) work on a list that could grow to thousands of rows.

### TDD fit (per §16)

**TDD: no** for this entire plan — this is frontend visual/layout/data-fetching work (component composition, TanStack Query wiring, chart rendering). Per this project's own TDD-fit rule (CLAUDE.md §6 / the design spec §13): "all frontend visual/layout work (verify by running + `react-doctor`)." Verify each task by running the dev server, clicking through the golden path, and `react-doctor`'s regression check — not by red-green unit tests. (The one arguable exception — a pure cursor-stack reducer function — is simple enough that a manual click-through covers it; add a unit test only if you find yourself unsure it's correct.)

---

## Task 1: Data layer — schemas, services, query hooks, shared components

Delivers everything Task 2 and Task 3 consume: the zod schemas for events + all seven analytics shapes, the two service files, the query-options files, and the two new shared components (`DateRangeFilter`, `CursorPagination`). No screens in this task.

**Files:**
- Modify: `apps/frontend/src/lib/schemas.ts` (append event + analytics schemas)
- Create: `apps/frontend/src/services/eventService.ts`
- Create: `apps/frontend/src/services/analyticsService.ts`
- Create: `apps/frontend/src/lib/events.ts`
- Create: `apps/frontend/src/lib/analytics.ts`
- Create: `apps/frontend/src/components/DateRangeFilter.tsx`
- Create: `apps/frontend/src/components/CursorPagination.tsx`
- Create: `apps/frontend/src/lib/duration.ts`

**Interfaces:**
- Consumes: `api` (`lib/api.ts`), `projectsQueryOptions` (`lib/projects.ts`, for the project filter dropdown in Tasks 2/3).
- Produces (Tasks 2/3 rely on these): `eventSchema`, `Event`, `eventCursorPageSchema`, `EventCursorPage`, `overviewSchema`, `Overview`, `tokenPointSchema`/`costPointSchema`/`latencyPointSchema`/`errorPointSchema` + their inferred types, `providerStatSchema`/`modelStatSchema` + types; `listEvents`, `getEvent`, `eventsExportUrl` (`services/eventService.ts`); `getOverview`, `getTokensSeries`, `getCostSeries`, `getLatencySeries`, `getErrorSeries`, `getProviders`, `getModels` (`services/analyticsService.ts`); `eventsQueryOptions`, `eventQueryOptions` (`lib/events.ts`); `overviewQueryOptions`, `tokensSeriesQueryOptions`, `costSeriesQueryOptions`, `latencySeriesQueryOptions`, `errorSeriesQueryOptions`, `providersQueryOptions`, `modelsQueryOptions` (`lib/analytics.ts`); `<DateRangeFilter>`, `<CursorPagination>`; `formatDuration`.

- [ ] **Step 1: Extend `schemas.ts` with the event + analytics shapes**

Read the current `apps/frontend/src/lib/schemas.ts` first (it already has `paginated()`, `projectSchema`, etc.) — APPEND the following, do not restructure what's there:

```ts
// --- events ---

export const eventSchema = z.object({
  id: z.string(),
  project_id: z.string(),
  provider: z.string(),
  model: z.string(),
  route_source: z.string(),
  agent: z.string(),
  input_tokens: z.number(),
  output_tokens: z.number(),
  cost_usd: z.string().nullable(),
  input_price_1m: z.string().nullable(),
  output_price_1m: z.string().nullable(),
  latency_ms: z.number().nullable(),
  status_code: z.number().nullable(),
  is_error: z.boolean(),
  error_message: z.string(),
  request_started_at: z.string(),
  request_finished_at: z.string().nullable(),
  metadata: z.unknown().optional(),
});
export type Event = z.infer<typeof eventSchema>;

/** GET /events — keyset-paginated, NOT the offset `paginated()` shape. */
export const eventCursorPageSchema = z.object({
  items: z.array(eventSchema),
  next_cursor: z.string(),
});
export type EventCursorPage = z.infer<typeof eventCursorPageSchema>;

// --- analytics ---

export const overviewSchema = z.object({
  total_requests: z.number(),
  total_input_tokens: z.number(),
  total_output_tokens: z.number(),
  total_cost_usd: z.string().nullable(),
  unpriced_count: z.number(),
  avg_latency_ms: z.number().nullable(),
  p95_latency_ms: z.number().nullable(),
  error_count: z.number(),
  error_rate: z.number(),
  most_used_provider: z.string(),
  most_used_model: z.string(),
  most_expensive_model: z.string(),
  top_projects: z.array(
    z.object({ project_id: z.string(), project_name: z.string(), request_count: z.number() }),
  ),
});
export type Overview = z.infer<typeof overviewSchema>;

export const tokenPointSchema = z.object({
  bucket: z.string(),
  input_tokens: z.number(),
  output_tokens: z.number(),
});
export type TokenPoint = z.infer<typeof tokenPointSchema>;

export const costPointSchema = z.object({ bucket: z.string(), cost_usd: z.string().nullable() });
export type CostPoint = z.infer<typeof costPointSchema>;

export const latencyPointSchema = z.object({
  bucket: z.string(),
  avg_latency_ms: z.number().nullable(),
  p95_latency_ms: z.number().nullable(),
});
export type LatencyPoint = z.infer<typeof latencyPointSchema>;

export const errorPointSchema = z.object({
  bucket: z.string(),
  request_count: z.number(),
  error_count: z.number(),
  error_rate: z.number(),
});
export type ErrorPoint = z.infer<typeof errorPointSchema>;

const distributionFields = {
  request_count: z.number(),
  input_tokens: z.number(),
  output_tokens: z.number(),
  cost_usd: z.string().nullable(),
};
export const providerStatSchema = z.object({ provider: z.string(), ...distributionFields });
export type ProviderStat = z.infer<typeof providerStatSchema>;
export const modelStatSchema = z.object({ provider: z.string(), model: z.string(), ...distributionFields });
export type ModelStat = z.infer<typeof modelStatSchema>;
```
NOTE: `cost_usd`/`total_cost_usd` etc. are `z.string().nullable()` — the backend renders `*decimal.Decimal` as a numeric string (never a JS `number`, to avoid float precision loss), matching `pricingRuleSchema`'s existing `input_price_per_1m: z.string()` convention in this same file.

- [ ] **Step 2: Write the date-range + duration helpers**

`apps/frontend/src/lib/duration.ts`:
```ts
/** Renders a millisecond duration compactly: "842ms" under 1s, "8.4s" at/above. */
export function formatDuration(ms: number | null): string {
  if (ms === null) return "—";
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}
```

- [ ] **Step 3: Write the event service**

`apps/frontend/src/services/eventService.ts`:
```ts
import { api } from "@/lib/api";
import { eventCursorPageSchema, eventSchema, type Event, type EventCursorPage } from "@/lib/schemas";

/** Shared filter shape for both the logs list and the CSV export link. */
export interface EventFilters {
  readonly projectId?: string;
  readonly preset?: "24h" | "7d" | "30d";
  readonly provider?: string;
  readonly model?: string;
  readonly isError?: boolean;
}

function filterParams(filters: EventFilters, extra: Record<string, string> = {}): Record<string, string> {
  const params: Record<string, string> = { ...extra };
  if (filters.projectId) params.project_id = filters.projectId;
  if (filters.preset) params.preset = filters.preset;
  if (filters.provider) params.provider = filters.provider;
  if (filters.model) params.model = filters.model;
  if (filters.isError !== undefined) params.is_error = String(filters.isError);
  return params;
}

/** GET /events — keyset-paginated. cursor is the opaque `next_cursor` from the previous page, or "" for the first page. */
export async function listEvents(filters: EventFilters, cursor: string): Promise<EventCursorPage> {
  const params = filterParams(filters, cursor ? { cursor } : {});
  const res = await api.get("/events", { params });
  return eventCursorPageSchema.parse(res.data);
}

/** GET /events/:id */
export async function getEvent(id: string): Promise<Event> {
  const res = await api.get(`/events/${id}`);
  return eventSchema.parse(res.data);
}

/** Builds the CSV export URL for an <a href> — never fetched with JS (see Global Constraints). */
export function eventsExportUrl(filters: EventFilters): string {
  const params = new URLSearchParams(filterParams(filters));
  const query = params.toString();
  return `/api/v1/events/export.csv${query ? `?${query}` : ""}`;
}
```

- [ ] **Step 4: Write the analytics service**

`apps/frontend/src/services/analyticsService.ts`:
```ts
import { api } from "@/lib/api";
import {
  costPointSchema,
  errorPointSchema,
  latencyPointSchema,
  modelStatSchema,
  overviewSchema,
  providerStatSchema,
  tokenPointSchema,
  type CostPoint,
  type ErrorPoint,
  type LatencyPoint,
  type ModelStat,
  type Overview,
  type ProviderStat,
  type TokenPoint,
} from "@/lib/schemas";

export interface AnalyticsFilters {
  readonly projectId?: string;
  readonly preset?: "24h" | "7d" | "30d";
}

export interface SeriesFilters extends AnalyticsFilters {
  readonly interval?: "hour" | "day" | "week";
}

function params(filters: AnalyticsFilters, interval?: string): Record<string, string> {
  const out: Record<string, string> = {};
  if (filters.projectId) out.project_id = filters.projectId;
  if (filters.preset) out.preset = filters.preset;
  if (interval) out.interval = interval;
  return out;
}

export async function getOverview(filters: AnalyticsFilters): Promise<Overview> {
  const res = await api.get("/analytics/overview", { params: params(filters) });
  return overviewSchema.parse(res.data);
}

export async function getTokensSeries(filters: SeriesFilters): Promise<TokenPoint[]> {
  const res = await api.get("/analytics/tokens", { params: params(filters, filters.interval) });
  return z_array(tokenPointSchema, res.data);
}

export async function getCostSeries(filters: SeriesFilters): Promise<CostPoint[]> {
  const res = await api.get("/analytics/cost", { params: params(filters, filters.interval) });
  return z_array(costPointSchema, res.data);
}

export async function getLatencySeries(filters: SeriesFilters): Promise<LatencyPoint[]> {
  const res = await api.get("/analytics/latency", { params: params(filters, filters.interval) });
  return z_array(latencyPointSchema, res.data);
}

export async function getErrorSeries(filters: SeriesFilters): Promise<ErrorPoint[]> {
  const res = await api.get("/analytics/errors", { params: params(filters, filters.interval) });
  return z_array(errorPointSchema, res.data);
}

export async function getProviders(filters: AnalyticsFilters): Promise<ProviderStat[]> {
  const res = await api.get("/analytics/providers", { params: params(filters) });
  return z_array(providerStatSchema, res.data);
}

export async function getModels(filters: AnalyticsFilters): Promise<ModelStat[]> {
  const res = await api.get("/analytics/models", { params: params(filters) });
  return z_array(modelStatSchema, res.data);
}

// Small local helper — every series/distribution endpoint returns a bare
// array; this is the one place that parses "an array of T", used 6 times
// above (S1192: shared, not copy-pasted per function).
function z_array<T>(item: { parse: (v: unknown) => T }, data: unknown): T[] {
  if (!Array.isArray(data)) throw new Error("expected an array response");
  return data.map((row) => item.parse(row));
}
```
NOTE: `z_array` takes `{ parse }` (structurally, any zod schema) rather than importing `z.ZodTypeAny` machinery — keeps the helper trivial. If you'd rather write `z.array(tokenPointSchema).parse(res.data)` inline six times, that's equally acceptable (S1192's "3+ duplication" threshold is about literal strings, not this kind of structural repetition) — use your judgment, but don't invent a heavier abstraction than either of these two options.

- [ ] **Step 5: Write the query-options files**

`apps/frontend/src/lib/events.ts`:
```ts
import { queryOptions } from "@tanstack/react-query";
import { getEvent, listEvents, type EventFilters } from "@/services/eventService";

export function eventsQueryOptions(filters: EventFilters, cursor: string) {
  return queryOptions({
    queryKey: ["events", filters, cursor],
    queryFn: () => listEvents(filters, cursor),
  });
}

export function eventQueryOptions(id: string) {
  return queryOptions({
    queryKey: ["events", id],
    queryFn: () => getEvent(id),
  });
}
```

`apps/frontend/src/lib/analytics.ts`:
```ts
import { queryOptions } from "@tanstack/react-query";
import {
  getCostSeries,
  getErrorSeries,
  getLatencySeries,
  getModels,
  getOverview,
  getProviders,
  getTokensSeries,
  type AnalyticsFilters,
  type SeriesFilters,
} from "@/services/analyticsService";

export function overviewQueryOptions(filters: AnalyticsFilters) {
  return queryOptions({ queryKey: ["analytics", "overview", filters], queryFn: () => getOverview(filters) });
}
export function tokensSeriesQueryOptions(filters: SeriesFilters) {
  return queryOptions({ queryKey: ["analytics", "tokens", filters], queryFn: () => getTokensSeries(filters) });
}
export function costSeriesQueryOptions(filters: SeriesFilters) {
  return queryOptions({ queryKey: ["analytics", "cost", filters], queryFn: () => getCostSeries(filters) });
}
export function latencySeriesQueryOptions(filters: SeriesFilters) {
  return queryOptions({ queryKey: ["analytics", "latency", filters], queryFn: () => getLatencySeries(filters) });
}
export function errorSeriesQueryOptions(filters: SeriesFilters) {
  return queryOptions({ queryKey: ["analytics", "errors", filters], queryFn: () => getErrorSeries(filters) });
}
export function providersQueryOptions(filters: AnalyticsFilters) {
  return queryOptions({ queryKey: ["analytics", "providers", filters], queryFn: () => getProviders(filters) });
}
export function modelsQueryOptions(filters: AnalyticsFilters) {
  return queryOptions({ queryKey: ["analytics", "models", filters], queryFn: () => getModels(filters) });
}
```

- [ ] **Step 6: Write the `DateRangeFilter` component**

`apps/frontend/src/components/DateRangeFilter.tsx`:
```tsx
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export type DateRangePreset = "24h" | "7d" | "30d";

interface DateRangeFilterProps {
  readonly value: DateRangePreset;
  readonly onChange: (preset: DateRangePreset) => void;
}

const PRESETS: readonly DateRangePreset[] = ["24h", "7d", "30d"];

/**
 * A 3-button segmented preset toggle — the backend's datetime.ParseRange only
 * accepts these three presets (or explicit from/to, unused here). Not a
 * calendar picker; see this plan's Global Constraints for why.
 */
export function DateRangeFilter({ value, onChange }: DateRangeFilterProps) {
  const { t } = useTranslation();
  return (
    <div role="group" aria-label={t("common.dateRange")} className="inline-flex rounded-md border border-border">
      {PRESETS.map((preset) => (
        <Button
          key={preset}
          type="button"
          variant="ghost"
          size="sm"
          aria-pressed={value === preset}
          onClick={() => onChange(preset)}
          className={cn(
            "rounded-none first:rounded-l-md last:rounded-r-md border-r border-border last:border-r-0",
            value === preset && "bg-accent text-accent-foreground",
          )}
        >
          {t(`common.preset.${preset}`)}
        </Button>
      ))}
    </div>
  );
}
```
Add to BOTH `en.json` and `id.json`, inside the existing `common` section: `"dateRange": "Date range"` / `"Rentang tanggal"`, and a nested `"preset": { "24h": "24h", "7d": "7d", "30d": "30d" }` (same labels both languages — these are units, not prose).

- [ ] **Step 7: Write the `CursorPagination` component**

`apps/frontend/src/components/CursorPagination.tsx`:
```tsx
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";

interface CursorPaginationProps {
  /** True when the current page's response returned a non-empty next_cursor. */
  readonly hasMore: boolean;
  /** True once at least one page back exists (cursor stack is non-empty). */
  readonly hasPrevious: boolean;
  readonly onNext: () => void;
  readonly onPrevious: () => void;
  readonly isLoading?: boolean;
}

/**
 * Forward-cursor pager: the backend only ever gives a next_cursor, never a
 * previous one. "Previous" is implemented by the CALLER maintaining a stack
 * of visited cursors (see the logs route for the reducer) — this component
 * is presentation-only, it does not own the stack.
 */
export function CursorPagination({ hasMore, hasPrevious, onNext, onPrevious, isLoading }: CursorPaginationProps) {
  const { t } = useTranslation();
  return (
    <div className="flex items-center justify-end gap-2">
      <Button variant="outline" size="sm" disabled={!hasPrevious || isLoading} onClick={onPrevious}>
        {t("common.previous")}
      </Button>
      <Button variant="outline" size="sm" disabled={!hasMore || isLoading} onClick={onNext}>
        {t("common.next")}
      </Button>
    </div>
  );
}
```
(`common.previous`/`common.next` already exist in both `en.json`/`id.json` from FE-03 — reused, not redefined.)

- [ ] **Step 8: Verify + report**

Run: `cd apps/frontend && bun run build`
Expected: TypeScript + Vite build succeed with zero errors (no screens consume these files yet, but everything must type-check and the schemas/services must compile standalone).

Do NOT git commit — this project commits once per plan, at the end, after the controller shows the full diff to the user (see `.superpowers/sdd/progress.md`'s established cadence — apply the same convention to this FE plan).

---

## Task 2: Request Logs screen

Replaces the `_app.logs.tsx` placeholder with a real filterable, cursor-paginated table + CSV export.

**Files:**
- Modify: `apps/frontend/src/routes/_app.logs.tsx` (replace the placeholder body entirely)
- Create: `apps/frontend/src/components/logs/LogFilters.tsx`
- Create: `apps/frontend/src/components/logs/LogStatusBadge.tsx`

**Interfaces:**
- Consumes: `eventsQueryOptions` (Task 1), `projectsQueryOptions` (existing, FE-03), `DateRangeFilter`/`CursorPagination` (Task 1), `DataTable`/`DataTableColumn` (existing), `formatTimestamp`/`formatTokens`/`formatUSD`/`UNPRICED` (existing), `formatDuration` (Task 1), `eventsExportUrl` (Task 1).

- [ ] **Step 1: Write the status badge**

`apps/frontend/src/components/logs/LogStatusBadge.tsx`:
```tsx
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import type { Event } from "@/lib/schemas";

/** Renders is_error + status_code as a single semantic badge — color is the
 * existing destructive/secondary tokens, never a new ad-hoc hex. */
export function LogStatusBadge({ event }: { readonly event: Event }) {
  const { t } = useTranslation();
  if (event.is_error) {
    return <Badge variant="destructive">{event.status_code ?? t("logs.error")}</Badge>;
  }
  return <Badge variant="secondary">{event.status_code ?? t("logs.ok")}</Badge>;
}
```
(If `components/ui/badge.tsx` isn't installed yet in this checkout, run `cd apps/frontend && bunx shadcn@latest add badge` first — it's a standard shadcn primitive.)

- [ ] **Step 2: Write the filter bar**

`apps/frontend/src/components/logs/LogFilters.tsx`:
```tsx
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
```
NOTE: a plain `<select>`/`<input type="checkbox">` is used deliberately (ponytail) rather than reaching for a shadcn `Select`/`Checkbox` component — these are simple, native-semantic form controls with no styling requirement beyond what's already here. Swap in the shadcn primitives later only if the visual design review asks for it.

- [ ] **Step 3: Rewrite the logs route**

Read the current `apps/frontend/src/routes/_app.logs.tsx` first (it's currently a `<Placeholder>` stub) — REPLACE its entire body:
```tsx
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
        <Button asChild variant="outline" size="sm">
          <a href={eventsExportUrl(filters)}>{t("logs.export")}</a>
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
```

- [ ] **Step 4: Add `logs.*` i18n keys**

Add to BOTH `apps/frontend/src/i18n/en.json` and `id.json`, as a new top-level `logs` section (alongside the existing `nav`/`common`/`auth`/`projects`/etc. sections):
```json
"logs": {
  "export": "Export CSV",
  "empty": "No events in this range yet.",
  "ok": "OK",
  "error": "Error",
  "columns": {
    "time": "Time",
    "provider": "Provider",
    "model": "Model",
    "tokens": "Tokens (in/out)",
    "cost": "Cost",
    "latency": "Latency",
    "status": "Status"
  },
  "filters": {
    "project": "Project",
    "allProjects": "All projects",
    "provider": "Provider",
    "model": "Model",
    "errorsOnly": "Errors only"
  }
}
```
(Indonesian `id.json` gets the same structure translated — e.g. `"export": "Ekspor CSV"`, `"empty": "Belum ada event di rentang ini."`, `"allProjects": "Semua proyek"`, `"errorsOnly": "Hanya error"`, etc. — translate every string, keep every key identical.)

- [ ] **Step 5: Verify + report**

Run: `cd apps/frontend && bun run build`
Expected: build succeeds, zero TypeScript errors.

Manually verify by running `bun run dev` (or the existing dev workflow) against a backend with seeded events: filters narrow the list, pagination Next/Previous works across at least 2 pages, CSV export downloads a file, empty state shows when a filter matches nothing, errors-only toggle isolates error rows.

Do NOT git commit (see Task 1's note).

---

## Task 3: Analytics screen

Adds the new `/analytics` route: overview stat cards, four time-series charts (tokens/cost/latency/errors), and two distribution tables (providers/models).

**Files:**
- Create: `apps/frontend/src/routes/_app.analytics.tsx`
- Create: `apps/frontend/src/components/analytics/StatCard.tsx`
- Create: `apps/frontend/src/components/analytics/ChartCard.tsx`
- Create: `apps/frontend/src/components/analytics/TokensChart.tsx`
- Create: `apps/frontend/src/components/analytics/CostChart.tsx`
- Create: `apps/frontend/src/components/analytics/LatencyChart.tsx`
- Create: `apps/frontend/src/components/analytics/ErrorsChart.tsx`
- Modify: `apps/frontend/src/components/layout/nav.ts` (add the `/analytics` entry)
- Modify: `apps/frontend/package.json` (add `recharts`)

**Interfaces:**
- Consumes: everything from Task 1's `lib/analytics.ts` + `services/analyticsService.ts`, `DateRangeFilter` (Task 1), `projectsQueryOptions` (existing), `DataTable` (existing), `formatUSD`/`UNPRICED`/`formatTokens`/`formatDuration` (existing + Task 1), the shadcn `Chart` primitives (`ChartContainer`, `ChartTooltip`, `ChartTooltipContent`, `ChartConfig`).

- [ ] **Step 1: Install Recharts + the shadcn Chart component**

```bash
cd apps/frontend
bun add recharts
bunx shadcn@latest add chart
```
This creates `apps/frontend/src/components/ui/chart.tsx` (the standard shadcn wrapper: `ChartContainer`, `ChartTooltip`, `ChartTooltipContent`, `ChartLegend`, `ChartLegendContent`, and a `ChartConfig` type). Do not hand-write this file — let the CLI generate it so it matches the installed shadcn version exactly.

- [ ] **Step 2: Write `StatCard`**

`apps/frontend/src/components/analytics/StatCard.tsx`:
```tsx
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
```

- [ ] **Step 3: Write `ChartCard`**

`apps/frontend/src/components/analytics/ChartCard.tsx`:
```tsx
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
```

- [ ] **Step 4: Write the four chart components**

`apps/frontend/src/components/analytics/TokensChart.tsx`:
```tsx
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
```

`apps/frontend/src/components/analytics/CostChart.tsx`:
```tsx
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
```

`apps/frontend/src/components/analytics/LatencyChart.tsx`:
```tsx
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
```

`apps/frontend/src/components/analytics/ErrorsChart.tsx`:
```tsx
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
```
NOTE: line/area for continuous trend metrics (tokens, cost, latency), bar for
a discrete count (error count per bucket) — matches the chart-type-by-data-type
rule (`chart-type`: trend→line, comparison/count→bar) from the UX reference
consulted while planning this task.

- [ ] **Step 5: Write the analytics route**

`apps/frontend/src/routes/_app.analytics.tsx` (new file):
```tsx
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
    { key: "cost", header: t("analytics.columns.cost"), cell: (s) => (s.cost_usd ? formatUSD(s.cost_usd) : UNPRICED) },
  ];
  const modelColumns: DataTableColumn<ModelStat>[] = [
    { key: "model", header: t("analytics.columns.model"), cell: (s) => `${s.provider} / ${s.model}` },
    { key: "requests", header: t("analytics.columns.requests"), cell: (s) => formatTokens(s.request_count) },
    { key: "tokens", header: t("analytics.columns.tokens"), cell: (s) => `${formatTokens(s.input_tokens)} / ${formatTokens(s.output_tokens)}` },
    { key: "cost", header: t("analytics.columns.cost"), cell: (s) => (s.cost_usd ? formatUSD(s.cost_usd) : UNPRICED) },
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
        <StatCard label={t("analytics.stats.cost")} value={o ? (o.total_cost_usd ? formatUSD(o.total_cost_usd) : UNPRICED) : "—"} hint={o && o.unpriced_count > 0 ? t("analytics.stats.unpricedHint", { count: o.unpriced_count }) : undefined} />
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
```

- [ ] **Step 6: Add the `/analytics` nav entry**

In `apps/frontend/src/components/layout/nav.ts`, add one entry to the existing `NAV` array, right after the `logs` entry:
```ts
{ to: "/analytics", key: "nav.analytics", icon: BarChart3 },
```
Add `BarChart3` to the existing `lucide-react` import line at the top of the file.

- [ ] **Step 7: Add `analytics.*` i18n keys + the `nav.analytics` key**

In BOTH `en.json` and `id.json`: add `"analytics": "Analytics"` to the existing `nav` section, and a new top-level `analytics` section:
```json
"analytics": {
  "stats": {
    "requests": "Requests",
    "tokens": "Tokens (in/out)",
    "cost": "Cost",
    "latency": "Latency (avg/P95)",
    "latencyHint": "Average / 95th percentile",
    "errorRate": "Error rate",
    "mostUsed": "Most-used model",
    "unpricedHint": "{{count}} unpriced event(s)"
  },
  "charts": {
    "tokens": "Tokens over time",
    "cost": "Cost over time",
    "latency": "Latency over time",
    "errors": "Errors over time",
    "empty": "No data in this range yet."
  },
  "providers": "By provider",
  "models": "By model",
  "columns": {
    "provider": "Provider",
    "model": "Model",
    "requests": "Requests",
    "tokens": "Tokens (in/out)",
    "cost": "Cost"
  }
}
```
(Translate every string into `id.json`, e.g. `"requests": "Permintaan"`, `"errorRate": "Tingkat error"`, `"empty": "Belum ada data di rentang ini."`, keeping every key identical.)

- [ ] **Step 8: Full verification**

Run: `cd apps/frontend && bun run build`
Expected: build succeeds, zero TypeScript errors. Confirm in the Vite build output that the analytics route produces its own JS chunk separate from the main bundle (per this plan's Performance discipline section) — look for a distinctly named chunk file whose size roughly matches Recharts' footprint, not merged into `index-*.js`.

Manually verify by running the dev server: overview stat cards populate, all four charts render with real data (or the empty state when a preset has no events), the unpriced hint appears when at least one event lacks a price, switching the date-range preset refetches everything, provider/model tables populate, and the `/analytics` nav link works and highlights correctly when active.

Run `react-doctor` (or your project's nearest lint/a11y/bundle audit) as the finishing pass before considering this task done, per this project's frontend-design chain.

- [ ] **Step 9: Commit**

Do NOT git commit — this project commits once per plan, at the end, after the controller shows the full diff to the user (see `.superpowers/sdd/progress.md`).

---

## Plan-level Definition of Done

- `/logs` shows a real, filterable (project/provider/model/error-only/date-range) request log table with working forward/backward cursor pagination and a working CSV export link — replacing the placeholder.
- `/analytics` is a new screen: six overview stat cards (requests, tokens, cost with an unpriced-count hint, latency avg/P95, error rate, most-used model), four time-series charts (tokens, cost, latency, errors) that share the same date-range preset control, and two distribution tables (providers, models).
- Unpriced cost never renders as `$0` anywhere on either screen (stat card, table cell, or chart gap).
- No duplicated logic: `DataTable`, `ConfirmDialog`, `api` client, and all existing `lib/*` formatters are reused; the only new shared components are `DateRangeFilter` and `CursorPagination`.
- Chart colors come from the existing `--chart-1`..`--chart-5` theme tokens — no new color values introduced.
- Recharts is isolated to the `/analytics` route's own bundle chunk (verified in the Vite build output).
- Both `en.json` and `id.json` have complete, parallel `logs.*` / `analytics.*` / `nav.analytics` keys.
- `bun run build` is clean; the golden path (filter → paginate → export on Logs; switch date range → see all four charts + tables update on Analytics) has been manually verified by running the app.
