import { request } from "@/utils/request";
import { useAuthStore } from "@/stores/authStore";

interface ApiEnvelope<T> {
  data: T;
}

export interface UserSummary {
  id: number;
  first_name: string;
  last_name: string;
  username: string;
  email: string;
  active: boolean;
  login_count: number;
  last_login?: string;
  role_ids: number[] | null;
}

export interface CreateUserPayload {
  first_name: string;
  last_name: string;
  username: string;
  email: string;
  password: string;
  active?: boolean;
  role_ids: number[];
}

export interface UpdateUserPayload {
  first_name: string;
  last_name: string;
  username: string;
  email: string;
  active: boolean;
  role_ids: number[];
}

function getAuthHeaders(contentType = false): HeadersInit {
  const accessToken = useAuthStore.getState().accessToken;
  return {
    ...(contentType ? { "Content-Type": "application/json" } : {}),
    ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
  };
}

function normalizeUser(user: UserSummary): UserSummary {
  return {
    ...user,
    role_ids: Array.isArray(user.role_ids) ? user.role_ids : [],
  };
}

export const usersApi = {
  async getUsers(): Promise<UserSummary[]> {
    const body = await request<ApiEnvelope<UserSummary[]>>("/api/v1/admin/users", {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });
    return body.data.map(normalizeUser);
  },

  async getUser(userId: number): Promise<UserSummary> {
    const body = await request<ApiEnvelope<UserSummary>>(`/api/v1/admin/users/${userId}`, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });
    return normalizeUser(body.data);
  },

  async createUser(payload: CreateUserPayload): Promise<UserSummary> {
    const body = await request<ApiEnvelope<UserSummary>>("/api/v1/admin/users", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });
    return normalizeUser(body.data);
  },

  async updateUser(userId: number, payload: UpdateUserPayload): Promise<UserSummary> {
    const body = await request<ApiEnvelope<UserSummary>>(`/api/v1/admin/users/${userId}`, {
      method: "PUT",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });
    return normalizeUser(body.data);
  },

  async deactivateUser(userId: number): Promise<void> {
    await request<{ data: boolean }>(`/api/v1/admin/users/${userId}`, {
      method: "DELETE",
      credentials: "include",
      headers: getAuthHeaders(),
    });
  },
};
