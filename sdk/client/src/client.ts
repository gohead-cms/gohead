import { GoHeadError } from "./errors.js";
import { buildQuery } from "./query.js";
import type {
  CollectionName,
  Collections,
  Entry,
  FindOptions,
  ListResponse,
  Locale,
  Me,
  Media,
  Pagination,
  ReadOptions,
  SingletonName,
  Singletons,
  TypeShape,
  WriteOptions,
} from "./types.js";

type Token = string | null | undefined;

export interface ClientOptions {
  /** GoHead's base URL, e.g. `https://cms.example.com` (no /api). */
  url: string;
  /**
   * An API token (`ght_…`) or a user session token, or a function returning
   * one (called per request). Without a token, requests are anonymous and
   * read what the public role grants.
   */
  token?: Token | (() => Token | Promise<Token>);
  /** Default locale of reads (`?locale=`). */
  locale?: string;
  /** Default depth of populated relations (0–5). */
  level?: ReadOptions["level"];
  /** fetch implementation (default: the global fetch). */
  fetch?: typeof fetch;
  /** Headers sent with every request. */
  headers?: Record<string, string>;
  /**
   * Default fetch options merged into every request, e.g.
   * `{ cache: "no-store" }` or, in Next.js, `{ next: { revalidate: 60 } }`.
   */
  init?: RequestInit;
  /** Abort requests after this many milliseconds (default: 30 000; 0: never). */
  timeoutMs?: number;
}

/** Options of one request. */
export interface RequestOptions {
  query?: string;
  body?: unknown;
  form?: FormData;
  /** fetch options for this request (merged over the client's `init`). */
  init?: RequestInit;
  /** Send no token (for requests that must stay anonymous). */
  anonymous?: boolean;
}

/** A decoded response envelope. */
export interface Envelope<T> {
  data: T;
  meta?: Record<string, unknown>;
}

type Untyped = { attributes: Record<string, unknown>; input: Record<string, unknown> };
type CollectionShape<K extends string> = K extends CollectionName ? Collections[K] & TypeShape : Untyped;
type SingletonShape<K extends string> = K extends SingletonName ? Singletons[K] & TypeShape : Untyped;
type Attr<S> = S extends { attributes: infer A extends object } ? A : Record<string, unknown>;
type In<S> = S extends { input: infer I extends object } ? I : Record<string, unknown>;

function trimSlash(url: string): string {
  let end = url.length;
  while (end > 0 && url[end - 1] === "/") end--;
  return url.slice(0, end);
}

function mergeInit(a: RequestInit | undefined, b: RequestInit | undefined): RequestInit {
  if (!a) return { ...b };
  if (!b) return { ...a };
  const out: RequestInit & { next?: unknown } = { ...a, ...b };
  const an = (a as { next?: Record<string, unknown> }).next;
  const bn = (b as { next?: Record<string, unknown> }).next;
  if (an || bn) {
    const next: Record<string, unknown> = { ...an, ...bn };
    const tags = [...((an?.tags as string[] | undefined) ?? []), ...((bn?.tags as string[] | undefined) ?? [])];
    if (tags.length) next.tags = [...new Set(tags)];
    out.next = next;
  }
  return out;
}

function enc(segment: string | number): string {
  return encodeURIComponent(String(segment));
}

/**
 * A GoHead API client. Create one with {@link createClient}.
 *
 * ```ts
 * const gohead = createClient({ url: "https://cms.example.com", token: process.env.GOHEAD_TOKEN });
 * const { data } = await gohead.collection("articles").find({ sort: "publishedAt:desc", pageSize: 10 });
 * ```
 */
export class GoHeadClient {
  readonly url: string;
  readonly options: Readonly<ClientOptions>;
  /** The media library. */
  readonly media: MediaClient;

  constructor(options: ClientOptions) {
    if (!options?.url) throw new TypeError("createClient: url is required (e.g. https://cms.example.com)");
    this.options = { ...options };
    this.url = trimSlash(options.url).replace(/\/api$/, "");
    this.media = new MediaClient(this);
  }

  /** A client with some options changed (e.g. another token or locale). */
  with(overrides: Partial<ClientOptions>): GoHeadClient {
    return new GoHeadClient({ ...this.options, ...overrides });
  }

  /** Items of a collection. */
  collection<K extends CollectionName>(name: K): CollectionClient<Attr<CollectionShape<K>>, In<CollectionShape<K>>>;
  collection<T extends object = Record<string, unknown>, I extends object = Partial<T>>(name: string): CollectionClient<T, I>;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  collection(name: string): CollectionClient<any, any> {
    return new CollectionClient(this, name);
  }

  /** A singleton (single type), e.g. a homepage or site settings. */
  singleton<K extends SingletonName>(name: K): SingletonClient<Attr<SingletonShape<K>>, In<SingletonShape<K>>>;
  singleton<T extends object = Record<string, unknown>, I extends object = Partial<T>>(name: string): SingletonClient<T, I>;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  singleton(name: string): SingletonClient<any, any> {
    return new SingletonClient(this, name);
  }

  /** Content locales, the default first. */
  async locales(init?: RequestInit): Promise<Locale[]> {
    return (await this.request<Locale[]>("GET", "/api/locales", { init })).data ?? [];
  }

  /** The default locale's code. */
  async defaultLocale(init?: RequestInit): Promise<string> {
    const list = await this.locales(init);
    return list.find((l) => l.isDefault)?.code ?? list[0]?.code ?? "en";
  }

  /**
   * Runs a GraphQL query or mutation and returns its `data`. GraphQL errors
   * throw a {@link GoHeadError} (status 400).
   */
  async graphql<T = Record<string, unknown>>(
    query: string,
    variables?: Record<string, unknown>,
    options: { operationName?: string; init?: RequestInit } = {},
  ): Promise<T> {
    const body = { query, variables, operationName: options.operationName };
    try {
      return (await this.request<T>("POST", "/api/graphql", { body, init: options.init })).data;
    } catch (err) {
      // GoHead puts the GraphQL error itself in details.
      if (err instanceof GoHeadError && typeof err.details === "string" && err.details) {
        throw new GoHeadError({ status: err.status, code: err.code, message: err.details, details: err.details, request: err.request, cause: err });
      }
      throw err;
    }
  }

  /** Signs in with a username and password; returns a session token to pass as `token`. */
  async login(username: string, password: string): Promise<string> {
    const res = await this.request<{ token: string }>("POST", "/auth/login", { body: { username, password }, anonymous: true });
    return res.data.token;
  }

  /** The account or token the client acts as. */
  async me(init?: RequestInit): Promise<Me> {
    return (await this.request<Me>("GET", "/auth/me", { init })).data;
  }

  /**
   * Sends a request to any endpoint and returns the decoded envelope.
   * Errors throw a {@link GoHeadError}.
   */
  async request<T>(method: string, path: string, opts: RequestOptions = {}): Promise<Envelope<T>> {
    const label = `${method} ${path}`;
    const init = mergeInit(this.options.init, opts.init);
    const headers = new Headers(this.options.headers);
    for (const extra of [this.options.init?.headers, opts.init?.headers]) {
      if (extra) new Headers(extra).forEach((v, k) => headers.set(k, v));
    }
    headers.set("Accept", "application/json");
    let body: BodyInit | undefined;
    if (opts.form) {
      body = opts.form;
    } else if (opts.body !== undefined) {
      headers.set("Content-Type", "application/json");
      body = JSON.stringify(opts.body);
    }
    if (!opts.anonymous) {
      const t = this.options.token;
      const token = typeof t === "function" ? await t() : t;
      if (token) headers.set("Authorization", `Bearer ${token}`);
    }
    const timeout = this.options.timeoutMs ?? 30_000;
    const callerSignal = init.signal ?? undefined;
    let signal = callerSignal;
    let timer: AbortSignal | undefined;
    if (timeout > 0 && typeof AbortSignal !== "undefined" && "timeout" in AbortSignal) {
      timer = AbortSignal.timeout(timeout);
      signal = callerSignal && "any" in AbortSignal ? AbortSignal.any([callerSignal, timer]) : (callerSignal ?? timer);
    }
    const doFetch = this.options.fetch ?? globalThis.fetch;
    if (!doFetch) throw new Error("No fetch available: pass `fetch` to createClient");

    let res: Response;
    try {
      res = await doFetch(this.url + path + (opts.query ?? ""), { ...init, method, headers, body, signal });
    } catch (cause) {
      if (callerSignal?.aborted) throw cause; // the caller cancelled: their AbortError
      const timedOut = !!timer?.aborted || (cause instanceof Error && cause.name === "TimeoutError");
      throw new GoHeadError({
        status: 0,
        code: timedOut ? "TimeoutError" : "NetworkError",
        message: timedOut ? `${label}: no answer from ${this.url}` : `${label}: cannot reach ${this.url} (${(cause as Error)?.message ?? cause})`,
        request: label,
        cause,
      });
    }

    let text: string;
    try {
      text = await res.text();
    } catch (cause) {
      if (callerSignal?.aborted) throw cause;
      throw new GoHeadError({ status: res.status, code: "NetworkError", message: `${label}: the answer was cut off`, request: label, cause });
    }
    let json: unknown = undefined;
    if (text) {
      try {
        json = JSON.parse(text);
      } catch {
        // handled below
      }
    }
    if (!res.ok) {
      const e = (json as { error?: { name?: string; message?: unknown; details?: unknown } } | undefined)?.error;
      throw new GoHeadError({
        status: res.status,
        code: e?.name ?? `HTTP${res.status}`,
        message: typeof e?.message === "string" ? e.message : `${label} failed with ${res.status}`,
        details: e?.details,
        request: label,
      });
    }
    if (json === undefined) {
      if (!text) return { data: null as T };
      throw new GoHeadError({ status: res.status, code: "InvalidResponse", message: `${label}: the answer is not JSON`, request: label });
    }
    return json as Envelope<T>;
  }

  /** @internal */
  queryOf(opts: (FindOptions<never> & ReadOptions) | undefined): string {
    return buildQuery(opts, { locale: this.options.locale, level: this.options.level });
  }
}

/** Items of one collection. `T` is the entry's attributes, `I` what writes accept. */
export class CollectionClient<T extends object, I extends object = Partial<T>> {
  constructor(
    private readonly client: GoHeadClient,
    readonly name: string,
  ) {}

  private get path(): string {
    return `/api/collections/${enc(this.name)}`;
  }

  /** A page of entries: filters, sort, pagination, status, locale, level. */
  async find(options?: FindOptions<T>, init?: RequestInit): Promise<ListResponse<Entry<T>>> {
    const res = await this.client.request<Entry<T>[]>("GET", this.path, { query: this.client.queryOf(options as unknown as FindOptions<never>), init });
    const pagination = (res.meta?.pagination as Pagination | undefined) ?? {
      page: 1,
      pageSize: res.data?.length ?? 0,
      pageCount: 1,
      total: res.data?.length ?? 0,
    };
    return { data: res.data ?? [], meta: { pagination } };
  }

  /** The first entry matching the options, or null (e.g. by slug). */
  async findFirst(options?: Omit<FindOptions<T>, "page" | "pageSize">, init?: RequestInit): Promise<Entry<T> | null> {
    const { data } = await this.find({ ...options, page: 1, pageSize: 1 } as FindOptions<T>, init);
    return data[0] ?? null;
  }

  /** One entry by ID. A missing entry throws a GoHeadError with status 404. */
  async findOne(id: number | string, options?: ReadOptions, init?: RequestInit): Promise<Entry<T>> {
    const res = await this.client.request<Entry<T>>("GET", `${this.path}/${enc(id)}`, { query: this.client.queryOf(options), init });
    return res.data;
  }

  /** Every page of entries, one page at a time. */
  async *pages(options?: Omit<FindOptions<T>, "page">, init?: RequestInit): AsyncGenerator<Entry<T>[], void, undefined> {
    let page = 1;
    for (;;) {
      const { data, meta } = await this.find({ pageSize: 100, ...options, page } as FindOptions<T>, init);
      if (data.length > 0) yield data;
      if (page >= meta.pagination.pageCount || data.length === 0) return;
      page++;
    }
  }

  /** Every matching entry (all pages). Mind large collections. */
  async findAll(options?: Omit<FindOptions<T>, "page">, init?: RequestInit): Promise<Entry<T>[]> {
    const out: Entry<T>[] = [];
    for await (const page of this.pages(options, init)) out.push(...page);
    return out;
  }

  /** How many entries match. */
  async count(options?: Pick<FindOptions<T>, "filters" | "status" | "locale">, init?: RequestInit): Promise<number> {
    const { meta } = await this.find({ ...options, page: 1, pageSize: 1, level: 0 } as FindOptions<T>, init);
    return meta.pagination.total;
  }

  /** Creates an entry. Pass `{ status: "published" }` to publish it at once. */
  async create(data: I, options: Omit<WriteOptions, "locale"> = {}): Promise<Entry<T>> {
    const query = options.level !== undefined ? `?level=${options.level}` : "";
    const body = options.status ? { data, status: options.status } : { data };
    return (await this.client.request<Entry<T>>("POST", this.path, { query, body })).data;
  }

  /** Creates several entries in one atomic batch. */
  async createMany(items: Array<{ data: I; status?: WriteOptions["status"] }>): Promise<Entry<T>[]> {
    return (await this.client.request<Entry<T>[]>("POST", this.path, { body: items })).data;
  }

  /**
   * Replaces an entry's data (send every attribute to keep). With `locale`,
   * writes that locale's localized values (a translation).
   */
  async update(id: number | string, data: Partial<I>, options: WriteOptions & { locale: string }): Promise<Entry<T>>;
  async update(id: number | string, data: I, options?: WriteOptions): Promise<Entry<T>>;
  async update(id: number | string, data: I | Partial<I>, options: WriteOptions = {}): Promise<Entry<T>> {
    const q = new URLSearchParams();
    if (options.level !== undefined) q.set("level", String(options.level));
    if (options.locale) q.set("locale", options.locale);
    const query = q.size ? `?${q}` : "";
    const body = options.status ? { data, status: options.status } : { data };
    return (await this.client.request<Entry<T>>("PUT", `${this.path}/${enc(id)}`, { query, body })).data;
  }

  /** Deletes an entry. */
  async delete(id: number | string): Promise<void> {
    await this.client.request<null>("DELETE", `${this.path}/${enc(id)}`);
  }

  /** Publishes a draft (draft-and-publish collections). */
  async publish(id: number | string): Promise<Entry<T>> {
    return (await this.client.request<Entry<T>>("POST", `${this.path}/${enc(id)}/publish`)).data;
  }

  /** Turns a published entry back into a draft. */
  async unpublish(id: number | string): Promise<Entry<T>> {
    return (await this.client.request<Entry<T>>("POST", `${this.path}/${enc(id)}/unpublish`)).data;
  }
}

/** A singleton's content. */
export class SingletonClient<T extends object, I extends object = Partial<T>> {
  constructor(
    private readonly client: GoHeadClient,
    readonly name: string,
  ) {}

  private get path(): string {
    return `/api/singleton/${enc(this.name)}`;
  }

  /** The content. Throws a GoHeadError (404) when it has none yet; see {@link find}. */
  async get(options?: ReadOptions, init?: RequestInit): Promise<Entry<T>> {
    return (await this.client.request<Entry<T>>("GET", this.path, { query: this.client.queryOf(options), init })).data;
  }

  /** The content, or null when it has none yet (or the singleton does not exist). */
  async find(options?: ReadOptions, init?: RequestInit): Promise<Entry<T> | null> {
    try {
      return await this.get(options, init);
    } catch (err) {
      if (err instanceof GoHeadError && err.status === 404) return null;
      throw err;
    }
  }

  /** Replaces the content (or, with `locale`, that locale's localized values). */
  async update(data: Partial<I>, options: Pick<WriteOptions, "level"> & { locale: string }): Promise<Entry<T>>;
  async update(data: I, options?: Pick<WriteOptions, "locale" | "level">): Promise<Entry<T>>;
  async update(data: I | Partial<I>, options: Pick<WriteOptions, "locale" | "level"> = {}): Promise<Entry<T>> {
    const q = new URLSearchParams();
    if (options.level !== undefined) q.set("level", String(options.level));
    if (options.locale) q.set("locale", options.locale);
    const query = q.size ? `?${q}` : "";
    return (await this.client.request<Entry<T>>("PUT", this.path, { query, body: { data } })).data;
  }
}

export interface MediaListOptions {
  page?: number;
  pageSize?: number;
  /** MIME type or prefix, e.g. `image/`. */
  mime?: string;
  search?: string;
}

/** A file to upload: a Blob/File, or bytes with a name. */
export type Upload = Blob | { name: string; data: Blob | ArrayBuffer | Uint8Array; type?: string };

/** The media library. */
export class MediaClient {
  constructor(private readonly client: GoHeadClient) {}

  /** Uploads files; `alt` and `caption` apply to each. */
  async upload(files: Upload | Upload[], meta: { alt?: string; caption?: string } = {}): Promise<Media[]> {
    const form = new FormData();
    for (const f of Array.isArray(files) ? files : [files]) {
      if (f instanceof Blob) {
        form.append("files", f, (f as File).name ?? "upload");
      } else {
        const data = f.data instanceof Blob ? f.data : new Blob([f.data as BlobPart], f.type ? { type: f.type } : undefined);
        form.append("files", data, f.name);
      }
    }
    if (meta.alt !== undefined) form.append("alt", meta.alt);
    if (meta.caption !== undefined) form.append("caption", meta.caption);
    return (await this.client.request<Media[]>("POST", "/api/media", { form })).data;
  }

  async list(options: MediaListOptions = {}, init?: RequestInit): Promise<ListResponse<Media>> {
    const q = new URLSearchParams();
    for (const [k, v] of Object.entries(options)) if (v !== undefined) q.set(k, String(v));
    const res = await this.client.request<Media[]>("GET", "/api/media", { query: q.size ? `?${q}` : "", init });
    const data = res.data ?? [];
    const pagination = (res.meta?.pagination as Pagination | undefined) ?? { page: 1, pageSize: data.length, pageCount: 1, total: data.length };
    return { data, meta: { pagination } };
  }

  async get(id: number, init?: RequestInit): Promise<Media> {
    return (await this.client.request<Media>("GET", `/api/media/${enc(id)}`, { init })).data;
  }

  async update(id: number, data: { name?: string; alt?: string; caption?: string }): Promise<Media> {
    return (await this.client.request<Media>("PUT", `/api/media/${enc(id)}`, { body: { data } })).data;
  }

  async delete(id: number): Promise<void> {
    await this.client.request<null>("DELETE", `/api/media/${enc(id)}`);
  }

  /**
   * The absolute URL of a file (local uploads are served by GoHead under
   * /uploads/…; S3/CDN URLs are returned as is). Accepts a Media object, a
   * URL or path, or null.
   */
  url(media: Pick<Media, "url"> | string | number | null | undefined): string | undefined {
    if (typeof media === "number") return undefined; // an unpopulated ID (level 0)
    const u = typeof media === "string" ? media : media?.url;
    if (!u) return undefined;
    if (/^[a-z][a-z0-9+.-]*:/i.test(u) || u.startsWith("//")) return u;
    return this.client.url + (u.startsWith("/") ? u : `/${u}`);
  }
}

/**
 * A relation or media value as populated content, or undefined when it is
 * an ID (deeper than the read's level), null or missing.
 *
 * ```ts
 * const author = populated(article.author); // Entry<Author> | undefined
 * ```
 */
export function populated<T extends object>(value: T | number | null | undefined): T | undefined {
  return value !== null && typeof value === "object" ? value : undefined;
}

/** Creates a GoHead client. */
export function createClient(options: ClientOptions): GoHeadClient {
  return new GoHeadClient(options);
}
