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
  

  // No body to parse: 204 No Content, or empty response
  const contentType = res.headers.get("content-type");
  const contentLength = res.headers.get("content-length");

  if (
    res.status === 204 ||
    contentLength === "0" ||
    !contentType?.includes("application/json")
  ) {
    return undefined as T;
  }

  return res.json() as Promise<T>;
}