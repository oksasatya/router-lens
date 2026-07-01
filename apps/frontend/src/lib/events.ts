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
