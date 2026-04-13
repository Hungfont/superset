import type { RegisterFormValues } from "@/lib/validations/register";
import type { LoginFormValues } from "@/lib/validations/login";
import { request } from "@/utils/request";

export type RegisterPayload = Omit<RegisterFormValues, "confirmPassword">;

export interface RegisterResponse {
  message: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
}

export interface RefreshResponse {
  access_token: string;
}

export interface LoginError extends Error {
  status: number;
  locked_until?: string;
}

export const authApi = {
  register: (data: RegisterPayload): Promise<RegisterResponse> =>
    request("/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  login: (data: LoginFormValues): Promise<LoginResponse> =>
    request("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
      credentials: "include", // send/receive HttpOnly cookie
    }),

  refresh: (): Promise<RefreshResponse> =>
    request("/api/v1/auth/refresh", {
      method: "POST",
      credentials: "include",
    }),

  verifyEmail: (hash: string): Promise<{ message?: string }> =>
    request(`/api/v1/auth/verify?hash=${encodeURIComponent(hash)}`, {
      method: "GET",
    }),

  logout: async (all = false, accessToken?: string | null): Promise<void> => {
    const headers: HeadersInit = {};
    if (accessToken) {
      headers.Authorization = `Bearer ${accessToken}`;
    }

    await request<Record<string, never>>(all ? "/api/v1/auth/logout?all=true" : "/api/v1/auth/logout", {
      method: "POST",
      credentials: "include",
      headers,
    });
  },

};
