import type { RegisterFormValues } from "@/lib/validations/register";
import type { LoginFormValues } from "@/lib/validations/login";

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

export interface ApiError {
  error: string;
  locked_until?: string;
}

async function request<T>(url: string, options: RequestInit): Promise<T> {
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) {
    const body = (await res.json().catch(() => ({ error: "Unknown error" }))) as ApiError;
    throw Object.assign(new Error(body.error ?? "Request failed"), { status: res.status });
  }
  return res.json() as Promise<T>;
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
      credentials: "include", // HttpOnly refresh token cookie
    }),
};
