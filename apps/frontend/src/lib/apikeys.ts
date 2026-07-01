import { queryOptions } from "@tanstack/react-query";
import { listApiKeys } from "@/services/apiKeyService";

export function apiKeysQueryOptions(projectId: string) {
  return queryOptions({
    queryKey: ["projects", projectId, "api-keys"],
    queryFn: () => listApiKeys(projectId),
  });
}
