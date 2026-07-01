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
