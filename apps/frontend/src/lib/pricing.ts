import { queryOptions } from "@tanstack/react-query";
import { listPricing } from "@/services/pricingService";

export const pricingQueryOptions = queryOptions({
  queryKey: ["pricing"],
  queryFn: listPricing,
});
