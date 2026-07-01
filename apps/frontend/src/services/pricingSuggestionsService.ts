import { z } from "zod";
import { api, ApiError } from "@/lib/api";

const priceSuggestionSchema = z.object({
  provider: z.string(),
  model: z.string(),
  input_price_per_1m: z.string(),
  output_price_per_1m: z.string(),
});

export type PriceSuggestion = z.infer<typeof priceSuggestionSchema>;

/**
 * GET /pricing/suggestions. Resolves to [] when the feature is disabled
 * (404) so callers can just render nothing rather than an error state;
 * any other failure (502, network) still rejects.
 */
export async function listPricingSuggestions(): Promise<PriceSuggestion[]> {
  try {
    const res = await api.get("/pricing/suggestions");
    return z.array(priceSuggestionSchema).parse(res.data);
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return [];
    throw err;
  }
}
