// Conversion between the admin API's schema JSON and the type editor's form
// state, plus client-side validation. Pure functions only.

export type ModelKind = "collections" | "components" | "singletons";

export const MODEL_KINDS: ModelKind[] = ["collections", "components", "singletons"];

export function isModelKind(value: string | undefined): value is ModelKind {
  return MODEL_KINDS.includes(value as ModelKind);
}

/** Labels for each kind of content type. */
export const KIND_LABELS: Record<ModelKind, { singular: string; plural: string }> = {
  collections: { singular: "collection", plural: "Collections" },
  components: { singular: "component", plural: "Components" },
  singletons: { singular: "singleton", plural: "Singletons" },
};

export const KIND_DESCRIPTIONS: Record<ModelKind, string> = {
  collections: "Lists of entries, such as articles or products.",
  components: "Reusable groups of attributes, used inside collections, singletons and other components.",
  singletons: "Content that exists exactly once, such as a home page or site settings.",
};

/** An attribute definition as the admin API sends and accepts it. */
export interface ApiAttribute {
  name?: string;
  type: string;
  required?: boolean;
  unique?: boolean;
  options?: string[] | null;
  min?: number | null;
  max?: number | null;
  pattern?: string;
  target?: string;
  relation?: string;
  component?: string;
  repeatable?: boolean;
  multiple?: boolean;
  allowed_types?: string[] | null;
}

/** GET /admin/collections(/{name}) item. */
export interface ApiCollection {
  id: number;
  uid: string;
  schema: {
    collectionName: string;
    /** Not returned by the current API; read if it ever is. */
    description?: string;
    options?: { draftAndPublish?: boolean };
    attributes: Record<string, ApiAttribute>;
  };
}

/** GET /admin/components and /admin/singletons items. */
export interface ApiNamedType {
  id: number;
  name: string;
  description?: string;
  attributes: ApiAttribute[] | null;
}

export interface AttributeForm {
  /** Stable React key; not sent to the API. */
  key: string;
  name: string;
  type: string;
  required: boolean;
  unique: boolean;
  pattern: string;
  min: string;
  max: string;
  options: string[];
  target: string;
  relation: string;
  component: string;
  repeatable: boolean;
  multiple: boolean;
  /** Comma-separated MIME patterns. */
  allowedTypes: string;
}

export interface TypeForm {
  name: string;
  description: string;
  draftAndPublish: boolean;
  attributes: AttributeForm[];
}

export const ATTRIBUTE_TYPES: { value: string; label: string }[] = [
  { value: "string", label: "String (short text)" },
  { value: "text", label: "Text (long text)" },
  { value: "richtext", label: "Rich text" },
  { value: "email", label: "Email" },
  { value: "password", label: "Password" },
  { value: "uid", label: "UID (slug, SKU)" },
  { value: "int", label: "Integer" },
  { value: "float", label: "Float" },
  { value: "bool", label: "Boolean" },
  { value: "date", label: "Date" },
  { value: "datetime", label: "Date and time" },
  { value: "enum", label: "Enumeration" },
  { value: "json", label: "JSON" },
  { value: "media", label: "Media" },
  { value: "relation", label: "Relation" },
  { value: "component", label: "Component" },
];

/** Aliases the API accepts, shown as their canonical type. */
const TYPE_ALIASES: Record<string, string> = {
  integer: "int",
  decimal: "float",
  boolean: "bool",
  enumeration: "enum",
};

export const RELATION_TYPES = [
  { value: "oneToOne", label: "One to one" },
  { value: "oneToMany", label: "One to many" },
  { value: "manyToMany", label: "Many to many" },
];

/** Attribute types available for a kind (components cannot hold relations). */
export function typesForKind(kind: ModelKind) {
  return kind === "components" ? ATTRIBUTE_TYPES.filter((t) => t.value !== "relation") : ATTRIBUTE_TYPES;
}

export function typeLabel(type: string): string {
  return ATTRIBUTE_TYPES.find((t) => t.value === type)?.label ?? type;
}

const PATTERN_TYPES = ["string", "text", "richtext", "email"];
const NO_UNIQUE_TYPES = ["relation", "component", "media", "json", "bool"];

export const supportsPattern = (a: AttributeForm) => PATTERN_TYPES.includes(a.type);
export const supportsUnique = (a: AttributeForm) => !NO_UNIQUE_TYPES.includes(a.type);
/** min/max: integer value bounds, or list length for multiple media and repeatable components. */
export const supportsMinMax = (a: AttributeForm) =>
  a.type === "int" || (a.type === "media" && a.multiple) || (a.type === "component" && a.repeatable);

export function minMaxLabels(a: AttributeForm): { min: string; max: string } {
  if (a.type === "int") return { min: "Minimum value", max: "Maximum value" };
  const unit = a.type === "media" ? "files" : "items";
  return { min: `Minimum ${unit}`, max: `Maximum ${unit}` };
}

let keySeq = 0;
function nextKey() {
  keySeq += 1;
  return `attr-${keySeq}`;
}

export function emptyAttribute(): AttributeForm {
  return {
    key: nextKey(),
    name: "",
    type: "string",
    required: false,
    unique: false,
    pattern: "",
    min: "",
    max: "",
    options: [],
    target: "",
    relation: "oneToOne",
    component: "",
    repeatable: false,
    multiple: false,
    allowedTypes: "",
  };
}

export function emptyForm(): TypeForm {
  return { name: "", description: "", draftAndPublish: false, attributes: [emptyAttribute()] };
}

/** "api::authors.authors" (the collection schema format) → "authors". */
export function relationTarget(target: string | undefined): string {
  if (!target) return "";
  const m = /^api::(.+)\.\1$/.exec(target);
  return m ? m[1] : target;
}

const numText = (n: number | null | undefined) => (typeof n === "number" ? String(n) : "");

export function attributeFromApi(name: string, a: ApiAttribute): AttributeForm {
  const type = TYPE_ALIASES[a.type] ?? a.type;
  return {
    ...emptyAttribute(),
    name,
    type,
    required: !!a.required,
    unique: !!a.unique,
    pattern: a.pattern ?? "",
    min: numText(a.min),
    max: numText(a.max),
    options: a.options ?? [],
    target: relationTarget(a.target),
    relation: a.relation || "oneToOne",
    component: a.component ?? "",
    repeatable: !!a.repeatable,
    multiple: !!a.multiple,
    allowedTypes: (a.allowed_types ?? []).join(", "),
  };
}

export function formFromCollection(c: ApiCollection): TypeForm {
  return {
    name: c.schema.collectionName,
    description: c.schema.description ?? "",
    draftAndPublish: !!c.schema.options?.draftAndPublish,
    attributes: Object.entries(c.schema.attributes ?? {}).map(([name, a]) => attributeFromApi(name, a)),
  };
}

export function formFromNamedType(t: ApiNamedType): TypeForm {
  return {
    name: t.name,
    description: t.description ?? "",
    draftAndPublish: false,
    attributes: (t.attributes ?? []).map((a) => attributeFromApi(a.name ?? "", a)),
  };
}

function attributeToApi(a: AttributeForm): ApiAttribute {
  const out: ApiAttribute = { type: a.type, required: a.required };
  if (a.unique && supportsUnique(a)) out.unique = true;
  if (supportsPattern(a) && a.pattern.trim()) out.pattern = a.pattern.trim();
  if (supportsMinMax(a)) {
    if (a.min.trim() !== "") out.min = Number(a.min);
    if (a.max.trim() !== "") out.max = Number(a.max);
  }
  switch (a.type) {
    case "enum":
      out.options = enumOptions(a);
      break;
    case "relation":
      out.relation = a.relation;
      out.target = a.target;
      break;
    case "component":
      out.component = a.component;
      out.repeatable = a.repeatable;
      break;
    case "media": {
      out.multiple = a.multiple;
      const types = splitList(a.allowedTypes);
      if (types.length) out.allowed_types = types;
      break;
    }
  }
  return out;
}

/** Enum options without blank lines. */
export function enumOptions(a: AttributeForm): string[] {
  return a.options.map((o) => o.trim()).filter(Boolean);
}

/** What a value of each simple type looks like. */
export const TYPE_HINTS: Record<string, string> = {
  string: "A short string, such as a title.",
  text: "Long plain text.",
  richtext: "Formatted text (stored as a string).",
  email: "An email address.",
  password: "Stored as given; prefer users for accounts.",
  uid: "An identifier such as a slug or SKU; usually unique.",
  int: "A whole number.",
  float: "A decimal number.",
  bool: "true or false.",
  date: 'A date, "YYYY-MM-DD".',
  datetime: 'An RFC 3339 date and time, such as "2026-10-18T09:00:00Z".',
  json: "Any JSON object.",
  enum: "One value from a fixed list.",
  media: "An uploaded file from the media library.",
  relation: "Links to items of another collection.",
  component: "A group of fields defined by a component.",
};

export function splitList(text: string): string[] {
  return text
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

/** The request body for POST/PUT of a type. */
export function formToPayload(kind: ModelKind, form: TypeForm): Record<string, unknown> {
  const attributes: Record<string, ApiAttribute> = {};
  for (const a of form.attributes) attributes[a.name.trim()] = attributeToApi(a);
  const body: Record<string, unknown> = {
    name: form.name.trim(),
    description: form.description.trim(),
    attributes,
  };
  if (kind === "collections") {
    body.kind = "collection";
    body.draft_and_publish = form.draftAndPublish;
  }
  return body;
}

/** Where to save: method and path for a new or existing type. */
export function saveRequest(kind: ModelKind, existingName: string | null): { method: string; path: string } {
  const enc = existingName ? encodeURIComponent(existingName) : "";
  switch (kind) {
    case "collections":
      return existingName ? { method: "PUT", path: `/admin/collections/${enc}` } : { method: "POST", path: "/admin/collections" };
    case "components":
      return existingName ? { method: "PUT", path: `/admin/components/${enc}` } : { method: "POST", path: "/admin/components" };
    case "singletons":
      return existingName ? { method: "PUT", path: `/admin/singleton/${enc}` } : { method: "POST", path: "/admin/singleton" };
  }
}

export function deletePath(kind: ModelKind, name: string): string {
  const enc = encodeURIComponent(name);
  return kind === "singletons" ? `/admin/singleton/${enc}` : `/admin/${kind}/${enc}`;
}

export function listPath(kind: ModelKind): string {
  return `/admin/${kind}`;
}

// -------------------- validation --------------------

export type AttributeField = "name" | "type" | "pattern" | "min" | "max" | "options" | "target" | "component" | "allowedTypes";

export interface FormErrors {
  name?: string;
  general?: string;
  attributes: Record<string, Partial<Record<AttributeField, string>>>;
}

export function hasErrors(e: FormErrors): boolean {
  return !!e.name || !!e.general || Object.values(e.attributes).some((a) => Object.keys(a).length > 0);
}

const TYPE_NAME = /^[A-Za-z][A-Za-z0-9_-]*$/;
const ATTRIBUTE_NAME = /^[A-Za-z_][A-Za-z0-9_]*$/;
const INTEGER = /^-?\d+$/;

function reservedNames(kind: ModelKind, draftAndPublish: boolean): string[] {
  if (kind === "collections") return ["id", "createdAt", "updatedAt", "publishedAt", ...(draftAndPublish ? ["status"] : [])];
  if (kind === "singletons") return ["id", "createdAt", "updatedAt"];
  return [];
}

export interface ValidationContext {
  kind: ModelKind;
  /** Names already used by types of this kind (excluding the one being edited). */
  takenNames: string[];
}

export function validateForm(form: TypeForm, ctx: ValidationContext): FormErrors {
  const errors: FormErrors = { attributes: {} };
  const name = form.name.trim();
  if (!name) errors.name = "Name is required.";
  else if (!TYPE_NAME.test(name)) errors.name = "Use letters, digits, '-' or '_', starting with a letter.";
  else if (ctx.takenNames.includes(name)) errors.name = `A ${KIND_LABELS[ctx.kind].singular} named "${name}" already exists.`;

  if (form.attributes.length === 0) errors.general = "Add at least one attribute.";

  const reserved = reservedNames(ctx.kind, form.draftAndPublish);
  const seen = new Set<string>();
  for (const a of form.attributes) {
    const e: Partial<Record<AttributeField, string>> = {};
    const attrName = a.name.trim();
    if (!attrName) e.name = "Name is required.";
    else if (!ATTRIBUTE_NAME.test(attrName)) e.name = "Use letters, digits and '_', not starting with a digit.";
    else if (reserved.includes(attrName)) e.name = `"${attrName}" is a reserved name.`;
    else if (seen.has(attrName)) e.name = "Duplicate attribute name.";
    seen.add(attrName);

    if (supportsPattern(a) && a.pattern.trim()) {
      try {
        new RegExp(a.pattern.trim());
      } catch {
        e.pattern = "Invalid regular expression.";
      }
    }
    if (supportsMinMax(a)) {
      const min = a.min.trim();
      const max = a.max.trim();
      if (min && !INTEGER.test(min)) e.min = "Must be a whole number.";
      if (max && !INTEGER.test(max)) e.max = "Must be a whole number.";
      if (!e.min && !e.max && min && max && Number(min) > Number(max)) e.max = "Must be greater than or equal to the minimum.";
    }
    if (a.type === "enum") {
      const opts = enumOptions(a);
      if (opts.length === 0) e.options = "Add at least one option.";
      else if (new Set(opts).size !== opts.length) e.options = "Options must be unique.";
    }
    if (a.type === "relation" && !a.target) e.target = "Choose a target collection.";
    if (a.type === "component" && !a.component) e.component = "Choose a component.";
    if (Object.keys(e).length > 0) errors.attributes[a.key] = e;
  }
  return errors;
}

/** A copy of list with the item at index moved by delta (clamped). */
export function moveItem<T>(list: T[], index: number, delta: number): T[] {
  const to = index + delta;
  if (to < 0 || to >= list.length) return list;
  const next = [...list];
  const [item] = next.splice(index, 1);
  next.splice(to, 0, item);
  return next;
}
