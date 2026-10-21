/** A tool a connected app offers (GET /admin/connections). */
export interface ConnectionTool {
  name: string;
  title?: string;
  description?: string;
  inputSchema?: Record<string, unknown>;
  /** The server says the tool changes nothing. */
  readOnly: boolean;
  destructive?: boolean;
}

export type ConnectionAuth = "none" | "bearer" | "header";

/** A connected app: a remote MCP server whose tools agents can use. */
export interface Connection {
  id: number;
  /** Prefix of its tools ("github" → "github.create_issue"). */
  name: string;
  label: string;
  preset?: string;
  url: string;
  authType: ConnectionAuth;
  headerName?: string;
  /** A token is stored (it is never returned). */
  hasToken: boolean;
  tools: ConnectionTool[];
  toolsUpdatedAt?: string;
  /** The error of the last refresh, empty when it worked. */
  lastError?: string;
}

/** A service the admin UI offers to connect (GET /admin/connection-presets). */
export interface ConnectionPreset {
  id: string;
  label: string;
  description: string;
  url: string;
  authType: ConnectionAuth;
  headerName?: string;
  tokenLabel: string;
  tokenHelp: string;
  tokenUrl?: string;
  docsUrl?: string;
}

export interface ConnectionInput {
  name?: string;
  label?: string;
  preset?: string;
  url?: string;
  authType?: ConnectionAuth;
  headerName?: string;
  token?: string;
}

/** The name a connection gets from its label: lowercase letters, digits and dashes. */
export function slugify(label: string): string {
  const s = label
    .toLowerCase()
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .replace(/^[^a-z]+/, "")
    .slice(0, 31)
    .replace(/-+$/, "");
  return s;
}

const RESERVED = ["content", "ai", "system", "collections", "llm", "app", "apps"];

/** Client-side check of a connection name (the server checks too). */
export function nameError(name: string): string | null {
  if (!name) return "A name is required.";
  if (!/^[a-z][a-z0-9-]{1,30}$/.test(name)) return "2 to 31 lowercase letters, digits or dashes, starting with a letter.";
  if (RESERVED.includes(name)) return `"${name}" is the name of built-in tools.`;
  return null;
}
