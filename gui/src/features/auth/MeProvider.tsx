import { type ReactNode, useMemo } from "react";
import { getJSON, useApiQuery } from "../../shared/hooks/useApiQuery";
import { type Me, MeContext, meState } from "./me";

/** Loads the signed-in user once for the whole app. */
export function MeProvider({ children }: { children: ReactNode }) {
  const { data, loading } = useApiQuery("/auth/me", async () => (await getJSON<Me>("/auth/me")).data);
  const value = useMemo(() => meState(data, loading), [data, loading]);
  return <MeContext.Provider value={value}>{children}</MeContext.Provider>;
}
