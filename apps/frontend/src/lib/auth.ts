import { queryOptions } from "@tanstack/react-query";
import { getMe, getSetupStatus } from "@/services/authService";

/** The current-user query — used by the route guard and the topbar user menu. */
export const meQueryOptions = queryOptions({
  queryKey: ["auth", "me"],
  queryFn: getMe,
  retry: false,
  staleTime: 60_000,
});

/** First-run setup status — used by the /login and /setup route guards. */
export const setupStatusQueryOptions = queryOptions({
  queryKey: ["auth", "setup-status"],
  queryFn: getSetupStatus,
  retry: false,
  staleTime: 60_000,
});
