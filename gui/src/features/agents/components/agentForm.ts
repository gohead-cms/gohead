import { type AgentFunctionInfo, type AgentLimits, type AgentNodeData } from "../../../shared/types";

/** The value the API returns in place of a stored secret; sending it back keeps the secret. */
export const SECRET_MASK = "********";

export const PROVIDERS = ["openai", "anthropic", "ollama"] as const;

export const TRIGGER_TYPES: { value: string; label: string }[] = [
  { value: "manual", label: "Manual (run from the admin UI only)" },
  { value: "collection_event", label: "Collection event" },
  { value: "cron", label: "Schedule (cron)" },
  { value: "webhook", label: "Webhook" },
];

export const EVENTS: { value: string; label: string; draftOnly?: boolean }[] = [
  { value: "item:created", label: "Created" },
  { value: "item:updated", label: "Updated" },
  { value: "item:deleted", label: "Deleted" },
  { value: "item:published", label: "Published", draftOnly: true },
  { value: "item:unpublished", label: "Unpublished", draftOnly: true },
];

/** Functions that change content: they should always be scoped. */
export const WRITE_FUNCTIONS = ["collections.upsert_item", "collections.delete_item"];

export const LIMIT_FIELDS: { key: keyof AgentLimits; label: string; help: string; decimal?: boolean }[] = [
  { key: "max_runs_per_hour", label: "Max runs per hour", help: "Runs started in the last 60 minutes." },
  { key: "max_runs_per_day", label: "Max runs per day", help: "Runs started in the last 24 hours." },
  { key: "max_tokens_per_run", label: "Max tokens per run", help: "A run stops once it has used this many tokens." },
  { key: "max_tokens_per_day", label: "Max tokens per day", help: "Input + output tokens over 24 hours." },
  { key: "max_cost_per_day", label: "Max cost per day (USD)", help: "Needs input/output prices.", decimal: true },
];

export interface FunctionDraft {
  /** Local key for React lists; never sent. */
  uid: string;
  name: string;
  description: string;
  impl_key: string;
  collections: string[];
  /** JSON text of the parameters schema. */
  parameters: string;
}

export interface AgentDraft {
  name: string;
  system_prompt: string;
  max_turns: string;
  provider: string;
  model: string;
  /** A new key typed by the user ("" = none typed). */
  api_key: string;
  /** A key is stored on the server and is kept unless replaced or cleared. */
  api_key_stored: boolean;
  /** Passed through unchanged (mask or ""): the UI does not edit it. */
  api_secret: string;
  base_url: string;
  input_cost: string;
  output_cost: string;
  trigger_type: string;
  collection: string;
  events: string[];
  cron: string;
  /** A new webhook token ("" = keep the stored one). */
  webhook_token: string;
  webhook_token_stored: boolean;
  memory_type: string;
  session_scope: string;
  limits: Record<keyof AgentLimits, string>;
  functions: FunctionDraft[];
}

let uidCounter = 0;
export function newUid(): string {
  uidCounter += 1;
  return `f${uidCounter}`;
}

/** A random hex token for webhook triggers (32 bytes, like `openssl rand -hex 32`). */
export function generateToken(): string {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

const num = (n: number | undefined) => (n ? String(n) : "");

export function emptyFunction(): FunctionDraft {
  return { uid: newUid(), name: "", description: "", impl_key: "", collections: [], parameters: "{}" };
}

/** The form state for an existing agent, or defaults for a new one. */
export function draftFromAgent(agent: AgentNodeData | null): AgentDraft {
  const s = agent?.schema;
  const llm = s?.llmConfig;
  const t = s?.trigger;
  const l = s?.limits ?? {};
  return {
    name: agent?.name ?? "",
    system_prompt: s?.systemPrompt ?? "",
    max_turns: String(s?.maxTurns || 4),
    provider: llm?.provider || "openai",
    model: llm?.model ?? "",
    api_key: "",
    api_key_stored: Boolean(llm?.api_key),
    api_secret: llm?.api_secret ?? "",
    base_url: llm?.base_url ?? "",
    input_cost: num(llm?.input_cost_per_million),
    output_cost: num(llm?.output_cost_per_million),
    trigger_type: t?.type || "manual",
    collection: t?.event_trigger?.collection ?? "",
    events: t?.event_trigger?.events ?? [],
    cron: t?.cron ?? "",
    webhook_token: agent ? "" : generateToken(),
    webhook_token_stored: Boolean(t?.webhook_token),
    memory_type: s?.memory?.type || "in-memory",
    session_scope: s?.memory?.session_scope || "user",
    limits: {
      max_runs_per_hour: num(l.max_runs_per_hour),
      max_runs_per_day: num(l.max_runs_per_day),
      max_tokens_per_run: num(l.max_tokens_per_run),
      max_tokens_per_day: num(l.max_tokens_per_day),
      max_cost_per_day: num(l.max_cost_per_day),
    },
    functions: (s?.functions ?? []).map((f) => ({
      uid: newUid(),
      name: f.name,
      description: f.description ?? "",
      impl_key: f.impl_key ?? "",
      collections: f.collections ?? [],
      parameters: JSON.stringify(f.parameters ?? {}, null, 2),
    })),
  };
}

/** Applies a catalog entry to a function: fills what is empty or still the previous default. */
export function applyCatalog(fn: FunctionDraft, info: AgentFunctionInfo | undefined, previous: AgentFunctionInfo | undefined): FunctionDraft {
  if (!info) return { ...fn, impl_key: "" };
  const prevParams = previous ? JSON.stringify(previous.parameters ?? {}, null, 2) : "";
  const paramsUntouched = fn.parameters.trim() === "" || fn.parameters.trim() === "{}" || fn.parameters === prevParams;
  const descUntouched = fn.description.trim() === "" || fn.description === previous?.description;
  const prevName = previous ? defaultName(previous.impl_key) : "";
  return {
    ...fn,
    impl_key: info.impl_key,
    name: fn.name.trim() === "" || fn.name === prevName ? defaultName(info.impl_key) : fn.name,
    description: descUntouched ? info.description : fn.description,
    parameters: paramsUntouched ? JSON.stringify(info.parameters ?? {}, null, 2) : fn.parameters,
    collections: info.scoped ? fn.collections : [],
  };
}

/** "collections.upsert_item" → "upsert_item". */
export function defaultName(implKey: string): string {
  return implKey.split(".").pop() ?? implKey;
}

/** Parses a parameters textarea: the schema object, or an error message. */
export function parseParameters(text: string): { value?: Record<string, unknown>; error?: string } {
  if (text.trim() === "") return { error: "Parameters are required; use {} for none." };
  try {
    const v: unknown = JSON.parse(text);
    if (typeof v !== "object" || v === null || Array.isArray(v)) return { error: "Parameters must be a JSON object (a JSON schema)." };
    return { value: v as Record<string, unknown> };
  } catch (e) {
    return { error: `Invalid JSON: ${e instanceof Error ? e.message : String(e)}` };
  }
}

const isNonNegNumber = (v: string) => v.trim() === "" || (!isNaN(Number(v)) && Number(v) >= 0);
const isNonNegInt = (v: string) => v.trim() === "" || /^\d+$/.test(v.trim());

/** Rough check of what robfig/cron ParseStandard accepts: 5 fields or a @descriptor. */
function cronLooksValid(expr: string): boolean {
  const e = expr.trim();
  if (e.startsWith("@")) return /^@(yearly|annually|monthly|weekly|daily|midnight|hourly|every \S+)$/.test(e);
  return e.split(/\s+/).length === 5;
}

export type DraftErrors = Record<string, string>;

/**
 * Client-side validation mirroring ValidateAgentSchema. Keys: field names, or
 * `fn.<uid>.<field>` for functions.
 */
export function validateDraft(d: AgentDraft, takenNames: string[]): DraftErrors {
  const e: DraftErrors = {};
  const name = d.name.trim();
  if (!name) e.name = "A name is required.";
  else if (/[/?#%\\]/.test(name)) e.name = "The name cannot contain / ? # % or \\ (it is used in URLs).";
  else if (takenNames.includes(name)) e.name = "Another agent already has this name.";
  if (!d.system_prompt.trim()) e.system_prompt = "A system prompt is required.";
  if (!/^\d+$/.test(d.max_turns.trim()) || Number(d.max_turns) < 1) e.max_turns = "A positive whole number.";

  if (!d.provider) e.provider = "Choose a provider.";
  if (d.provider === "ollama" && !d.model.trim()) e.model = "Ollama needs a model.";
  if (d.base_url.trim() && !/^https?:\/\/\S+$/.test(d.base_url.trim())) e.base_url = "Must start with http:// or https://.";
  if (!isNonNegNumber(d.input_cost)) e.input_cost = "A positive number.";
  if (!isNonNegNumber(d.output_cost)) e.output_cost = "A positive number.";

  if (d.trigger_type === "collection_event") {
    if (!d.collection) e.collection = "Choose a collection.";
    if (d.events.length === 0) e.events = "Choose at least one event.";
  } else if (d.trigger_type === "cron") {
    if (!d.cron.trim()) e.cron = "A cron expression is required.";
    else if (!cronLooksValid(d.cron)) e.cron = "Use five fields (minute hour day month weekday), e.g. 0 7 * * 1-5.";
  } else if (d.trigger_type === "webhook") {
    if (!d.webhook_token_stored && !d.webhook_token.trim()) e.webhook_token = "Generate a token.";
  }

  if (!d.memory_type.trim()) e.memory_type = "Required.";
  if (!d.session_scope.trim()) e.session_scope = "Required.";

  for (const f of LIMIT_FIELDS) {
    const v = d.limits[f.key];
    if (!(f.decimal ? isNonNegNumber(v) : isNonNegInt(v))) e[`limits.${f.key}`] = f.decimal ? "A positive number." : "A positive whole number.";
  }
  const priced = Number(d.input_cost || 0) > 0 || Number(d.output_cost || 0) > 0;
  if (Number(d.limits.max_cost_per_day || 0) > 0 && !priced) {
    e["limits.max_cost_per_day"] = "Set input or output prices in the Model section to use a cost limit.";
  }

  const names = new Map<string, number>();
  const impls = new Map<string, number>();
  for (const f of d.functions) {
    names.set(f.name.trim(), (names.get(f.name.trim()) ?? 0) + 1);
    if (f.impl_key) impls.set(f.impl_key, (impls.get(f.impl_key) ?? 0) + 1);
  }
  for (const f of d.functions) {
    const p = `fn.${f.uid}`;
    const n = f.name.trim();
    if (!n) e[`${p}.name`] = "A name is required.";
    else if (!/^[a-zA-Z0-9_-]{1,64}$/.test(n)) e[`${p}.name`] = "Letters, digits, _ or - (max 64), as LLM providers require.";
    else if ((names.get(n) ?? 0) > 1) e[`${p}.name`] = "Two functions have this name.";
    if (!f.impl_key) e[`${p}.impl_key`] = "Choose an implementation.";
    else if ((impls.get(f.impl_key) ?? 0) > 1) e[`${p}.impl_key`] = "The server accepts each implementation once per agent.";
    const parsed = parseParameters(f.parameters);
    if (parsed.error) e[`${p}.parameters`] = parsed.error;
  }
  return e;
}

/** Warnings that do not block saving. */
export function functionWarning(f: FunctionDraft): string | null {
  if (WRITE_FUNCTIONS.includes(f.impl_key) && f.collections.length === 0) {
    return "This function can change items in every collection. Restrict it to the collections it needs.";
  }
  return null;
}

const optNumber = (v: string) => (v.trim() === "" ? undefined : Number(v));

/** The body for POST /admin/agents and PUT /admin/agents/{name}. */
export function buildPayload(d: AgentDraft): Record<string, unknown> {
  const apiKey = d.api_key.trim() ? d.api_key.trim() : d.api_key_stored ? SECRET_MASK : "";
  const trigger: Record<string, unknown> = { type: d.trigger_type };
  if (d.trigger_type === "collection_event") trigger.event_trigger = { collection: d.collection, events: d.events };
  if (d.trigger_type === "cron") trigger.cron = d.cron.trim();
  if (d.trigger_type === "webhook") trigger.webhook_token = d.webhook_token.trim() || SECRET_MASK;

  const limits: Record<string, number> = {};
  for (const f of LIMIT_FIELDS) {
    const v = optNumber(d.limits[f.key]);
    if (v) limits[f.key] = v;
  }

  return {
    name: d.name.trim(),
    system_prompt: d.system_prompt,
    max_turns: Number(d.max_turns),
    llm_config: {
      provider: d.provider,
      model: d.model.trim(),
      api_key: apiKey,
      api_secret: d.api_secret,
      base_url: d.base_url.trim() || undefined,
      input_cost_per_million: optNumber(d.input_cost),
      output_cost_per_million: optNumber(d.output_cost),
    },
    memory: { type: d.memory_type.trim(), session_scope: d.session_scope.trim() },
    trigger,
    functions: d.functions.map((f) => ({
      name: f.name.trim(),
      description: f.description.trim(),
      impl_key: f.impl_key,
      parameters: parseParameters(f.parameters).value ?? {},
      ...(f.collections.length > 0 ? { collections: f.collections } : {}),
    })),
    limits,
  };
}
