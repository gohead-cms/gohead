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
  isAdmin: boolean;
  /** Editors and admins can write content. */
  canWrite: boolean;
}

export const MeContext = createContext<MeState>({ me: null, loading: true, isAdmin: false, canWrite: false });

/** The signed-in user and what they may do. */
export function useMe(): MeState {
  return useContext(MeContext);
}

export function meState(me: Me | null, loading: boolean): MeState {
  const role = me?.role ?? "";
  return { me, loading, isAdmin: role === "admin", canWrite: role === "admin" || role === "editor" };
}
