/**
 * Authenticated fetch client.
 *
 * Adds `Authorization: Bearer <token>` to every request.
 * On 401, attempts a silent token refresh (POST /api/v1/auth/refresh) once,
 * then retries the original request. On second failure, clears auth and
 * redirects to /login.
 */

import { useAuthStore } from "@/stores/authStore";
import { request } from "@/utils/request";

let isRefreshing = false;
// Queued callbacks waiting for the in-flight refresh to resolve.
let refreshQueue: Array<(token: string | null) => void> = [];

function drainQueue(token: string | null) {
  refreshQueue.forEach((cb) => cb(token));
  refreshQueue = [];
}

async function attemptRefresh(): Promise<string | null> {
  try {
    const data = await request<{ access_token: string }>("/api/v1/auth/refresh", {
      method: "POST",
      credentials: "include",
    });
    return data.access_token;
  } catch {
    return null;
  }
}

export function isTokenExpired(token: string): boolean {
  try {
    const claims = JSON.parse(atob(token.split(".")[1])) as { exp?: number };
    // No exp claim or malformed → treat as expired (fail-secure)
    if (typeof claims.exp !== "number") return true;
    return claims.exp * 1000 <= Date.now();
  } catch {
    return true; // malformed token → force re-login
  }
}

export async function apiFetch<T>(url: string, options: RequestInit = {}): Promise<T> {
  let token = useAuthStore.getState().accessToken;

  // Client-side exp check: refresh proactively before sending to avoid a round-trip 401.
  if (token && isTokenExpired(token)) {
    const refreshed = await attemptRefresh();
    if (refreshed === null) {
      useAuthStore.getState().clearAuth();
      window.location.replace("/login");
      throw Object.assign(new Error("Session expired"), { status: 401 });
    }
    useAuthStore.getState().setAccessToken(refreshed);
    token = refreshed;
  }

  const headers = new Headers(options.headers);
  headers.set("Content-Type", "application/json");
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  try {
    return await request<T>(url, { ...options, headers, credentials: "include" });
  } catch (error) {
    const status = (error as { status?: number })?.status;
    if (status !== 401) {
      throw error;
    }
  }

  // --- 401 path: attempt silent refresh ---

  // If another request is already refreshing, queue this one
  if (isRefreshing) {
    return new Promise<T>((resolve, reject) => {
      refreshQueue.push((newToken) => {
        if (newToken === null) {
          reject(Object.assign(new Error("Session expired"), { status: 401 }));
          return;
        }
        const retryHeaders = new Headers(headers);
        retryHeaders.set("Authorization", `Bearer ${newToken}`);
        request<T>(url, { ...options, headers: retryHeaders, credentials: "include" })
          .then(resolve)
          .catch(reject);
      });
    });
  }

  isRefreshing = true;
  const newToken = await attemptRefresh();
  isRefreshing = false;

  if (newToken === null) {
    drainQueue(null);
    useAuthStore.getState().clearAuth();
    window.location.replace("/login");
    throw Object.assign(new Error("Session expired"), { status: 401 });
  }

  useAuthStore.getState().setAccessToken(newToken);
  drainQueue(newToken);

  // Retry original request with new token
  const retryHeaders = new Headers(headers);
  retryHeaders.set("Authorization", `Bearer ${newToken}`);
  return request<T>(url, { ...options, headers: retryHeaders, credentials: "include" });
}
