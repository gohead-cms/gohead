import { api, type Envelope } from "../../services/api";

/** "baseline": the content an entry had before history was recorded (made by the system before its first recorded change). */
export type RevisionAction = "baseline" | "create" | "update" | "publish" | "unpublish" | "delete" | "restore";
export type ActorKind = "user" | "token" | "agent" | "system";

/** One change of an entry (lists leave the data out). */
export interface Revision {
  id: number;
  entityType: "item" | "singleton";
  entityId: number;
  version: number;
  typeName: string;
  action: RevisionAction;
  status?: "draft" | "published";
  actorKind: ActorKind;
  actorName: string;
  agentRunId?: number;
  restoredFrom?: number;
  createdAt: string;
}

export interface FieldChange {
  field: string;
  change: "added" | "removed" | "changed";
  before?: unknown;
  after?: unknown;
}

/** A revision with its stored data and the changes from another version. */
export interface RevisionDetail extends Revision {
  data: Record<string, unknown> | null;
  comparedTo: number | null;
  changes: FieldChange[];
}

export interface RevisionPagination {
  page: number;
  pageSize: number;
  total: number;
  pageCount: number;
}

export interface RestoreMeta {
  /** The new revision, or the current one when nothing changed. */
  revision?: Revision;
  dropped?: string[];
  undeleted?: boolean;
  /** The version was already the current content: no new revision. */
  unchanged?: boolean;
}

/** The entry a history is about: a collection item or a singleton. */
export type HistoryTarget =
  | { kind: "item"; collection: string; id: number | string; label?: string }
  | { kind: "singleton"; name: string; label?: string };

/** The API path of an entry's history. */
export function historyPath(target: HistoryTarget): string {
  return target.kind === "item"
    ? `/api/collections/${encodeURIComponent(target.collection)}/${encodeURIComponent(String(target.id))}/revisions`
    : `/api/singleton/${encodeURIComponent(target.name)}/revisions`;
}

/** A stable key for a target, for React keys and query keys. */
export function targetKey(target: HistoryTarget): string {
  return historyPath(target);
}

/** The entry a revision belongs to. */
export function revisionTarget(rev: Revision): HistoryTarget {
  return rev.entityType === "singleton"
    ? { kind: "singleton", name: rev.typeName }
    : { kind: "item", collection: rev.typeName, id: rev.entityId };
}

/** "articles #12" or the singleton name. */
export function targetLabel(target: HistoryTarget): string {
  if (target.label) return target.label;
  return target.kind === "item" ? `${target.collection} #${target.id}` : target.name;
}

/** Makes a version the entry's current content. */
export function restoreRevision(target: HistoryTarget, version: number): Promise<Envelope<Record<string, unknown>>> {
  return api<Record<string, unknown>>("POST", `${historyPath(target)}/${version}/restore`);
}

/** The toast after a restore: info when nothing changed, success otherwise. */
export function restoreToast(version: number, meta: RestoreMeta | undefined): { status: "success" | "info"; title: string; description: string } {
  if (meta?.unchanged) {
    return { status: "info", title: "Already the current content", description: `Version ${version} matches the current content: nothing changed.` };
  }
  return {
    status: "success",
    title: meta?.undeleted ? "Entry brought back" : "Version restored",
    description: restoredDescription(version, meta),
  };
}

/** The toast text after a restore: what came back and what was left out. */
export function restoredDescription(version: number, meta: RestoreMeta | undefined): string {
  const parts = [meta?.undeleted ? `Version ${version} restored: the entry is brought back.` : `Version ${version} is now the current content.`];
  const dropped = meta?.dropped ?? [];
  if (dropped.length > 0) {
    parts.push(`Left out (no longer in the type): ${dropped.join(", ")}.`);
  }
  return parts.join(" ");
}

export const actionColor: Record<RevisionAction, string> = {
  baseline: "gray",
  create: "green",
  update: "blue",
  publish: "teal",
  unpublish: "orange",
  delete: "red",
  restore: "purple",
};

export const actorKindColor: Record<ActorKind, string> = {
  user: "gray",
  token: "cyan",
  agent: "purple",
  system: "yellow",
};

/** Badge text of an action. */
export const actionLabel: Record<RevisionAction, string> = {
  baseline: "before history",
  create: "create",
  update: "update",
  publish: "publish",
  unpublish: "unpublish",
  delete: "delete",
  restore: "restore",
};

/** Password attributes come back redacted as this string. */
export const REDACTED = "********";

/** A readable title for an entry's data: its title, name or first short text. */
export function dataTitle(data: Record<string, unknown> | null | undefined): string | undefined {
  if (!data) return undefined;
  for (const key of ["title", "name", "label", "slug"]) {
    const v = data[key];
    if (typeof v === "string" && v.trim()) return v;
  }
  for (const v of Object.values(data)) {
    if (typeof v === "string" && v.trim() && v.length <= 80) return v;
  }
  return undefined;
}

/** The agents page with one of an agent's runs open. */
export function agentRunLink(agentName: string, runId: number): string {
  return `/agents?${new URLSearchParams({ name: agentName, run: String(runId) })}`;
}
