// src/api.ts
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL;
import { useAuthStore } from "../store/auth";

export function apiFetch(path: string, options?: RequestInit) {
  return fetch(`${API_BASE_URL}${path}`, options)
}

export async function apiFetchWithAuth(path: string, options: RequestInit = {}) {
  const token = useAuthStore.getState().token;
  const headers = {
    ...options.headers,
    Authorization: `Bearer ${token}`,
  };
  return fetch(`${import.meta.env.VITE_API_URL || "http://localhost:8080"}${path}`, {
    ...options,
    headers,
  });
}
