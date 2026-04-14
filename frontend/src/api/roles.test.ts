import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { rolesApi } from "@/api/roles";

describe("rolesApi", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
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

  it("maps 409 error response on delete", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 409,
      json: () => Promise.resolve({ error: "role has assigned users" }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(rolesApi.deleteRole(5)).rejects.toMatchObject({ message: "role has assigned users", status: 409 });
  });
});
