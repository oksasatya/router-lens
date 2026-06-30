import { z } from "zod";

// Schemas validate the UNWRAPPED payload — the axios interceptor already strips
// the { data } envelope. zod v4: z.email() (not z.string().email()).

export const userSchema = z.object({
  id: z.string(),
  email: z.email(),
  name: z.string(),
});
export type User = z.infer<typeof userSchema>;

export const paginationSchema = z.object({
  page: z.number(),
  limit: z.number(),
  total: z.number(),
});

/** Wraps an item schema in the unwrapped list shape `{ items, pagination }`. */
export function paginated<T extends z.ZodTypeAny>(item: T) {
  return z.object({ items: z.array(item), pagination: paginationSchema });
}
