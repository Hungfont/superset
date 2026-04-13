import { request } from "@/utils/request";
import { useAuthStore } from "@/stores/authStore";

export interface Role {
  id: number;
  name: string;
  user_count: number;
  permission_count: number;
  built_in: boolean;
}

export interface RolePayload {
  name: string;
}

interface ApiEnvelope<T> {
  data: T;
}

export const rolesApi = {
  async getRoles(): Promise<Role[]> {
    const accessToken = useAuthStore.getState().accessToken;
    const headers: HeadersInit = accessToken ? { Authorization: `Bearer ${accessToken}` } : {};
    const body = await request<ApiEnvelope<Role[]>>("/api/v1/roles", {
      method: "GET",
      credentials: "include",
      headers,
    });
    return body.data;
  },

  async createRole(payload: RolePayload): Promise<Role> {
    const accessToken = useAuthStore.getState().accessToken;
    const headers: HeadersInit = {
      "Content-Type": "application/json",
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
    };
    const body = await request<ApiEnvelope<Role>>("/api/v1/roles", {
      method: "POST",
      credentials: "include",
      headers,
      body: JSON.stringify(payload),
    });
    return body.data;
  },

  async updateRole(roleId: number, payload: RolePayload): Promise<Role> {
    const accessToken = useAuthStore.getState().accessToken;
    const headers: HeadersInit = {
      "Content-Type": "application/json",
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
    };
    const body = await request<ApiEnvelope<Role>>(`/api/v1/roles/${roleId}`, {
      method: "PUT",
      credentials: "include",
      headers,
      body: JSON.stringify(payload),
    });
    return body.data;
  },

  async deleteRole(roleId: number): Promise<void> {
    const accessToken = useAuthStore.getState().accessToken;
    const headers: HeadersInit = accessToken ? { Authorization: `Bearer ${accessToken}` } : {};
    await request<{ data?: unknown }>(`/api/v1/roles/${roleId}`, {
      method: "DELETE",
      credentials: "include",
      headers,
    });
  },
};
