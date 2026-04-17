import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { datasetsApi } from "@/api/datasets";
import { useAuthStore } from "@/stores/authStore";

function createJsonResponse(body: unknown) {
  return {
    ok: true,
    status: 200,
    headers: {
      get: (name: string) => {
        if (name.toLowerCase() === "content-type") {
          return "application/json";
        }
        if (name.toLowerCase() === "content-length") {
          return "1";
        }
        return null;
      },
    },
    json: () => Promise.resolve(body),
  };
}

describe("datasetsApi", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    useAuthStore.setState({ accessToken: null });
  });

  it("calls POST /api/v1/datasets to create physical dataset", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      createJsonResponse({
        data: {
          id: 42,
          table_name: "orders",
          background_sync: true,
        },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const created = await datasetsApi.createDataset({
      database_id: 7,
      schema: "public",
      table_name: "orders",
    });

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/datasets");
    expect(init.method).toBe("POST");
    expect(created.id).toBe(42);
    expect(created.background_sync).toBe(true);
  });

  it("sends Authorization header when token exists", async () => {
    useAuthStore.setState({ accessToken: "tok-ds-123" });

    const fetchMock = vi.fn().mockResolvedValue(
      createJsonResponse({
        data: {
          id: 42,
          table_name: "orders",
          background_sync: true,
        },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await datasetsApi.createDataset({
      database_id: 7,
      schema: "public",
      table_name: "orders",
    });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect((init.headers as Record<string, string>)?.Authorization).toBe("Bearer tok-ds-123");
  });
});
