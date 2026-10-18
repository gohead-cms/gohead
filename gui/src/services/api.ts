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
import { useAuthStore } from "./auth";

export function apiFetch(path: string, options?: RequestInit) {
  return fetch(`${API_BASE_URL}${path}`, options);
}

export async function apiFetchWithAuth(path: string, options: RequestInit = {}) {
  const token = useAuthStore.getState().token;
  const headers = {
    ...options.headers,
    Authorization: `Bearer ${token}`,
  };
  
  return fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers,
  });
}