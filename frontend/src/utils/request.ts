export interface ApiError {
  error: string;
  locked_until?: string;
}

export async function request<T>(url: string, options: RequestInit): Promise<T> {
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) {
    const body = (await res.json().catch(() => ({ error: "Unknown error" }))) as ApiError;
    throw Object.assign(new Error(body.error ?? "Request failed"), { status: res.status, locked_until: body.locked_until });
  }
  return res.json() as Promise<T>;
}