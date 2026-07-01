import { z } from "zod";
import { api } from "@/lib/api";
import { apiKeyCreatedSchema, apiKeySchema, type ApiKey, type ApiKeyCreated } from "@/lib/schemas";

/** GET /projects/:projectId/api-keys — plain array, no pagination (small, per-project). */
export async function listApiKeys(projectId: string): Promise<ApiKey[]> {
  const res = await api.get(`/projects/${projectId}/api-keys`);
  return z.array(apiKeySchema).parse(res.data);
}

/** POST /projects/:projectId/api-keys — the response's `key` field is shown exactly once. */
export async function createApiKey(projectId: string, name: string): Promise<ApiKeyCreated> {
  const res = await api.post(`/projects/${projectId}/api-keys`, { name });
  return apiKeyCreatedSchema.parse(res.data);
}

/** DELETE /api-keys/:id */
export async function revokeApiKey(id: string): Promise<void> {
  await api.delete(`/api-keys/${id}`);
}
