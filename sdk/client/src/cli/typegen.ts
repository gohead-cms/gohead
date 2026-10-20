/**
 * TypeScript types for a content model: one interface per component,
 * collection and singleton (as read, and as written), and the module
 * augmentation that makes `client.collection("articles")` typed.
 */
import type { AttributeDef, Attributes, Model } from "./model.js";

const RESERVED = new Set(["Entry", "Media", "JsonValue", "Collections", "Singletons"]);

/** articles → Article, blog_posts → BlogPost, categories → Category. */
export function typeName(name: string, singular = true): string {
  const words = name.split(/[^A-Za-z0-9]+/).filter(Boolean);
  if (words.length === 0) return "Type";
  if (singular) words[words.length - 1] = singularize(words[words.length - 1]!);
  let out = words.map((w) => w[0]!.toUpperCase() + w.slice(1)).join("");
  if (/^[0-9]/.test(out)) out = `T${out}`;
  return out;
}

export function singularize(w: string): string {
  if (/ies$/i.test(w) && w.length > 4) return w.slice(0, -3) + "y";
  if (/(ss|us|is)$/i.test(w)) return w;
  if (/(ches|shes|xes|zes|sses)$/i.test(w)) return w.slice(0, -2);
  if (/s$/i.test(w) && w.length > 1) return w.slice(0, -1);
  return w;
}

function key(name: string): string {
  return /^[A-Za-z_$][A-Za-z0-9_$]*$/.test(name) ? name : JSON.stringify(name);
}

function scalarType(a: AttributeDef): string {
  switch (a.type) {
    case "int":
    case "integer":
    case "float":
    case "decimal":
      return "number";
    case "bool":
    case "boolean":
      return "boolean";
    case "enum":
    case "enumeration":
      return (a.options ?? []).length ? a.options!.map((o) => JSON.stringify(o)).join(" | ") : "string";
    case "json":
      return "JsonValue";
    default:
      // string, text, richtext, email, uid, password, date, datetime, time
      return "string";
  }
}

interface Names {
  collections: Map<string, string>;
  singletons: Map<string, string>;
  components: Map<string, string>;
}

function assignNames(model: Model): Names {
  const used = new Set<string>(RESERVED);
  const take = (base: string, suffix: string) => {
    let n = base;
    if (used.has(n) || used.has(`${n}Input`)) n = `${base}${suffix}`;
    let i = 2;
    while (used.has(n) || used.has(`${n}Input`)) n = `${base}${suffix}${i++}`;
    used.add(n);
    used.add(`${n}Input`);
    return n;
  };
  const names: Names = { collections: new Map(), singletons: new Map(), components: new Map() };
  for (const c of Object.keys(model.collections ?? {})) names.collections.set(c, take(typeName(c), "Entry"));
  for (const s of Object.keys(model.singletons ?? {})) names.singletons.set(s, take(typeName(s, false), "Single"));
  for (const c of Object.keys(model.components ?? {})) names.components.set(c, take(typeName(c, false), "Component"));
  return names;
}

function fieldType(a: AttributeDef, names: Names, input: boolean): string {
  switch (a.type) {
    case "relation": {
      const target = a.target ? names.collections.get(a.target) : undefined;
      const many = a.relation === "manyToMany";
      if (input) {
        const one = target ? `number | ${target}Input` : "number | Record<string, unknown>";
        return many ? `Array<${one}>` : one;
      }
      const one = target ? `Entry<${target}>` : "Entry";
      return many ? `${one}[]` : one;
    }
    case "component": {
      const c = a.component ? names.components.get(a.component) : undefined;
      const t = c ? (input ? `${c}Input` : c) : "Record<string, unknown>";
      return a.repeatable ? `${t}[]` : t;
    }
    case "media":
      if (input) return a.multiple ? "number[]" : "number";
      return a.multiple ? "Media[]" : "Media";
    default:
      return scalarType(a);
  }
}

function iface(name: string, attrs: Attributes, names: Names, input: boolean, doc?: string): string {
  const lines: string[] = [];
  if (doc) lines.push(`/** ${doc.replace(/\*\//g, "*\\/")} */`);
  lines.push(`export interface ${name} {`);
  for (const [field, a] of Object.entries(attrs)) {
    if (!input && a.type === "password") continue; // never returned
    let t = fieldType(a, names, input);
    const list = t.endsWith("[]") || t.startsWith("Array<");
    const optional = !a.required;
    if (optional && !list) t += " | null";
    const notes: string[] = [];
    if (a.localized) notes.push("localized");
    if (a.unique) notes.push("unique");
    if (a.type === "relation" && !input) notes.push(`relation (${a.relation ?? "oneToOne"}): populated when read at a deep enough level, an ID otherwise`);
    if (["date", "datetime", "time"].includes(a.type)) notes.push(a.type);
    if (a.type === "richtext") notes.push("rich text");
    if (notes.length) lines.push(`  /** ${notes.join(", ")} */`);
    lines.push(`  ${key(field)}${optional ? "?" : ""}: ${t};`);
  }
  lines.push("}");
  return lines.join("\n");
}

export interface GenerateOptions {
  /** Where the model came from, for the header. */
  source?: string;
  /** The package to augment (default @gohead-cms/client). */
  module?: string;
}

/** Generates the types of a model as a TypeScript module. */
export function generateTypes(model: Model, opts: GenerateOptions = {}): string {
  const names = assignNames(model);
  const mod = opts.module ?? "@gohead-cms/client";
  const out: string[] = [
    `// Generated by ${mod}${opts.source ? ` from ${opts.source}` : ""}.`,
    "// Do not edit: run `gohead-client types` (npx @gohead-cms/client types) again after changing the content model.",
    "",
    `import type { Entry, JsonValue, Media } from "${mod}";`,
    "",
  ];
  const sections: [string, Map<string, string>, Record<string, { attributes: Attributes; description?: string }> | undefined][] = [
    ["Components", names.components, model.components],
    ["Collections", names.collections, model.collections],
    ["Singletons", names.singletons, model.singletons],
  ];
  for (const [title, map, defs] of sections) {
    if (!defs || Object.keys(defs).length === 0) continue;
    out.push(`// ---- ${title} ----`, "");
    for (const [n, def] of Object.entries(defs)) {
      const t = map.get(n)!;
      const label = title === "Components" ? `Component "${n}"` : title === "Collections" ? `An entry of the "${n}" collection` : `The "${n}" singleton`;
      out.push(iface(t, def.attributes ?? {}, names, false, def.description ? `${label}: ${def.description}` : label), "");
      out.push(iface(`${t}Input`, def.attributes ?? {}, names, true, `Values written to ${label.toLowerCase().replace(/^an entry/, "entries")}`), "");
    }
  }
  const entries = (map: Map<string, string>) =>
    [...map].map(([n, t]) => `    ${key(n)}: { attributes: ${t}; input: ${t}Input };`).join("\n");
  out.push(
    `declare module "${mod}" {`,
    "  interface Collections {",
    entries(names.collections),
    "  }",
    "  interface Singletons {",
    entries(names.singletons),
    "  }",
    "}",
    "",
  );
  // Keep the JsonValue/Media/Entry import used even for tiny models.
  out.push("export type { Entry, JsonValue, Media };", "");
  return out.join("\n").replace(/\n{3,}/g, "\n\n");
}
