import { useState } from "react";
import { getJSON, useApiQuery } from "../../shared/hooks/useApiQuery";
import { type HistoryTarget, type Revision, type RevisionDetail, type RevisionPagination, dataTitle, historyPath } from "./history";

const PAGE_SIZE = 20;

/**
 * An entry's revisions, newest first, PAGE_SIZE at a time. loadMore() adds
 * the next page; refetch() reloads what is shown (after a restore).
 */
export function useRevisions(target: HistoryTarget | null) {
  const path = target ? historyPath(target) : null;
  const [pages, setPages] = useState<{ path: string | null; count: number }>({ path, count: 1 });
  const pageCount = pages.path === path ? pages.count : 1;
  const { data, loading, error, refetch } = useApiQuery(path, async () => {
    const revisions: Revision[] = [];
    let total = 0;
    for (let page = 1; page <= pageCount; page++) {
      const json = await getJSON<Revision[]>(`${path}?page=${page}&pageSize=${PAGE_SIZE}`, "Failed to load the history");
      const batch = Array.isArray(json.data) ? json.data : [];
      revisions.push(...batch);
      total = (json.meta?.pagination as RevisionPagination | undefined)?.total ?? revisions.length;
      if (batch.length < PAGE_SIZE) break;
    }
    return { revisions, total };
  });
  const revisions = data?.revisions ?? [];
  const total = data?.total ?? 0;
  const loadMore = () => {
    setPages({ path, count: pageCount + 1 });
    refetch();
  };
  return { revisions, total, loading, error, refetch, hasMore: revisions.length < total, loadMore };
}

/** One revision with its changes from the previous version, or from the current one. */
export function useRevision(target: HistoryTarget | null, version: number | null, compare: "previous" | "current") {
  const key = target && version ? `${historyPath(target)}/${version}${compare === "current" ? "?compare=current" : ""}` : null;
  const { data, loading, error } = useApiQuery(key, async () => (await getJSON<RevisionDetail>(key!, "Failed to load the revision")).data);
  return { revision: data, loading, error };
}

export interface FeedFilters {
  collection?: string;
  singleton?: string;
  action?: string;
  actorKind?: string;
  actor?: string;
  agentRun?: number;
}

/** The change feed across entries (GET /api/revisions), newest first. */
export function useRevisionFeed(filters: FeedFilters, page: number, pageSize = 25) {
  const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
  for (const [k, v] of Object.entries(filters)) {
    if (v !== undefined && v !== "") params.set(k, String(v));
  }
  const key = `/api/revisions?${params}`;
  const { data, loading, error, refetch } = useApiQuery(key, async () => {
    const json = await getJSON<Revision[]>(key, "Failed to load content changes");
    return {
      revisions: Array.isArray(json.data) ? json.data : [],
      pagination: (json.meta?.pagination as RevisionPagination | undefined) ?? null,
    };
  });
  return { revisions: data?.revisions ?? [], pagination: data?.pagination ?? null, loading, error, refetch };
}

/** A deleted entry that is still deleted, with the title it had. */
export interface DeletedEntry {
  revision: Revision;
  title?: string;
}

/** Delete revisions scanned for "Recently deleted" (the newest ones). */
export const DELETES_SCANNED = 50;

/** Deleted entries listed, at most: each costs two requests. */
export const DELETED_LIMIT = 20;

/**
 * The collection's recently deleted entries (null loads nothing). An entry
 * restored since (it has a newer revision than its delete) is left out.
 */
export function useDeletedEntries(collection: string | null) {
  const key = collection
    ? `/api/revisions?${new URLSearchParams({ collection, action: "delete", pageSize: String(DELETES_SCANNED) })}`
    : null;
  const { data, loading, error, refetch } = useApiQuery(key, async (): Promise<DeletedEntry[]> => {
    const json = await getJSON<Revision[]>(key!, "Failed to load deleted entries");
    const seen = new Set<number>();
    const deletes = (Array.isArray(json.data) ? json.data : []).filter((r) => {
      if (seen.has(r.entityId)) return false; // newest delete first
      seen.add(r.entityId);
      return true;
    });
    const results = await Promise.all(
      deletes.slice(0, DELETED_LIMIT).map(async (rev): Promise<DeletedEntry | null> => {
        const base = historyPath({ kind: "item", collection: collection!, id: rev.entityId });
        const [latest, detail] = await Promise.allSettled([
          getJSON<Revision[]>(`${base}?page=1&pageSize=1`),
          getJSON<RevisionDetail>(`${base}/${rev.version}`),
        ]);
        if (latest.status !== "fulfilled" || latest.value.data?.[0]?.version !== rev.version) return null;
        const title = detail.status === "fulfilled" ? dataTitle(detail.value.data?.data) : undefined;
        return { revision: rev, title };
      }),
    );
    return results.filter((r): r is DeletedEntry => r !== null);
  });
  return { entries: data ?? [], loading, error, refetch };
}
