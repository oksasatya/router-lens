import { format, formatDistanceToNow, parseISO } from "date-fns";

/** Absolute local timestamp for tables/detail views. */
export function formatTimestamp(iso: string): string {
  return format(parseISO(iso), "yyyy-MM-dd HH:mm:ss");
}

/** Relative time ("3 minutes ago") for activity feeds. */
export function formatRelative(iso: string): string {
  return formatDistanceToNow(parseISO(iso), { addSuffix: true });
}
