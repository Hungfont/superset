import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { authApi } from "@/api/auth";

describe("authApi.logout", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("calls POST /api/v1/auth/logout and sends credentials", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      json: () => Promise.resolve({}),
    });
    vi.stubGlobal("fetch", fetchMock);

    await authApi.logout(false, "access-token");

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/auth/logout", {
      method: "POST",
      credentials: "include",
      headers: {
        Authorization: "Bearer access-token",
      },
    });
  });

  it("appends all=true query when logout-all is requested", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      json: () => Promise.resolve({}),
    });
    vi.stubGlobal("fetch", fetchMock);

    await authApi.logout(true, "access-token");

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/auth/logout?all=true", {
      method: "POST",
      credentials: "include",
      headers: {
        Authorization: "Bearer access-token",
      },
    });
  });
});
