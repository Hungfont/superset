import { request } from "@/utils/request";
import { useAuthStore } from "@/stores/authStore";

interface ApiEnvelope<T> {
  data: T;
}

export interface CreateDatabasePayload {
  database_name: string;
  sqlalchemy_uri: string;
  allow_dml: boolean;
  expose_in_sqllab: boolean;
  allow_run_async: boolean;
  allow_file_upload: boolean;
  strict_test: boolean;
}

export interface UpdateDatabasePayload {
  database_name?: string;
  sqlalchemy_uri?: string;
  allow_dml?: boolean;
  expose_in_sqllab?: boolean;
  allow_run_async?: boolean;
  allow_file_upload?: boolean;
  strict_test?: boolean;
}

export interface DatabaseDetail {
  id: number;
  database_name: string;
  backend: string;
  sqlalchemy_uri: string;
  allow_dml: boolean;
  expose_in_sqllab: boolean;
  allow_run_async: boolean;
  allow_file_upload: boolean;
  dataset_count?: number;
}

export interface DatabaseListResponse {
  items: DatabaseDetail[];
  pagination: {
    total: number;
    page: number;
    page_size: number;
  };
}

export interface DatabaseListFilters {
  q?: string;
  backend?: string;
  page?: number;
  pageSize?: number;
}

export interface TestConnectionResult {
  success: boolean;
  latency_ms?: number;
  db_version?: string;
  driver?: string;
  error?: string;
}

export interface DatabaseTable {
  name: string;
}

export interface DatabaseColumn {
  name: string;
  data_type: string;
  is_nullable: boolean;
  default_value?: string;
  is_dttm: boolean;
}

export interface DatabaseTablesResponse {
  items: DatabaseTable[];
  pagination: {
    total: number;
    page: number;
    page_size: number;
  };
}

export interface GetDatabaseTablesParams {
  schema: string;
  page?: number;
  pageSize?: number;
  forceRefresh?: boolean;
}

export interface GetDatabaseColumnsParams {
  schema: string;
  table: string;
  forceRefresh?: boolean;
}

function getAuthHeaders(contentType = false): HeadersInit {
  const accessToken = useAuthStore.getState().accessToken;
  return {
    ...(contentType ? { "Content-Type": "application/json" } : {}),
    ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
  };
}

export const databasesApi = {
  async getDatabases(filters: DatabaseListFilters): Promise<DatabaseListResponse> {
    const query = new URLSearchParams();
    if (filters.q) {
      query.set("q", filters.q);
    }
    if (filters.backend) {
      query.set("backend", filters.backend);
    }
    if (filters.page) {
      query.set("page", String(filters.page));
    }
    if (filters.pageSize) {
      query.set("page_size", String(filters.pageSize));
    }

    const queryString = query.toString();
    const url = queryString === "" ? "/api/v1/admin/databases" : `/api/v1/admin/databases?${queryString}`;

    const body = await request<{ data: DatabaseDetail[]; pagination: DatabaseListResponse["pagination"] }>(url, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });

    return {
      items: body.data,
      pagination: body.pagination,
    };
  },

  async getDatabase(databaseId: number): Promise<DatabaseDetail> {
    const body = await request<ApiEnvelope<DatabaseDetail>>(`/api/v1/admin/databases/${databaseId}`, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });
    return body.data;
  },

  async getSchemas(databaseId: number, forceRefresh = false): Promise<string[]> {
    const query = new URLSearchParams();
    if (forceRefresh) {
      query.set("force_refresh", "true");
    }

    const queryString = query.toString();
    const url = queryString === ""
      ? `/api/v1/admin/databases/${databaseId}/schemas`
      : `/api/v1/admin/databases/${databaseId}/schemas?${queryString}`;

    const body = await request<ApiEnvelope<string[]>>(url, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });
    return body.data;
  },

  async getTables(databaseId: number, params: GetDatabaseTablesParams): Promise<DatabaseTablesResponse> {
    const query = new URLSearchParams();
    query.set("schema", params.schema);

    if (params.page) {
      query.set("page", String(params.page));
    }
    if (params.pageSize) {
      query.set("page_size", String(params.pageSize));
    }
    if (params.forceRefresh) {
      query.set("force_refresh", "true");
    }

    const body = await request<{ data: DatabaseTable[]; pagination: DatabaseTablesResponse["pagination"] }>(
      `/api/v1/admin/databases/${databaseId}/tables?${query.toString()}`,
      {
        method: "GET",
        credentials: "include",
        headers: getAuthHeaders(),
      },
    );

    return {
      items: body.data,
      pagination: body.pagination,
    };
  },

  async getColumns(databaseId: number, params: GetDatabaseColumnsParams): Promise<DatabaseColumn[]> {
    const query = new URLSearchParams();
    query.set("schema", params.schema);
    query.set("table", params.table);
    if (params.forceRefresh) {
      query.set("force_refresh", "true");
    }

    const body = await request<ApiEnvelope<DatabaseColumn[]>>(
      `/api/v1/admin/databases/${databaseId}/columns?${query.toString()}`,
      {
        method: "GET",
        credentials: "include",
        headers: getAuthHeaders(),
      },
    );
    return body.data;
  },

  async testConnection(payload: CreateDatabasePayload): Promise<TestConnectionResult> {
    const body = await request<ApiEnvelope<TestConnectionResult>>("/api/v1/admin/databases/test", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });
    return body.data;
  },

  async testConnectionById(databaseId: number): Promise<TestConnectionResult> {
    const body = await request<ApiEnvelope<TestConnectionResult>>(`/api/v1/admin/databases/${databaseId}/test`, {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(),
    });
    return body.data;
  },

  async createDatabase(payload: CreateDatabasePayload): Promise<DatabaseDetail> {
    const body = await request<ApiEnvelope<DatabaseDetail>>("/api/v1/admin/databases", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });
    return body.data;
  },

  async updateDatabase(databaseId: number, payload: UpdateDatabasePayload): Promise<DatabaseDetail> {
    const body = await request<ApiEnvelope<DatabaseDetail>>(`/api/v1/admin/databases/${databaseId}`, {
      method: "PUT",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });
    return body.data;
  },

  async deleteDatabase(databaseId: number): Promise<void> {
    await request<{ data?: unknown }>(`/api/v1/admin/databases/${databaseId}`, {
      method: "DELETE",
      credentials: "include",
      headers: getAuthHeaders(),
    });
  },
};
