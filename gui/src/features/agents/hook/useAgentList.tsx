import { useState, useEffect } from "react";
import { apiFetchWithAuth } from "../../../services/api";
import { AgentNodeData } from "../../../shared/types";

export function useAgentsList() {
  const [agents, setAgents] = useState<AgentNodeData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setError(null);
    apiFetchWithAuth("/admin/agents")
      .then(async (res) => {
        if (!res.ok) throw new Error("Failed to fetch agents");
        const json = await res.json();
        setAgents(Array.isArray(json.data) ? json.data : []);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  return { agents, loading, error };
}
