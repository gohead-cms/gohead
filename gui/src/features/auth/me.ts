import { createContext, useContext } from "react";

/** The signed-in user (GET /auth/me). */
export interface Me {
  kind: "user" | "token";
  id?: number;
  username: string;
  email?: string;
  role: "admin" | "editor" | "viewer" | string;
}

export interface MeState {
  me: Me | null;
  loading: boolean;
  /** Set when /auth/me failed (other than 401, which signs out). */
  error: string | null;
  retry: () => void;
  isAdmin: boolean;
  /** Editors and admins can write content. */
  canWrite: boolean;
}

export const MeContext = createContext<MeState>({
  me: null,
  loading: true,
  error: null,
  retry: () => {},
  isAdmin: false,
  canWrite: false,
});

/** The signed-in user and what they may do. */
export function useMe(): MeState {
  return useContext(MeContext);
}

export function meState(me: Me | null, loading: boolean, error: string | null, retry: () => void): MeState {
  const role = me?.role ?? "";
  return { me, loading, error, retry, isAdmin: role === "admin", canWrite: role === "admin" || role === "editor" };
}
