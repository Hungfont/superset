import { request } from "@/utils/request";
import { useAuthStore } from "@/stores/authStore";

function getAuthHeaders(): HeadersInit {
  const token = useAuthStore.getState().accessToken;
  return {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
}

export interface TabStateResponse {
  id: number;
  label: string;
  db_id: number;
  schema: string;
  catalog: string;
  sql: string;
  active: boolean;
  query_limit: number;
  latest_query_id: string | null;
  latest_query_status: string;
  hide_left_bar: boolean;
  created_on: string;
}

export interface CreateTabRequest {
  db_id: number;
  schema?: string;
  catalog?: string;
  sql?: string;
  query_limit?: number;
}

export interface CreateTabResponse {
  id: number;
  label: string;
  active: boolean;
}

interface FetchTabsOptions {
  include_closed?: boolean;
}

export async function fetchTabs(options?: FetchTabsOptions): Promise<TabStateResponse[]> {
  const params = options?.include_closed ? "?include_closed=true" : "";
  return request<TabStateResponse[]>(`/api/v1/sqllab/tabs${params}`, {
    method: "GET",
    headers: getAuthHeaders(),
  });
}

export async function createTab(data: CreateTabRequest): Promise<CreateTabResponse> {
  return request<CreateTabResponse>("/api/v1/sqllab/tabs", {
    method: "POST",
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
}

export async function fetchTab(id: number): Promise<TabStateResponse> {
  return request<TabStateResponse>(`/api/v1/sqllab/tabs/${id}`, {
    method: "GET",
    headers: getAuthHeaders(),
  });
}

export interface UpdateTabRequest {
  label?: string;
  sql?: string;
  schema?: string;
  catalog?: string;
  query_limit?: number;
  db_id?: number;
  latest_query_id?: string;
  hide_left_bar?: boolean;
  extra_json?: string;
}

export async function updateTab(id: number, data: UpdateTabRequest): Promise<TabStateResponse> {
  return request<TabStateResponse>(`/api/v1/sqllab/tabs/${id}`, {
    method: "PUT",
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
}

export async function closeTab(id: number): Promise<{ closed: boolean }> {
  return request<{ closed: boolean }>(`/api/v1/sqllab/tabs/${id}/close`, {
    method: "PUT",
    headers: getAuthHeaders(),
  });
}

export async function closeAllTabs(exceptId?: number): Promise<{ closed: number }> {
  const qs = exceptId !== undefined ? `?except_id=${exceptId}` : "";
  return request<{ closed: number }>(`/api/v1/sqllab/tabs${qs}`, {
    method: "DELETE",
    headers: getAuthHeaders(),
  });
}
