import { api } from "@/lib/api";
import { userSchema, type User } from "@/lib/schemas";

/** GET /auth/me — the current session user. Throws ApiError(401) when unauthenticated. */
export async function getMe(): Promise<User> {
  const res = await api.get("/auth/me");
  return userSchema.parse(res.data);
}
