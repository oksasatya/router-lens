import { AxiosError, type InternalAxiosRequestConfig } from "axios";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api, ApiError } from "./api";

// Drive the real interceptors by swapping the adapter so it rejects with an
// AxiosError carrying a synthetic response — exactly the shape the error
// interceptor receives from a real non-2xx backend response.
function mockStatus(status: number, data: unknown) {
  api.defaults.adapter = async (config: InternalAxiosRequestConfig) => {
    throw new AxiosError("request failed", AxiosError.ERR_BAD_REQUEST, config, null, {
      data,
      status,
      statusText: "",
      headers: {},
      config,
    });
  };
}

const unauthorized = { error: { code: "unauthorized", message: "Authentication required" } };
const badCreds = { error: { code: "auth.invalid_credentials", message: "Invalid email or password" } };

// jsdom's location.assign is non-configurable, so replace the whole location.
let assign: ReturnType<typeof vi.fn>;
beforeEach(() => {
  assign = vi.fn();
  vi.stubGlobal("location", { assign });
});
afterEach(() => {
  delete api.defaults.adapter;
  vi.unstubAllGlobals();
});

describe("api 401 interceptor", () => {
  it("redirects to /login on 401 from a protected call (session expired)", async () => {
    mockStatus(401, unauthorized);
    await expect(api.get("/projects")).rejects.toBeInstanceOf(ApiError);
    expect(assign).toHaveBeenCalledWith("/login");
  });

  it("does NOT redirect on 401 from /auth/login (form error must survive)", async () => {
    mockStatus(401, badCreds);
    const err = await api.post("/auth/login", {}).catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).code).toBe("auth.invalid_credentials");
    expect(assign).not.toHaveBeenCalled();
  });

  it("does NOT redirect on 401 from the /auth/me probe", async () => {
    mockStatus(401, unauthorized);
    await expect(api.get("/auth/me")).rejects.toBeInstanceOf(ApiError);
    expect(assign).not.toHaveBeenCalled();
  });

  it("does NOT redirect on 4xx from /setup (first-run form error)", async () => {
    mockStatus(401, unauthorized);
    await expect(api.post("/setup", {})).rejects.toBeInstanceOf(ApiError);
    expect(assign).not.toHaveBeenCalled();
  });
});
