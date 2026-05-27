import { request } from "@/utils/request";
import { useAuthStore } from "@/stores/authStore";

// QE-005: WebSocket event types
export interface WsProgressEvent {
  type: "progress";
  query_id: string;
  progress?: string;
  percent?: number;
}

export interface WsDoneEvent {
  type: "done";
  query_id: string;
  data: { rows: Record<string, unknown>[]; columns: { name: string; type: string }[] };
}

export interface WsResultReadyEvent {
  type: "result_ready";
  query_id: string;
  download_url: string;
}

export interface WsErrorEvent {
  type: "error";
  query_id: string;
  message: string;
}

export type WsEvent = WsProgressEvent | WsDoneEvent | WsResultReadyEvent | WsErrorEvent;

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

export interface CancelQueryResponse {
  status: string;      // "stopping" | "stopped" | "success"
  query_id: string;
  message?: string;
}

export interface QueryHistoryItem {
  id: string;
  status: string;
  sql: string;
  database_id: number;
  database_name: string;
  rows: number;
  start_time: string;
  end_time: string;
  duration_ms: number;
  error_message: string | null;
  results_key: string | null;
  user_id: number;
}

export interface QueryHistoryResponse {
  queries: QueryHistoryItem[];
  total: number;
  page: number;
  page_size: number;
}

export interface DeleteHistoryResponse {
  deleted: number;
}

export interface EstimateRequest {
  sql: string;
  database_id: number;
}

export interface EstimateResult {
  supported: boolean;
  driver?: string;
  total_cost?: number;
  estimated_rows?: number;
  bytes_processed?: number;
  estimated_cost_usd?: number;
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

  cancel: (queryId: string): Promise<CancelQueryResponse> =>
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
  }): Promise<QueryHistoryResponse> => {
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

  deleteHistory: (olderThan: string): Promise<DeleteHistoryResponse> =>
    request(`/api/v1/query/history?older_than=${encodeURIComponent(olderThan)}`, {
      method: "DELETE",
      credentials: "include",
      headers: getAuthHeaders(),
    }),

  getResult: (
    queryId: string,
    params?: { offset?: number; limit?: number }
  ): Promise<{ data: Record<string, unknown>[]; columns: QueryColumn[]; rows: number }> => {
    const searchParams = new URLSearchParams();
    if (params?.offset !== undefined) searchParams.append("offset", String(params.offset));
    if (params?.limit !== undefined) searchParams.append("limit", String(params.limit));
    const qs = searchParams.toString();
    return request(`/api/v1/query/${queryId}/result${qs ? `?${qs}` : ""}`, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });
  },

  estimate: (data: EstimateRequest): Promise<EstimateResult> =>
    request("/api/v1/query/estimate", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(data),
    }),

  download: async (queryId: string, format: "csv" | "xlsx" | "json"): Promise<void> => {
    const accessToken = useAuthStore.getState().accessToken;
    const res = await fetch(`/api/v1/query/${queryId}/download?format=${format}`, {
      method: "GET",
      credentials: "include",
      headers: {
        ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
      },
    });

    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: "Download failed" }));
      if (res.status === 410) {
        throw new Error("Result expired. Re-run query to download.");
      }
      if (res.status === 429) {
        throw new Error("Download limit reached. Try again later.");
      }
      if (res.status === 403) {
        throw new Error("Not authorized to download this query.");
      }
      throw new Error(body.error || body.message || "Download failed");
    }

    const blob = await res.blob();

    const disposition = res.headers.get("Content-Disposition");
    let filename = `query_${queryId}_${Date.now()}.${format}`;
    if (disposition) {
      const match = disposition.match(/filename="?([^"]+)"?/);
      if (match) filename = match[1];
    }

    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  },
};