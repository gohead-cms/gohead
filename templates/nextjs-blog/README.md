# GoHead + Next.js blog starter

A multilingual blog built with [Next.js](https://nextjs.org) on top of [GoHead](https://github.com/gohead-cms/gohead), a headless CMS written in Go. Edit articles in the GoHead admin; the site reads them through the typed [`@gohead-cms/client`](https://gohead-cms.github.io/gohead/frontend/sdk/) and refreshes the pages you changed as soon as you publish.

- Articles, authors and categories, with drafts and publishing
- English and French, with a language switcher that keeps the current page
- Home page hero and featured articles edited in GoHead
- Markdown article bodies, cover images through `next/image`
- Static pages, refreshed on publish by a signed webhook
- Open Graph metadata, sitemap, robots.txt and an RSS feed per language
- Light and dark themes, from the visitor’s system setting
- TypeScript types generated from the content model

Stack: Next.js 16 (App Router), React 19, TypeScript, Tailwind CSS 4.

## Quick start

You need Node.js 20.9 or later, and Docker to run GoHead locally.

```bash
npm create @gohead-cms@latest my-blog
cd my-blog
docker compose up -d     # GoHead on http://localhost:8080, its admin on http://localhost:3001
npm run gohead:setup     # content model, sample articles, TypeScript types
npm run dev              # http://localhost:3000
```

`npm create` copies this template, installs dependencies and writes `.env.local`. Starting from a clone of the GoHead repository instead:

```bash
cp -r gohead/templates/nextjs-blog my-blog && cd my-blog
cp .env.example .env.local
npm install
```

Sign in to the admin at <http://localhost:3001> with `admin` / `gohead-local-password` (the defaults of `docker-compose.yml` and `.env.example`; they are for your machine only, and the ports are published on localhost).

If the site can’t reach GoHead, or GoHead doesn’t have the content model yet, `npm run dev` shows a setup page that says what to run.

### Using a GoHead you already run

Set `GOHEAD_URL` in `.env.local`, and `GOHEAD_USERNAME` / `GOHEAD_PASSWORD` to one of its admin accounts, then run `npm run gohead:setup`. The setup scripts only add: they never delete types or entries, and they skip collections that already have entries.

## Scripts

| Script | What it does |
|--------|--------------|
| `npm run dev` | Development server on port 3000 |
| `npm run build`, `npm start` | Production build and server |
| `npm run lint` | ESLint |
| `npm run typecheck` | Route types (`next typegen`), then `tsc` |
| `npm run gohead:push` | Applies `gohead/model.json` to GoHead: locales, the `seo` component, collections, singletons and what anonymous visitors may read |
| `npm run gohead:seed` | Uploads `gohead/media/` and creates the entries of `gohead/seed.json` |
| `npm run gohead:types` | Writes `src/gohead.d.ts` from `gohead/model.json` |
| `npm run gohead:setup` | The three above, in order |

The `gohead:*` scripts use the `gohead-client` CLI of `@gohead-cms/client`, which reads `GOHEAD_URL`, `GOHEAD_USERNAME` and `GOHEAD_PASSWORD` from `.env.local` or `.env`. Pushing the model needs an admin account; the site itself never does.

## Environment

| Variable | Used by | |
|----------|---------|--|
| `GOHEAD_URL` | site, scripts | GoHead’s base URL, without `/api`. Default `http://localhost:8080` |
| `GOHEAD_TOKEN` | site | Optional API token, for a GoHead whose content isn’t public. Server side only |
| `GOHEAD_WEBHOOK_SECRET` | site | Signing secret of the revalidation webhook |
| `GOHEAD_REVALIDATE` | site | Seconds a GoHead read stays cached (default `60`; `false`: until a webhook says otherwise) |
| `GOHEAD_MEDIA_URL` | site | Public URL of uploads when GoHead stores them in S3 or behind a CDN |
| `NEXT_PUBLIC_SITE_URL` | site | This site’s public URL, for canonical links, Open Graph, the sitemap and RSS. On Vercel it defaults to the production domain |
| `GOHEAD_USERNAME`, `GOHEAD_PASSWORD` | scripts | An admin account, for `gohead:push` and `gohead:seed` |

Anonymous visitors can read the blog’s content types because `gohead/model.json` says so under `public`, so the site needs no token. To keep a GoHead private, remove `public` from the model, then give the site a read-only token:

```bash
npx gohead-client token create --name website --role viewer   # prints ght_…
```

and set it as `GOHEAD_TOKEN`.

## How content maps to pages

| GoHead | Page |
|--------|------|
| `site` singleton: name, tagline, description, footer, links, seo | Header, footer and default metadata of every page |
| `homepage` singleton: hero title and text, featured articles | `/en`, `/fr`: the hero, then the featured articles (the first one large) |
| `articles` collection, published entries | `/[locale]` lists them newest first, 5 per page (`?page=2`); `/[locale]/articles/[slug]` shows one |
| `categories` collection | Header navigation and `/[locale]/categories/[slug]` |
| `authors` collection | Bylines and `/[locale]/authors/[slug]` |

`/` redirects to `/en` or `/fr` from the browser’s language. Localized attributes (titles, bodies, category names…) are read in the page’s language and fall back to English when they have no translation. Slugs are shared between languages, which is what lets the language switcher keep the current page.

Article bodies are Markdown (GitHub flavored); raw HTML in them is dropped, never rendered. An article’s metadata comes from its `seo` component (in English; other languages use the translated title and excerpt), its excerpt, and its cover or the site’s share image.

All reads live in [`src/lib/gohead.ts`](src/lib/gohead.ts). Start there to add a page: the client is typed from `src/gohead.d.ts`, so `gohead.collection("articles").find({ filters: { featured: true } })` autocompletes field names and returns typed entries.

Category and author pages filter the list of all articles in Next.js, because GoHead filters don’t match on relations yet. That is fine for hundreds of articles; for more, add a scalar attribute to filter on (such as a `category_slug`).

## Publishing and revalidation

Pages are built statically and cached. Every GoHead read is tagged `gohead` and `gohead:<collection>`, and refetched after `GOHEAD_REVALIDATE` seconds (60 by default). With a webhook, a change shows up on the next visit instead:

1. In the GoHead admin, open **Settings → Webhooks** and create a webhook:
   - URL: `https://your-site.example.com/api/revalidate`
   - Events: all (or `item:published`, `item:unpublished`, `item:updated`, `item:deleted`, `singleton:updated`)
   - Collections: leave empty for all
   - Sign deliveries: on (the default)
2. Copy the signing secret GoHead shows once (`whsec_…`) into `GOHEAD_WEBHOOK_SECRET` where the site runs, and redeploy or restart.
3. Send a test from the webhook’s row: GoHead should report HTTP 200 (the site then refreshes every GoHead read).

[`src/app/api/revalidate/route.ts`](src/app/api/revalidate/route.ts) checks the signature with `parseWebhook` from the SDK (deliveries older than five minutes are refused too) and expires the reads of the collection the event is about. Articles embed their author and category, so editing an author refreshes the articles that show it.

Good to know:

- Singleton changes (`site`, `homepage`) arrive as `singleton:updated` events, for the `singleton:<name>` cache tags.
- Webhooks are delivered by GoHead’s `worker` and `dispatcher` processes, which need Redis. The provided `docker-compose.yml` runs both.
- GoHead refuses webhook URLs on private and loopback addresses unless `webhooks.allow_private_networks` is on (`GOHEAD_WEBHOOKS_ALLOW_PRIVATE_NETWORKS=true`). The provided `docker-compose.yml` turns it on, so in development you can point a webhook at `http://host.docker.internal:3000/api/revalidate`. Keep it off in production.
- `npm run dev` never caches, so revalidation is only visible with `npm run build && npm start`.

## Deploying

The site is a standard Next.js app: deploy it to Vercel, or to any host that runs Node.js (`npm run build`, then `npm start`), or as a container.

1. Make GoHead reachable from the site’s server. Covers and avatars go through the Next.js image optimizer; images inside article bodies are linked directly, so uploads must be public too (served by GoHead, or from S3/a CDN with `GOHEAD_MEDIA_URL`).
2. Set `GOHEAD_URL`, `GOHEAD_WEBHOOK_SECRET`, `NEXT_PUBLIC_SITE_URL` and, if needed, `GOHEAD_TOKEN`. Leave out `GOHEAD_USERNAME` and `GOHEAD_PASSWORD`.
3. Build with GoHead reachable, so articles are prerendered. A build that can’t reach it still succeeds (it logs which pages it skipped) and renders them on their first visit instead.
4. Create the revalidation webhook pointing at the deployed site.

`next.config.ts` allows `next/image` to load files from `GOHEAD_URL` and `GOHEAD_MEDIA_URL`, which are read at build time: rebuild after changing them. When GoHead runs on a private address (localhost, a Docker service name), image optimization is allowed for it.

When GoHead is down, pages already rendered keep being served from the cache; a page never rendered before shows an error page and the server log says why. In development you get the setup page instead.

## Changing the content model

The model is code: [`gohead/model.json`](gohead/model.json). After editing it, or after changing types in the admin and pulling them with `npx gohead-client model pull`:

```bash
npm run gohead:push    # apply it to GoHead (npm run gohead:push -- --dry-run to preview)
npm run gohead:types   # regenerate src/gohead.d.ts
npm run typecheck      # see what the change broke
```

`gohead:push` creates and updates, but never deletes a type, and refuses to remove attributes unless you run `npx gohead-client model push gohead/model.json --allow-removals`. Commit `src/gohead.d.ts` with the model, so builds don’t need GoHead to type-check.

If you add a locale, add it to `locales` in [`src/i18n.ts`](src/i18n.ts) too, with its interface text.

## Customizing

- **Design**: colors and fonts are tokens at the top of [`src/app/globals.css`](src/app/globals.css); fonts are in `src/fonts` (Schibsted Grotesk and Literata, SIL Open Font License), loaded with `next/font/local` so builds work offline.
- **Interface text**: [`src/i18n.ts`](src/i18n.ts).
- **Components**: `src/components` (article list and cards, header, footer, Markdown, setup page).

## Project structure

```
gohead/
  model.json             content model (npm run gohead:push)
  seed.json, media/      sample content (npm run gohead:seed)
src/
  app/[locale]/          pages, per language
  app/api/revalidate/    webhook endpoint
  app/sitemap.ts, robots.ts
  components/
  fonts/
  lib/gohead.ts          GoHead client and reads
  gohead.d.ts            generated content types
  i18n.ts                languages and interface text
  proxy.ts               redirects / to a language
docker-compose.yml       GoHead for local development
```
