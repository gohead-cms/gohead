/**
 * End-to-end tests against a live GoHead (CI starts one):
 *   GOHEAD_URL=http://localhost:8080 GOHEAD_USERNAME=admin GOHEAD_PASSWORD=… npm run test:integration
 * They create their own types (suffixed) and leave them behind.
 */
import { describe, expect, it, beforeAll } from "vitest";

import { createClient, GoHeadError, type GoHeadClient } from "../src/index.js";
import { pullModel, pushModel, type Model } from "../src/cli/model.js";
import { seed } from "../src/cli/seed.js";
import { generateTypes } from "../src/cli/typegen.js";

const url = process.env.GOHEAD_URL ?? "http://localhost:8080";
const username = process.env.GOHEAD_USERNAME ?? process.env.GOHEAD_ADMIN_USERNAME ?? "";
const password = process.env.GOHEAD_PASSWORD ?? process.env.GOHEAD_ADMIN_PASSWORD ?? "";
const suffix = Date.now().toString(36);
const posts = `posts_${suffix}`;
const people = `people_${suffix}`;
const settings = `settings_${suffix}`;

const model: Model = {
  locales: [{ code: "fr", name: "Français" }],
  components: { [`meta_${suffix}`]: { attributes: { title: { type: "string" } } } },
  collections: {
    [posts]: {
      draftAndPublish: true,
      attributes: {
        title: { type: "string", required: true, localized: true },
        slug: { type: "uid", unique: true },
        views: { type: "int" },
        kind: { type: "enum", options: ["news", "guide"] },
        author: { type: "relation", relation: "oneToOne", target: people },
        meta: { type: "component", component: `meta_${suffix}` },
      },
    },
    [people]: { attributes: { name: { type: "string", required: true }, best: { type: "relation", relation: "oneToOne", target: posts } } },
  },
  singletons: { [settings]: { attributes: { headline: { type: "string", localized: true } } } },
  public: { content: { [posts]: ["read"], [`singleton:${settings}`]: ["read"] } },
};

let admin: GoHeadClient;
const anon = createClient({ url });

describe.skipIf(!username || !password)("GoHead SDK against a live server", () => {
  beforeAll(async () => {
    const token = await createClient({ url }).login(username, password);
    admin = createClient({ url, token });
  });

  it("pushes a model with relation cycles, idempotently", async () => {
    const plan = await pushModel(admin, model);
    expect(plan.filter((s) => s.action === "create").map((s) => `${s.kind}:${s.name}`)).toEqual(
      expect.arrayContaining([`collection:${posts}`, `collection:${people}`, `singleton:${settings}`, `component:meta_${suffix}`]),
    );
    const again = await pushModel(admin, model);
    expect(again.filter((s) => s.action !== "unchanged" && s.kind !== "locale")).toEqual([]);

    const pulled = await pullModel(admin);
    expect(pulled.collections?.[people]?.attributes.best).toEqual({ type: "relation", relation: "oneToOne", target: posts });
    expect(pulled.collections?.[posts]?.draftAndPublish).toBe(true);
    expect(generateTypes(pulled)).toContain(`${posts}: { attributes:`);
  });

  it("refuses to drop attributes unless allowed", async () => {
    const smaller = structuredClone(model);
    delete smaller.collections![posts]!.attributes.views;
    await expect(pushModel(admin, smaller, { dryRun: true })).rejects.toThrow(/removes attributes/);
  });

  it("seeds with references and translations", async () => {
    const res = await seed(
      admin,
      {
        collections: {
          [people]: [{ $id: "ada", name: "Ada" }],
          [posts]: [
            { $id: "a", $status: "published", title: "Alpha", slug: `alpha-${suffix}`, views: 10, kind: "news", author: `@ref:${people}/ada`, $translations: { fr: { title: "Alpha FR" } } },
            { $id: "b", $status: "published", title: "Beta", slug: `beta-${suffix}`, views: 20, kind: "guide", meta: { title: "B" } },
            { $id: "c", title: "Draft", slug: `draft-${suffix}`, views: 30 },
          ],
        },
        singletons: { [settings]: { headline: "Hello", $translations: { fr: { headline: "Bonjour" } } } },
      },
      { baseDir: "." },
    );
    expect(res).toMatchObject({ entries: 4, singletons: 1 });
    const again = await seed(admin, { collections: { [posts]: [{ title: "x" }] } }, { baseDir: "." });
    expect(again.skipped).toEqual([posts]);
  });

  it("reads anonymously what the public role grants", async () => {
    const list = await anon.collection(posts).find({ sort: "views:desc", level: 1 });
    expect(list.data.map((p) => p.title)).toEqual(["Beta", "Alpha"]);
    expect(list.meta.pagination.total).toBe(2);
    expect(list.data[1]?.author).toBeNull(); // people are not public

    const fr = await anon.collection(posts).findFirst({ filters: { slug: `alpha-${suffix}` }, locale: "fr" });
    expect(fr?.title).toBe("Alpha FR");
    expect((await anon.singleton(settings).get({ locale: "fr" })).headline).toBe("Bonjour");

    const err = (await anon.collection(people).find().catch((e: unknown) => e)) as GoHeadError;
    expect(err).toBeInstanceOf(GoHeadError);
    expect(err.status).toBe(401);
    const draft = (await anon.collection(posts).find({ status: "draft" }).catch((e: unknown) => e)) as GoHeadError;
    expect(draft.status).toBe(401);
  });

  it("filters, sorts, pages and counts", async () => {
    const c = admin.collection(posts);
    expect((await c.find({ filters: { views: { $gte: 15 } }, status: "all" })).data.map((p) => p.title).sort()).toEqual(["Beta", "Draft"]);
    expect((await c.find({ filters: { kind: "news" } })).data.map((p) => p.title)).toEqual(["Alpha"]);
    expect(await c.count({ status: "all" })).toBe(3);
    const pages: number[] = [];
    for await (const page of c.pages({ pageSize: 1, status: "all", sort: "views" })) pages.push(page.length);
    expect(pages).toEqual([1, 1, 1]);
    expect((await c.findAll({ pageSize: 2, status: "all" })).length).toBe(3);
    const bad = (await c.find({ filters: { nope: 1 } }).catch((e: unknown) => e)) as GoHeadError;
    expect(bad.status).toBe(400);
  });

  it("creates, updates, publishes and deletes", async () => {
    const c = admin.collection(posts);
    const created = await c.create({ title: "Gamma", slug: `gamma-${suffix}` });
    expect(created.status).toBe("draft");
    const published = await c.publish(created.id);
    expect(published.status).toBe("published");
    const updated = await c.update(created.id, { title: "Gamma 2", slug: `gamma-${suffix}` });
    expect(updated.title).toBe("Gamma 2");
    await c.update(created.id, { title: "Gamma FR" }, { locale: "fr" });
    expect((await c.findOne(created.id, { locale: "fr" })).title).toBe("Gamma FR");
    await c.delete(created.id);
    const gone = (await c.findOne(created.id).catch((e: unknown) => e)) as GoHeadError;
    expect(gone.isNotFound).toBe(true);
    const invalid = (await c.create({ slug: "no-title" }).catch((e: unknown) => e)) as GoHeadError;
    expect([invalid.status, invalid.code]).toEqual([400, "ValidationError"]);
  });

  it("uploads media and runs GraphQL", async () => {
    const [file] = await admin.media.upload({ name: `note-${suffix}.txt`, data: new TextEncoder().encode("hello"), type: "text/plain" }, { alt: "note" });
    expect(file?.id).toBeGreaterThan(0);
    expect(admin.media.url(file)).toMatch(/^https?:\/\//);
    const listed = await admin.media.list({ search: `note-${suffix}` });
    expect(listed.data.some((m) => m.id === file?.id)).toBe(true);
    await admin.media.delete(file!.id);

    const data = await admin.graphql<Record<string, Array<{ title: string }>>>(`{ ${posts} { title } }`);
    expect(data[posts]?.map((p) => p.title)).toContain("Alpha");
    const err = (await anon.graphql("{ __schema { types { name } } }").catch((e: unknown) => e)) as GoHeadError;
    expect(err.status).toBe(401);
  });

  it("knows who it is", async () => {
    expect((await admin.me()).role).toBe("admin");
    expect((await admin.locales()).map((l) => l.code)).toContain("fr");
  });
});
