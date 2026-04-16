import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useAuthStore } from "@/stores/authStore";
import { databasesApi } from "@/api/databases";

describe("databasesApi", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    useAuthStore.setState({ accessToken: null });
  });

  it("calls POST /api/v1/admin/databases/test for test connection", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({
        data: {
          success: true,
          latency_ms: 42,
          db_version: "PostgreSQL 15.4",
        },
      }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await databasesApi.testConnection({
      database_name: "analytics",
      sqlalchemy_uri: "postgresql://alice:secret@localhost:5432/analytics",
      allow_dml: false,
      expose_in_sqllab: true,
      allow_run_async: false,
      allow_file_upload: false,
      strict_test: true,
    });

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/databases/test");
    expect(init.method).toBe("POST");
    expect(result.success).toBe(true);
    expect(result.latency_ms).toBe(42);
  });

  it("calls POST /api/v1/admin/databases for create", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          data: {
            id: 12,
            database_name: "analytics",
            sqlalchemy_uri: "postgresql://alice:***@localhost:5432/analytics",
            allow_dml: false,
            expose_in_sqllab: true,
            allow_run_async: false,
            allow_file_upload: false,
          },
        }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const created = await databasesApi.createDatabase({
      database_name: "analytics",
      sqlalchemy_uri: "postgresql://alice:secret@localhost:5432/analytics",
      allow_dml: false,
      expose_in_sqllab: true,
      allow_run_async: false,
      allow_file_upload: false,
      strict_test: true,
    });

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/databases");
    expect(init.method).toBe("POST");
    expect(created.id).toBe(12);
  });

  it("sends Authorization header when accessToken exists", async () => {
    useAuthStore.setState({ accessToken: "tok-db-123" });

    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { success: true } }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await databasesApi.testConnection({
      database_name: "analytics",
      sqlalchemy_uri: "postgresql://alice:secret@localhost:5432/analytics",
      allow_dml: false,
      expose_in_sqllab: true,
      allow_run_async: false,
      allow_file_upload: false,
      strict_test: true,
    });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect((init.headers as Record<string, string>)?.Authorization).toBe("Bearer tok-db-123");
  });

  it("calls POST /api/v1/admin/databases/:id/test for existing database", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          data: {
            success: true,
            latency_ms: 29,
            db_version: "PostgreSQL 16.2",
            driver: "postgresql",
          },
        }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await databasesApi.testConnectionById(7);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/databases/7/test");
    expect(init.method).toBe("POST");
    expect(result.success).toBe(true);
    expect(result.driver).toBe("postgresql");
  });
});
