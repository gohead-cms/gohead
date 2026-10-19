import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import type { AgentNodeData } from "../../../shared/types";
import type { ApiCollection } from "../../model/schema";

export interface WorkspaceData {
  collections: ApiCollection[];
  agents: AgentNodeData[];
}

/** The saved collections and agents shown on the Schema Designer canvas. */
export function useWorkspaceData() {
  return useApiQuery<WorkspaceData>("workspace", async () => {
    const [collections, agents] = await Promise.all([
      getJSON<ApiCollection[]>("/admin/collections", "Failed to load collections"),
      getJSON<AgentNodeData[]>("/admin/agents", "Failed to load agents"),
    ]);
    return { collections: collections.data ?? [], agents: agents.data ?? [] };
  });
}
