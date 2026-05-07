import { request } from "@/utils/request";
import { useAuthStore } from "@/stores/authStore";



export interface QueryColumn {
  name: string;
  type: string;
}

export interface QueryMeta {
  id: string;
  client_id?: string;
  sql: string;
  executed_sql: string;
  start_time: string;
  start_running_time?: string;
  end_time: string;
  rows: number;
  limit: number;
  limiting_factor: number;
  status: string;
  progress?: string;
  results_key?: string;
  select_as_cta_used?: boolean;
}

export interface ExecuteQueryRequest {
  database_id: number;
  sql: string;
  limit?: number;
  schema?: string;
  catalog?: string;
  tab_name?: string;
  sql_editor_id?: string;
  client_id?: string;
  force_refresh?: boolean;
  select_as_cta?: boolean;
}

export interface ExecuteQueryResponse {
  data: Record<string, unknown>[];
  columns: QueryColumn[];
  from_cache: boolean;
  results_truncated?: boolean;
  query: QueryMeta;
}

// Async query types
export interface SubmitQueryRequest {
  database_id: number;
  sql: string;
  limit?: number;
  schema?: string;
  catalog?: string;
  tab_name?: string;
  sql_editor_id?: string;
  client_id?: string;
  force_refresh?: boolean;
  select_as_cta?: boolean;
}

export interface SubmitQueryResponse {
  query_id: string;
  status: string;
  queue: string;
}

export interface QueryStatusResponse {
  query_id: string;
  status: string;
  progress?: string;
  start_time?: string;
  end_time?: string;
  rows: number;
  results_key?: string;
  error?: string;
  elapsed_ms: number;
  timeout_at?: string;
}

function getAuthHeaders(contentType = false): HeadersInit {
  const accessToken = useAuthStore.getState().accessToken;
  return {
    ...(contentType ? { "Content-Type": "application/json" } : {}),
    ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
  };
}

export const queriesApi = {
  execute: (data: ExecuteQueryRequest): Promise<ExecuteQueryResponse> =>
    request("/api/v1/query/execute", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(data),
    }),

  submit: (data: SubmitQueryRequest): Promise<SubmitQueryResponse> =>
    request("/api/v1/query/submit", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(data),
    }),

  getStatus: (queryId: string): Promise<QueryStatusResponse> =>
    request(`/api/v1/query/${queryId}/status`, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    }),

  cancel: (queryId: string): Promise<{ status: string }> =>
    request(`/api/v1/query/${queryId}`, {
      method: "DELETE",
      credentials: "include",
      headers: getAuthHeaders(),
    }),

  getHistory: (params?: {
    status?: string;
    database_id?: number;
    sql_contains?: string;
    page?: number;
    page_size?: number;
  }): Promise<{
    queries: QueryMeta[];
    total: number;
    page: number;
    page_size: number;
  }> => {
    const searchParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          searchParams.append(key, String(value));
        }
      });
    }
    return request(`/api/v1/query/history?${searchParams}`, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });
  },

  getResult: (
    queryId: string
  ): Promise<{ data: Record<string, unknown>[]; columns: QueryColumn[]; rows: number }> =>
    request(`/api/v1/query/${queryId}/result`, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    }),
};