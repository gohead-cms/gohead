import { AgentRun, AgentUsage, RunStatus } from "../../../shared/types";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";

export interface RunPagination {
  page: number;
  pageSize: number;
  total: number;
  pageCount: number;
}

interface RunsPage {
  runs: AgentRun[];
  pagination: RunPagination | null;
  usage: AgentUsage | null;
}

/** Lists an agent's runs (newest first) and its usage against its limits. */
export function useAgentRuns(agentName: string | null, status: RunStatus | "", page: number) {
  const name = agentName ? encodeURIComponent(agentName) : null;
  const params = new URLSearchParams({ page: String(page), pageSize: "20" });
  if (status) params.set("status", status);
  const key = name ? `/admin/agents/${name}/runs?${params}` : null;

  const { data, loading, error, refetch } = useApiQuery(key, async (): Promise<RunsPage> => {
    const [runs, usage] = await Promise.all([
      getJSON<AgentRun[]>(key!, "Failed to load runs"),
      getJSON<AgentUsage>(`/admin/agents/${name}/usage`, "Failed to load usage"),
    ]);
    const pagination = (runs.meta?.pagination as RunPagination | undefined) ?? null;
    return { runs: runs.data || [], pagination, usage: usage.data || null };
  });

  return {
    runs: data?.runs ?? [],
    pagination: data?.pagination ?? null,
    usage: data?.usage ?? null,
    loading,
    error,
    refresh: refetch,
  };
}

/** Loads one run with its steps. */
export function useAgentRun(agentName: string | null, runId: number | null) {
  const key = agentName && runId ? `/admin/agents/${encodeURIComponent(agentName)}/runs/${runId}` : null;
  const { data, loading, error } = useApiQuery(key, async () => (await getJSON<AgentRun>(key!, "Failed to load run")).data ?? null);
  return { run: data, loading, error };
}
