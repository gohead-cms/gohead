/**
 * Role permissions, as GET /auth/me and /admin/roles return them, and the
 * checks the server makes (pkg/auth/permissions.go).
 */

export type ContentAction = "read" | "create" | "update" | "delete" | "publish";
export type MediaAction = "read" | "create" | "update" | "delete";
export type Action = ContentAction;

/** Actions on collection items, single types and the media library. */
export const COLLECTION_ACTIONS: readonly ContentAction[] = ["read", "create", "update", "delete", "publish"];
export const SINGLETON_ACTIONS: readonly ContentAction[] = ["read", "update"];
export const MEDIA_ACTIONS: readonly MediaAction[] = ["read", "create", "update", "delete"];
/** The only action the public role (anonymous requests) may be given. */
export const PUBLIC_ACTIONS: readonly ContentAction[] = ["read"];

/** The permission key of every content type without its own entry. */
export const ANY_TYPE = "*";
/** The media library, as a resource of can(). */
export const MEDIA = "media";
const SINGLETON_PREFIX = "singleton:";

/** The permission key of a single type. */
export function singletonResource(name: string): string {
  return SINGLETON_PREFIX + name;
}

/** The single type a permission key names, or null for a collection or "*". */
export function singletonName(key: string): string | null {
  return key.startsWith(SINGLETON_PREFIX) ? key.slice(SINGLETON_PREFIX.length) : null;
}

/**
 * What a role may do. content maps a collection name, "singleton:<name>"
 * or "*" (every other type) to actions. The server may send null for an
 * empty map or list.
 */
export interface Permissions {
  content: Record<string, string[] | null> | null;
  media: string[] | null;
}

/** No access at all. */
export const NO_PERMISSIONS: Permissions = { content: {}, media: [] };

/** Everything (the admin and editor roles). */
export const ALL_PERMISSIONS: Permissions = {
  content: { [ANY_TYPE]: [...COLLECTION_ACTIONS] },
  media: [...MEDIA_ACTIONS],
};

/** Permissions of built-in roles, for an API that does not send them. */
export function builtinPermissions(role: string): Permissions {
  if (role === "admin" || role === "editor") return ALL_PERMISSIONS;
  if (role === "viewer") return { content: { [ANY_TYPE]: ["read"] }, media: [] };
  return NO_PERMISSIONS;
}

/**
 * The actions granted on a resource: a collection name, singletonResource(name)
 * or MEDIA. A type without its own entry gets those of "*".
 */
export function actionsOn(p: Permissions, resource: string): string[] {
  if (resource === MEDIA) return p.media ?? [];
  const content = p.content ?? {};
  const own = Object.prototype.hasOwnProperty.call(content, resource) ? content[resource] : content[ANY_TYPE];
  return own ?? [];
}

/** Whether the permissions grant action on resource (see actionsOn). */
export function allows(p: Permissions, action: string, resource: string): boolean {
  return actionsOn(p, resource).includes(action);
}

/** Whether action is granted on at least one content type. */
export function allowsAny(p: Permissions, action: string): boolean {
  return Object.values(p.content ?? {}).some((actions) => (actions ?? []).includes(action));
}

/**
 * Adds or removes an action, as the server reads it: any action implies
 * read, so checking one adds read and unchecking read clears them all.
 */
export function toggleAction(actions: readonly string[], action: string, on: boolean): string[] {
  if (on) {
    const next = new Set(actions);
    next.add(action);
    next.add("read");
    return [...next];
  }
  if (action === "read") return [];
  return actions.filter((a) => a !== action);
}

/** Actions sorted in the order of the list given (unknown ones last). */
export function sortActions(actions: readonly string[], order: readonly string[] = COLLECTION_ACTIONS): string[] {
  const rank = (a: string) => {
    const i = order.indexOf(a as ContentAction);
    return i === -1 ? order.length : i;
  };
  return [...new Set(actions)].sort((a, b) => rank(a) - rank(b));
}
