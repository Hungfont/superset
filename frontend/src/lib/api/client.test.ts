import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { useAuthStore } from "@/stores/authStore";
import { apiFetch } from "@/lib/api/client";

/** Creates a minimal JWT-like token with a future exp so isTokenExpired returns false. */
function makeValidToken(id = "1"): string {
  const exp = Math.floor(Date.now() / 1000) + 3600; // 1 hour from now
  const header = btoa(JSON.stringify({ alg: "HS256" }));
  const payload = btoa(JSON.stringify({ sub: id, exp }));
  return `${header}.${payload}.sig`;
}

// Reset Zustand store and fetch mock between tests
beforeEach(() => {
  useAuthStore.setState({ user: null, accessToken: null, isAuthenticated: false, refreshTimer: null });
  vi.restoreAllMocks();
});

afterEach(() => {
  vi.restoreAllMocks();
});

function makeFetchMock(responses: Array<{ status: number; body: unknown }>) {
  let calls = 0;
  return vi.fn().mockImplementation(() => {
    const r = responses[calls] ?? responses[responses.length - 1];
    calls++;
    return Promise.resolve({
      ok: r.status >= 200 && r.status < 300,
      status: r.status,
      json: () => Promise.resolve(r.body),
    });
  });
}

describe("apiFetch", () => {
  it("adds Authorization header when token is present", async () => {
    const token = makeValidToken();
    useAuthStore.setState({ accessToken: token, isAuthenticated: true, user: null, refreshTimer: null });
    const mockFetch = makeFetchMock([{ status: 200, body: { data: 1 } }]);
    vi.stubGlobal("fetch", mockFetch);

    await apiFetch("/api/v1/me");

    const calledHeaders = mockFetch.mock.calls[0][1].headers as Headers;
    expect(calledHeaders.get("Authorization")).toBe(`Bearer ${token}`);
  });

  it("returns response body on success", async () => {
    vi.stubGlobal("fetch", makeFetchMock([{ status: 200, body: { id: 42 } }]));

    const result = await apiFetch<{ id: number }>("/api/v1/me");
    expect(result).toEqual({ id: 42 });
  });

  it("throws with status on non-401 error", async () => {
    vi.stubGlobal("fetch", makeFetchMock([{ status: 403, body: { error: "Forbidden" } }]));

    await expect(apiFetch("/api/v1/me")).rejects.toMatchObject({ status: 403, message: "Forbidden" });
  });

  it("retries with new token after successful refresh on 401", async () => {
    useAuthStore.setState({ accessToken: makeValidToken(), isAuthenticated: true, user: null, refreshTimer: null });

    const mockFetch = vi.fn()
      .mockResolvedValueOnce({ ok: false, status: 401, json: () => Promise.resolve({ error: "Unauthorized" }) })
      .mockResolvedValueOnce({ ok: true, status: 200, json: () => Promise.resolve({ access_token: "new-token" }) })
      .mockResolvedValueOnce({ ok: true, status: 200, json: () => Promise.resolve({ id: 99 }) });

    vi.stubGlobal("fetch", mockFetch);

    const result = await apiFetch<{ id: number }>("/api/v1/me");

    expect(result).toEqual({ id: 99 });
    expect(useAuthStore.getState().accessToken).toBe("new-token");
  });

  it("clears auth and throws when refresh fails after 401", async () => {
    useAuthStore.setState({ accessToken: makeValidToken(), isAuthenticated: true, user: null, refreshTimer: null });
    // Stub window.location.replace to avoid jsdom error
    const replaceSpy = vi.fn();
    vi.stubGlobal("location", { replace: replaceSpy });

    const mockFetch = vi.fn()
      .mockResolvedValueOnce({ ok: false, status: 401, json: () => Promise.resolve({ error: "Unauthorized" }) })
      .mockResolvedValueOnce({ ok: false, status: 401, json: () => Promise.resolve({ error: "Unauthorized" }) });

    vi.stubGlobal("fetch", mockFetch);

    await expect(apiFetch("/api/v1/me")).rejects.toMatchObject({ status: 401 });
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(replaceSpy).toHaveBeenCalledWith("/login");
  });
});
