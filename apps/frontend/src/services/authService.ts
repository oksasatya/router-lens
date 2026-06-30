import { api } from "@/lib/api";
import { setupStatusSchema, userSchema, type SetupStatus, type User } from "@/lib/schemas";

/** GET /auth/me — the current session user. Throws ApiError(401) when unauthenticated. */
export async function getMe(): Promise<User> {
  const res = await api.get("/auth/me");
  return userSchema.parse(res.data);
}

/** GET /setup/status — whether the first-run setup is still open. */
export async function getSetupStatus(): Promise<SetupStatus> {
  const res = await api.get("/setup/status");
  return setupStatusSchema.parse(res.data);
}

export interface Credentials {
  email: string;
  password: string;
}

/** POST /auth/login — sets the httpOnly session cookie on success. */
export async function login(creds: Credentials): Promise<void> {
  await api.post("/auth/login", creds);
}

export interface SetupInput extends Credentials {
  name: string;
}

/** POST /setup — creates the first-run admin. Locked (403) once a user exists. */
export async function setup(input: SetupInput): Promise<void> {
  await api.post("/setup", input);
}

/** POST /auth/logout — revokes the session server-side and clears the cookie. */
export async function logout(): Promise<void> {
  await api.post("/auth/logout");
}
