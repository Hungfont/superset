import { request } from "@/utils/request";
import { useAuthStore } from "@/stores/authStore";

export interface CreateChartPayload {
  slice_name: string;
  viz_type: string;
  datasource_id: string;
  datasource_type: string;
  params?: string;
  query_context?: string;
  description?: string;
  cache_timeout?: number;
  certified_by?: string;
  certification_details?: string;
}

export interface ChartResponse {
  id: number;
  slice_name: string;
  viz_type: string;
  datasource_id: string;
  datasource_type: string;
  datasource_name: string;
  params: string;
  query_context: string;
  description: string;
  cache_timeout: number;
  perm: string;
  schema_perm: string;
  certified_by: string;
  certification_details: string;
  last_saved_at: string;
  last_saved_by_fk: number;
  created_on: string;
  changed_on: string;
}

interface ApiEnvelope<T> {
  data: T;
}

function getAuthHeaders(contentType = false): HeadersInit {
  const accessToken = useAuthStore.getState().accessToken;
  return {
    ...(contentType ? { "Content-Type": "application/json" } : {}),
    ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
  };
}

export interface ChartListParams {
  q?: string;
  viz_type?: string;
  datasource_id?: number;
  owner?: number;
  certified?: boolean;
  page?: number;
  page_size?: number;
}

export interface ChartListItem {
  id: number;
  slice_name: string;
  viz_type: string;
  datasource_name: string;
  last_saved_at: string;
  last_saved_by: { id: number; username: string; first_name: string; last_name: string } | null;
  certified_by: string;
  dashboard_count: number;
}

export interface ChartListResponse {
  items: ChartListItem[];
  total: number;
  page: number;
  page_size: number;
}

export interface ChartDetail extends ChartListItem {
  datasource_id: string;
  datasource_type: string;
  params: string;
  query_context: string;
  description: string;
  cache_timeout: number;
  perm: string;
  certification_details: string;
  created_by: { id: number; username: string; first_name: string; last_name: string } | null;
}

export const chartsApi = {
  create: (payload: CreateChartPayload): Promise<ChartResponse> =>
    request<ApiEnvelope<ChartResponse>>("/api/v1/charts", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    }).then((r) => r.data),

  list: (params: ChartListParams = {}): Promise<ChartListResponse> => {
    const sp = new URLSearchParams();
    if (params.q) sp.set("q", params.q);
    if (params.viz_type) sp.set("viz_type", params.viz_type);
    if (params.datasource_id) sp.set("datasource_id", String(params.datasource_id));
    if (params.owner) sp.set("owner", String(params.owner));
    if (params.certified !== undefined) sp.set("certified", String(params.certified));
    if (params.page) sp.set("page", String(params.page));
    if (params.page_size) sp.set("page_size", String(params.page_size));
    const qs = sp.toString();
    return request<ApiEnvelope<ChartListResponse>>(`/api/v1/charts${qs ? "?" + qs : ""}`, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    }).then((r) => r.data);
  },

  get: (id: number): Promise<ChartDetail> =>
    request<ApiEnvelope<ChartDetail>>(`/api/v1/charts/${id}`, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    }).then((r) => r.data),
};
