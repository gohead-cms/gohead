# Next.js starter

A multilingual blog on Next.js 16 (App Router) and GoHead: articles, authors and
categories with drafts, English and French, a home page edited in GoHead, Markdown bodies,
Open Graph metadata, a sitemap and RSS feeds, light and dark themes, and pages that refresh
when you publish. Its source is in
[`templates/nextjs-blog`](https://github.com/gohead-cms/gohead/tree/master/templates/nextjs-blog).
To see every GoHead feature at work (forms, previews, GraphQL, agents), use the conference
starter of the [feature tour](feature-tour.md) instead.

## Create it

You need Node.js 20.9+ and, to run GoHead locally, Docker.

```bash
npm create @gohead-cms@latest my-blog     # pnpm create @gohead-cms, yarn create @gohead-cms…
cd my-blog
docker compose up -d                      # GoHead on :8080, its admin UI on :3001
npm run gohead:setup                      # content model, sample articles, types
npm run dev                               # http://localhost:3000
```

`npm create` copies the template, installs its dependencies and writes `.env.local`
(`--template`, `--url <your GoHead>`, `--no-install`, `--yes`). The template's
`docker-compose.yml` runs GoHead with an `admin` / `gohead-local-password` account bound
to localhost; with a GoHead you already run, set `GOHEAD_URL`, `GOHEAD_USERNAME` and
`GOHEAD_PASSWORD` in `.env.local` instead.

| Script | |
|--------|-|
| `gohead:push` | Applies `gohead/model.json`: locales, the `seo` component, collections, singletons, public access ([models as code](model-as-code.md)) |
| `gohead:seed` | Uploads `gohead/media/` and creates the entries of `gohead/seed.json` |
| `gohead:types` | Writes `src/gohead.d.ts` from the model |
| `gohead:setup` | The three, in order |
| `dev`, `build`, `start`, `lint`, `typecheck` | The usual |

## How it is built

| GoHead | Site |
|--------|------|
| `site` singleton | Header, footer and default metadata of every page |
| `homepage` singleton | Hero and featured articles of `/en` and `/fr` |
| `articles` (draft and publish) | The article list (`?page=`) and `/[locale]/articles/[slug]` |
| `categories`, `authors` | Navigation, bylines, `/[locale]/categories/[slug]`, `/[locale]/authors/[slug]` |

- **Reads** live in `src/lib/gohead.ts`: one server-only client from `@gohead-cms/client`,
  typed by `src/gohead.d.ts`. The site reads published content through
  [public access](../guides/users-and-roles.md#public-access), opened by the model's
  `public` key, so it needs no token; set `GOHEAD_TOKEN` for a private GoHead.
- **Languages:** `/` redirects to `/en` or `/fr` from the browser's language; localized
  attributes fall back to English; slugs are shared, so the language switcher keeps the page.
- **Bodies** are Markdown, rendered without raw HTML.

## Publishing and revalidation

Pages are static. Every GoHead read is cached with the tags `gohead` and
`gohead:<collection>` (`gohead:singleton:<name>` for singletons) and refetched after
`GOHEAD_REVALIDATE` seconds (60). With a webhook, changes show up on the next visit:

1. In GoHead, **Settings → Webhooks**: URL `https://your-site/api/revalidate`, all events.
2. Put the signing secret GoHead shows (`whsec_…`) in `GOHEAD_WEBHOOK_SECRET`.

`src/app/api/revalidate/route.ts` checks the signature with `parseWebhook` and expires the
tags of the collection or singleton the event is about. GoHead refuses webhook URLs on
private addresses unless `webhooks.allow_private_networks` is on: the template's
`docker-compose.yml` turns it on for local work, so a webhook can target
`http://host.docker.internal:3000/api/revalidate`.

## Make it yours

1. Change `gohead/model.json` (the `$schema` gives your editor completion), then
   `npm run gohead:push` and `npm run gohead:types`.
2. Fix what TypeScript reports, and build pages with the typed client.
3. Deploy anywhere Next.js runs (Vercel, a Node server, a container) with `GOHEAD_URL`,
   `GOHEAD_WEBHOOK_SECRET` and `NEXT_PUBLIC_SITE_URL`, and point the webhook at it.

A build that can't reach GoHead still succeeds and renders pages on their first visit.
