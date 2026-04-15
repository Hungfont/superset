import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { permissionsApi } from "@/api/permissions";
import { useAuthStore } from "@/stores/authStore";

describe("permissionsApi", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    useAuthStore.setState({ accessToken: null });
  });

  it("fetches GET /api/v1/admin/permissions", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: [{ id: 1, name: "can_read" }] }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const permissions = await permissionsApi.getPermissions();

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/permissions");
    expect(init.method).toBe("GET");
    expect(permissions[0]?.name).toBe("can_read");
  });

  it("creates permission view via POST /api/v1/admin/permission-views", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { id: 8, permission_id: 1, view_menu_id: 2 } }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const permissionView = await permissionsApi.createPermissionView({ permission_id: 1, view_menu_id: 2 });

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/permission-views");
    expect(init.method).toBe("POST");
    expect(init.body).toBe(JSON.stringify({ permission_id: 1, view_menu_id: 2 }));
    expect(permissionView.id).toBe(8);
  });

  it("deletes permission view via DELETE /api/v1/admin/permission-views/:id", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: true }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await permissionsApi.deletePermissionView(8);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/permission-views/8");
    expect(init.method).toBe("DELETE");
  });

  it("maps duplicate permission-view conflict to thrown API error", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 409,
      json: () => Promise.resolve({ error: "permission view already exists" }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(permissionsApi.createPermissionView({ permission_id: 1, view_menu_id: 2 })).rejects.toMatchObject({
      message: "permission view already exists",
      status: 409,
    });
  });
});
