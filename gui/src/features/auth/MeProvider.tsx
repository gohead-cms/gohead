import { Alert, Button } from "@chakra-ui/react";
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
        <Alert.Root status="error" variant="subtle" borderStartWidth="4px" borderStartColor="colorPalette.solid" alignItems="center">
          <Alert.Indicator />
          <Alert.Description flex="1">
            Could not load your account ({error}). Menus and permissions may be incomplete.
          </Alert.Description>
          <Button size="xs" variant="subtle" onClick={refetch} loading={loading}>
            Retry
          </Button>
        </Alert.Root>
      )}
      {children}
    </MeContext.Provider>
  );
}
