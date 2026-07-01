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
