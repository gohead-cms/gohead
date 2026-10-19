import type { RunStatus } from "../../shared/types";

/** GET /admin/overview (see pkg/storage/overview.go). */
export interface OverviewResponse {
  version: string;
  overview: Overview;
}

export interface Overview {
  counts: {
    collections: number;
    singletons: number;
    components: number;
    items: number;
    media: number;
    mediaBytes: number;
    users: number;
    agents: number;
    webhooks: number;
    apiTokens: number;
  };
  runs24h: {
    total: number;
    succeeded: number;
    failed: number;
    limited: number;
    running: number;
    inputTokens: number;
    outputTokens: number;
    costUsd: number;
  };
  deliveries24h: { total: number; failed: number };
  recentItems: RecentItem[] | null;
  recentRuns: RecentRun[] | null;
  recentDeliveries: RecentDelivery[] | null;
}

export interface RecentItem {
  id: number;
  collection: string;
  title: string;
  updatedAt: string;
}

export interface RecentRun {
  id: number;
  agentId: number;
  agentName: string;
  status: RunStatus;
  triggerType: string;
  startedAt: string;
  durationMs: number;
  costUsd: number;
}

export interface RecentDelivery {
  id: number;
  webhookId: number;
  event: string;
  collection: string;
  itemId: number;
  attempt: number;
  statusCode: number;
  error: string;
  createdAt: string;
}

/** "1.4 MB" */
export function formatBytes(bytes: number): string {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  const v = bytes / 1024 ** i;
  return `${v >= 10 || i === 0 ? Math.round(v) : v.toFixed(1)} ${units[i]}`;
}

/** A delivery failed unless the endpoint answered 2xx (0: no answer). */
export const deliveryFailed = (statusCode: number) => statusCode < 200 || statusCode >= 300;

/** "5 min ago", "3 h ago", "2 d ago", relative to now. */
export function timeAgo(iso: string, now = Date.now()): string {
  const t = new Date(iso).getTime();
  if (isNaN(t)) return iso;
  const s = Math.max(0, Math.round((now - t) / 1000));
  if (s < 60) return "just now";
  if (s < 3600) return `${Math.floor(s / 60)} min ago`;
  if (s < 86400) return `${Math.floor(s / 3600)} h ago`;
  return `${Math.floor(s / 86400)} d ago`;
}

/** Share of runs that succeeded, among finished ones, as "92%" (or "–"). */
export function successRate(runs: Overview["runs24h"]): string {
  const finished = runs.succeeded + runs.failed + runs.limited;
  return finished ? `${Math.round((runs.succeeded / finished) * 100)}%` : "–";
}
