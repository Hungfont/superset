import { request } from "@/utils/request";
import { useAuthStore } from "@/stores/authStore";

interface ApiEnvelope<T> {
  data: T;
}

export interface CreateDatasetPayload {
  database_id: number;
  schema: string;
  table_name: string;
}

export interface CreateVirtualDatasetPayload {
  database_id: number;
  table_name: string;
  sql: string;
  validate_sql: boolean;
}

export interface CreateDatasetResponse {
  id: number;
  table_name: string;
  background_sync: boolean;
}

export interface Column {
  id: number;
  column_name: string;
  type: string;
  is_dttm: boolean;
  is_active: boolean;
}

export interface CreateVirtualDatasetResponse {
  id: number;
  table_name: string;
  background_sync: boolean;
  columns?: Column[];
}

function getAuthHeaders(contentType = false): HeadersInit {
  const accessToken = useAuthStore.getState().accessToken;
  return {
    ...(contentType ? { "Content-Type": "application/json" } : {}),
    ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
  };
}

export const datasetsApi = {
  async createDataset(payload: CreateDatasetPayload): Promise<CreateDatasetResponse> {
    const body = await request<ApiEnvelope<CreateDatasetResponse>>("/api/v1/datasets", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });

    return body.data;
  },

  async createVirtualDataset(payload: CreateVirtualDatasetPayload): Promise<CreateVirtualDatasetResponse> {
    const body = await request<ApiEnvelope<CreateVirtualDatasetResponse>>("/api/v1/datasets/virtual", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });

    return body.data;
  },
};
