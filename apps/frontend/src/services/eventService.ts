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

/** Builds the CSV export URL for an <a href> — never fetched with JS (see the plan's Global Constraints). */
export function eventsExportUrl(filters: EventFilters): string {
  const params = new URLSearchParams(filterParams(filters));
  const query = params.toString();
  return `/api/v1/events/export.csv${query ? `?${query}` : ""}`;
}
