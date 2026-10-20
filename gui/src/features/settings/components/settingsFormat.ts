import { formatShortTime } from "../../agents/components/runFormat";
import type { AdminUser, HeaderRow, WebhookDelivery } from "./types";

export const roleColor: Record<string, string> = {
  admin: "purple",
  editor: "blue",
  viewer: "gray",
  // The role of anonymous requests (see roles.ts).
  public: "orange",
};

/** Custom roles share one color. */
export const CUSTOM_ROLE_COLOR = "teal";

export const tokenStatusColor: Record<string, string> = {
  active: "green",
  expired: "orange",
  revoked: "red",
};

export function userRoleName(u: AdminUser): string {
  return u.role?.name ?? "";
}

export function userCreatedAt(u: AdminUser): string | undefined {
  const t = u.created_at || u.CreatedAt;
  return t && !t.startsWith("0001-") ? t : undefined;
}

/** A nullable timestamp for a table cell. */
export function optTime(iso: string | null | undefined, empty = "–"): string {
  return iso ? formatShortTime(iso) : empty;
}

/** Loose e-mail check: catches typos, the server has the final say. */
export function isEmail(s: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(s.trim());
}

/** Whether a URL is http(s); the server also blocks private networks. */
export function isHttpUrl(s: string): boolean {
  try {
    const u = new URL(s.trim());
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}

export const MIN_PASSWORD_LENGTH = 6;

/** A delivery failed when no 2xx response came back. */
export function deliveryFailed(d: Pick<WebhookDelivery, "statusCode" | "error">): boolean {
  return !!d.error || d.statusCode < 200 || d.statusCode >= 300;
}

let rowSeq = 0;

export function newHeaderRow(name = "", value = ""): HeaderRow {
  rowSeq += 1;
  return { key: `h${rowSeq}`, name, value };
}

/** Rows with a name become the headers map; blank rows are ignored. */
export function headersFromRows(rows: HeaderRow[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const r of rows) {
    const name = r.name.trim();
    if (name) out[name] = r.value;
  }
  return out;
}

// Mirrors webhooks.ValidateHeaderName on the server.
const headerNameRE = /^[A-Za-z0-9_-]+$/;
const reservedHeaders = ["content-type", "content-length", "host", "user-agent", "transfer-encoding", "connection"];

/** Client-side checks for header rows; returns an error message or null. */
export function headerRowsError(rows: HeaderRow[]): string | null {
  const seen = new Set<string>();
  for (const r of rows) {
    const name = r.name.trim();
    if (!name) {
      if (r.value) return "A header has a value but no name";
      continue;
    }
    if (!headerNameRE.test(name)) return `"${name}" is not a valid header name (letters, digits, - and _)`;
    const lower = name.toLowerCase();
    if (reservedHeaders.includes(lower) || lower.startsWith("x-gohead-")) return `"${name}" is set by GoHead and cannot be overridden`;
    if (!r.value) return `Enter a value for "${name}" (or remove it)`;
    if (seen.has(name.toLowerCase())) return `"${name}" is listed twice`;
    seen.add(name.toLowerCase());
  }
  return null;
}
