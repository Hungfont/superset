import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useAuthStore } from "@/stores/authStore";
import { databasesApi } from "@/api/databases";

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

describe("databasesApi", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    useAuthStore.setState({ accessToken: null });
  });

  it("calls POST /api/v1/admin/databases/test for test connection", async () => {
    const fetchMock = vi.fn().mockResolvedValue(createJsonResponse({
        data: {
          success: true,
          latency_ms: 42,
          db_version: "PostgreSQL 15.4",
        },
      }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await databasesApi.testConnection({
      database_name: "analytics",
      password: "",
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
    const fetchMock = vi.fn().mockResolvedValue(createJsonResponse(
        {
          data: {
            id: 12,
            database_name: "analytics",
            sqlalchemy_uri: "postgresql://alice:***@localhost:5432/analytics",
            allow_dml: false,
            expose_in_sqllab: true,
            allow_run_async: false,
            allow_file_upload: false,
          },
        },
    ));
    vi.stubGlobal("fetch", fetchMock);

    const created = await databasesApi.createDatabase({
      database_name: "analytics",
      password: "secret",
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

    const fetchMock = vi.fn().mockResolvedValue(createJsonResponse({ data: { success: true } }));
    vi.stubGlobal("fetch", fetchMock);

    await databasesApi.testConnection({
      database_name: "analytics",
      password: "",
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
    const fetchMock = vi.fn().mockResolvedValue(createJsonResponse(
        {
          data: {
            success: true,
            latency_ms: 29,
            db_version: "PostgreSQL 16.2",
            driver: "postgresql",
          },
        },
    ));
    vi.stubGlobal("fetch", fetchMock);

    const result = await databasesApi.testConnectionById(7);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/databases/7/test");
    expect(init.method).toBe("POST");
    expect(result.success).toBe(true);
    expect(result.driver).toBe("postgresql");
  });

  it("calls GET /api/v1/admin/databases with query filters", async () => {
    const fetchMock = vi.fn().mockResolvedValue(createJsonResponse(
        {
          data: [
            {
              id: 7,
              database_name: "analytics",
              backend: "postgresql",
              sqlalchemy_uri: "postgresql://superset:***@localhost:5432/analytics",
              expose_in_sqllab: true,
              allow_run_async: false,
            },
          ],
          pagination: { total: 1, page: 2, page_size: 10 },
        },
    ));
    vi.stubGlobal("fetch", fetchMock);

    const result = await databasesApi.getDatabases({ q: "ana", backend: "postgresql", page: 2, pageSize: 10 });

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/databases?q=ana&backend=postgresql&page=2&page_size=10");
    expect(init.method).toBe("GET");
    expect(result.items).toHaveLength(1);
    expect(result.pagination.total).toBe(1);
  });

  it("calls GET /api/v1/admin/databases/:id for detail", async () => {
    const fetchMock = vi.fn().mockResolvedValue(createJsonResponse(
        {
          data: {
            id: 8,
            database_name: "warehouse",
            backend: "postgresql",
            sqlalchemy_uri: "postgresql://superset:***@localhost:5432/warehouse",
            dataset_count: 12,
            allow_dml: false,
            expose_in_sqllab: true,
            allow_run_async: true,
            allow_file_upload: false,
          },
        },
    ));
    vi.stubGlobal("fetch", fetchMock);

    const result = await databasesApi.getDatabase(8);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/databases/8");
    expect(init.method).toBe("GET");
    expect(result.dataset_count).toBe(12);
  });

  it("calls DELETE /api/v1/admin/databases/:id", async () => {
    const fetchMock = vi.fn().mockResolvedValue(createJsonResponse({ data: true }));
    vi.stubGlobal("fetch", fetchMock);

    await databasesApi.deleteDatabase(5);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/databases/5");
    expect(init.method).toBe("DELETE");
  });

  it("calls GET /api/v1/admin/databases/:id/schemas", async () => {
    const fetchMock = vi.fn().mockResolvedValue(createJsonResponse({ data: ["analytics", "public"] }));
    vi.stubGlobal("fetch", fetchMock);

    const schemas = await databasesApi.getSchemas(9);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/databases/9/schemas");
    expect(init.method).toBe("GET");
    expect(schemas).toEqual(["analytics", "public"]);
  });

  it("calls GET /api/v1/admin/databases/:id/tables with schema pagination and force_refresh", async () => {
    const fetchMock = vi.fn().mockResolvedValue(createJsonResponse(
        {
          data: [{ name: "orders" }],
          pagination: { total: 1, page: 1, page_size: 25 },
        },
    ));
    vi.stubGlobal("fetch", fetchMock);

    const result = await databasesApi.getTables(9, {
      schema: "public",
      page: 1,
      pageSize: 25,
      forceRefresh: true,
    });

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/databases/9/tables?schema=public&page=1&page_size=25&force_refresh=true");
    expect(init.method).toBe("GET");
    expect(result.items).toHaveLength(1);
    expect(result.items[0].name).toBe("orders");
    expect(result.pagination.total).toBe(1);
  });

  it("calls GET /api/v1/admin/databases/:id/columns with schema and table", async () => {
    const fetchMock = vi.fn().mockResolvedValue(createJsonResponse(
        {
          data: [
            {
              name: "created_at",
              data_type: "timestamp",
              is_nullable: false,
              is_dttm: true,
            },
          ],
        },
    ));
    vi.stubGlobal("fetch", fetchMock);

    const result = await databasesApi.getColumns(9, {
      schema: "public",
      table: "orders",
    });

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/admin/databases/9/columns?schema=public&table=orders");
    expect(init.method).toBe("GET");
    expect(result).toHaveLength(1);
    expect(result[0].is_dttm).toBe(true);
  });
});
