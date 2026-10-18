import { RunStatus } from "../../../shared/types";

export const statusColor: Record<RunStatus, string> = {
  running: "blue",
  succeeded: "green",
  failed: "red",
  limited: "orange",
};

export function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 10_000) return `${Math.round(n / 1000)}k`;
  return n.toLocaleString();
}

export function formatCost(usd: number): string {
  if (!usd) return "–";
  if (usd < 0.01) return `$${usd.toFixed(4)}`;
  return `$${usd.toFixed(2)}`;
}

export function formatDuration(ms: number): string {
  if (!ms) return "–";
  if (ms < 1000) return `${ms} ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)} s`;
  return `${Math.floor(ms / 60_000)}m ${Math.round((ms % 60_000) / 1000)}s`;
}

export function formatTime(iso: string): string {
  const d = new Date(iso);
  return isNaN(d.getTime()) ? iso : d.toLocaleString();
}

/** Compact local time for tables: "Oct 18, 15:11". */
export function formatShortTime(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

/** Pretty-prints JSON text, or returns it unchanged. */
export function prettyJSON(text: string): string {
  if (!text) return "";
  try {
    return JSON.stringify(JSON.parse(text), null, 2);
  } catch {
    return text;
  }
}

/** One-line summary of what triggered a run. */
export function triggerSummary(triggerType: string, trigger: string): string {
  try {
    const t = JSON.parse(trigger);
    const ev = t?.collection_event;
    if (triggerType === "collection_event" && ev && typeof ev === "object") {
      const id = ev.item_id ?? ev.itemId;
      return [ev.collection, ev.event].filter(Boolean).join(" ") + (id !== undefined ? ` #${id}` : "");
    }
  } catch {
    /* not JSON */
  }
  return triggerType.replace(/_/g, " ");
}
