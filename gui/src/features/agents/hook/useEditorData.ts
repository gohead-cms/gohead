import { type AgentFunctionInfo, type Schema } from "../../../shared/types";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";

/** The built-in functions agents can use (GET /admin/agent-functions). */
export function useFunctionCatalog() {
  const { data, loading, error } = useApiQuery("/admin/agent-functions", async () => {
    const json = await getJSON<AgentFunctionInfo[]>("/admin/agent-functions", "Failed to load the function catalog");
    return Array.isArray(json.data) ? json.data : [];
  });
  return { catalog: data ?? [], loading, error };
}

export interface CollectionChoice {
  name: string;
  draftAndPublish: boolean;
}

/** Collection names for trigger and scope pickers. */
export function useCollectionChoices() {
  const path = "/admin/collections?pageSize=1000";
  const { data, loading, error } = useApiQuery(path, async () => {
    const json = await getJSON<{ schema: Schema }[]>(path, "Failed to load collections");
    const list = Array.isArray(json.data) ? json.data : [];
    return list
      .map((c): CollectionChoice => ({
        name: c.schema.collectionName ?? "",
        draftAndPublish: Boolean(c.schema.options?.draftAndPublish),
      }))
      .filter((c) => c.name)
      .sort((a, b) => a.name.localeCompare(b.name));
  });
  return { collections: data ?? [], loading, error };
}
