/**
 * Content models as code: a JSON file describing locales, components,
 * collections, singletons and public access, pulled from and pushed to a
 * GoHead instance.
 */
import type { GoHeadClient } from "../client.js";
import { GoHeadError } from "../errors.js";

export interface AttributeDef {
  type: string;
  required?: boolean;
  unique?: boolean;
  localized?: boolean;
  /** enum values */
  options?: string[];
  min?: number;
  max?: number;
  pattern?: string;
  /** relation: the target collection */
  target?: string;
  relation?: "oneToOne" | "oneToMany" | "manyToMany";
  /** component: its name */
  component?: string;
  repeatable?: boolean;
  /** media: a list of files */
  multiple?: boolean;
  /** media: MIME patterns, e.g. image/* */
  allowed_types?: string[];
}

export type Attributes = Record<string, AttributeDef>;

export interface CollectionDef {
  description?: string;
  draftAndPublish?: boolean;
  attributes: Attributes;
}

export interface SingletonDef {
  description?: string;
  attributes: Attributes;
}

export interface ComponentDef {
  description?: string;
  attributes: Attributes;
}

export interface LocaleDef {
  code: string;
  name?: string;
  default?: boolean;
}

export interface Model {
  $schema?: string;
  locales?: LocaleDef[];
  components?: Record<string, ComponentDef>;
  collections?: Record<string, CollectionDef>;
  singletons?: Record<string, SingletonDef>;
  /** What anonymous visitors may read (the public role): collection names, "singleton:<name>", "*", and media. */
  public?: { content?: Record<string, string[]>; media?: string[] };
}

const ATTR_KEYS: (keyof AttributeDef)[] = [
  "type", "required", "unique", "localized", "options", "min", "max", "pattern",
  "target", "relation", "component", "repeatable", "multiple", "allowed_types",
];

/** Normalizes an attribute as the server returns it (both list and map forms). */
export function normalizeAttribute(raw: Record<string, unknown>): AttributeDef {
  const out: Record<string, unknown> = {};
  for (const k of ATTR_KEYS) {
    let v = raw[k];
    if (v === undefined || v === null || v === false || v === "" || (Array.isArray(v) && v.length === 0)) continue;
    if (k === "target" && typeof v === "string") v = v.replace(/^api::([^.]+)\..*$/, "$1");
    out[k] = v;
  }
  if (out.type === "relation" && out.component) delete out.component;
  return out as unknown as AttributeDef;
}

function attributesFrom(raw: unknown): Attributes {
  const out: Attributes = {};
  if (Array.isArray(raw)) {
    for (const a of raw as Record<string, unknown>[]) {
      if (typeof a?.name === "string") out[a.name] = normalizeAttribute(a);
    }
  } else if (raw && typeof raw === "object") {
    for (const [name, a] of Object.entries(raw as Record<string, Record<string, unknown>>)) out[name] = normalizeAttribute(a);
  }
  return sortKeys(out);
}

function sortKeys<T>(obj: Record<string, T>): Record<string, T> {
  return Object.fromEntries(Object.entries(obj).sort(([a], [b]) => a.localeCompare(b)));
}

async function listAll(client: GoHeadClient, path: string): Promise<Record<string, unknown>[]> {
  const out: Record<string, unknown>[] = [];
  for (let page = 1; page < 1000; page++) {
    const res = await client.request<Record<string, unknown>[]>("GET", path, { query: `?page=${page}&pageSize=100` });
    const list = res.data ?? [];
    out.push(...list);
    const total = Number((res.meta as { total?: unknown } | undefined)?.total ?? NaN);
    if (list.length < 100 || (Number.isFinite(total) && out.length >= total)) break;
  }
  return out;
}

/** Reads the content model of an instance. */
export async function pullModel(client: GoHeadClient): Promise<Model> {
  const model: Model = { locales: [], components: {}, collections: {}, singletons: {} };

  for (const l of await client.locales()) {
    model.locales!.push(l.isDefault ? { code: l.code, name: l.name, default: true } : { code: l.code, name: l.name });
  }
  for (const c of await listAll(client, "/api/schema/components")) {
    model.components![String(c.name)] = withDescription({ attributes: attributesFrom(c.attributes) }, c.description);
  }
  for (const c of await listAll(client, "/api/schema/collections")) {
    const schema = (c.schema ?? {}) as Record<string, unknown>;
    const name = String(schema.collectionName ?? c.name);
    const options = (schema.options ?? {}) as { draftAndPublish?: boolean };
    const def: CollectionDef = options.draftAndPublish
      ? { draftAndPublish: true, attributes: attributesFrom(schema.attributes) }
      : { attributes: attributesFrom(schema.attributes) };
    model.collections![name] = withDescription(def, schema.description);
  }
  for (const s of await listAll(client, "/api/schema/singletons")) {
    model.singletons![String(s.name)] = withDescription({ attributes: attributesFrom(s.attributes) }, s.description);
  }
  try {
    const role = (await client.request<{ permissions: Model["public"] }>("GET", "/admin/roles/public")).data;
    const p = role?.permissions;
    if (p && (Object.keys(p.content ?? {}).length > 0 || (p.media ?? []).length > 0)) model.public = p;
  } catch (err) {
    if (!(err instanceof GoHeadError) || ![401, 403, 404].includes(err.status)) throw err;
  }
  model.components = sortKeys(model.components!);
  model.collections = sortKeys(model.collections!);
  model.singletons = sortKeys(model.singletons!);
  return model;
}

function withDescription<T extends { attributes: Attributes; description?: string }>(def: T, description: unknown): T {
  if (typeof description === "string" && description) def.description = description;
  return def;
}

// ---- push ----

export interface PushOptions {
  dryRun?: boolean;
  /** Allow removing attributes that exist on the server but not in the model. */
  allowRemovals?: boolean;
  log?: (line: string) => void;
}

export interface PlanStep {
  kind: "locale" | "component" | "collection" | "singleton" | "public";
  name: string;
  action: "create" | "update" | "unchanged";
  removed?: string[];
  changed?: string[];
  added?: string[];
}

function same(a: unknown, b: unknown): boolean {
  return JSON.stringify(canonical(a)) === JSON.stringify(canonical(b));
}

function canonical(v: unknown): unknown {
  if (Array.isArray(v)) return v.map(canonical);
  if (v && typeof v === "object") {
    return Object.fromEntries(
      Object.entries(v as Record<string, unknown>)
        .filter(([, x]) => x !== undefined && x !== false && x !== "" && !(Array.isArray(x) && x.length === 0))
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([k, x]) => [k, canonical(x)]),
    );
  }
  return v;
}

function diffAttributes(current: Attributes | undefined, wanted: Attributes): Pick<PlanStep, "added" | "removed" | "changed"> {
  const cur = current ?? {};
  const wantedNorm = Object.fromEntries(Object.entries(wanted).map(([k, v]) => [k, normalizeAttribute(v as unknown as Record<string, unknown>)]));
  return {
    added: Object.keys(wantedNorm).filter((k) => !(k in cur)),
    removed: Object.keys(cur).filter((k) => !(k in wantedNorm)),
    changed: Object.keys(wantedNorm).filter((k) => k in cur && !same(cur[k], wantedNorm[k])),
  };
}

/** The admin API's attribute map. */
function attributesBody(attrs: Attributes): Record<string, AttributeDef> {
  return Object.fromEntries(Object.entries(attrs).map(([k, v]) => [k, normalizeAttribute(v as unknown as Record<string, unknown>)]));
}

/** Collections ordered so relation targets come first (cycles are fine: see pushModel). */
function relationTargets(def: CollectionDef | SingletonDef): string[] {
  return Object.values(def.attributes ?? {})
    .filter((a) => a.type === "relation" && a.target)
    .map((a) => a.target!);
}

/**
 * Applies a model to an instance: creates or updates locales, components,
 * collections and singletons, and sets public access. Nothing that exists
 * only on the server is deleted; removing attributes of a type needs
 * allowRemovals. Needs an admin user session (roles are managed by users).
 */
export async function pushModel(client: GoHeadClient, model: Model, opts: PushOptions = {}): Promise<PlanStep[]> {
  const log = opts.log ?? (() => {});
  validateModel(model);
  const current = await pullModel(client);
  const plan: PlanStep[] = [];

  // Locales
  const curLocales = new Map((current.locales ?? []).map((l) => [l.code, l]));
  for (const l of model.locales ?? []) {
    const cur = curLocales.get(l.code);
    if (!cur) plan.push({ kind: "locale", name: l.code, action: "create" });
    else if ((l.name && l.name !== cur.name) || (l.default && !cur.default)) plan.push({ kind: "locale", name: l.code, action: "update" });
  }
  // Types
  const typeSteps = (kind: "component" | "collection" | "singleton", wanted: Record<string, { attributes: Attributes; description?: string; draftAndPublish?: boolean }> | undefined, existing: Record<string, { attributes: Attributes; description?: string; draftAndPublish?: boolean }> | undefined) => {
    for (const [name, def] of Object.entries(wanted ?? {})) {
      const cur = existing?.[name];
      if (!cur) {
        plan.push({ kind, name, action: "create", added: Object.keys(def.attributes ?? {}) });
        continue;
      }
      const d = diffAttributes(cur.attributes, def.attributes ?? {});
      const meta = (def.description ?? "") !== (cur.description ?? "") || (kind === "collection" && !!def.draftAndPublish !== !!cur.draftAndPublish);
      const changed = d.added!.length + d.removed!.length + d.changed!.length > 0 || meta;
      plan.push({ kind, name, action: changed ? "update" : "unchanged", ...d });
    }
  };
  typeSteps("component", model.components, current.components);
  typeSteps("collection", model.collections, current.collections);
  typeSteps("singleton", model.singletons, current.singletons);
  if (model.public && !same(model.public, current.public ?? { content: {}, media: [] })) {
    plan.push({ kind: "public", name: "public", action: "update" });
  }

  const removals = plan.filter((s) => (s.removed ?? []).length > 0);
  for (const s of plan) {
    if (s.action === "unchanged") continue;
    const parts = [s.added?.length ? `+${s.added.join(", +")}` : "", s.changed?.length ? `~${s.changed.join(", ~")}` : "", s.removed?.length ? `-${s.removed.join(", -")}` : ""].filter(Boolean);
    log(`${s.action === "create" ? "create" : "update"} ${s.kind === "public" ? "public access" : `${s.kind} ${s.name}`}${parts.length ? ` (${parts.join(" ")})` : ""}`);
  }
  if (removals.length > 0 && !opts.allowRemovals) {
    throw new Error(
      `the model removes attributes (${removals.map((s) => `${s.name}: ${s.removed!.join(", ")}`).join("; ")}). ` +
        "Their values stay in the database but are no longer part of the API. Pass --allow-removals to apply it.",
    );
  }
  if (opts.dryRun) return plan;

  // Locales first (localized attributes need them), then components.
  for (const l of model.locales ?? []) {
    const cur = curLocales.get(l.code);
    if (!cur) {
      await client.request("POST", "/admin/locales", { body: { code: l.code, name: l.name ?? l.code } });
      if (l.default) await client.request("PUT", `/admin/locales/${encodeURIComponent(l.code)}`, { body: { isDefault: true } });
    } else if ((l.name && l.name !== cur.name) || (l.default && !cur.default)) {
      await client.request("PUT", `/admin/locales/${encodeURIComponent(l.code)}`, { body: { name: l.name ?? cur.name, ...(l.default ? { isDefault: true } : {}) } });
    }
  }
  for (const [name, def] of Object.entries(model.components ?? {})) {
    const body = { name, description: def.description ?? "", attributes: attributesBody(def.attributes) };
    if (!current.components?.[name]) await client.request("POST", "/admin/components", { body });
    else if (plan.find((s) => s.kind === "component" && s.name === name)?.action === "update") {
      await client.request("PUT", `/admin/components/${encodeURIComponent(name)}`, { body });
    }
  }

  // Collections: new ones are created without relations to collections
  // that do not exist yet; every changed or new collection then gets its
  // full definition, so relation cycles work.
  const collections = Object.entries(model.collections ?? {});
  const exists = new Set(Object.keys(current.collections ?? {}));
  const pending = collections.filter(([name]) => !exists.has(name));
  let progress = true;
  while (pending.length > 0 && progress) {
    progress = false;
    for (let i = 0; i < pending.length; i++) {
      const [name, def] = pending[i]!;
      const attrs = Object.fromEntries(
        Object.entries(def.attributes).filter(([, a]) => a.type !== "relation" || exists.has(a.target!) || a.target === name),
      );
      await client.request("POST", "/admin/collections", { body: collectionBody(name, { ...def, attributes: attrs }) });
      exists.add(name);
      pending.splice(i, 1);
      i--;
      progress = true;
    }
  }
  for (const [name, def] of collections) {
    const step = plan.find((s) => s.kind === "collection" && s.name === name);
    const createdPartially = step?.action === "create" && relationTargets(def).some((t) => t !== name);
    if (step?.action === "update" || createdPartially) {
      await client.request("PUT", `/admin/collections/${encodeURIComponent(name)}`, { body: collectionBody(name, def) });
    }
  }
  for (const [name, def] of Object.entries(model.singletons ?? {})) {
    const step = plan.find((s) => s.kind === "singleton" && s.name === name);
    if (step?.action === "unchanged") continue;
    const body = { name, description: def.description ?? "", attributes: attributesBody(def.attributes) };
    if (step?.action === "create") await client.request("POST", "/admin/singleton", { body });
    else await client.request("PUT", `/admin/singleton/${encodeURIComponent(name)}`, { body });
  }
  if (plan.some((s) => s.kind === "public")) {
    await client.request("PUT", "/admin/roles/public", {
      body: { description: "Anonymous visitors (requests without a token)", permissions: { content: model.public!.content ?? {}, media: model.public!.media ?? [] } },
    });
  }
  return plan;
}

function collectionBody(name: string, def: CollectionDef) {
  return {
    name,
    kind: "collection",
    description: def.description ?? "",
    draft_and_publish: !!def.draftAndPublish,
    attributes: attributesBody(def.attributes),
  };
}

/** Checks a model's internal references before sending anything. */
export function validateModel(model: Model): void {
  const problems: string[] = [];
  const collections = new Set(Object.keys(model.collections ?? {}));
  const components = new Set(Object.keys(model.components ?? {}));
  const check = (where: string, attrs: Attributes | undefined) => {
    if (!attrs || typeof attrs !== "object") {
      problems.push(`${where}: "attributes" must be an object`);
      return;
    }
    for (const [name, a] of Object.entries(attrs)) {
      if (!a?.type) problems.push(`${where}.${name}: "type" is required`);
      if (a?.type === "relation" && !a.target) problems.push(`${where}.${name}: relations need a "target"`);
      if (a?.type === "component" && !a.component) problems.push(`${where}.${name}: components need a "component"`);
      if (a?.type === "component" && a.component && !components.has(a.component)) {
        problems.push(`${where}.${name}: component "${a.component}" is not in the model`);
      }
      if (a?.type === "relation" && a.target && !collections.has(a.target)) {
        problems.push(`${where}.${name}: target "${a.target}" is not a collection of the model`);
      }
      if ((a?.type === "enum" || a?.type === "enumeration") && !(a.options ?? []).length) problems.push(`${where}.${name}: enums need "options"`);
    }
  };
  for (const [n, d] of Object.entries(model.components ?? {})) check(`components.${n}`, d.attributes);
  for (const [n, d] of Object.entries(model.collections ?? {})) check(`collections.${n}`, d.attributes);
  for (const [n, d] of Object.entries(model.singletons ?? {})) check(`singletons.${n}`, d.attributes);
  const defaults = (model.locales ?? []).filter((l) => l.default);
  if (defaults.length > 1) problems.push("locales: only one locale can be the default");
  if (problems.length) throw new Error(`invalid model:\n  - ${problems.join("\n  - ")}`);
}
