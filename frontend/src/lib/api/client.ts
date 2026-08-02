/*
  Browser-side API client. Every request goes through the Next proxy at
  /api/proxy/*, so no base URL, token, or CORS handling lives here. A 401 means
  the session expired (no refresh endpoint exists); the client redirects to the
  login page rather than surfacing a raw error.
*/

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = "ApiError";
  }
}

const PROXY_PREFIX = "/api/proxy";

async function request<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const res = await fetch(`${PROXY_PREFIX}${path}`, {
    ...init,
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });

  if (res.status === 401 && typeof window !== "undefined") {
    // Session gone. Bounce to login preserving where we were.
    const from = window.location.pathname + window.location.search;
    window.location.href = `/login?from=${encodeURIComponent(from)}`;
    throw new ApiError(401, "session expired");
  }

  const text = await res.text();
  const data = text ? JSON.parse(text) : null;

  if (!res.ok) {
    const message =
      (data && (data.message || data.error)) || `request failed (${res.status})`;
    throw new ApiError(res.status, message);
  }

  return data as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "POST", body: body ? JSON.stringify(body) : undefined }),
  patch: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PATCH", body: body ? JSON.stringify(body) : undefined }),
  delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),
};

/** buildTicketQuery turns the filter object into the backend's query string. */
export function buildTicketQuery(filters: Record<string, unknown>): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(filters)) {
    if (value === undefined || value === null || value === "") continue;
    if (Array.isArray(value)) {
      for (const v of value) params.append(key, String(v));
    } else {
      params.set(key, String(value));
    }
  }
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}
