import { z } from "zod";
import { api } from "@/lib/api";
import { pricingRuleSchema, type PricingRule } from "@/lib/schemas";

export interface PricingInput {
  provider: string;
  model: string;
  input_price_per_1m: string;
  output_price_per_1m: string;
}

/** GET /pricing — plain array, no pagination (small, admin-curated list). */
export async function listPricing(): Promise<PricingRule[]> {
  const res = await api.get("/pricing");
  return z.array(pricingRuleSchema).parse(res.data);
}

/** POST /pricing — upserts on (provider, model). */
export async function createPricing(input: PricingInput): Promise<PricingRule> {
  const res = await api.post("/pricing", input);
  return pricingRuleSchema.parse(res.data);
}

/** PUT /pricing/:id — 204 No Content, nothing to parse. */
export async function updatePricing(id: string, input: PricingInput): Promise<void> {
  await api.put(`/pricing/${id}`, input);
}

/** DELETE /pricing/:id */
export async function deletePricing(id: string): Promise<void> {
  await api.delete(`/pricing/${id}`);
}
