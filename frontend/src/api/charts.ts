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

export const chartsApi = {
  create: (payload: CreateChartPayload): Promise<ChartResponse> =>
    request<ApiEnvelope<ChartResponse>>("/api/v1/charts", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    }).then((r) => r.data),
};
