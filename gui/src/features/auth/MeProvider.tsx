import { Alert, AlertDescription, AlertIcon, Button } from "@chakra-ui/react";
import { type ReactNode, useMemo } from "react";
import { getJSON, useApiQuery } from "../../shared/hooks/useApiQuery";
import { type Me, MeContext, meState } from "./me";

/** Loads the signed-in user once for the whole app. */
export function MeProvider({ children }: { children: ReactNode }) {
  const { data, loading, error, refetch } = useApiQuery("/auth/me", async () => (await getJSON<Me>("/auth/me")).data);
  const value = useMemo(() => meState(data, loading, error, refetch), [data, loading, error, refetch]);
  return (
    <MeContext.Provider value={value}>
      {error && (
        <Alert status="error" variant="left-accent">
          <AlertIcon />
          <AlertDescription flex="1">
            Could not load your account ({error}). Menus and permissions may be incomplete.
          </AlertDescription>
          <Button size="sm" onClick={refetch} isLoading={loading}>
            Retry
          </Button>
        </Alert>
      )}
      {children}
    </MeContext.Provider>
  );
}
