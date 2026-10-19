import { useEffect } from "react";
import { getJSON, useApiQuery } from "../../shared/hooks/useApiQuery";
import {
  PROPOSALS_CHANGED,
  type Proposal,
  type ProposalDetail,
  type ProposalFilters,
  type ProposalPagination,
  proposalsQuery,
} from "./proposals";

export const PAGE_SIZE = 20;

/** How often the pending count of the sidebar is reloaded. */
const COUNT_POLL_MS = 60_000;

/** Reloads when a proposal is decided anywhere in the app, and when the window gets the focus back. */
function useReloadOnChange(refetch: () => void, onFocus: boolean) {
  useEffect(() => {
    window.addEventListener(PROPOSALS_CHANGED, refetch);
    if (onFocus) window.addEventListener("focus", refetch);
    return () => {
      window.removeEventListener(PROPOSALS_CHANGED, refetch);
      window.removeEventListener("focus", refetch);
    };
  }, [refetch, onFocus]);
}

/**
 * A page of proposals, newest first. Pending ones are checked for
 * staleness (the entry changed since), which only the details tell.
 */
export function useProposals(filters: ProposalFilters, page: number) {
  const key = proposalsQuery(filters, page, PAGE_SIZE);
  const { data, loading, error, refetch } = useApiQuery(key, async () => {
    const json = await getJSON<Proposal[]>(key, "Failed to load proposals");
    const proposals = Array.isArray(json.data) ? json.data : [];
    const pending = proposals.filter((p) => p.status === "pending" && p.action !== "create");
    const details = await Promise.allSettled(pending.map((p) => getJSON<ProposalDetail>(`/api/proposals/${p.id}`)));
    const stale = new Set<number>();
    details.forEach((d, i) => {
      if (d.status === "fulfilled" && d.value.data?.stale) stale.add(pending[i].id);
    });
    return { proposals, stale, pagination: (json.meta?.pagination as ProposalPagination | undefined) ?? null };
  });
  useReloadOnChange(refetch, false);
  return {
    proposals: data?.proposals ?? [],
    stale: data?.stale ?? new Set<number>(),
    pagination: data?.pagination ?? null,
    loading,
    error,
    refetch,
  };
}

/** One proposal with the entry as it is now and the changes it makes. */
export function useProposal(id: number | null) {
  const key = id ? `/api/proposals/${id}` : null;
  const { data, loading, error, refetch } = useApiQuery(key, async () => (await getJSON<ProposalDetail>(key!, "Failed to load the proposal")).data ?? null);
  return { detail: data, loading, error, refetch };
}

/**
 * The number of pending proposals the user may review (null while unknown
 * or when disabled). Reloads every minute, on focus, and after a decision.
 */
export function usePendingCount(enabled: boolean): number | null {
  const key = enabled ? "/api/proposals?status=pending&pageSize=1" : null;
  const { data, refetch } = useApiQuery(key, async () => {
    const json = await getJSON<Proposal[]>(key!, "Failed to count proposals");
    return (json.meta?.pagination as ProposalPagination | undefined)?.total ?? 0;
  });
  useReloadOnChange(refetch, enabled);
  useEffect(() => {
    if (!enabled) return;
    const id = window.setInterval(refetch, COUNT_POLL_MS);
    return () => window.clearInterval(id);
  }, [enabled, refetch]);
  return data;
}

/** The proposals of one agent run (all statuses), for the run details. */
export function useRunProposals(runId: number | null) {
  const key = runId ? proposalsQuery({ status: "all", agentRun: runId }, 1, 5) : null;
  const { data, loading, error } = useApiQuery(key, async () => {
    const json = await getJSON<Proposal[]>(key!, "Failed to load the run's proposals");
    return {
      proposals: Array.isArray(json.data) ? json.data : [],
      total: (json.meta?.pagination as ProposalPagination | undefined)?.total ?? 0,
    };
  });
  return { proposals: data?.proposals ?? [], total: data?.total ?? 0, loading, error };
}
