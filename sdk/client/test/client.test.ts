import { describe, expect, it, vi } from "vitest";

import { createClient, GoHeadError, isGoHeadError } from "../src/index.js";
import { buildQuery } from "../src/query.js";

type Call = { url: string; init: RequestInit };

function mockFetch(responses: Array<{ status?: number; body?: unknown } | Error>) {
  const calls: Call[] = [];
  const fn = vi.fn(async (url: string | URL | Request, init?: RequestInit) => {
    calls.push({ url: String(url), init: init ?? {} });
    const r = responses.shift() ?? { body: { data: null } };
    if (r instanceof Error) throw r;
    const text = r.body === undefined ? "" : typeof r.body === "string" ? r.body : JSON.stringify(r.body);
    return new Response(text, { status: r.status ?? 200, headers: { "Content-Type": "application/json" } });
  });
  return { fetch: fn as unknown as typeof fetch, calls };
}

const page = (data: unknown[], p = { page: 1, pageSize: 10, pageCount: 1, total: data.length }) => ({ body: { data, meta: { pagination: p } } });

describe("queries", () => {
  it("serializes filters, sort, pagination, status, locale and level", () => {
    const q = buildQuery(
      { filters: { category: "tech", views: { $gte: 10 }, title: { $contains: "go" }, published: true }, sort: ["views:desc", "title"], page: 2, pageSize: 5, status: "all", level: 2, locale: "fr", fallback: false } as never,
      {},
    );
    expect(decodeURIComponent(q)).toBe(
      "?filters[category]=tech&filters[views][$gte]=10&filters[title][$contains]=go&filters[published]=true&sort=views:desc,title&page=2&pageSize=5&status=all&level=2&locale=fr&fallback=false",
    );
  });

  it("applies the client's defaults, overridden per call", () => {
    expect(buildQuery(undefined, { locale: "de", level: 1 })).toBe("?level=1&locale=de");
    expect(buildQuery({ locale: "fr" } as never, { locale: "de" })).toBe("?locale=fr");
    expect(buildQuery(undefined, {})).toBe("");
  });

  it("refuses values filters cannot express", () => {
    expect(() => buildQuery({ filters: { a: null } } as never, {})).toThrow(/null/);
    expect(() => buildQuery({ filters: { a: { gte: 1 } } } as never, {})).toThrow(/start with \$/);
    expect(() => buildQuery({ filters: { a: [1, 2] } } as never, {})).toThrow(TypeError);
    expect(decodeURIComponent(buildQuery({ filters: { at: { $lt: new Date("2026-01-01T00:00:00Z") } } } as never, {}))).toBe(
      "?filters[at][$lt]=2026-01-01T00:00:00.000Z",
    );
  });
});

describe("client", () => {
  it("lists entries with the token and default options", async () => {
    const m = mockFetch([page([{ id: 1, title: "Hi" }])]);
    const gohead = createClient({ url: "https://cms.example.com/api/", token: "ght_x", fetch: m.fetch, locale: "fr", init: { cache: "no-store" } });
    const res = await gohead.collection("articles").find({ sort: "title" });
    expect(res.data[0]?.title).toBe("Hi");
    expect(res.meta.pagination.total).toBe(1);
    const { url, init } = m.calls[0]!;
    expect(url).toBe("https://cms.example.com/api/collections/articles?sort=title&locale=fr");
    expect(new Headers(init.headers).get("authorization")).toBe("Bearer ght_x");
    expect(init.cache).toBe("no-store");
  });

  it("is anonymous without a token, and calls token functions per request", async () => {
    const m = mockFetch([page([]), page([])]);
    await createClient({ url: "http://x", fetch: m.fetch }).collection("a").find();
    expect(new Headers(m.calls[0]!.init.headers).has("authorization")).toBe(false);
    let n = 0;
    await createClient({ url: "http://x", fetch: m.fetch, token: async () => `t${++n}` }).collection("a").find();
    expect(new Headers(m.calls[1]!.init.headers).get("authorization")).toBe("Bearer t1");
  });

  it("merges fetch options: Next.js tags and headers", async () => {
    const m = mockFetch([page([])]);
    const gohead = createClient({ url: "http://x", fetch: m.fetch, init: { next: { revalidate: 60, tags: ["all"] }, headers: { "X-A": "1" } } as RequestInit });
    await gohead.collection("a").find({}, { next: { tags: ["a"] }, headers: { "X-B": "2" } } as RequestInit);
    expect((m.calls[0]!.init as { next?: unknown }).next).toEqual({ revalidate: 60, tags: ["all", "a"] });
    const h = new Headers(m.calls[0]!.init.headers);
    expect([h.get("x-a"), h.get("x-b"), h.get("accept")]).toEqual(["1", "2", "application/json"]);
  });

  it("lets the caller's abort through, and reports timeouts", async () => {
    const hang = (async (_u: unknown, init?: RequestInit) =>
      new Promise((_, reject) => init?.signal?.addEventListener("abort", () => reject(init.signal!.reason)))) as unknown as typeof fetch;
    const ctl = new AbortController();
    const p = createClient({ url: "http://x", fetch: hang }).locales({ signal: ctl.signal });
    ctl.abort();
    await expect(p).rejects.toMatchObject({ name: "AbortError" });
    const slow = (await createClient({ url: "http://x", fetch: hang, timeoutMs: 20 }).locales().catch((e: unknown) => e)) as GoHeadError;
    expect(slow.code).toBe("TimeoutError");
  });

  it("writes", async () => {
    const m = mockFetch([{ body: { data: { id: 3 } } }, { body: { data: { id: 3 } } }, { body: { data: null } }, { body: { data: { id: 3 } } }, { body: { data: [{ id: 4 }] } }]);
    const a = createClient({ url: "http://x", fetch: m.fetch }).collection("articles");
    await a.create({ title: "x" }, { status: "published" });
    await a.update(3, { title: "y" }, { locale: "fr" });
    await a.delete(3);
    await a.publish(3);
    await a.createMany([{ data: { title: "z" } }]);
    expect(m.calls.map((c) => `${c.init.method} ${c.url.replace("http://x", "")}`)).toEqual([
      "POST /api/collections/articles",
      "PUT /api/collections/articles/3?locale=fr",
      "DELETE /api/collections/articles/3",
      "POST /api/collections/articles/3/publish",
      "POST /api/collections/articles",
    ]);
    expect(JSON.parse(String(m.calls[0]!.init.body))).toEqual({ data: { title: "x" }, status: "published" });
    expect(JSON.parse(String(m.calls[4]!.init.body))).toEqual([{ data: { title: "z" } }]);
  });

  it("pages through everything", async () => {
    const m = mockFetch([
      page([{ id: 1 }, { id: 2 }], { page: 1, pageSize: 2, pageCount: 2, total: 3 }),
      page([{ id: 3 }], { page: 2, pageSize: 2, pageCount: 2, total: 3 }),
    ]);
    const all = await createClient({ url: "http://x", fetch: m.fetch }).collection("a").findAll({ pageSize: 2 });
    expect(all.map((e) => e.id)).toEqual([1, 2, 3]);
    expect(m.calls[1]!.url).toContain("page=2");
  });

  it("finds the first match or null", async () => {
    const m = mockFetch([page([{ id: 7, slug: "x" }]), page([])]);
    const a = createClient({ url: "http://x", fetch: m.fetch }).collection("a");
    expect((await a.findFirst({ filters: { slug: "x" } }))?.id).toBe(7);
    expect(await a.findFirst({ filters: { slug: "y" } })).toBeNull();
    expect(m.calls[0]!.url).toContain("pageSize=1");
  });

  it("turns error envelopes into GoHeadErrors", async () => {
    const m = mockFetch([{ status: 400, body: { error: { status: 400, name: "ValidationError", message: "missing required attribute: 'title'", details: { item_index: 1 } } } }]);
    const err = await createClient({ url: "http://x", fetch: m.fetch }).collection("a").create({}).catch((e: unknown) => e);
    expect(isGoHeadError(err)).toBe(true);
    const e = err as GoHeadError;
    expect([e.status, e.code, e.message, e.isValidation, e.request]).toEqual([400, "ValidationError", "missing required attribute: 'title'", true, "POST /api/collections/a"]);
    expect(e.details).toEqual({ item_index: 1 });
  });

  it("reports network failures and non-JSON answers", async () => {
    const m = mockFetch([new TypeError("fetch failed"), { status: 502, body: "<html>bad gateway</html>" }, { status: 200, body: "<html>" }]);
    const gohead = createClient({ url: "http://x", fetch: m.fetch });
    const e1 = (await gohead.locales().catch((e: unknown) => e)) as GoHeadError;
    expect([e1.status, e1.code]).toEqual([0, "NetworkError"]);
    const e2 = (await gohead.locales().catch((e: unknown) => e)) as GoHeadError;
    expect([e2.status, e2.code]).toEqual([502, "HTTP502"]);
    const e3 = (await gohead.locales().catch((e: unknown) => e)) as GoHeadError;
    expect(e3.code).toBe("InvalidResponse");
  });

  it("returns null for a singleton without content", async () => {
    const m = mockFetch([{ status: 404, body: { error: { status: 404, name: "NotFoundError", message: "no single item found" } } }, { body: { data: { id: 1, headline: "Hi" } } }]);
    const s = createClient({ url: "http://x", fetch: m.fetch }).singleton("homepage");
    expect(await s.find()).toBeNull();
    expect((await s.get({ locale: "fr" })).headline).toBe("Hi");
    expect(m.calls[1]!.url).toBe("http://x/api/singleton/homepage?locale=fr");
  });

  it("resolves media URLs", () => {
    const gohead = createClient({ url: "https://cms.example.com/" });
    expect(gohead.media.url({ url: "/uploads/a.png" })).toBe("https://cms.example.com/uploads/a.png");
    expect(gohead.media.url("https://cdn.example.com/a.png")).toBe("https://cdn.example.com/a.png");
    expect(gohead.media.url(null)).toBeUndefined();
    expect(gohead.media.url(4)).toBeUndefined();
  });

  it("uploads files as multipart", async () => {
    const m = mockFetch([{ body: { data: [{ id: 1, url: "/uploads/a.txt" }] } }]);
    const files = await createClient({ url: "http://x", fetch: m.fetch }).media.upload({ name: "a.txt", data: new TextEncoder().encode("hi"), type: "text/plain" }, { alt: "A" });
    expect(files[0]?.id).toBe(1);
    const form = m.calls[0]!.init.body as FormData;
    expect((form.get("files") as File).name).toBe("a.txt");
    expect(form.get("alt")).toBe("A");
    expect(new Headers(m.calls[0]!.init.headers).has("content-type")).toBe(false);
  });

  it("surfaces GraphQL errors", async () => {
    const m = mockFetch([{ status: 400, body: { error: { status: 400, name: "ValidationError", message: "GraphQL request failed", details: 'Cannot query field "x"' } } }]);
    const err = (await createClient({ url: "http://x", fetch: m.fetch }).graphql("{ x }").catch((e: unknown) => e)) as GoHeadError;
    expect(err.message).toBe('Cannot query field "x"');
  });

  it("runs GraphQL and signs in", async () => {
    const m = mockFetch([{ body: { data: { articles: [] } } }, { body: { data: { token: "jwt" } } }]);
    const gohead = createClient({ url: "http://x", fetch: m.fetch, token: "ght_x" });
    expect(await gohead.graphql("{ articles { title } }")).toEqual({ articles: [] });
    expect(await gohead.login("u", "p")).toBe("jwt");
    expect(new Headers(m.calls[1]!.init.headers).has("authorization")).toBe(false);
  });

  it("requires a URL", () => {
    expect(() => createClient({ url: "" })).toThrow(/url is required/);
  });
});
