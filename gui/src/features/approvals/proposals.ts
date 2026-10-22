import { api, ApiError, type Envelope } from "../../services/api";
import { type FieldChange } from "../history/history";

/** applying: being decided right now (only listed with status=all). */
export type ProposalStatus = "pending" | "applying" | "approved" | "rejected" | "failed";
/** call: a call to a connected app's tool (admins review those). */
export type ProposalAction = "create" | "update" | "delete" | "call";

/**
 * A content change an agent with require_approval asked for (GET
 * /api/proposals). Nothing is written until an editor approves it.
 */
export interface Proposal {
  id: number;
  agentName: string;
  agentRunId?: number;
  action: ProposalAction;
  collection: string;
  itemId?: number;
  /** A translation: data holds that locale's values. */
  locale?: string;
  /** The entry's values after the change (for a locale, that locale's values). */
  data: Record<string, unknown> | null;
  /** The agent's explanation, when it gave one. */
  reason?: string;
  baseVersion: number;
  /** Lists only: a pending update or delete whose entry changed since (omitted when not). */
  stale?: boolean;
  status: ProposalStatus;
  reviewedBy?: string;
  reviewNote?: string;
  error?: string;
  createdAt: string;
  decidedAt?: string;
}

/**
 * GET /api/proposals/:id: the proposal and, while it is pending, the entry
 * as it is now and what approving would write.
 */
export interface ProposalDetail {
  proposal: Proposal;
  /** The entry changed since the agent proposed this. */
  stale: boolean;
  /** Pending: the entry's values now (or that locale's). */
  current?: Record<string, unknown>;
  currentStatus?: "draft" | "published";
  /**
   * Pending: the values approving would write. For a stale update, the
   * entry as it is now with only the agent's changed attributes.
   */
  proposed?: Record<string, unknown>;
  /** Pending: current → proposed. */
  changes?: FieldChange[];
  /** A call to a connected app's tool. */
  call?: ProposedCall;
  /** A call whose connection was deleted. */
  missing?: boolean;
}

/** The call a "call" proposal asks for, with the tool as the connection lists it now. */
export interface ProposedCall {
  connection: string;
  connectionLabel?: string;
  tool: string;
  title?: string;
  description?: string;
  arguments: Record<string, unknown> | null;
  readOnly?: boolean;
  destructive?: boolean;
}

export interface ProposalPagination {
  page: number;
  pageSize: number;
  total: number;
  pageCount: number;
}

export interface ProposalFilters {
  /** "all" or a status; pending by default. */
  status: ProposalStatus | "all";
  collection?: string;
  agent?: string;
  agentRun?: number;
}

export const STATUS_TABS: { value: ProposalFilters["status"]; label: string }[] = [
  { value: "pending", label: "Pending" },
  { value: "approved", label: "Approved" },
  { value: "rejected", label: "Rejected" },
  { value: "failed", label: "Failed" },
  { value: "all", label: "All" },
];

export const proposalStatusColor: Record<ProposalStatus, string> = {
  pending: "yellow",
  applying: "gray",
  approved: "green",
  rejected: "gray",
  failed: "red",
};

export const proposalActionColor: Record<ProposalAction, string> = {
  create: "green",
  update: "blue",
  delete: "red",
  call: "purple",
};

/** Badge text of an action: a translation is an update of one locale. */
export function proposalActionLabel(p: Pick<Proposal, "action" | "locale">): string {
  if (p.action === "update" && p.locale) return "translate";
  return p.action;
}

/** Prefix of the collection of call proposals ("app:github"). */
const APP_PREFIX = "app:";

/** The connection and tool of a call proposal ("github", "create_issue"). */
export function callTarget(p: Pick<Proposal, "collection" | "data">): { connection: string; tool: string } {
  const tool = typeof p.data?.tool === "string" ? p.data.tool : "";
  return { connection: p.collection.startsWith(APP_PREFIX) ? p.collection.slice(APP_PREFIX.length) : p.collection, tool };
}

/** "articles #12", "articles (new entry)", or "github.create_issue" for a call. */
export function proposalEntryLabel(p: Pick<Proposal, "action" | "collection" | "itemId" | "data">): string {
  if (p.action === "call") {
    const { connection, tool } = callTarget(p);
    return `${connection}.${tool}`;
  }
  return p.itemId ? `${p.collection} #${p.itemId}` : `${p.collection} (new entry)`;
}

/** The content editor with the entry open. */
export function entryLink(collection: string, itemId: number): string {
  return `/contrib?${new URLSearchParams({ collection, item: String(itemId) })}`;
}

/** The Approvals page with filters. */
export function approvalsLink(filters: Partial<ProposalFilters>): string {
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(filters)) {
    if (v !== undefined && v !== "") params.set(k, String(v));
  }
  const q = params.toString();
  return q ? `/approvals?${q}` : "/approvals";
}

/** The query string of GET /api/proposals. */
export function proposalsQuery(filters: ProposalFilters, page: number, pageSize: number): string {
  const params = new URLSearchParams({ status: filters.status, page: String(page), pageSize: String(pageSize) });
  if (filters.collection) params.set("collection", filters.collection);
  if (filters.agent) params.set("agent", filters.agent);
  if (filters.agentRun) params.set("agentRun", String(filters.agentRun));
  return `/api/proposals?${params}`;
}

/** Fired after a proposal was decided, so counts and lists reload. */
export const PROPOSALS_CHANGED = "gohead:proposals-changed";

export function notifyProposalsChanged() {
  window.dispatchEvent(new Event(PROPOSALS_CHANGED));
}

export interface ApproveResult {
  proposal: Proposal;
  item?: Record<string, unknown>;
}

/** Applies a proposal: the agent's values, or edited ones; force applies it although the entry changed since. */
export function approveProposal(id: number, body: { data?: Record<string, unknown>; force?: boolean }): Promise<Envelope<ApproveResult>> {
  return api<ApproveResult>("POST", `/api/proposals/${id}/approve`, body);
}

export interface CallResult {
  proposal: Proposal;
  /** What the tool returned. */
  result: string;
}

/** Makes a proposed call: with the agent's arguments, or edited ones. */
export function approveCall(id: number, args?: Record<string, unknown>): Promise<Envelope<CallResult>> {
  return api<CallResult>("POST", `/api/proposals/${id}/approve`, args ? { arguments: args } : {});
}

export function rejectProposal(id: number, note: string): Promise<Envelope<{ proposal: Proposal }>> {
  return api<{ proposal: Proposal }>("POST", `/api/proposals/${id}/reject`, note.trim() ? { note: note.trim() } : {});
}

/** The code of a 409 from the approve endpoint: "stale" or "decided". */
function conflictCode(err: ApiError): string | undefined {
  const d = err.details;
  if (d && typeof d === "object" && "code" in d && typeof (d as { code: unknown }).code === "string") return (d as { code: string }).code;
  return undefined;
}

/** An approval refused because the entry changed since the proposal (409, details.code "stale"). */
export function isStaleConflict(err: unknown): boolean {
  if (!(err instanceof ApiError) || err.status !== 409) return false;
  const code = conflictCode(err);
  if (code !== undefined) return code === "stale";
  // Older servers: a 409 that suggests force.
  const details = typeof err.details === "string" ? err.details : "";
  return /force/i.test(details) || /changed/i.test(err.message);
}

/** Whether two JSON values are the same (object key order aside). */
export function sameJSON(a: unknown, b: unknown): boolean {
  if (a === b) return true;
  if (typeof a !== "object" || typeof b !== "object" || a === null || b === null) return false;
  if (Array.isArray(a) !== Array.isArray(b)) return false;
  if (Array.isArray(a) && Array.isArray(b)) return a.length === b.length && a.every((v, i) => sameJSON(v, b[i]));
  const ao = a as Record<string, unknown>;
  const bo = b as Record<string, unknown>;
  const keys = Object.keys(ao);
  return keys.length === Object.keys(bo).length && keys.every((k) => Object.prototype.hasOwnProperty.call(bo, k) && sameJSON(ao[k], bo[k]));
}

/** Parses the "edit and approve" JSON: an object, or an error message. */
export function parseEditedData(text: string): { value?: Record<string, unknown>; error?: string } {
  if (!text.trim()) return { error: "Enter the values as a JSON object." };
  try {
    const v: unknown = JSON.parse(text);
    if (typeof v !== "object" || v === null || Array.isArray(v)) return { error: "The values must be a JSON object, like the proposal's." };
    return { value: v as Record<string, unknown> };
  } catch (e) {
    return { error: `Invalid JSON: ${e instanceof Error ? e.message : String(e)}` };
  }
}
