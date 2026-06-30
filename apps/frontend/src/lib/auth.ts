import { queryOptions } from "@tanstack/react-query";
import { getMe } from "@/services/authService";

/** The current-user query — used by the route guard and the topbar user menu. */
export const meQueryOptions = queryOptions({
  queryKey: ["auth", "me"],
  queryFn: getMe,
  retry: false,
  staleTime: 60_000,
});
