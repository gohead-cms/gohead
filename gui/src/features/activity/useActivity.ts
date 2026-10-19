import { type AgentNodeData, type AgentRun, type AgentUsage, type RunStatus } from "../../shared/types";
import { getJSON, useApiQuery } from "../../shared/hooks/useApiQuery";
import { type RunPagination } from "../agents/hook/useAgentRuns";

/** Usage is loaded for at most this many agents, to keep the page light. */
export const USAGE_AGENT_LIMIT = 20;

export const PAGE_SIZE = 25;

/** Runs of every agent, newest first (GET /admin/runs). */
export function useAllRuns(status: RunStatus | "", page: number) {
  const params = new URLSearchParams({ page: String(page), pageSize: String(PAGE_SIZE) });
  if (status) params.set("status", status);
  const key = `/admin/runs?${params}`;
  const { data, loading, error, refetch } = useApiQuery(key, async () => {
    const json = await getJSON<AgentRun[]>(key, "Failed to load runs");
    return {
      runs: Array.isArray(json.data) ? json.data : [],
      pagination: (json.meta?.pagination as RunPagination | undefined) ?? null,
    };
  });
  return { runs: data?.runs ?? [], pagination: data?.pagination ?? null, loading, error, refetch };
}

export interface AgentUsageRow {
  id?: number;
  name: string;
  usage: AgentUsage | null;
  error: string | null;
}

export interface UsageSummary {
  rows: AgentUsageRow[];
  /** Agents left out because there are more than USAGE_AGENT_LIMIT. */
  skipped: number;
}

/**
 * The agents, and the usage of each over the last hour and 24 hours. One
 * request per agent (up to USAGE_AGENT_LIMIT), in parallel; a failing agent
 * does not fail the others.
 */
export function useUsageSummary() {
  const { data, loading, error, refetch } = useApiQuery("activity-usage", async (): Promise<UsageSummary> => {
    const json = await getJSON<AgentNodeData[]>("/admin/agents", "Failed to load agents");
    const agents = Array.isArray(json.data) ? json.data : [];
    const shown = agents.slice(0, USAGE_AGENT_LIMIT);
    const results = await Promise.allSettled(
      shown.map((a) => getJSON<AgentUsage>(`/admin/agents/${encodeURIComponent(a.name)}/usage`, "Failed to load usage")),
    );
    const rows = shown.map((a, i): AgentUsageRow => {
      const r = results[i];
      return r.status === "fulfilled"
        ? { id: a.id, name: a.name, usage: r.value.data ?? null, error: null }
        : { id: a.id, name: a.name, usage: null, error: r.reason instanceof Error ? r.reason.message : "Failed to load usage" };
    });
    return { rows, skipped: agents.length - shown.length };
  });
  return { summary: data, loading, error, refetch };
}
