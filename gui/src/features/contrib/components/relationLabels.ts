import type { Schema } from "../../../shared/types";

/** An entry of the target collection, as listed or populated. */
export interface RelatedEntry {
  id: number | string;
  status?: string;
  [key: string]: unknown;
}

const LABEL_FIELDS = ["title", "name", "label", "displayName", "slug", "username", "email", "code"];
const TEXT_TYPES = new Set(["string", "text", "uid", "email", "enum", "enumeration"]);

/** The attribute that names the entries of a collection, if any. */
export function labelField(schema: Schema | null | undefined): string | null {
  if (!schema) return null;
  const attrs = schema.attributes ?? {};
  const named = LABEL_FIELDS.find((f) => attrs[f] && TEXT_TYPES.has(attrs[f].type));
  if (named) return named;
  const first = Object.entries(attrs).find(([, a]) => a.type === "string" || a.type === "uid");
  return first ? first[0] : null;
}

/** A short name for an entry: its label field, then #id. */
export function entryLabel(entry: RelatedEntry, field: string | null): string {
  const v = field ? entry[field] : undefined;
  if (typeof v === "string" && v.trim() !== "") return v;
  if (typeof v === "number") return String(v);
  return `#${entry.id}`;
}
