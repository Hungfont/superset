import { request } from "@/utils/request";
import { useAuthStore } from "@/stores/authStore";

interface ApiEnvelope<T> {
  data: T;
}

interface UserRolesPayload {
  user_id: number;
  role_ids: number[];
}

function getAuthHeaders(contentType = false): HeadersInit {
  const accessToken = useAuthStore.getState().accessToken;
  return {
    ...(contentType ? { "Content-Type": "application/json" } : {}),
    ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
  };
}

export const userRolesApi = {
  async getUserRoles(userId: number): Promise<number[]> {
    const body = await request<ApiEnvelope<UserRolesPayload>>(`/api/v1/admin/users/${userId}/roles`, {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });
    return body.data.role_ids;
  },

  async setUserRoles(userId: number, roleIds: number[]): Promise<number[]> {
    const body = await request<ApiEnvelope<UserRolesPayload>>(`/api/v1/admin/users/${userId}/roles`, {
      method: "PUT",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify({ role_ids: roleIds }),
    });
    return body.data.role_ids;
  },
};
