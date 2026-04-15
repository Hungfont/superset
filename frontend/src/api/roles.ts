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

interface RolePermissionsPayload {
  role_id: number;
  permission_view_ids: number[];
}

interface RolePermissionsRequest {
  permission_view_ids: number[];
}

function getAuthHeaders(contentType = false): HeadersInit {
  const accessToken = useAuthStore.getState().accessToken;
  return {
    ...(contentType ? { "Content-Type": "application/json" } : {}),
    ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
  };
}

export const rolesApi = {
  async getRoles(): Promise<Role[]> {
    const body = await request<ApiEnvelope<Role[]>>("/api/v1/admin/roles", {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });
    return body.data;
  },

  async createRole(payload: RolePayload): Promise<Role> {
    const body = await request<ApiEnvelope<Role>>("/api/v1/admin/roles", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });
    return body.data;
  },

  async updateRole(roleId: number, payload: RolePayload): Promise<Role> {
    const body = await request<ApiEnvelope<Role>>(`/api/v1/admin/roles/${roleId}`, {
      method: "PUT",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });
    return body.data;
  },

  async deleteRole(roleId: number): Promise<void> {
    await request<{ data?: unknown }>(`/api/v1/admin/roles/${roleId}`, {
      method: "DELETE",
      credentials: "include",
      headers: getAuthHeaders(),
    });
  },

  async getRolePermissions(roleId: number): Promise<number[]> {
    const body = await request<ApiEnvelope<RolePermissionsPayload>>(`/api/v1/admin/roles/${roleId}/permissions`, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });
    return body.data.permission_view_ids;
  },

  async setRolePermissions(roleId: number, permissionViewIds: number[]): Promise<number[]> {
    const payload: RolePermissionsRequest = { permission_view_ids: permissionViewIds };
    const body = await request<ApiEnvelope<RolePermissionsPayload>>(`/api/v1/admin/roles/${roleId}/permissions`, {
      method: "PUT",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });
    return body.data.permission_view_ids;
  },

  async addRolePermissions(roleId: number, permissionViewIds: number[]): Promise<number[]> {
    const payload: RolePermissionsRequest = { permission_view_ids: permissionViewIds };
    const body = await request<ApiEnvelope<RolePermissionsPayload>>(`/api/v1/admin/roles/${roleId}/permissions/add`, {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });
    return body.data.permission_view_ids;
  },

  async removeRolePermission(roleId: number, permissionViewId: number): Promise<number[]> {
    const body = await request<ApiEnvelope<RolePermissionsPayload>>(`/api/v1/admin/roles/${roleId}/permissions/${permissionViewId}`, {
      method: "DELETE",
      credentials: "include",
      headers: getAuthHeaders(),
    });
    return body.data.permission_view_ids;
  },
};
