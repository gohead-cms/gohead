import { type AgentNodeData } from "../../../shared/types";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";

export function useAgentsList() {
  const { data, loading, error, refetch } = useApiQuery("/admin/agents", async () => {
    const json = await getJSON<AgentNodeData[]>("/admin/agents?pageSize=100", "Failed to fetch agents");
    return Array.isArray(json.data) ? json.data : [];
  });
  return { agents: data ?? [], loading: loading && data === null, error, refetch };
}
