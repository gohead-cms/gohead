import { useCallback, useEffect, useState } from "react";
import { apiFetchWithAuth } from "../../../services/api";
import { AgentRun, AgentUsage, RunStatus } from "../../../shared/types";

export interface RunPagination {
  page: number;
  pageSize: number;
  total: number;
  pageCount: number;
}

/** Lists an agent's runs (newest first) and its usage against its limits. */
export function useAgentRuns(agentName: string | null, status: RunStatus | "", page: number) {
  const [runs, setRuns] = useState<AgentRun[]>([]);
  const [pagination, setPagination] = useState<RunPagination | null>(null);
  const [usage, setUsage] = useState<AgentUsage | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tick, setTick] = useState(0);

  const refresh = useCallback(() => setTick((t) => t + 1), []);

  useEffect(() => {
    if (!agentName) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    const params = new URLSearchParams({ page: String(page), pageSize: "20" });
    if (status) params.set("status", status);
    const name = encodeURIComponent(agentName);
    Promise.all([
      apiFetchWithAuth(`/admin/agents/${name}/runs?${params}`).then(async (res) => {
        if (!res.ok) throw new Error("Failed to load runs");
        return res.json();
      }),
      apiFetchWithAuth(`/admin/agents/${name}/usage`).then(async (res) => {
        if (!res.ok) throw new Error("Failed to load usage");
        return res.json();
      }),
    ])
      .then(([runsJson, usageJson]) => {
        if (cancelled) return;
        setRuns(runsJson.data || []);
        setPagination(runsJson.meta?.pagination || null);
        setUsage(usageJson.data || null);
      })
      .catch((err) => !cancelled && setError(err.message))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [agentName, status, page, tick]);

  return { runs, pagination, usage, loading, error, refresh };
}

/** Loads one run with its steps. */
export function useAgentRun(agentName: string | null, runId: number | null) {
  const [run, setRun] = useState<AgentRun | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!agentName || !runId) {
      setRun(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError(null);
    apiFetchWithAuth(`/admin/agents/${encodeURIComponent(agentName)}/runs/${runId}`)
      .then(async (res) => {
        if (!res.ok) throw new Error("Failed to load run");
        const json = await res.json();
        if (!cancelled) setRun(json.data || null);
      })
      .catch((err) => !cancelled && setError(err.message))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [agentName, runId]);

  return { run, loading, error };
}
