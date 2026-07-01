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
// array; this is the one place that parses "an array of T", reused 6 times
// above.
function z_array<T>(item: { parse: (v: unknown) => T }, data: unknown): T[] {
  if (!Array.isArray(data)) throw new Error("expected an array response");
  return data.map((row) => item.parse(row));
}
