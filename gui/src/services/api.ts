import { useAuthStore } from "./auth";

declare global {
  interface Window {
    // Injected at container start by /config.js (see gui/nginx.conf.template).
    __GOHEAD_CONFIG__?: { apiBaseUrl?: string };
  }
}

// Runtime config wins, so one image works in every environment; the build-time
// variable and localhost remain as fallbacks for `npm run dev`.
export const API_BASE_URL =
  window.__GOHEAD_CONFIG__?.apiBaseUrl ||
  import.meta.env.VITE_API_BASE_URL ||
  "http://localhost:8080";

export function apiFetch(path: string, options?: RequestInit) {
  return fetch(`${API_BASE_URL}${path}`, options);
}

/** fetch with the session token. An expired or revoked session signs out. */
export async function apiFetchWithAuth(path: string, options: RequestInit = {}) {
  const token = useAuthStore.getState().token;
  const headers = {
    ...options.headers,
    Authorization: `Bearer ${token}`,
  };
  const res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers });
  if (res.status === 401 && token) {
    useAuthStore.getState().logout();
    if (!window.location.pathname.startsWith("/login")) {
      window.location.assign("/login");
    }
  }
  return res;
}

/** An API error with the server's message and details. */
export class ApiError extends Error {
  status: number;
  details?: unknown;
  constructor(status: number, message: string, details?: unknown) {
    super(message);
    this.status = status;
    this.details = details;
  }
}

export interface Envelope<T> {
  data: T;
  meta?: Record<string, unknown>;
}

/**
 * Calls the API with a JSON body and returns the response envelope. Errors
 * throw an ApiError with the server's message (and its details appended).
 */
export async function api<T = unknown>(method: string, path: string, body?: unknown): Promise<Envelope<T>> {
  const res = await apiFetchWithAuth(path, {
    method,
    headers: body === undefined ? {} : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const json = await res.json().catch(() => null);
  if (!res.ok) {
    const err = json?.error;
    let message: string = err?.message || `Request failed (${res.status})`;
    if (typeof err?.details === "string" && err.details) message += `: ${err.details}`;
    throw new ApiError(res.status, message, err?.details);
  }
  return (json ?? { data: null }) as Envelope<T>;
}
