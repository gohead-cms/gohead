import { Attribute } from "../../../shared/types";

const NUMBER_TYPES = new Set(["int", "integer", "float", "decimal", "number"]);
const BOOLEAN_TYPES = new Set(["bool", "boolean"]);

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
