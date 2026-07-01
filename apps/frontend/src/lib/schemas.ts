import { z } from "zod";

// Schemas validate the UNWRAPPED payload — the axios interceptor already strips
// the { data } envelope. zod v4: z.email() (not z.string().email()).

export const userSchema = z.object({
  id: z.string(),
  email: z.email(),
  name: z.string(),
});
export type User = z.infer<typeof userSchema>;

/** GET /setup/status — whether the first-run admin still needs creating. */
export const setupStatusSchema = z.object({ needs_setup: z.boolean() });
export type SetupStatus = z.infer<typeof setupStatusSchema>;

const paginationSchema = z.object({
  page: z.number(),
  limit: z.number(),
  total: z.number(),
});

/** Wraps an item schema in the unwrapped list shape `{ items, pagination }`. */
export function paginated<T extends z.ZodTypeAny>(item: T) {
  return z.object({ items: z.array(item), pagination: paginationSchema });
}

export const projectSchema = z.object({
  id: z.string(),
  name: z.string(),
  slug: z.string(),
  description: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});
export type Project = z.infer<typeof projectSchema>;

export const apiKeySchema = z.object({
  id: z.string(),
  name: z.string(),
  key_prefix: z.string(),
  last_used_at: z.string().nullable(),
  revoked_at: z.string().nullable(),
  created_at: z.string(),
});
export type ApiKey = z.infer<typeof apiKeySchema>;

/** POST /projects/:id/api-keys response — carries the plaintext key exactly once. */
export const apiKeyCreatedSchema = apiKeySchema.extend({ key: z.string() });
export type ApiKeyCreated = z.infer<typeof apiKeyCreatedSchema>;

export const pricingRuleSchema = z.object({
  id: z.string(),
  provider: z.string(),
  model: z.string(),
  input_price_per_1m: z.string(),
  output_price_per_1m: z.string(),
  currency: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});
export type PricingRule = z.infer<typeof pricingRuleSchema>;

// --- events ---

export const eventSchema = z.object({
  id: z.string(),
  project_id: z.string(),
  provider: z.string(),
  model: z.string(),
  route_source: z.string(),
  agent: z.string(),
  input_tokens: z.number(),
  output_tokens: z.number(),
  cost_usd: z.string().nullable(),
  input_price_1m: z.string().nullable(),
  output_price_1m: z.string().nullable(),
  latency_ms: z.number().nullable(),
  status_code: z.number().nullable(),
  is_error: z.boolean(),
  error_message: z.string(),
  request_started_at: z.string(),
  request_finished_at: z.string().nullable(),
  metadata: z.unknown().optional(),
});
export type Event = z.infer<typeof eventSchema>;

/** GET /events — keyset-paginated, NOT the offset `paginated()` shape. */
export const eventCursorPageSchema = z.object({
  items: z.array(eventSchema),
  next_cursor: z.string(),
});
export type EventCursorPage = z.infer<typeof eventCursorPageSchema>;

// --- analytics ---

export const overviewSchema = z.object({
  total_requests: z.number(),
  total_input_tokens: z.number(),
  total_output_tokens: z.number(),
  total_cost_usd: z.string().nullable(),
  unpriced_count: z.number(),
  avg_latency_ms: z.number().nullable(),
  p95_latency_ms: z.number().nullable(),
  error_count: z.number(),
  error_rate: z.number(),
  most_used_provider: z.string(),
  most_used_model: z.string(),
  most_expensive_model: z.string(),
  top_projects: z.array(
    z.object({ project_id: z.string(), project_name: z.string(), request_count: z.number() }),
  ),
});
export type Overview = z.infer<typeof overviewSchema>;

export const tokenPointSchema = z.object({
  bucket: z.string(),
  input_tokens: z.number(),
  output_tokens: z.number(),
});
export type TokenPoint = z.infer<typeof tokenPointSchema>;

export const costPointSchema = z.object({ bucket: z.string(), cost_usd: z.string().nullable() });
export type CostPoint = z.infer<typeof costPointSchema>;

export const latencyPointSchema = z.object({
  bucket: z.string(),
  avg_latency_ms: z.number().nullable(),
  p95_latency_ms: z.number().nullable(),
});
export type LatencyPoint = z.infer<typeof latencyPointSchema>;

export const errorPointSchema = z.object({
  bucket: z.string(),
  request_count: z.number(),
  error_count: z.number(),
  error_rate: z.number(),
});
export type ErrorPoint = z.infer<typeof errorPointSchema>;

const distributionFields = {
  request_count: z.number(),
  input_tokens: z.number(),
  output_tokens: z.number(),
  cost_usd: z.string().nullable(),
};
export const providerStatSchema = z.object({ provider: z.string(), ...distributionFields });
export type ProviderStat = z.infer<typeof providerStatSchema>;
export const modelStatSchema = z.object({ provider: z.string(), model: z.string(), ...distributionFields });
export type ModelStat = z.infer<typeof modelStatSchema>;
