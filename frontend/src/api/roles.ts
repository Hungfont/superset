import { apiFetch } from "@/lib/api/client";

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
    const body = await apiFetch<ApiEnvelope<Role[]>>("/api/v1/roles", { method: "GET" });
    return body.data;
  },

  async createRole(payload: RolePayload): Promise<Role> {
    const body = await apiFetch<ApiEnvelope<Role>>("/api/v1/roles", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    return body.data;
  },

  async updateRole(roleId: number, payload: RolePayload): Promise<Role> {
    const body = await apiFetch<ApiEnvelope<Role>>(`/api/v1/roles/${roleId}`, {
      method: "PUT",
      body: JSON.stringify(payload),
    });
    return body.data;
  },

  async deleteRole(roleId: number): Promise<void> {
    await apiFetch<{ data?: unknown }>(`/api/v1/roles/${roleId}`, { method: "DELETE" });
  },
};
