# JavaScript client

`@gohead-cms/client` is GoHead's client for JavaScript and TypeScript. It has no
dependencies and runs wherever `fetch` does: Node 20+, browsers, Next.js (server
components, route handlers, edge), Astro, Remix, Bun, Deno and Workers.

```bash
npm install @gohead-cms/client
```

## Connect

```ts
import { createClient } from "@gohead-cms/client";

export const gohead = createClient({
  url: process.env.GOHEAD_URL!,       // e.g. https://cms.example.com (without /api)
  token: process.env.GOHEAD_TOKEN,    // optional
});
```

| Option | |
|--------|-|
| `url` | GoHead's base URL |
| `token` | An [API token](../guides/integrations.md#api-tokens) (`ght_…`) or a session token, or a function returning one (called per request). Without one, requests are anonymous: they read what [public access](../guides/users-and-roles.md#public-access) allows. |
| `locale` | Default locale of reads |
| `level` | Default depth of populated relations (0–5) |
| `init` | Default `fetch` options, e.g. `{ cache: "no-store" }` or Next.js `{ next: { revalidate: 60 } }` |
| `headers` | Extra headers on every request |
| `fetch` | A `fetch` implementation (default: the global one) |
| `timeoutMs` | Abort after this long (default 30 s; 0: never) |

`gohead.with({ locale: "fr" })` returns a copy with some options changed, e.g. per request.

## Read entries

```ts
const articles = gohead.collection("articles");

const { data, meta } = await articles.find({
  filters: { category: "tech", views: { $gte: 100 }, title: { $contains: "Go" } },
  sort: ["publishedAt:desc", "title"],
  page: 1,
  pageSize: 10,
  level: 1,
  locale: "fr",
});
meta.pagination; // { page, pageSize, pageCount, total }

const one = await articles.findOne(42);                                  // throws a 404 GoHeadError if missing
const bySlug = await articles.findFirst({ filters: { slug: "hello" } }); // or null
const all = await articles.findAll({ filters: { featured: true } });     // every page
for await (const page of articles.pages({ pageSize: 100 })) { /* … */ }
const total = await articles.count({ filters: { category: "tech" } });
```

| Option | |
|--------|-|
| `filters` | Conditions combined with AND: a value (`$eq`) or operators `$eq`, `$ne`, `$lt`, `$lte`, `$gt`, `$gte`, `$contains`. See [filtering](../guides/rest-api.md#filtering-and-sorting) for the fields that can be filtered. |
| `sort` | `"field"`, `"field:desc"`, or a list |
| `page`, `pageSize` | 1-based page, 1–100 entries |
| `status` | `published` (default), `draft` or `all` in [draft-and-publish](../guides/content-modeling.md#draft-and-publish) collections. Drafts need a token whose role may edit the type. |
| `level` | How deep relations are populated (anonymous requests: 2 at most) |
| `locale`, `fallback` | [Localized](../guides/localization.md) values, falling back to the default locale unless `fallback: false` |

Entries have their attributes at the top level, with `id`, `createdAt`, `updatedAt`, and
`status` and `publishedAt` in draft-and-publish collections.

### Singletons, locales, GraphQL

```ts
const home = await gohead.singleton("homepage").find({ locale: "fr" }); // null when empty
const site = await gohead.singleton("site").get();                    // throws when empty

const locales = await gohead.locales();          // [{ code: "en", isDefault: true, … }, …]
const data = await gohead.graphql(`query { articles(limit: 3) { title } }`);
```

### Caching (Next.js and others)

Every read takes `fetch` options as its last argument:

```ts
await articles.find({ pageSize: 3 }, { next: { tags: ["gohead:articles"], revalidate: 60 } });
```

## Write

Writes need a token whose role allows them.

```ts
const post = await articles.create({ title: "Hello", author: 3 }, { status: "published" });
await articles.update(post.id, { title: "Hello!", author: 3 });            // replaces the data
await articles.update(post.id, { title: "Bonjour !" }, { locale: "fr" }); // a translation
await articles.publish(post.id);
await articles.unpublish(post.id);
await articles.delete(post.id);
await articles.createMany([{ data: { title: "A" } }, { data: { title: "B" } }]); // atomic

await gohead.singleton("homepage").update({ headline: "Welcome" });
```

Relations take IDs (or objects, to create related entries in the same transaction), media
attributes take file IDs, components take objects.

## Media

```ts
const [file] = await gohead.media.upload(fileFromInput, { alt: "Cover" }); // Blob, File or { name, data }
await gohead.media.list({ mime: "image/", search: "cover" });
await gohead.media.update(file.id, { alt: "New alt text" });
await gohead.media.delete(file.id);

gohead.media.url(article.cover); // absolute URL (local uploads live under /uploads/ on GoHead)
```

## Errors

Failures throw a `GoHeadError`:

| Property | |
|----------|-|
| `status` | HTTP status, `0` when GoHead could not be reached |
| `code` | GoHead's error name: `ValidationError`, `NotFoundError`, `UnauthorizedError`… (`NetworkError`, `TimeoutError` without an answer) |
| `message`, `details` | What went wrong (validation details, failing batch item…) |
| `request` | e.g. `POST /api/collections/articles` |
| `isNotFound`, `isUnauthorized`, `isForbidden`, `isValidation` | Shortcuts |

```ts
import { isGoHeadError } from "@gohead-cms/client";

try {
  await articles.create({ title: "" });
} catch (err) {
  if (isGoHeadError(err) && err.isValidation) showError(err.message);
  else throw err;
}
```

## Types

[Generate types](model-as-code.md#typescript-types) from your content model, and the client
knows your collections:

```ts
import { populated } from "@gohead-cms/client";

const { data } = await gohead.collection("articles").find({ filters: { featured: true } });
data[0].title;                     // string
populated(data[0].author)?.name;   // string | undefined

gohead.collection("articles").find({ filters: { cover: 1 } });   // error: media can't be filtered
gohead.collection("articles").create({ slug: "x" });            // error: title is required
```

Relations are populated down to the read's `level` and IDs below it (media are IDs at
level 0), so they are typed `Entry<Author> | number`: `populated(value)` returns the
entry, or `undefined` for an ID. A translation (`update(id, data, { locale })`) takes only
the attributes it changes.

Without generated types, entries are `Record<string, unknown>`; you can also pass your own
types: `gohead.collection<Article, ArticleInput>("articles")`.

## Webhooks

GoHead signs [webhook](../guides/integrations.md#outbound-webhooks) deliveries.
`parseWebhook` checks the signature and the timestamp (5 minutes by default) and returns
the event, or `null`:

```ts
import { parseWebhook } from "@gohead-cms/client";

export async function POST(request: Request) {
  const event = await parseWebhook(request, process.env.GOHEAD_WEBHOOK_SECRET!);
  if (!event) return new Response("invalid signature", { status: 401 });
  // event.event: item:created | item:updated | item:deleted | item:published | item:unpublished
  //              | singleton:updated | singleton:deleted
  // event.collection (singleton:<name> for singletons), event.data (the entry)
  return Response.json({ ok: true });
}
```

`verifyWebhook({ body, secret, signature, timestamp })` does the same for frameworks without
`Request` objects (pass the raw body).

## Sign in

```ts
const token = await gohead.login("alice", process.env.PASSWORD!); // a session token (72 h)
const me = await gohead.with({ token }).me();                     // { username, role, permissions, … }
```

Prefer [API tokens](../guides/integrations.md#api-tokens) for servers and build scripts.
