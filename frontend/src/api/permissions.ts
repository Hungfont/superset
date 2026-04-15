import { useAuthStore } from "@/stores/authStore";
import { request } from "@/utils/request";

interface ApiEnvelope<T> {
  data: T;
}

export interface Permission {
  id: number;
  name: string;
}

export interface ViewMenu {
  id: number;
  name: string;
}

export interface PermissionView {
  id: number;
  permission_id: number;
  view_menu_id: number;
  permission_name?: string;
  view_menu_name?: string;
}

export interface NamePayload {
  name: string;
}

export interface CreatePermissionViewPayload {
  permission_id: number;
  view_menu_id: number;
}

function getAuthHeaders(contentType = false): HeadersInit {
  const accessToken = useAuthStore.getState().accessToken;
  return {
    ...(contentType ? { "Content-Type": "application/json" } : {}),
    ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
  };
}

export const permissionsApi = {
  async getPermissions(): Promise<Permission[]> {
    const body = await request<ApiEnvelope<Permission[]>>("/api/v1/admin/permissions", {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });
    return body.data;
  },

  async createPermission(payload: NamePayload): Promise<Permission> {
    const body = await request<ApiEnvelope<Permission>>("/api/v1/admin/permissions", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });
    return body.data;
  },

  async getViewMenus(): Promise<ViewMenu[]> {
    const body = await request<ApiEnvelope<ViewMenu[]>>("/api/v1/admin/view-menus", {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });
    return body.data;
  },

  async createViewMenu(payload: NamePayload): Promise<ViewMenu> {
    const body = await request<ApiEnvelope<ViewMenu>>("/api/v1/admin/view-menus", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });
    return body.data;
  },

  async getPermissionViews(): Promise<PermissionView[]> {
    const body = await request<ApiEnvelope<PermissionView[]>>("/api/v1/admin/permission-views", {
      method: "GET",
      credentials: "include",
      headers: getAuthHeaders(),
    });
    return body.data;
  },

  async createPermissionView(payload: CreatePermissionViewPayload): Promise<PermissionView> {
    const body = await request<ApiEnvelope<PermissionView>>("/api/v1/admin/permission-views", {
      method: "POST",
      credentials: "include",
      headers: getAuthHeaders(true),
      body: JSON.stringify(payload),
    });
    return body.data;
  },

  async deletePermissionView(permissionViewId: number): Promise<void> {
    await request<{ data?: unknown }>(`/api/v1/admin/permission-views/${permissionViewId}`, {
      method: "DELETE",
      credentials: "include",
      headers: getAuthHeaders(),
    });
  },
};
