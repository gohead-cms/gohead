import { createContext, useContext } from "react";
import { type Action, allows, allowsAny, builtinPermissions, NO_PERMISSIONS, type Permissions } from "./permissions";

/** The signed-in user (GET /auth/me). */
export interface Me {
  kind: "user" | "token";
  id?: number;
  username: string;
  email?: string;
  role: "admin" | "editor" | "viewer" | string;
  /** What the role may do (built-in roles expanded). */
  permissions?: Permissions;
}

export interface MeState {
  me: Me | null;
  loading: boolean;
  /** Set when /auth/me failed (other than 401, which signs out). */
  error: string | null;
  retry: () => void;
  isAdmin: boolean;
  /** The role can create or update at least one content type. */
  canWrite: boolean;
  /** The role can read at least one content type. */
  canReadContent: boolean;
  /**
   * Whether the role may perform action on resource: a collection name,
   * singletonResource(name) or MEDIA ("media"). Mirrors the server: a type
   * without its own entry uses "*".
   */
  can: (action: Action, resource: string) => boolean;
}

export const MeContext = createContext<MeState>({
  me: null,
  loading: true,
  error: null,
  retry: () => {},
  isAdmin: false,
  canWrite: false,
  canReadContent: false,
  can: () => false,
});

/** The signed-in user and what they may do. */
export function useMe(): MeState {
  return useContext(MeContext);
}

/** The permissions of the signed-in user (none while loading). */
export function mePermissions(me: Me | null): Permissions {
  if (!me) return NO_PERMISSIONS;
  return me.permissions ?? builtinPermissions(me.role);
}

export function meState(me: Me | null, loading: boolean, error: string | null, retry: () => void): MeState {
  const perms = mePermissions(me);
  return {
    me,
    loading,
    error,
    retry,
    isAdmin: me?.role === "admin",
    canWrite: allowsAny(perms, "create") || allowsAny(perms, "update"),
    canReadContent: allowsAny(perms, "read"),
    can: (action, resource) => allows(perms, action, resource),
  };
}
