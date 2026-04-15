import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useAuthStore } from "@/stores/authStore";
import { userRolesApi } from "@/api/userRoles";

describe("userRolesApi", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    useAuthStore.setState({ accessToken: null });
  });

  it("fetches GET /api/v1/admin/users/:id/roles", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { user_id: 7, role_ids: [1, 3] } }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const roleIds = await userRolesApi.getUserRoles(7);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/users/7/roles");
    expect(init.method).toBe("GET");
    expect(init.credentials).toBe("include");
    expect(roleIds).toEqual([1, 3]);
  });

  it("updates roles via PUT /api/v1/admin/users/:id/roles", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { user_id: 7, role_ids: [1, 2] } }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const roleIds = await userRolesApi.setUserRoles(7, [1, 2]);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/users/7/roles");
    expect(init.method).toBe("PUT");
    expect(init.body).toBe(JSON.stringify({ role_ids: [1, 2] }));
    expect(roleIds).toEqual([1, 2]);
  });

  it("sends Authorization header when accessToken exists", async () => {
    useAuthStore.setState({ accessToken: "tok-user-role-123" });
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { user_id: 7, role_ids: [1] } }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await userRolesApi.getUserRoles(7);

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect((init.headers as Record<string, string>)?.Authorization).toBe("Bearer tok-user-role-123");
  });
});
