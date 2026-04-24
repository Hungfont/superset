import { apiFetch } from "@/lib/api/client";

export interface QueryColumn {
  name: string;
  type: string;
}

export interface QueryMeta {
  id: number;
  sql: string;
  executed_sql: string;
  start_time: string;
  end_time: string;
  rows: number;
  status: string;
}

export interface ExecuteQueryRequest {
  database_id: number;
  sql: string;
  limit?: number;
  schema?: string;
  client_id?: string;
  force_refresh?: boolean;
}

export interface ExecuteQueryResponse {
  data: Record<string, unknown>[];
  columns: QueryColumn[];
  from_cache: boolean;
  results_truncated?: boolean;
  query: QueryMeta;
}

export const queriesApi = {
  execute: (data: ExecuteQueryRequest): Promise<ExecuteQueryResponse> =>
    apiFetch("/api/v1/query/execute", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  getStatus: (queryId: string): Promise<{ status: string; progress?: number }> =>
    apiFetch(`/api/v1/query/${queryId}/status`, { method: "GET" }),

  cancel: (queryId: string): Promise<{ status: string }> =>
    apiFetch(`/api/v1/query/${queryId}`, { method: "DELETE" }),

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
    return apiFetch(`/api/v1/query/history?${searchParams}`, { method: "GET" });
  },

  getResult: (
    queryId: string
  ): Promise<{ data: Record<string, unknown>[]; columns: QueryColumn[]; rows: number }> =>
    apiFetch(`/api/v1/query/${queryId}/result`, { method: "GET" }),
};