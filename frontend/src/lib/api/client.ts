/**
 * Authenticated fetch client.
 *
 * Adds `Authorization: Bearer <token>` to every request.
 * On 401, attempts a silent token refresh (POST /api/v1/auth/refresh) once,
 * then retries the original request. On second failure, clears auth and
 * redirects to /login.
 */

import { useAuthStore } from "@/stores/authStore";

let isRefreshing = false;
// Queued callbacks waiting for the in-flight refresh to resolve.
let refreshQueue: Array<(token: string | null) => void> = [];

function drainQueue(token: string | null) {
  refreshQueue.forEach((cb) => cb(token));
  refreshQueue = [];
}

async function attemptRefresh(): Promise<string | null> {
  try {
    const res = await fetch("/api/v1/auth/refresh", {
      method: "POST",
      credentials: "include",
    });
    if (!res.ok) return null;
    const data = (await res.json()) as { access_token: string };
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

  const res = await fetch(url, { ...options, headers, credentials: "include" });

  if (res.status !== 401) {
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: "Unknown error" })) as { error?: string };
      throw Object.assign(new Error(body.error ?? "Request failed"), { status: res.status });
    }
    return res.json() as Promise<T>;
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
        fetch(url, { ...options, headers: retryHeaders, credentials: "include" })
          .then((r) => r.json() as Promise<T>)
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
  const retryRes = await fetch(url, { ...options, headers: retryHeaders, credentials: "include" });
  if (!retryRes.ok) {
    const body = await retryRes.json().catch(() => ({ error: "Unknown error" })) as { error?: string };
    throw Object.assign(new Error(body.error ?? "Request failed"), { status: retryRes.status });
  }
  return retryRes.json() as Promise<T>;
}
