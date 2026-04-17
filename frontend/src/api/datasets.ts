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

export interface CreateDatasetResponse {
  id: number;
  table_name: string;
  background_sync: boolean;
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
};
