import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import {
  ANY_TYPE,
  COLLECTION_ACTIONS,
  MEDIA_ACTIONS,
  type Permissions,
  SINGLETON_ACTIONS,
  singletonName,
  sortActions,
} from "../../auth/permissions";

/** A role, GET /admin/roles. */
export interface Role {
  id: number;
  name: string;
  description: string;
  builtin: boolean;
  /** The role of anonymous requests (no token): read permissions only. */
  public?: boolean;
  permissions: Permissions;
}

/** Help shown for built-in roles whose description is empty. */
export const BUILTIN_ROLE_HELP: Record<string, string> = {
  admin: "Everything, including the content model, users, tokens and settings",
  editor: "Create, edit, publish and delete all content; manage media",
  viewer: "Read-only access to all content",
};

/** The name of the role of anonymous requests. */
export const PUBLIC_ROLE = "public";

/** Help shown for the public role when its description is empty. */
export const PUBLIC_ROLE_HELP = "Anonymous visitors: API calls without a token";

/** Whether a role is the public role, which accounts and tokens cannot have. */
export function isPublicRole(r: Pick<Role, "public">): boolean {
  return !!r.public;
}

/** Mirrors the server's role name rule. */
export const ROLE_NAME_RE = /^[a-z][a-z0-9_-]{1,31}$/;

export function roleNameError(name: string): string | null {
  if (!name) return "Name is required";
  if (name.length < 2 || name.length > 32) return "Between 2 and 32 characters";
  if (!/^[a-z]/.test(name)) return "Must start with a lowercase letter";
  if (!ROLE_NAME_RE.test(name)) return "Only lowercase letters, digits, - and _";
  return null;
}

export function roleHelp(r: Pick<Role, "name" | "description">): string {
  return r.description || BUILTIN_ROLE_HELP[r.name] || (r.name === PUBLIC_ROLE ? PUBLIC_ROLE_HELP : "Custom role");
}

/** Every role: built-in ones first, then the public role, then custom ones. */
export function useRoles() {
  return useApiQuery("/admin/roles", async () => {
    const res = await getJSON<Role[]>("/admin/roles", "Failed to load roles");
    return Array.isArray(res.data) ? res.data : [];
  });
}

/** Ready-made sets of actions for a row of the permission matrix. */
export interface Preset {
  label: string;
  collection: string[];
  singleton: string[];
}

export const PRESETS: Preset[] = [
  { label: "No access", collection: [], singleton: [] },
  { label: "Read only", collection: ["read"], singleton: ["read"] },
  { label: "Writer (no publish)", collection: ["read", "create", "update"], singleton: ["read", "update"] },
  { label: "Editor", collection: [...COLLECTION_ACTIONS], singleton: [...SINGLETON_ACTIONS] },
];

export const MEDIA_PRESETS: { label: string; actions: string[] }[] = [
  { label: "No access", actions: [] },
  { label: "Read only", actions: ["read"] },
  { label: "Upload (no delete)", actions: ["read", "create", "update"] },
  { label: "Full", actions: [...MEDIA_ACTIONS] },
];

function keyLabel(key: string, only: boolean): string {
  if (key === ANY_TYPE) return only ? "all types" : "other types";
  const single = singletonName(key);
  return single !== null ? `${single} (single)` : key;
}

/**
 * A one-line summary of a role's permissions, e.g.
 * "articles: read, create, update · other types: read · media: read".
 */
export function permissionSummary(p: Permissions): string {
  const content = Object.entries(p.content ?? {});
  // "*" last, the specific types first, by name.
  content.sort(([a], [b]) => (a === ANY_TYPE ? 1 : b === ANY_TYPE ? -1 : a.localeCompare(b)));
  const parts = content.map(([key, actions]) => {
    const list = sortActions(actions ?? []);
    return `${keyLabel(key, content.length === 1)}: ${list.length ? list.join(", ") : "no access"}`;
  });
  const media = sortActions(p.media ?? [], MEDIA_ACTIONS);
  if (media.length) parts.push(`media: ${media.join(", ")}`);
  return parts.length ? parts.join(" · ") : "No access";
}

/**
 * What the public role lets anyone read, e.g. "articles, about (single),
 * media", or null when it grants nothing.
 */
export function publicAccessSummary(p: Permissions): string | null {
  const content = Object.entries(p.content ?? {}).filter(([, actions]) => (actions ?? []).includes("read"));
  content.sort(([a], [b]) => (a === ANY_TYPE ? 1 : b === ANY_TYPE ? -1 : a.localeCompare(b)));
  const parts = content.map(([key]) => keyLabel(key, content.length === 1));
  if ((p.media ?? []).includes("read")) parts.push("media");
  return parts.length ? parts.join(", ") : null;
}
