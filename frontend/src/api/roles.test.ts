import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { rolesApi } from "@/api/roles";
import { useAuthStore } from "@/stores/authStore";

describe("rolesApi", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    useAuthStore.setState({ accessToken: null });
  });

  it("fetches GET /api/v1/admin/roles", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: [{ id: 1, name: "Admin", user_count: 1, permission_count: 10, built_in: true }] }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const roles = await rolesApi.getRoles();

    expect(fetchMock).toHaveBeenCalled();
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/roles");
    expect(init.method).toBe("GET");
    expect(init.credentials).toBe("include");
    expect(roles[0]?.permission_count).toBe(10);
  });

  it("creates role via POST /api/v1/admin/roles", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { id: 2, name: "Analyst" } }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const role = await rolesApi.createRole({ name: "Analyst" });

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/roles");
    expect(init.method).toBe("POST");
    expect(init.body).toBe(JSON.stringify({ name: "Analyst" }));
    expect(role.name).toBe("Analyst");
  });

  it("updates role via PUT /api/v1/admin/roles/:id", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { id: 3, name: "Editor" } }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const role = await rolesApi.updateRole(3, { name: "Editor" });

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/roles/3");
    expect(init.method).toBe("PUT");
    expect(init.body).toBe(JSON.stringify({ name: "Editor" }));
    expect(role.name).toBe("Editor");
  });

  it("deletes role via DELETE /api/v1/admin/roles/:id", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: null }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await rolesApi.deleteRole(7);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/roles/7");
    expect(init.method).toBe("DELETE");
    expect(init.credentials).toBe("include");
  });

  it("maps 409 error response on delete", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 409,
      json: () => Promise.resolve({ error: "role has assigned users" }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(rolesApi.deleteRole(5)).rejects.toMatchObject({ message: "role has assigned users", status: 409 });
  });

  it("sends Authorization header in getRoles when accessToken is present", async () => {
    useAuthStore.setState({ accessToken: "tok-get-123" });
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: [] }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await rolesApi.getRoles();

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect((init.headers as Record<string, string>)?.["Authorization"]).toBe("Bearer tok-get-123");
  });

  it("sends Authorization header in createRole when accessToken is present", async () => {
    useAuthStore.setState({ accessToken: "tok-post-456" });
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { id: 1, name: "Test" } }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await rolesApi.createRole({ name: "Test" });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect((init.headers as Record<string, string>)?.["Authorization"]).toBe("Bearer tok-post-456");
  });

  it("sends Authorization header in updateRole when accessToken is present", async () => {
    useAuthStore.setState({ accessToken: "tok-put-789" });
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { id: 1, name: "Test" } }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await rolesApi.updateRole(1, { name: "Test" });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect((init.headers as Record<string, string>)?.["Authorization"]).toBe("Bearer tok-put-789");
  });

  it("sends Authorization header in deleteRole when accessToken is present", async () => {
    useAuthStore.setState({ accessToken: "tok-del-abc" });
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: null }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await rolesApi.deleteRole(1);

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect((init.headers as Record<string, string>)?.["Authorization"]).toBe("Bearer tok-del-abc");
  });
});
