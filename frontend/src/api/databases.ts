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

export interface DatabaseDetail {
  id: number;
  database_name: string;
  sqlalchemy_uri: string;
  allow_dml: boolean;
  expose_in_sqllab: boolean;
  allow_run_async: boolean;
  allow_file_upload: boolean;
}

export interface TestConnectionResult {
  success: boolean;
  latency_ms?: number;
  db_version?: string;
  driver?: string;
  error?: string;
}

function getAuthHeaders(contentType = false): HeadersInit {
  const accessToken = useAuthStore.getState().accessToken;
  return {
    ...(contentType ? { "Content-Type": "application/json" } : {}),
    ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
  };
}

export const databasesApi = {
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
};
