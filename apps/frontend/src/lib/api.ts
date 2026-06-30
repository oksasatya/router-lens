import axios from "axios";
import i18n from "@/i18n";

export const api = axios.create({ baseURL: "/api/v1", withCredentials: true });

/**
 * ApiError keeps real Error semantics (stack, instanceof) while carrying the
 * backend's localized { error } envelope + HTTP status, so callers / TanStack
 * Query / error boundaries get a typed Error, never a bare object.
 */
export class ApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly details?: unknown;
  // Explicit fields (not constructor parameter-properties) — tsconfig has
  // erasableSyntaxOnly, which forbids parameter-properties.
  constructor(code: string, message: string, status: number, details?: unknown) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
    this.details = details;
  }
}

// Attach the active locale so the backend localizes error messages.
api.interceptors.request.use((config) => {
  config.headers.set("Accept-Language", i18n.language || "en");
  return config;
});

// Success: unwrap the { data } envelope so services/hooks get the payload directly
// (paginated payloads arrive as { items, pagination }). Error: 401 (except the
// /auth/me probe) -> /login; otherwise reject a typed ApiError.
api.interceptors.response.use(
  (res) => {
    if (res.data && typeof res.data === "object" && "data" in res.data) {
      res.data = (res.data as { data: unknown }).data;
    }
    return res;
  },
  (err) => {
    const status: number = err.response?.status ?? 0;
    const url: string = err.config?.url ?? "";
    if (status === 401 && !url.includes("/auth/me")) {
      globalThis.location?.assign("/login");
    }
    const envelope = err.response?.data?.error;
    if (envelope) {
      return Promise.reject(new ApiError(envelope.code, envelope.message, status, envelope.details));
    }
    return Promise.reject(err instanceof Error ? err : new Error(String(err)));
  },
);
