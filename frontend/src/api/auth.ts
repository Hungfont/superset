import type { RegisterFormValues } from "@/lib/validations/register";

export type RegisterPayload = Omit<RegisterFormValues, "confirmPassword">;

export interface RegisterResponse {
  message: string;
}

export interface ApiError {
  error: string;
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
};
