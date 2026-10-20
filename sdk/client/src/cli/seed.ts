/**
 * Sample content: uploads media and creates entries and singleton content
 * from a JSON file, with references between them.
 *
 * ```json
 * {
 *   "media": { "ada": { "file": "media/ada.jpg", "alt": "Ada Lovelace" } },
 *   "collections": {
 *     "authors": [{ "$id": "ada", "name": "Ada Lovelace", "avatar": "@media:ada" }],
 *     "articles": [{ "$status": "published", "title": "Hello", "author": "@ref:authors/ada",
 *                    "$translations": { "fr": { "title": "Bonjour" } } }]
 *   },
 *   "singletons": { "homepage": { "headline": "Welcome", "featured": ["@ref:articles/hello"] } }
 * }
 * ```
 *
 * `@ref:<collection>/<$id>` becomes the ID of an entry created earlier in
 * the file, `@media:<key>` the ID of an uploaded file. `$status` publishes
 * entries of draft-and-publish collections; `$translations` writes
 * localized values per locale.
 */
import { readFile } from "node:fs/promises";
import { basename, dirname, extname, resolve } from "node:path";

import type { GoHeadClient } from "../client.js";

export interface SeedFile {
  media?: Record<string, { file: string; alt?: string; caption?: string }>;
  collections?: Record<string, Array<Record<string, unknown>>>;
  singletons?: Record<string, Record<string, unknown>>;
}

export interface SeedOptions {
  /** Directory media paths are relative to (default: the seed file's). */
  baseDir: string;
  /** Seed collections that already have entries, and singletons that have content. */
  force?: boolean;
  log?: (line: string) => void;
}

export interface SeedResult {
  media: number;
  entries: number;
  singletons: number;
  skipped: string[];
}

const MIME: Record<string, string> = {
  ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".gif": "image/gif", ".webp": "image/webp",
  ".svg": "image/svg+xml", ".avif": "image/avif", ".pdf": "application/pdf", ".mp4": "video/mp4", ".txt": "text/plain",
};

export async function readSeed(path: string): Promise<SeedFile> {
  return JSON.parse(await readFile(path, "utf8")) as SeedFile;
}

export async function seed(client: GoHeadClient, file: SeedFile, opts: SeedOptions): Promise<SeedResult> {
  const log = opts.log ?? (() => {});
  const result: SeedResult = { media: 0, entries: 0, singletons: 0, skipped: [] };
  const mediaIds = new Map<string, number>();
  const refs = new Map<string, number>();

  // Which collections and singletons to fill.
  const todo: [string, Array<Record<string, unknown>>][] = [];
  for (const [name, entries] of Object.entries(file.collections ?? {})) {
    if (!opts.force && (await client.collection(name).count({ status: "all" })) > 0) {
      result.skipped.push(name);
      log(`skip ${name}: it already has entries (use --force to add the samples anyway)`);
      continue;
    }
    todo.push([name, entries]);
  }
  const singletons: [string, Record<string, unknown>][] = [];
  for (const [name, values] of Object.entries(file.singletons ?? {})) {
    if (!opts.force && (await client.singleton(name).find({ level: 0 })) !== null) {
      result.skipped.push(`singleton:${name}`);
      log(`skip singleton ${name}: it already has content`);
      continue;
    }
    singletons.push([name, values]);
  }

  // References: to entries this run creates, or to entries of skipped
  // collections that already exist (found by their slug). Everything is
  // checked before anything is uploaded or created.
  const produced = new Set<string>();
  for (const [name, entries] of todo) for (const e of entries) if (typeof e.$id === "string") produced.add(`${name}/${e.$id}`);
  const wanted = new Set<string>();
  const scanRefs = (v: unknown): void => {
    if (typeof v === "string" && v.startsWith("@ref:")) wanted.add(v.slice(5));
    else if (Array.isArray(v)) v.forEach(scanRefs);
    else if (v && typeof v === "object") Object.values(v).forEach(scanRefs);
  };
  todo.forEach(([, e]) => scanRefs(e));
  singletons.forEach(([, v]) => scanRefs(v));
  const missing: string[] = [];
  for (const ref of wanted) {
    if (produced.has(ref)) continue;
    const slash = ref.indexOf("/");
    const [collection, id] = [ref.slice(0, slash), ref.slice(slash + 1)];
    const source = (file.collections?.[collection] ?? []).find((e) => e.$id === id);
    const slug = typeof source?.slug === "string" ? source.slug : undefined;
    const existing = slug && result.skipped.includes(collection)
      ? await client.collection(collection).findFirst({ filters: { slug }, status: "all", level: 0 })
      : null;
    if (existing) refs.set(ref, existing.id);
    else missing.push(`@ref:${ref}`);
  }
  // Entries can only reference entries created before them.
  const soFar = new Set<string>();
  for (const [name, entries] of todo) {
    for (const [i, e] of entries.entries()) {
      const refsHere = new Set<string>();
      const collect = (v: unknown): void => {
        if (typeof v === "string" && v.startsWith("@ref:")) refsHere.add(v.slice(5));
        else if (Array.isArray(v)) v.forEach(collect);
        else if (v && typeof v === "object") Object.values(v).forEach(collect);
      };
      collect(e);
      for (const r of refsHere) {
        if (produced.has(r) && !soFar.has(r)) missing.push(`@ref:${r} (used by ${name}[${i}] before it is created)`);
      }
      if (typeof e.$id === "string") soFar.add(`${name}/${e.$id}`);
    }
  }
  if (missing.length) {
    throw new Error(
      `cannot resolve ${missing.join(", ")}: reference entries of the file (with "$id"), listed before; ` +
        "an entry of a collection that already has entries is found by its slug",
    );
  }

  // Media used by what will be seeded.
  const usedMedia = new Set<string>();
  const scan = (v: unknown): void => {
    if (typeof v === "string" && v.startsWith("@media:")) usedMedia.add(v.slice(7));
    else if (Array.isArray(v)) v.forEach(scan);
    else if (v && typeof v === "object") Object.values(v).forEach(scan);
  };
  todo.forEach(([, e]) => scan(e));
  singletons.forEach(([, v]) => scan(v));
  for (const key of usedMedia) {
    const m = file.media?.[key];
    if (!m) throw new Error(`@media:${key} is used but not declared under "media"`);
    const path = resolve(opts.baseDir, m.file);
    const bytes = await readFile(path);
    const [uploaded] = await client.media.upload(
      { name: basename(path), data: new Uint8Array(bytes), type: MIME[extname(path).toLowerCase()] ?? "application/octet-stream" },
      { alt: m.alt, caption: m.caption },
    );
    if (!uploaded) throw new Error(`uploading ${m.file} returned nothing`);
    mediaIds.set(key, uploaded.id);
    result.media++;
    log(`upload ${m.file} → media ${uploaded.id}`);
  }

  const resolveValue = (v: unknown, where: string): unknown => {
    if (typeof v === "string") {
      if (v.startsWith("@media:")) return mediaIds.get(v.slice(7));
      if (v.startsWith("@ref:")) {
        const id = refs.get(v.slice(5));
        if (id === undefined) throw new Error(`${where}: ${v} is not an entry created before (list the referenced collection first)`);
        return id;
      }
      return v;
    }
    if (Array.isArray(v)) return v.map((x, i) => resolveValue(x, `${where}[${i}]`));
    if (v && typeof v === "object") {
      return Object.fromEntries(Object.entries(v).map(([k, x]) => [k, resolveValue(x, `${where}.${k}`)]));
    }
    return v;
  };

  for (const [name, entries] of todo) {
    const col = client.collection(name);
    for (const [i, raw] of entries.entries()) {
      const where = `${name}[${i}]`;
      const { $id, $status, $translations, ...values } = raw;
      const data = resolveValue(values, where) as Record<string, unknown>;
      const created = await col.create(data, $status === "published" || $status === "draft" ? { status: $status, level: 0 } : { level: 0 });
      if (typeof $id === "string") refs.set(`${name}/${$id}`, created.id);
      for (const [locale, t] of Object.entries(($translations ?? {}) as Record<string, Record<string, unknown>>)) {
        await col.update(created.id, resolveValue(t, `${where}.$translations.${locale}`) as Record<string, unknown>, { locale, level: 0 });
      }
      result.entries++;
    }
    log(`create ${entries.length} ${name}`);
  }

  for (const [name, raw] of singletons) {
    const { $translations, ...values } = raw;
    const s = client.singleton(name);
    await s.update(resolveValue(values, `singletons.${name}`) as Record<string, unknown>, { level: 0 });
    for (const [locale, t] of Object.entries(($translations ?? {}) as Record<string, Record<string, unknown>>)) {
      await s.update(resolveValue(t, `singletons.${name}.$translations.${locale}`) as Record<string, unknown>, { locale, level: 0 });
    }
    result.singletons++;
    log(`fill singleton ${name}`);
  }
  return result;
}

export function seedBaseDir(seedPath: string): string {
  return dirname(resolve(seedPath));
}
