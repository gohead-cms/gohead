import { AgentNodeData } from "../../../shared/types";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";

export function useAgentsList() {
  const { data, loading, error } = useApiQuery("/admin/agents", async () => {
    const json = await getJSON<AgentNodeData[]>("/admin/agents", "Failed to fetch agents");
    return Array.isArray(json.data) ? json.data : [];
  });
  return { agents: data ?? [], loading, error };
}
