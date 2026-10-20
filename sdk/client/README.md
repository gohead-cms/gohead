# @gohead-cms/client

The JavaScript and TypeScript client for [GoHead](https://github.com/gohead-cms/gohead), the headless CMS written in Go.

- **Typed content**: generate types from your content model and get autocompletion for collections, fields, filters and sorts.
- **Everything the content API does**: entries, drafts and publishing, singletons, locales, media, GraphQL.
- **Runs anywhere `fetch` does**: Node 20+, browsers, Next.js (server components, route handlers, edge), Remix, Astro, Bun, Deno, Workers. No dependencies.
- **A CLI** for type generation, content models as code and sample content.
- **Webhook verification** for on-demand revalidation.

```bash
npm install @gohead-cms/client
```

It is an ES module (`import`); `require()` works on Node 20.19+ and 22.12+.

## Quick start

```ts
import { createClient } from "@gohead-cms/client";

const gohead = createClient({
  url: "https://cms.example.com",     // your GoHead, without /api
  token: process.env.GOHEAD_TOKEN,    // optional: without a token, you read what public access allows
});

const { data, meta } = await gohead.collection("articles").find({
  filters: { category: "tech", views: { $gte: 100 } },
  sort: ["publishedAt:desc", "title"],
  pageSize: 10,
  level: 1,            // populate relations one level deep
  locale: "fr",        // localized values in French (falling back to the default locale)
});

const article = await gohead.collection("articles").findFirst({ filters: { slug: "hello" } });
const home = await gohead.singleton("homepage").find(); // null until it has content
```

## Typed content

Generate types from your instance (or from a model file):

```bash
npx @gohead-cms/client types --url https://cms.example.com --token $GOHEAD_TOKEN --out src/gohead.d.ts
```

The file declares one interface per collection, singleton and component, and teaches the client about them:

```ts
const { data } = await gohead.collection("articles").find({ filters: { featured: true } });
data[0].title;                    // string
populated(data[0].author)?.name;  // string | undefined: a relation, an ID when not populated

gohead.collection("articles").find({ filters: { nope: 1 } }); // compile error: unknown field
```

Regenerate it whenever the content model changes. Without generated types, everything works with `Record<string, unknown>`.

## Reading

| Method | Returns |
|--------|---------|
| `collection(name).find(options?)` | `{ data, meta: { pagination } }` |
| `collection(name).findFirst(options?)` | the first match, or `null` |
| `collection(name).findOne(id, options?)` | one entry (a missing one throws a 404 `GoHeadError`) |
| `collection(name).findAll(options?)` / `.pages(options?)` | every page, as an array or an async iterator |
| `collection(name).count(options?)` | how many entries match |
| `singleton(name).get(options?)` / `.find(options?)` | the content (`find` returns `null` when there is none) |
| `locales()`, `defaultLocale()` | content locales |
| `graphql(query, variables?)` | the query's `data` |
| `me()` | the account or token you act as |

Options: `filters` (`$eq`, `$ne`, `$lt`, `$lte`, `$gt`, `$gte`, `$contains`; a plain value means `$eq`), `sort`, `page`, `pageSize` (≤ 100), `status` (`published`, `draft`, `all`: drafts need a token that may edit), `level` (0–5), `locale`, `fallback`.

Every read takes fetch options as its last argument, e.g. for Next.js caching:

```ts
await gohead.collection("articles").find({ pageSize: 3 }, { next: { tags: ["gohead:articles"], revalidate: 60 } });
```

Client-wide defaults go in `createClient({ init, locale, level })`, and `gohead.with({ locale: "fr" })` returns a copy with other options.

## Writing

Writes need a token whose role allows them.

```ts
const articles = gohead.collection("articles");
const draft = await articles.create({ title: "Hello", author: 3 });           // relations by ID
await articles.update(draft.id, { title: "Hello!", author: 3 });              // replaces the data
await articles.update(draft.id, { title: "Bonjour !" }, { locale: "fr" });   // a translation
await articles.publish(draft.id);
await articles.delete(draft.id);
await articles.createMany([{ data: { title: "A" } }, { data: { title: "B" }, status: "published" }]);

await gohead.singleton("homepage").update({ headline: "Welcome" });

const [file] = await gohead.media.upload(new File([bytes], "cover.jpg", { type: "image/jpeg" }), { alt: "Cover" });
await articles.update(draft.id, { title: "Hello!", cover: file.id });
```

`gohead.media.url(entry.cover)` turns a file into an absolute URL (local uploads are served by GoHead under `/uploads/`).

## Errors

Failures throw a `GoHeadError` with `status` (0 when the server could not be reached), `code` (`ValidationError`, `NotFoundError`…), `message`, `details` and `request`.

```ts
import { isGoHeadError } from "@gohead-cms/client";

try {
  await gohead.collection("articles").create({});
} catch (err) {
  if (isGoHeadError(err) && err.isValidation) console.log(err.message); // missing required attribute: 'title'
}
```

## Webhooks

Verify GoHead's signed webhooks, e.g. to revalidate a Next.js site when content is published:

```ts
// app/api/revalidate/route.ts
import { parseWebhook } from "@gohead-cms/client";
import { revalidateTag } from "next/cache";

export async function POST(request: Request) {
  const event = await parseWebhook(request, process.env.GOHEAD_WEBHOOK_SECRET!);
  if (!event) return new Response("invalid signature", { status: 401 });
  revalidateTag(`gohead:${event.collection}`, "max");
  return Response.json({ revalidated: true });
}
```

## CLI

```bash
npx @gohead-cms/client types --out src/gohead.d.ts        # TypeScript types
npx @gohead-cms/client model pull --out gohead/model.json # the content model, as a file
npx @gohead-cms/client model push gohead/model.json       # apply a model file (never deletes types)
npx @gohead-cms/client seed gohead/seed.json              # sample entries and media
npx @gohead-cms/client token create --name website        # an API token
```

Connection: `--url` / `GOHEAD_URL`, `--token` / `GOHEAD_TOKEN`, or `--username` and `--password` (`GOHEAD_USERNAME`, `GOHEAD_PASSWORD`). `.env.local` and `.env` are read. `model push` and `token create` need an admin user.

A model file describes locales, components, collections, singletons and what anonymous visitors may read:

```json
{
  "locales": [{ "code": "en", "default": true }, { "code": "fr", "name": "Français" }],
  "collections": {
    "articles": {
      "draftAndPublish": true,
      "attributes": {
        "title": { "type": "string", "required": true, "localized": true },
        "slug": { "type": "uid", "required": true, "unique": true },
        "author": { "type": "relation", "relation": "oneToOne", "target": "authors" }
      }
    },
    "authors": { "attributes": { "name": { "type": "string", "required": true } } }
  },
  "public": { "content": { "articles": ["read"], "authors": ["read"] } }
}
```

## Starters

`npm create @gohead-cms@latest` creates a site from a template (Next.js blog) with its content model, sample content and setup scripts.

## Documentation

[gohead-cms.github.io/gohead/frontend](https://gohead-cms.github.io/gohead/frontend/sdk/) · MIT license
