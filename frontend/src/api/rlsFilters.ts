import { request } from "@/utils/request";
import { useAuthStore } from "@/stores/authStore";

export interface RLSFilter {
  id: number;
  name: string;
  filter_type: "Regular" | "Base";
  clause: string;
  group_key: string;
  description: string;
  roles: Role[];
  tables: RLSTable[];
  created_by: number;
  created_on: string;
  changed_on: string;
}

export interface Role {
  id: number;
  name: string;
}

export interface RLSTable {
  datasource_id: number;
  datasource_type: string;
  table_name: string;
  database_name: string;
}

export interface CreateRLSFilterRequest {
  name: string;
  filter_type: "Regular" | "Base";
  clause: string;
  group_key?: string;
  description?: string;
  role_ids: number[];
  table_ids: number[];
}

export interface UpdateRLSFilterRequest {
  name?: string;
  filter_type?: "Regular" | "Base";
  clause?: string;
  group_key?: string;
  description?: string;
  role_ids?: number[];
  table_ids?: number[];
}

export interface RLSFilterListParams {
  page?: number;
  page_size?: number;
  q?: string;
  filter_type?: string;
  role_id?: number;
  datasource_id?: number;
}

export interface RLSFilterListResponse {
  total: number;
  page: number;
  pages: number;
  data: RLSFilter[];
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

export const rlsFiltersApi = {
  async getFilters(params?: RLSFilterListParams): Promise<RLSFilterListResponse> {
    const searchParams = new URLSearchParams();
    if (params?.page) searchParams.set("page", params.page.toString());
    if (params?.page_size) searchParams.set("page_size", params.page_size.toString());
    if (params?.q) searchParams.set("q", params.q);
    if (params?.filter_type) searchParams.set("filter_type", params.filter_type);
    if (params?.role_id) searchParams.set("role_id", params.role_id.toString());
    if (params?.datasource_id) searchParams.set("datasource_id", params.datasource_id.toString());

    const query = searchParams.toString();
    const body = await request<ApiEnvelope<RLSFilterListResponse>>(
      `/api/v1/admin/rls${query ? `?${query}` : ""}`,
      {
        method: "GET",
        credentials: "include",
        headers: getAuthHeaders(),
      }
    );
    return body.data;
  },

  async getFilter(id: number): Promise<RLSFilter> {
    const body = await request<ApiEnvelope<RLSFilter>>(`/api/v1/admin/rls/${id}`, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });
    return body.data;
  },

  async createFilter(payload: CreateRLSFilterRequest): Promise<RLSFilter> {
    const body = await request<ApiEnvelope<RLSFilter>>("/api/v1/admin/rls", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });
    return body.data;
  },

  async updateFilter(id: number, payload: UpdateRLSFilterRequest): Promise<RLSFilter> {
    const body = await request<ApiEnvelope<RLSFilter>>(`/api/v1/admin/rls/${id}`, {
      method: "PUT",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });
    return body.data;
  },

  async deleteFilter(id: number): Promise<void> {
    await request<{ data?: unknown }>(`/api/v1/admin/rls/${id}`, {
      method: "DELETE",
      credentials: "include",
      headers: getAuthHeaders(),
    });
  },
};