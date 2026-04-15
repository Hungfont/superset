import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useAuthStore } from "@/stores/authStore";
import { usersApi } from "@/api/users";

describe("usersApi", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    useAuthStore.setState({ accessToken: null });
  });

  it("fetches GET /api/v1/admin/users", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: [{ id: 7, username: "alice", role_ids: [1] }] }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const users = await usersApi.getUsers();

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/users");
    expect(init.method).toBe("GET");
    expect(users).toHaveLength(1);
  });

  it("creates user via POST /api/v1/admin/users", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { id: 8, username: "newuser", role_ids: [1] } }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await usersApi.createUser({
      first_name: "New",
      last_name: "User",
      username: "newuser",
      email: "new@example.com",
      password: "StrongPass@123",
      role_ids: [1],
    });

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/users");
    expect(init.method).toBe("POST");
    expect(init.body).toContain("new@example.com");
  });

  it("updates user via PUT /api/v1/admin/users/:id", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { id: 8, username: "edited", role_ids: [1, 3] } }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await usersApi.updateUser(8, {
      first_name: "Edit",
      last_name: "User",
      username: "edited",
      email: "edited@example.com",
      active: true,
      role_ids: [1, 3],
    });

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/users/8");
    expect(init.method).toBe("PUT");
    expect(init.body).toContain("edited@example.com");
  });

  it("deactivates user via DELETE /api/v1/admin/users/:id", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: true }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await usersApi.deactivateUser(8);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/users/8");
    expect(init.method).toBe("DELETE");
  });

  it("sends Authorization header when accessToken exists", async () => {
    useAuthStore.setState({ accessToken: "tok-users-123" });
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: [] }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await usersApi.getUsers();

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect((init.headers as Record<string, string>)?.Authorization).toBe("Bearer tok-users-123");
  });
});
