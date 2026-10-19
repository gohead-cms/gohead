import { useEffect, useState } from "react";
import { type AgentRun, type AgentUsage, type RunStatus } from "../../../shared/types";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import { errorMessage } from "../../../shared/utils/errors";

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

/** How often a run that has not ended is reloaded (after the previous load settled). */
const RUN_POLL_MS = 2000;

const runInProgress = (run: AgentRun | null) => run?.status === "queued" || run?.status === "running";

interface RunState {
  key: string;
  run: AgentRun | null;
  error: string | null;
}

/**
 * Loads one run with its steps, and reloads it until it ends (queued or
 * running). Each reload is scheduled once the previous one settled, so a
 * slow server is never flooded; a failed reload keeps the run loaded last
 * and tries again.
 */
export function useAgentRun(agentName: string | null, runId: number | null) {
  const key = agentName && runId ? `/admin/agents/${encodeURIComponent(agentName)}/runs/${runId}` : null;
  const [state, setState] = useState<RunState | null>(null);

  useEffect(() => {
    if (key === null) return;
    let active = true;
    let timer: number | undefined;
    let last: AgentRun | null = null;
    const load = async () => {
      try {
        last = (await getJSON<AgentRun>(key, "Failed to load run")).data ?? null;
        if (!active) return;
        setState({ key, run: last, error: null });
      } catch (err) {
        if (!active) return;
        // Before the first load, the error is all there is to show.
        setState({ key, run: last, error: last ? null : errorMessage(err) });
        if (!last) return;
      }
      if (runInProgress(last)) timer = window.setTimeout(load, RUN_POLL_MS);
    };
    void load();
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [key]);

  const current = key !== null && state?.key === key ? state : null;
  const run = current?.run ?? null;
  return { run, loading: key !== null && current === null, error: current?.error ?? null, inProgress: runInProgress(run) };
}
