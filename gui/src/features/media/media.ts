import { API_BASE_URL, apiFetchWithAuth } from "../../services/api";

/** A file of the media library, as GET /api/media returns it. */
export interface MediaItem {
  id: number;
  url: string;
  name: string;
  mime: string;
  size: number;
  width: number | null;
  height: number | null;
  hash?: string;
  alt: string;
  caption: string;
  uploadedBy?: string;
  createdAt: string;
  updatedAt: string;
}

export interface MediaPagination {
  page: number;
  pageSize: number;
  total: number;
  pageCount: number;
}

export const PAGE_SIZE = 24;

/** Type filters, mapped to the API's `mime` prefix filter. */
export const TYPE_FILTERS = [
  { value: "", label: "All types" },
  { value: "image/", label: "Images" },
  { value: "video/", label: "Video" },
  { value: "audio/", label: "Audio" },
  { value: "application/", label: "Documents" },
  { value: "text/", label: "Text" },
] as const;

export function listPath(page: number, mime: string, search: string): string {
  const q = new URLSearchParams({ page: String(page), pageSize: String(PAGE_SIZE) });
  if (mime) q.set("mime", mime);
  if (search.trim()) q.set("search", search.trim());
  return `/api/media?${q.toString()}`;
}

/** Relative URLs (/uploads/...) are served by the API, not by the admin UI. */
export function absoluteUrl(url: string): string {
  return url.startsWith("/") ? `${API_BASE_URL}${url}` : url;
}

export function isImage(mime: string): boolean {
  return mime.startsWith("image/");
}

/** Upper-case extension of a file name ("PDF"), or the mime subtype. */
export function extension(name: string, mime: string): string {
  const dot = name.lastIndexOf(".");
  if (dot > 0 && dot < name.length - 1) return name.slice(dot + 1).toUpperCase().slice(0, 5);
  return (mime.split("/")[1] || "file").toUpperCase().slice(0, 5);
}

export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MB`;
  return `${(n / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

export function dimensions(m: Pick<MediaItem, "width" | "height">): string | null {
  return m.width && m.height ? `${m.width} × ${m.height}` : null;
}

/**
 * Uploads one file to POST /api/media. The browser sets the multipart
 * Content-Type (with its boundary), so none is passed here. Throws an Error
 * with the server's message (type not allowed, too large…).
 */
export async function uploadFile(file: File): Promise<MediaItem> {
  const body = new FormData();
  body.append("files", file);
  const res = await apiFetchWithAuth("/api/media", { method: "POST", body });
  const json: unknown = await res.json().catch(() => null);
  if (!res.ok) throw new Error(uploadError(res.status, json));
  const data = (json as { data?: MediaItem[] } | null)?.data;
  if (!Array.isArray(data) || data.length === 0) throw new Error("The server returned no file");
  return data[0];
}

function uploadError(status: number, json: unknown): string {
  const err = (json as { error?: { message?: string; details?: unknown } } | null)?.error;
  let msg = err?.message || defaultUploadError(status);
  if (typeof err?.details === "string" && err.details) msg += `: ${err.details}`;
  return msg;
}

function defaultUploadError(status: number): string {
  if (status === 413) return "File too large";
  if (status === 415) return "File type not allowed";
  return `Upload failed (${status})`;
}

export type UploadStatus = "pending" | "uploading" | "done" | "error";

export interface UploadTask {
  id: string;
  name: string;
  size: number;
  status: UploadStatus;
  error?: string;
}
