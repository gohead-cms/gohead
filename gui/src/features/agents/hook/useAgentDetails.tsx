import { useState, useEffect } from "react";
import { apiFetchWithAuth } from "../../../services/api";
import { AgentNodeData } from "../../../shared/types";

export function useAgentDetails(agentName: string | null) {
  const [agentDetails, setAgentDetails] = useState<AgentNodeData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!agentName) {
      setAgentDetails(null);
      return;
    }
    setLoading(true);
    setError(null);
    apiFetchWithAuth(`/admin/agents/${agentName}`)
      .then(async (res) => {
        if (!res.ok) throw new Error("Failed to fetch agent details");
        const json = await res.json();
        setAgentDetails(json.data || null);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [agentName]);

  return { agentDetails, loading, error };
}
