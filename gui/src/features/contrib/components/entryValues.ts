import type { Attribute } from "../../../shared/types";
import { toMediaRef } from "./mediaRef";

const NUMBER_TYPES = new Set(["int", "integer", "float", "decimal", "number"]);
const BOOLEAN_TYPES = new Set(["bool", "boolean"]);
const SYSTEM_FIELDS = new Set(["id", "createdAt", "updatedAt", "publishedAt", "status"]);

export const isNumberType = (type: string) => NUMBER_TYPES.has(type);
export const isBooleanType = (type: string) => BOOLEAN_TYPES.has(type);

/** The HTML input type for a simple attribute. */
export function inputType(type: string): string {
  if (isNumberType(type)) return "number";
  if (type === "date") return "date";
  if (type === "datetime") return "datetime-local";
  return "text";
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
    return Object.fromEntries(Object.entries(entry).filter(([key]) => !SYSTEM_FIELDS.has(key) || key in attributes));
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
    if (value === "" || value === undefined) continue;
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
    .filter(([, attr]) => attr.required && attr.type !== "relation" && attr.type !== "component" && !isBooleanType(attr.type))
    .filter(([key]) => {
      const v = values[key];
      return v === undefined || v === null || v === "" || (Array.isArray(v) && v.length === 0);
    })
    .map(([key]) => key);
}
