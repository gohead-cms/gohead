import { type AgentNodeData } from "../../../shared/types";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";

export function useAgentDetails(agentName: string | null) {
  const path = agentName ? `/admin/agents/${encodeURIComponent(agentName)}` : null;
  const { data, loading, error, refetch } = useApiQuery(path, async () => {
    const json = await getJSON<AgentNodeData>(path!, "Failed to fetch agent details");
    return json.data ?? null;
  });
  return { agentDetails: data, loading, error, refetch };
}
