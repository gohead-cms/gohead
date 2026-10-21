import { type AgentDefinition, type AgentFunction, type AgentFunctionInfo, type AgentLimits, type AgentNodeData, type AgentTool } from "../../../shared/types";

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

export const LIMIT_FIELDS: { key: keyof AgentLimits; label: string; help: string; decimal?: boolean }[] = [
  { key: "max_runs_per_hour", label: "Max runs per hour", help: "Runs started in the last 60 minutes." },
  { key: "max_runs_per_day", label: "Max runs per day", help: "Runs started in the last 24 hours." },
  { key: "max_tokens_per_run", label: "Max tokens per run", help: "A run stops once it has used this many tokens." },
  { key: "max_tokens_per_day", label: "Max tokens per day", help: "Input + output tokens over 24 hours." },
  { key: "max_cost_per_day", label: "Max cost per day (USD)", help: "Needs input/output prices.", decimal: true },
];

/**
 * A tool in the form: which one and where, plus optional overrides. Empty
 * overrides use the catalog's name, description and parameters.
 */
export interface ToolDraft {
  /** Local key for React lists; never sent. */
  uid: string;
  /** "content.update" (or an implementation key the catalog doesn't list). */
  use: string;
  on: string[];
  name: string;
  description: string;
  /** JSON text of a parameters schema ("" = the tool's own). */
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
  tools: ToolDraft[];
  /** Content writes become proposals an editor approves. */
  require_approval: boolean;
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

/** A new tool from the catalog; content tools start on the trigger's collection, if any. */
export function newTool(info: AgentFunctionInfo, collection = ""): ToolDraft {
  return { uid: newUid(), use: info.use, on: info.scoped && collection ? [collection] : [], name: "", description: "", parameters: "" };
}

/** The tool's catalog entry, by use name or implementation key. */
export function toolInfo(catalog: AgentFunctionInfo[], use: string): AgentFunctionInfo | undefined {
  return catalog.find((c) => c.use === use || c.impl_key === use);
}

/** The name the model sees: the override, else the catalog's. */
export function toolName(t: ToolDraft, catalog: AgentFunctionInfo[]): string {
  return t.name.trim() || toolInfo(catalog, t.use)?.name || t.use.replaceAll(".", "_");
}

function draftTool(t: AgentTool): ToolDraft {
  return {
    uid: newUid(),
    use: t.use,
    on: t.on ?? [],
    name: t.name ?? "",
    description: t.description ?? "",
    parameters: t.parameters && Object.keys(t.parameters).length > 0 ? JSON.stringify(t.parameters, null, 2) : "",
  };
}

/** Long-form functions (older API responses, old definitions) as tools. */
function toolOfFunction(f: AgentFunction): AgentTool {
  return { use: f.impl_key ?? "", on: f.collections, name: f.name, description: f.description, parameters: f.parameters };
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
    tools: (s?.tools ?? (s?.functions ?? []).map(toolOfFunction)).map(draftTool),
    require_approval: Boolean(s?.requireApproval ?? s?.require_approval),
  };
}

/**
 * The form state for a new agent from a definition (what a template
 * renders, the body of POST /admin/agents): nothing is stored yet.
 */
export function draftFromDefinition(def: AgentDefinition): AgentDraft {
  const draft = draftFromAgent({
    label: def.name,
    name: def.name,
    schema: {
      systemPrompt: def.system_prompt,
      maxTurns: def.max_turns,
      memory: def.memory,
      functions: def.functions ?? [],
      tools: def.tools,
      llmConfig: { ...def.llm_config, provider: def.llm_config?.provider ?? "", model: def.llm_config?.model ?? "" },
      trigger: def.trigger ?? { type: "manual" },
      limits: def.limits,
      requireApproval: def.require_approval,
    },
  });
  return {
    ...draft,
    // A new agent: a key or token in the definition is typed, not stored.
    api_key: def.llm_config?.api_key && def.llm_config.api_key !== SECRET_MASK ? def.llm_config.api_key : "",
    api_key_stored: false,
    api_secret: "",
    webhook_token: def.trigger?.webhook_token && def.trigger.webhook_token !== SECRET_MASK ? def.trigger.webhook_token : generateToken(),
    webhook_token_stored: false,
  };
}

/** Parses a parameters override: the schema object ("" = none), or an error message. */
export function parseParameters(text: string): { value?: Record<string, unknown>; error?: string } {
  if (text.trim() === "") return {};
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
 * `tool.<uid>.<field>` for tools.
 */
export function validateDraft(d: AgentDraft, takenNames: string[], catalog: AgentFunctionInfo[] = []): DraftErrors {
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
  for (const t of d.tools) names.set(toolName(t, catalog), (names.get(toolName(t, catalog)) ?? 0) + 1);
  for (const t of d.tools) {
    const p = `tool.${t.uid}`;
    const n = toolName(t, catalog);
    if (!t.use) e[`${p}.use`] = "Choose a tool.";
    if (!/^[a-zA-Z0-9_-]{1,64}$/.test(n)) e[`${p}.name`] = "Letters, digits, _ or - (max 64), as model providers require.";
    else if ((names.get(n) ?? 0) > 1) e[`${p}.name`] = `Two tools are named ${n}: give this one its own name.`;
    const parsed = parseParameters(t.parameters);
    if (parsed.error) e[`${p}.parameters`] = parsed.error;
  }
  return e;
}

/** Warnings that do not block saving. */
export function toolWarning(t: ToolDraft, catalog: AgentFunctionInfo[]): string | null {
  if (toolInfo(catalog, t.use)?.writes && t.on.length === 0) {
    return "This tool can change items in every collection. Choose the collections it needs.";
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
    tools: d.tools.map((t) => {
      const tool: AgentTool = { use: t.use };
      if (t.on.length > 0) tool.on = t.on;
      if (t.name.trim()) tool.name = t.name.trim();
      if (t.description.trim()) tool.description = t.description.trim();
      const params = parseParameters(t.parameters).value;
      if (params) tool.parameters = params;
      return tool;
    }),
    limits,
    require_approval: d.require_approval,
  };
}
