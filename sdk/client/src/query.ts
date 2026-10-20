import type { FindOptions, ReadOptions } from "./types.js";

type Params = Record<string, unknown>;

/** Appends filters[field][op]=value pairs. */
function appendFilters(q: URLSearchParams, filters: Params): void {
  for (const [field, cond] of Object.entries(filters)) {
    if (cond === undefined) continue;
    if (cond !== null && typeof cond === "object" && !Array.isArray(cond)) {
      for (const [op, value] of Object.entries(cond as Params)) {
        if (value === undefined) continue;
        if (!op.startsWith("$")) throw new TypeError(`filters.${field}: operators start with $ (got "${op}")`);
        q.append(`filters[${field}][${op}]`, scalar(value, `filters.${field}.${op}`));
      }
    } else {
      q.append(`filters[${field}]`, scalar(cond, `filters.${field}`));
    }
  }
}

function scalar(v: unknown, where: string): string {
  if (v === null) throw new TypeError(`${where}: null is not supported in filters`);
  if (typeof v === "string" || typeof v === "number" || typeof v === "boolean") return String(v);
  if (v instanceof Date) return v.toISOString();
  throw new TypeError(`${where}: expected a string, number, boolean or Date`);
}

/** Builds the query string of a read. */
export function buildQuery(opts: (FindOptions<never> & ReadOptions) | undefined, defaults: { locale?: string; level?: number }): string {
  const q = new URLSearchParams();
  const o = (opts ?? {}) as FindOptions & ReadOptions;
  if (o.filters) appendFilters(q, o.filters as Params);
  if (o.sort !== undefined) q.set("sort", Array.isArray(o.sort) ? o.sort.join(",") : String(o.sort));
  if (o.page !== undefined) q.set("page", String(o.page));
  if (o.pageSize !== undefined) q.set("pageSize", String(o.pageSize));
  if (o.status !== undefined) q.set("status", o.status);
  const level = o.level ?? defaults.level;
  if (level !== undefined) q.set("level", String(level));
  const locale = o.locale ?? defaults.locale;
  if (locale !== undefined) q.set("locale", locale);
  if (o.fallback !== undefined) q.set("fallback", String(o.fallback));
  const s = q.toString();
  return s ? `?${s}` : "";
}
