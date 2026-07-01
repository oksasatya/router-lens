import { queryOptions } from "@tanstack/react-query";
import { listPricingSuggestions } from "@/services/pricingSuggestionsService";

export const pricingSuggestionsQueryOptions = queryOptions({
  queryKey: ["pricing", "suggestions"],
  queryFn: listPricingSuggestions,
  staleTime: 60 * 60 * 1000, // 1h — matches the backend's own cache window
  retry: false,
});
