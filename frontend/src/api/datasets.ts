import { request } from "@/utils/request";
import { useAuthStore } from "@/stores/authStore";

interface ApiEnvelope<T> {
  data: T;
}

export interface DatasetListFilters {
  q?: string;
  database_id?: number;
  schema?: string;
  type?: "physical" | "virtual";
  owner?: number;
  page?: number;
  page_size?: number;
  order_by?: string;
}

export interface DatasetWithCounts {
  id: number;
  table_name: string;
  schema?: string;
  database_id: number;
  database_name?: string;
  type: "physical" | "virtual";
  perm: string;
  created_by_fk: number;
  owner_name?: string;
  column_count: number;
  metric_count: number;
  changed_on: string;
}

export interface Column {
  id: number;
  column_name: string;
  type: string;
  is_dttm: boolean;
  is_active: boolean;
}

export interface SqlMetric {
  id: number;
  metric_name: string;
  metric_type: string;
  expression: string;
  created_on: string;
}

export interface DatasetDetail extends DatasetWithCounts {
  table_columns: Column[];
  sql_metrics: SqlMetric[];
}

export interface DatasetListResponse {
  items: DatasetWithCounts[];
  total: number;
  page: number;
  page_size: number;
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

export interface CreateVirtualDatasetResponse {
  id: number;
  table_name: string;
  background_sync: boolean;
  columns?: Column[];
}

type PaginationParams = {
  page?: number;
  page_size?: number;
};

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

  async getDatasets(filters: DatasetListFilters & PaginationParams): Promise<DatasetListResponse> {
    const params = new URLSearchParams();
    if (filters.q) params.set("q", filters.q);
    if (filters.database_id) params.set("database_id", String(filters.database_id));
    if (filters.schema) params.set("schema", filters.schema);
    if (filters.type) params.set("type", filters.type);
    if (filters.owner) params.set("owner", String(filters.owner));
    if (filters.page) params.set("page", String(filters.page));
    if (filters.page_size) params.set("page_size", String(filters.page_size));
    if (filters.order_by) params.set("order_by", filters.order_by);

    const body = await request<ApiEnvelope<DatasetListResponse>>(`/api/v1/datasets?${params.toString()}`, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });

    return body.data;
  },

  async getDataset(id: number): Promise<DatasetDetail> {
    const body = await request<ApiEnvelope<DatasetDetail>>(`/api/v1/datasets/${id}`, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });

    return body.data;
  },

  async deleteDataset(id: number, force = false): Promise<void> {
    await request<{ data: null }>(`/api/v1/datasets/${id}${force ? "?force=true" : ""}`, {
      method: "DELETE",
      credentials: "include",
      headers: getAuthHeaders(),
    });
  },
};
