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
  latest_query?: {
    id: string;
    status: string;
    rows: number;
    error_message?: string;
  } | null;
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

export interface SavedQueryResponse {
  id: number;
  label: string;
  db_id: number;
  schema: string;
  catalog: string;
  sql: string;
  description: string;
  sql_tables: string;
  published: boolean;
  created_on: string;
  changed_on: string;
}

export interface SaveQueryRequest {
  db_id: number;
  label: string;
  schema?: string;
  catalog?: string;
  sql: string;
  description?: string;
  published?: boolean;
}

export interface SavedQueryListParams {
  q?: string;
  published?: boolean;
  page?: number;
  limit?: number;
}

export interface SavedQueryListResponse {
  items: SavedQueryResponse[];
  meta: {
    total: number;
    page: number;
    limit: number;
  };
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

export async function saveQuery(data: SaveQueryRequest): Promise<{ id: number; label: string; sql_tables: string }> {
  return request<{ id: number; label: string; sql_tables: string }>("/api/v1/sqllab/saved-queries", {
    method: "POST",
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
}

export async function fetchSavedQueries(params?: SavedQueryListParams): Promise<SavedQueryListResponse> {
  const qs = new URLSearchParams();
  if (params?.q) qs.set("q", params.q);
  if (params?.published !== undefined) qs.set("published", String(params.published));
  if (params?.page) qs.set("page", String(params.page));
  if (params?.limit) qs.set("limit", String(params.limit));
  const queryString = qs.toString();
  return request<SavedQueryListResponse>(`/api/v1/sqllab/saved-queries${queryString ? "?" + queryString : ""}`, {
    method: "GET",
    headers: getAuthHeaders(),
  });
}

export interface UpdateSavedQueryRequest {
  label?: string;
  sql?: string;
  schema?: string;
  catalog?: string;
  description?: string;
  published?: boolean;
  extra_json?: string;
}

export async function updateSavedQuery(id: number, data: UpdateSavedQueryRequest): Promise<SavedQueryResponse> {
  return request<SavedQueryResponse>(`/api/v1/sqllab/saved-queries/${id}`, {
    method: "PUT",
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
}

export async function deleteSavedQuery(id: number): Promise<{ deleted: boolean }> {
  return request<{ deleted: boolean }>(`/api/v1/sqllab/saved-queries/${id}`, {
    method: "DELETE",
    headers: getAuthHeaders(),
  });
}

export async function forkSavedQuery(id: number): Promise<SavedQueryResponse> {
  return request<SavedQueryResponse>(`/api/v1/sqllab/saved-queries/${id}/fork`, {
    method: "POST",
    headers: getAuthHeaders(),
  });
}

export interface SavedQueryUsageResponse {
  tab_count: number;
}

export async function getSavedQueryUsage(id: number): Promise<SavedQueryUsageResponse> {
  return request<SavedQueryUsageResponse>(`/api/v1/sqllab/saved-queries/${id}/usage`, {
    method: "GET",
    headers: getAuthHeaders(),
  });
}

// ── Autocomplete (SQL-010) ──

export interface AutocompleteRequest {
  word: string;
  prefix: string;
  db_id: number;
  schema: string;
}

export interface AutocompleteSuggestion {
  text: string;
  type: "keyword" | "schema" | "table" | "column" | "function";
  score: number;
  detail: string;
}

export interface AutocompleteResponse {
  suggestions: AutocompleteSuggestion[];
  cache_miss: boolean;
}

export async function autocomplete(data: AutocompleteRequest): Promise<AutocompleteResponse> {
  return request<AutocompleteResponse>("/api/v1/sqllab/autocomplete", {
    method: "POST",
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
}

// ── Schema Browser (SQL-006) ──

export interface SchemaColumnItem {
  name: string;
  data_type: string;
  is_nullable: boolean;
  default_value?: string;
  is_dttm: boolean;
}

export interface SchemaTableItem {
  table_name: string;
  table_type: string;
  expanded: boolean;
  columns?: SchemaColumnItem[] | null;
}

export interface GetSchemaResponse {
  schemas: string[];
  tables: SchemaTableItem[];
}

export interface ExpandTableResponse {
  table_name: string;
  columns: SchemaColumnItem[];
}

export async function getSchemaTables(tabId: number, forceRefresh?: boolean): Promise<GetSchemaResponse> {
  const qs = forceRefresh ? "?force_refresh=true" : "";
  return request<GetSchemaResponse>(`/api/v1/sqllab/tabs/${tabId}/schema${qs}`, {
    method: "GET",
    headers: getAuthHeaders(),
  });
}

export async function expandTable(tabId: number, tableName: string): Promise<ExpandTableResponse> {
  return request<ExpandTableResponse>(`/api/v1/sqllab/tabs/${tabId}/schema`, {
    method: "POST",
    headers: getAuthHeaders(),
    body: JSON.stringify({ table_name: tableName }),
  });
}

export async function collapseTable(tabId: number, tableName: string): Promise<void> {
  return request<void>(`/api/v1/sqllab/tabs/${tabId}/schema/${encodeURIComponent(tableName)}`, {
    method: "DELETE",
    headers: getAuthHeaders(),
  });
}

export async function clearSchemaState(tabId: number): Promise<void> {
  return request<void>(`/api/v1/sqllab/tabs/${tabId}/schema`, {
    method: "DELETE",
    headers: getAuthHeaders(),
  });
}
