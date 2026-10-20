import type { Attribute } from "../../../shared/types";
import { toMediaRef } from "./mediaRef";

const NUMBER_TYPES = new Set(["int", "integer", "float", "decimal", "number"]);
const BOOLEAN_TYPES = new Set(["bool", "boolean"]);
// locale and locales describe a response of a type with localized attributes.
const SYSTEM_FIELDS = new Set(["id", "createdAt", "updatedAt", "publishedAt", "status", "locale", "locales"]);

export const isNumberType = (type: string) => NUMBER_TYPES.has(type);
export const isBooleanType = (type: string) => BOOLEAN_TYPES.has(type);

/** The HTML input type for a simple attribute. */
export function inputType(type: string): string {
  if (isNumberType(type)) return "number";
  if (type === "date") return "date";
  if (type === "datetime") return "datetime-local";
  return "text";
}

/**
 * A field value that cannot be saved as it is (JSON that does not parse, a
 * component with such a field): saving fails with its message.
 */
export class InvalidValue {
  readonly message: string;
  constructor(message: string) {
    this.message = message;
  }
}

/** Files (objects with an id and a url) inside a component value, as their IDs. */
export function filesToIDs(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(filesToIDs);
  if (value && typeof value === "object" && !(value instanceof InvalidValue)) {
    const o = value as Record<string, unknown>;
    if ("id" in o && typeof o.url === "string") return o.id;
    return Object.fromEntries(Object.entries(o).map(([k, v]) => [k, filesToIDs(v)]));
  }
  return value;
}

/** A relation value as the API stores it: an ID or a list of IDs. */
export function relationIDs(value: unknown): unknown {
  const id = (v: unknown) => (v !== null && typeof v === "object" && "id" in v ? (v as { id: unknown }).id : v);
  if (value === null || value === undefined) return undefined;
  if (Array.isArray(value)) return value.map(id).filter((v) => v !== null && v !== undefined);
  return id(value);
}

/** An ISO time as a datetime-local input value (local time, minutes). */
function toLocalInput(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** The blank value a new entry's field starts with. */
export function blankValue(attr: Attribute): unknown {
  if (isBooleanType(attr.type)) return false;
  return "";
}

/** Attributes as a map by name (singletons list them as an array). */
export function attributeMap(attributes: Attribute[] | Record<string, Attribute> | undefined): Record<string, Attribute> {
  if (!attributes) return {};
  if (!Array.isArray(attributes)) return attributes;
  return Object.fromEntries(attributes.map((a) => [a.name, a]));
}

/**
 * The form's starting values: an existing entry's attributes (entries are
 * flat; system fields are dropped unless the schema defines them), or blank
 * values for a new one.
 */
export function initialValues(attributes: Record<string, Attribute>, entry?: Record<string, unknown> | null): Record<string, unknown> {
  if (entry) {
    const values: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(entry)) {
      if (SYSTEM_FIELDS.has(key) && !(key in attributes)) continue;
      const attr = attributes[key];
      if (attr?.type === "relation") {
        // Populated entries are kept for their labels; saving sends IDs.
        if (value !== null && value !== undefined) values[key] = value;
        continue;
      }
      if (attr?.type === "datetime" && typeof value === "string") {
        values[key] = toLocalInput(value);
        continue;
      }
      values[key] = value;
    }
    return values;
  }
  const values: Record<string, unknown> = {};
  for (const [key, attr] of Object.entries(attributes)) {
    if (attr.type === "relation") continue;
    if (attr.type === "media") {
      if (attr.multiple) values[key] = [];
      continue;
    }
    values[key] = blankValue(attr);
  }
  return values;
}

/**
 * Turns form values into the item data the API expects: inputs give
 * strings, so numbers are converted, and fields left empty are omitted
 * (the API validates required ones).
 */
export function toItemData(attributes: Record<string, Attribute>, values: Record<string, unknown>): Record<string, unknown> {
  const data: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(values)) {
    const attr = attributes[key];
    if (value instanceof InvalidValue) throw new Error(`${key}: ${value.message}`);
    // Empty means unset: the entry's data is replaced, so the field goes.
    if (value === "" || value === undefined || value === null) continue;
    if (attr?.type === "relation") {
      const ids = relationIDs(value);
      if (ids !== undefined) data[key] = ids;
      continue;
    }
    if (attr?.type === "component") {
      // An untouched component holds files as the API returned them.
      data[key] = filesToIDs(value);
      continue;
    }
    if (attr?.type === "datetime" && typeof value === "string") {
      const d = new Date(value);
      data[key] = Number.isNaN(d.getTime()) ? value : d.toISOString();
      continue;
    }
    if (attr && isNumberType(attr.type) && typeof value === "string") {
      const n = Number(value);
      data[key] = Number.isNaN(n) ? value : n;
      continue;
    }
    data[key] = value;
  }
  return data;
}

/**
 * Like toItemData, and media fields (which hold uploaded files) become the
 * IDs or URLs the API accepts.
 */
export function toEntryPayload(attributes: Record<string, Attribute>, values: Record<string, unknown>): Record<string, unknown> {
  const data = toItemData(attributes, values);
  for (const [key, attr] of Object.entries(attributes)) {
    if (attr.type !== "media") continue;
    if (data[key] === null || data[key] === undefined) delete data[key];
    else data[key] = toMediaRef(data[key]);
  }
  return data;
}

/** The required, editable fields left empty. */
export function missingRequired(attributes: Record<string, Attribute>, values: Record<string, unknown>): string[] {
  return Object.entries(attributes)
    .filter(([, attr]) => attr.required && attr.type !== "component" && !isBooleanType(attr.type))
    .filter(([key]) => {
      const v = values[key];
      return v === undefined || v === null || v === "" || (Array.isArray(v) && v.length === 0);
    })
    .map(([key]) => key);
}

/** The table columns of a collection: id, then up to four preferred attributes. */
export function tableColumns(schema: { attributes: Record<string, Attribute> }): string[] {
  const attributes = Object.keys(schema.attributes).filter((a) => schema.attributes[a].type !== "relation");
  const preferred = ["title", "name", "slug"];
  const ordered = [...attributes.filter((a) => preferred.includes(a.toLowerCase())), ...attributes.filter((a) => !preferred.includes(a.toLowerCase()))];
  return ["id", ...ordered.filter((h) => h !== "id")].slice(0, 5);
}

/** A short text for a table cell. */
export function displayValue(value: unknown): string {
  if (value === null || value === undefined || value === "") return "–";
  if (typeof value === "object") {
    const s = JSON.stringify(value);
    return s.length > 30 ? `${s.substring(0, 30)}…` : s;
  }
  const s = String(value);
  return s.length > 50 ? `${s.substring(0, 50)}…` : s;
}

// -------------------- localization --------------------

/** The attributes with a value per locale. */
export function localizedAttributes(attributes: Record<string, Attribute>): Record<string, Attribute> {
  return Object.fromEntries(Object.entries(attributes).filter(([, a]) => a.localized && a.type !== "relation"));
}

export function hasLocalizedAttributes(attributes: Record<string, Attribute>): boolean {
  return Object.values(attributes).some((a) => a.localized && a.type !== "relation");
}

/**
 * The form values of a translation: the localized attributes of an entry
 * read with fallback=false (a missing value is a field left empty).
 */
export function translationValues(localized: Record<string, Attribute>, entry: Record<string, unknown> | null): Record<string, unknown> {
  const all = initialValues(localized, entry ?? {});
  return Object.fromEntries(Object.entries(all).filter(([key]) => key in localized));
}

/**
 * The data of PUT ?locale=: the localized fields that have a value. A field
 * left empty is not sent, so it falls back to the default locale's value
 * (the request replaces the whole translation).
 */
export function toTranslationPayload(localized: Record<string, Attribute>, values: Record<string, unknown>): Record<string, unknown> {
  const data = toEntryPayload(localized, values);
  for (const key of Object.keys(data)) {
    if (!(key in localized)) delete data[key];
    else if (Array.isArray(data[key]) && (data[key] as unknown[]).length === 0) delete data[key];
  }
  return data;
}

/** The locale codes of an entry that have content: the default first (from its "locales"). */
export function entryLocales(entry: Record<string, unknown> | null | undefined): string[] {
  const l = entry?.locales;
  return Array.isArray(l) ? l.filter((c): c is string => typeof c === "string") : [];
}

/** A one-line preview of a default-locale value, for hints under translated fields. */
export function hintValue(value: unknown): string {
  if (value === null || value === undefined || value === "") return "empty";
  if (typeof value === "boolean") return value ? "on" : "off";
  if (typeof value === "object") {
    const name = (value as { name?: unknown; filename?: unknown }).name ?? (value as { filename?: unknown }).filename;
    if (typeof name === "string") return name;
    const s = JSON.stringify(value);
    return s.length > 80 ? `${s.substring(0, 80)}…` : s;
  }
  const s = String(value).replace(/\s+/g, " ");
  return s.length > 120 ? `${s.substring(0, 120)}…` : s;
}
