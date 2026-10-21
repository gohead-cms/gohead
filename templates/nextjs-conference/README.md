# GoHead + Next.js conference starter

A bilingual conference site, **GoHead Conf**, built with [Next.js](https://nextjs.org) on top of [GoHead](https://github.com/gohead-cms/gohead). It is a working site and a guide at the same time: every GoHead feature has a job on it, and the [feature tour](https://gohead-cms.github.io/gohead/frontend/feature-tour/) walks you through each one, with the file that uses it and the steps to try it.

- **Content model as code**: every attribute type, components inside components, repeatable components, one-to-one and many-to-many relations, three singletons, draft and publish, English and French
- **Reads**: filters, sorting and pagination (the schedule by day, level and format), search, GraphQL (sponsors), a calendar file (`.ics`), RSS
- **Writes from visitors**: questions under talks, a call for papers with a PDF upload, newsletter sign-ups, through an API token limited to a custom role
- **Draft mode**: editors preview unpublished talks and news on the real site
- **Fresh pages**: a signed webhook expires exactly what changed
- **Seven agents**: the four templates, a hand-written one with LLM helper functions, one started by a webhook, one on a schedule; translations wait for an editor's approval
- **History**: every change is a revision, an agent's run can be undone

Stack: Next.js 16 (App Router, Server Actions), React 19, TypeScript, Tailwind CSS 4, `@gohead-cms/client`.

## Quick start

You need Node.js 20.9 or later, and Docker to run GoHead locally.

```bash
npm create @gohead-cms@latest my-conf -- --template nextjs-conference
cd my-conf
docker compose up -d     # GoHead on http://localhost:8080, its admin on http://localhost:3001
npm run gohead:setup     # model, sample programme, types, roles, tokens, webhooks
npm run dev              # http://localhost:3000
```

Sign in to the admin at <http://localhost:3001> with `admin` / `gohead-local-password` (the defaults of `docker-compose.yml` and `.env.example`; they are for your machine only, and the ports are published on localhost).

Starting from a clone of the GoHead repository instead:

```bash
cp -r gohead/templates/nextjs-conference my-conf && cd my-conf
cp .env.example .env.local
npm install
```

### Agents

Agents need a language model. Put its settings in a `.env` file next to `docker-compose.yml`, restart GoHead, and create the agents:

```bash
cat > .env <<'EOF'
GOHEAD_LLM_PROVIDER=openai
GOHEAD_LLM_MODEL=gpt-4o-mini
GOHEAD_LLM_API_KEY=sk-...
EOF
docker compose up -d
npm run gohead:agents    # then restart npm run dev (it reads the feedback agent's token)
```

`anthropic` and `ollama` work too (for Ollama, set `GOHEAD_LLM_BASE_URL`). Every agent has run limits; try one with a **Test run** in the admin before relying on it.

## Scripts

| Script | What it does |
|--------|--------------|
| `npm run dev` | Development server on port 3000 |
| `npm run build`, `npm start` | Production build and server |
| `npm run lint`, `npm run typecheck` | ESLint; route types and `tsc` |
| `npm run gohead:push` | Applies `gohead/model.json`: locales, components, collections, singletons, public read access |
| `npm run gohead:seed` | Uploads `gohead/media/` and creates the entries of `gohead/seed.json` |
| `npm run gohead:types` | Writes `src/gohead.d.ts` from the model |
| `npm run gohead:access` | Roles `site-forms`, `site-preview`, `program-committee`; the site's two API tokens; a `committee` user; the revalidation webhook and a Slack example; the draft mode secret ([`scripts/setup-access.mjs`](scripts/setup-access.mjs)) |
| `npm run gohead:agents` | The seven agents ([`scripts/setup-agents.mjs`](scripts/setup-agents.mjs)) |
| `npm run gohead:setup` | `push`, `seed`, `types` and `access`, in order |

The scripts sign in with `GOHEAD_USERNAME` and `GOHEAD_PASSWORD` (an admin account), write the tokens and secrets they create to `.env.local`, and can run again: they update roles, and keep tokens, webhooks, users, agents and entries that already exist. Restart `npm run dev` after running them.

## Environment

| Variable | Used by | |
|----------|---------|--|
| `GOHEAD_URL` | site, scripts | GoHead's base URL, without `/api`. Default `http://localhost:8080` |
| `GOHEAD_FORMS_TOKEN` | site | API token of the `site-forms` role: the forms write with it |
| `GOHEAD_PREVIEW_TOKEN`, `GOHEAD_PREVIEW_SECRET` | site | Draft mode: a token that sees drafts, and the secret of `/api/preview` |
| `GOHEAD_WEBHOOK_SECRET` | site | Signing secret of the revalidation webhook |
| `GOHEAD_FEEDBACK_AGENT_ID`, `GOHEAD_FEEDBACK_AGENT_TOKEN` | site | The `feedback-desk` agent's inbound webhook |
| `GOHEAD_TOKEN` | site | Optional read token, for a GoHead whose content isn't public |
| `GOHEAD_REVALIDATE` | site | Seconds a read stays cached (default `60`; `false`: until a webhook says otherwise) |
| `GOHEAD_PUBLIC_URL`, `GOHEAD_MEDIA_URL` | site | Public addresses of GoHead and of its files, when they differ from `GOHEAD_URL` |
| `NEXT_PUBLIC_SITE_URL` | site | This site's public URL (canonical links, sitemap, RSS, calendar) |
| `GOHEAD_USERNAME`, `GOHEAD_PASSWORD` | scripts | An admin account. Don't set them where the site is deployed |
| `GOHEAD_REVALIDATE_URL` | scripts | The site's `/api/revalidate` as GoHead reaches it (default `http://host.docker.internal:3000/api/revalidate`) |

All tokens stay on the site's server: pages and Server Actions use them, the browser never sees them.

## How content maps to pages

| GoHead | Page |
|--------|------|
| `site` singleton | Header (name, dates, navigation), footer (venue, social links), default metadata |
| `homepage` singleton | `/en`: hero, featured talks and speakers, FAQ |
| `tickets` singleton | `/en/tickets` |
| `talks` (draft and publish) | `/en/schedule` (filters), `/en/talks/<slug>`, `/en/schedule.ics` |
| `speakers`, `tracks` | `/en/speakers`, `/en/speakers/<slug>`, track chips and the track filter |
| `posts` (draft and publish) | `/en/news` (paginated), `/en/news/<slug>`, `/en/feed.xml` |
| `sponsors` | `/en/sponsors` (GraphQL), logos on the home page |
| `pages` | `/en/<slug>`: code of conduct, practical information |
| `questions` (draft and publish) | Under each talk once published; created by the form as drafts |
| `proposals`, `subscribers`, `feedback` | Private: written by the forms and agents, read in the admin |

`/` redirects to `/en` or `/fr` from the browser's language. All reads are in [`src/lib/gohead.ts`](src/lib/gohead.ts), all writes in [`src/lib/forms.ts`](src/lib/forms.ts) and [`src/app/actions.ts`](src/app/actions.ts).

## Changing the content model

The model is [`gohead/model.json`](gohead/model.json). After editing it, or after changing types in the admin and pulling them with `npx gohead-client model pull`:

```bash
npm run gohead:push    # apply it (npm run gohead:push -- --dry-run to preview)
npm run gohead:types   # regenerate src/gohead.d.ts
npm run typecheck      # see what the change broke
```

`gohead:push` creates and updates, but never deletes a type, and refuses to remove attributes unless you pass `--allow-removals`. Commit `src/gohead.d.ts` with the model.

## Deploying

A standard Next.js app: Vercel, any Node.js host (`npm run build`, `npm start`) or a container.

1. Make GoHead reachable from the site's server, and its uploads public.
2. Set `GOHEAD_URL`, `NEXT_PUBLIC_SITE_URL`, the tokens and secrets of `.env.local` (`GOHEAD_FORMS_TOKEN`, `GOHEAD_PREVIEW_TOKEN`, `GOHEAD_PREVIEW_SECRET`, `GOHEAD_WEBHOOK_SECRET`, the feedback agent's), not `GOHEAD_USERNAME` / `GOHEAD_PASSWORD`.
3. Point the `website revalidation` webhook at `https://your-site/api/revalidate` (Settings → Webhooks).
4. Turn off `GOHEAD_WEBHOOKS_ALLOW_PRIVATE_NETWORKS` on GoHead: `docker-compose.yml` turns it on for local development only.

Forms are protected by validation, length limits and a hidden honeypot field; put a rate limit or a CAPTCHA in front of them before a real call for papers.

## Project structure

```
gohead/
  model.json             content model (npm run gohead:push)
  seed.json, media/      sample programme (npm run gohead:seed)
scripts/                 roles, tokens, webhooks, agents (npm run gohead:access, gohead:agents)
src/
  app/[locale]/          pages, per language
  app/actions.ts         Server Actions of the forms
  app/api/preview/       draft mode
  app/api/revalidate/    webhook endpoint
  components/            header, footer, talks, speakers, forms…
  lib/gohead.ts          GoHead clients and every read
  lib/forms.ts           every write
  gohead.d.ts            generated content types
  i18n.ts                languages, interface text, date formats
  proxy.ts               redirects / to a language
docker-compose.yml       GoHead (API, admin, worker, dispatcher, scheduler) for local development
```

Fonts (Schibsted Grotesk, Literata) are under the SIL Open Font License. The sample people, companies and talks are made up; their pictures are generated.
