# Feature tour

**GoHead Conf** is a bilingual conference site built on GoHead and Next.js, and this page is
its guide. Every GoHead feature has a real job on the site: the call for papers uploads a PDF
through a scoped API token, the schedule is filtered by GoHead, an agent sorts feedback sent
to its webhook, and translations wait for an editor's approval. Each section below says what
the feature is, which file uses it, and how to try it.

The source is in
[`templates/nextjs-conference`](https://github.com/gohead-cms/gohead/tree/master/templates/nextjs-conference).
For a smaller starting point, see the [blog starter](nextjs.md).

## Run it

You need Node.js 20.9+ and Docker.

```bash
npm create @gohead-cms@latest my-conf -- --template nextjs-conference
cd my-conf
docker compose up -d     # GoHead on :8080, its admin UI on :3001 (admin / gohead-local-password)
npm run gohead:setup     # model, sample programme, types, roles, tokens, webhooks
npm run dev              # http://localhost:3000
```

The agents need a language model, so they are a separate step. Put the model's settings in a
`.env` file next to `docker-compose.yml`, then:

```bash
docker compose up -d     # picks up GOHEAD_LLM_PROVIDER, GOHEAD_LLM_MODEL, GOHEAD_LLM_API_KEY
npm run gohead:agents
```

Restart `npm run dev` after each setup script: they write tokens and secrets to `.env.local`.

| Feature | Where it is used | Section |
|---------|------------------|---------|
| Content model as code, every attribute type | `gohead/model.json` | [Content model](#content-model) |
| Localization | everything, `/en` and `/fr` | [Two languages](#two-languages) |
| Filters, sorting, pagination, counts | schedule, news, home | [Reading content](#reading-content) |
| GraphQL | sponsors | [GraphQL](#graphql) |
| Public access, API tokens, custom roles | the site's three clients | [Who may do what](#who-may-do-what) |
| Writes, uploads, validation | the four forms | [Visitor forms](#visitor-forms) |
| Draft and publish | talks, posts, questions | [Drafts and preview](#drafts-and-preview) |
| Outbound webhooks | revalidation, Slack | [Fresh pages](#fresh-pages) |
| Agents: templates, triggers, functions, limits, approvals | seven agents | [Agents](#agents) |
| Content history, restore, revert a run | any entry | [History](#history) |

## Content model

The whole model is one file, [`gohead/model.json`](https://github.com/gohead-cms/gohead/tree/master/templates/nextjs-conference/gohead/model.json),
applied by `npm run gohead:push` ([models as code](model-as-code.md)). It uses every
attribute type:

| Type | Attribute | Shown as |
|------|-----------|----------|
| `string`, `text`, `richtext` | `talks.title`, `talks.summary`, `talks.abstract` | title, "In short" box, Markdown |
| `uid` (unique) | every `slug` | URLs |
| `email` (unique) | `subscribers.email`, `proposals.speaker_email` | validated by GoHead |
| `int` with `min`/`max` | `talks.duration_minutes` (5–480), `proposals.review_score` (0–10) | "45 min" |
| `float` | `ticket_tier.price` | "€249" |
| `bool` | `speakers.featured`, `ticket_tier.sold_out` | featured speakers first, "Sold out" |
| `date`, `datetime`, `time` | `site.starts_on`, `talks.starts_at`, `venue.doors_open` | dates, times in the venue's time zone |
| `enum` | `talks.format`, `level`, `room`, `sponsors.tier` | labels, filters |
| `json` | `talks.tags`, `ticket_tier.perks`, `site.social` | tags, perks |
| `media` | `speakers.photo`, `talks.slides` (PDF only), `talks.gallery` (`multiple`, at most 6) | `next/image`, downloads |
| `relation` | `talks.track` (one-to-one), `talks.speakers` (many-to-many) | track chips, speaker lists |
| `component` | `seo`, `venue`; `faq_item` and `link` lists (`repeatable`); `cta` holds two `link` components | metadata, FAQ, hero buttons |
| `string` with `pattern` | `tracks.color` (`^#[0-9a-fA-F]{6}$`) | track colors |

It has three singletons (`site`, `homepage`, `tickets`), three draft-and-publish collections
(`talks`, `posts`, `questions`), and three private ones that only the forms and agents write
to (`proposals`, `subscribers`, `feedback`).

`npm run gohead:types` turns the model into [`src/gohead.d.ts`](sdk.md#types): the client
then checks field names in filters and sorts, and a renamed attribute breaks `npm run
typecheck` instead of the site.

**Try it:** add `"hashtag": { "type": "string" }` to `talks`, run `npm run gohead:push --
--dry-run`, then without `--dry-run`, then `npm run gohead:types`. The new field shows in the
admin's editor and in your editor's autocompletion.

## Two languages

`locales` in the model declares English (default) and French. Attributes marked
`"localized": true` have a value per language; the others are shared. Whole components can be
localized: `site.venue`, `site.nav` and the ticket tiers differ by language, `talks.seo`
doesn't.

Every read passes the page's `locale`; untranslated values fall back to English. Slugs are
shared, so the language switcher keeps you on the same page. `gohead/seed.json` writes the
French values with `$translations`. See [localization](../guides/localization.md).

**Try it:** in the admin, open a talk, switch the editor to Français and change its title;
`/fr/talks/<slug>` shows it, `/en/…` doesn't change.

## Reading content

All reads are in [`src/lib/gohead.ts`](https://github.com/gohead-cms/gohead/tree/master/templates/nextjs-conference/src/lib/gohead.ts),
with the [JavaScript client](sdk.md). The site reads anonymously: `public` in the model gives
the public role `read` on the types the site shows.

| Page | Read |
|------|------|
| `/en/schedule?day=2027-03-12&level=beginner&format=talk` | `talks.findAll` with `starts_at: { $gte, $lt }` (the day in the venue's time zone), `level` and `format` filters, `sort: ["starts_at", "room"]` |
| `/en/news?page=2` | `posts.find` with `page`, `pageSize: 6`, `sort: "publishedAt:desc"`; the pager uses `meta.pagination` |
| `/en` | `homepage` with `level: 2` (featured talks with their speakers), and `count()` of talks, speakers, tracks |
| `/en/talks/<slug>` | `findFirst({ filters: { slug } })`; published questions filtered on `talk_slug` |
| `/en/search?q=agents` | `$contains` on talk and post titles and speaker names |
| `/en/schedule.ics` | the programme as a calendar, from `starts_at` and `duration_minutes` |

Two limits of filters show up here, with the usual workaround:

- **Relations can't be filtered.** The track filter and a speaker's talks filter in Next.js;
  questions store a plain copy of their talk's slug (`talk_slug`) so GoHead can filter them.
- **`$contains` is a SQL `LIKE`**: case-insensitive on SQLite and MySQL, case-sensitive on
  Postgres. Search also tries the lowercase and capitalized query.

Every read is cached by Next.js and tagged `gohead:<type>`, so a webhook can expire exactly
what changed ([fresh pages](#fresh-pages)).

## GraphQL

`/en/sponsors` reads with one [GraphQL](../guides/graphql.md) query instead of REST:

```graphql
query Sponsors($locale: String) {
  sponsors(limit: 100, locale: $locale) {
    id name slug tier website blurb rank
    logo { url width height alt }
  }
}
```

Anonymous GraphQL queries can read public types (introspection needs a token). Singletons
aren't in GraphQL, so the site reads them over REST.

## Who may do what

The site talks to GoHead with three identities, all on its server:

| Client | Identity | May |
|--------|----------|-----|
| `gohead` | no token: the **public role** | read published talks, speakers, posts… (`public` in the model) |
| `forms` | API token of the custom role **`site-forms`** | create questions, proposals, subscribers; upload files |
| `preview` | API token of the custom role **`site-preview`** | read drafts of talks and posts (it may `update` them, which is what lets a role see drafts) |

`npm run gohead:access` ([`scripts/setup-access.mjs`](https://github.com/gohead-cms/gohead/tree/master/templates/nextjs-conference/scripts/setup-access.mjs))
creates the two roles and their [API tokens](../guides/integrations.md#api-tokens) with the
admin API, and a third role for people: **`program-committee`** reviews proposals, drafts
talks but can't publish them (no `publish` permission), and moderates questions. It also
creates a `committee` user with that role and prints its password once.

Every action implies `read` of its type: the forms token can list proposals and subscribers,
which is why it never leaves the server. It gets `403` on everything else.

**Try it:** sign in to the admin as `committee`. Proposals and questions are editable, a talk
you create stays a draft, and the content model, agents and settings stay admin-only.

```bash
curl -s -o /dev/null -w '%{http_code}\n' localhost:8080/api/collections/proposals                 # 401: not public
curl -s -o /dev/null -w '%{http_code}\n' localhost:8080/api/collections/talks -H "Authorization: Bearer $GOHEAD_FORMS_TOKEN"  # 403
```

## Visitor forms

The public role never writes. The four forms are [Server Actions](https://nextjs.org/docs/app/guides/forms)
([`src/app/actions.ts`](https://github.com/gohead-cms/gohead/tree/master/templates/nextjs-conference/src/app/actions.ts))
that validate the input, then write with the forms token
([`src/lib/forms.ts`](https://github.com/gohead-cms/gohead/tree/master/templates/nextjs-conference/src/lib/forms.ts)).
They work without JavaScript.

| Form | GoHead |
|------|--------|
| Ask a question (talk pages) | `questions.create({ talk, talk_slug, name, body })`: a **draft**, invisible until published |
| Call for papers (`/en/cfp`) | `media.upload(pdf)`, then `proposals.create({ …, attachment: file.id })`. GoHead checks the file's real type and the attribute's `allowed_types` |
| Newsletter (footer) | `subscribers.create({ email })`. `email` is unique: a second sign-up fails validation, and the site answers the same either way |
| Feedback (`/en/feedback`) | not a write: `POST /agents/webhook/<id>` starts the `feedback-desk` agent ([below](#agents)) |

Errors come back as a `GoHeadError` (`isValidation`, `status`); the site logs them and shows
a generic message.

**Try it:** send a proposal with a PDF, then find it under **Content → proposals**, with the
file in the media library. A text file renamed `.pdf` is refused: the site checks the file's
first bytes, and GoHead, which detects types from the content too, would store it as
`text/plain` and refuse it as the attachment of a proposal.

## Drafts and preview

`talks`, `posts` and `questions` have **draft and publish**: readers, including anonymous
ones, only ever get published entries. The seed has a draft talk (`surprise-act`) and a draft
post (`workshops-move`).

[`/api/preview`](https://github.com/gohead-cms/gohead/tree/master/templates/nextjs-conference/src/app/api/preview/route.ts)
turns on Next.js draft mode after checking `GOHEAD_PREVIEW_SECRET` and that the entry exists.
In draft mode, reads switch to the preview token with `status: "all"`, skip every cache, and
a banner offers to leave:

```
http://localhost:3000/api/preview?secret=<GOHEAD_PREVIEW_SECRET>&type=talks&slug=surprise-act
```

**Try it:** open that URL: the talk shows with a **Draft** badge, and it is in the schedule.
Exit draft mode: `/en/talks/surprise-act` is a 404 again. Publish it in the admin, and it
appears for everyone.

## Fresh pages

Talk, speaker, news and page routes are prerendered at build time and cached. `npm run
gohead:access` creates a [webhook](../guides/integrations.md#outbound-webhooks) **website
revalidation** that posts content events to
[`/api/revalidate`](https://github.com/gohead-cms/gohead/tree/master/templates/nextjs-conference/src/app/api/revalidate/route.ts).
The route checks the signature with `parseWebhook` and expires the reads tagged with the
event's type (`gohead:talks`, `gohead:singleton:site`…).

Changes agents make send no events, so a summary written by `talk-summarizer` shows after
`GOHEAD_REVALIDATE` seconds (60 by default), not at once.

It also creates **slack: new talks**, disabled: `"format": "slack"` posts a one-line message
on `item:published` of `talks` to a Slack incoming webhook. Set its URL and enable it.

**Try it:** `npm run build && npm start`, change a talk's title in the admin, reload its
page: the new title is there. **Settings → Webhooks → website revalidation** lists the
deliveries; **Send test** expires every read.

In development, GoHead in Docker reaches the site at `host.docker.internal`; the template's
`docker-compose.yml` sets `GOHEAD_WEBHOOKS_ALLOW_PRIVATE_NETWORKS` for that. Leave it off in
production.

## Agents

`npm run gohead:agents` ([`scripts/setup-agents.mjs`](https://github.com/gohead-cms/gohead/tree/master/templates/nextjs-conference/scripts/setup-agents.mjs))
creates seven [agents](../agents/index.md), covering every trigger and the main function
families:

| Agent | Trigger | Does | Shows |
|-------|---------|------|-------|
| `talk-summarizer` | `talks` created or updated | writes `summary` (the "In short" box) | [template](../agents/templates.md#summarizer) |
| `talk-tagger` | `talks` created or updated | picks `tags` from a fixed list | [template](../agents/templates.md#tagger) |
| `question-moderator` | `questions` created or updated | sets `flagged` and `flag_reason` on spam | [template](../agents/templates.md#moderator) |
| `post-translator` | `posts` created | translates title, excerpt and body into French, **as proposals** | [template](../agents/templates.md#translator), [approvals](../agents/approvals.md) |
| `proposal-triage` | `proposals` created | scores it with `llm.score_quality`, saves `review_score` and `review_notes` | [LLM helpers](../agents/functions.md#llm-helper-functions), [scopes](../agents/functions.md#limiting-functions-to-collections), [limits](../agents/runs-and-limits.md#limits) |
| `feedback-desk` | **webhook** from the site | `llm.analyze_sentiment`, `llm.classify_text`, then stores the message in `feedback` | [webhook trigger](../agents/triggers.md#webhook) |
| `weekly-digest` | **cron** `0 7 * * 1` | lists talks, drafts a news post for an editor | [cron trigger](../agents/triggers.md#cron) |

Every writing function is scoped to one collection (`"collections": ["proposals"]`): GoHead
refuses anything else, whatever the model says. Agents are not triggered by their own
writes, and writes made by agents send no events, so they never trigger each other.

**Try it:**

- **Test run.** Agents → `talk-summarizer` → Test run, pick a talk: the run shows what it
  would write (`would_update`) and changes nothing.
- **Approvals.** Create a news post in the admin. A minute later **Approvals** has a French
  translation to approve, edit or reject; once approved, `/fr/news/<slug>` shows it.
- **Webhook trigger.** Send the feedback form, then open **Content → feedback**: the message
  is there with its sentiment and topic. The agent's **Runs** tab shows each step.
- **Cron.** Agents → `weekly-digest` → Run now: a draft post appears under News in the admin.
- **Limits.** Each agent has run limits (`max_runs_per_hour`…); a run over them is recorded as
  `limited`, not run.

## History

Every change is a [revision](../guides/content-history.md), by whom (a person, an API token,
an agent and its run).

**Try it:**

- Open a talk, then **History** (the clock icon): pick an older version to compare it with
  the current one and **Restore this version**.
- Delete a sponsor, then **Recently deleted** in the sponsors list brings it back with the
  same ID.
- Open a `weekly-digest` run and **Revert** it: the post it created is deleted (and can be
  restored from its history).
- **Activity → Content changes** filters every change by agents, tokens or people.

## What it doesn't cover

- **One process:** `docker-compose.yml` runs the API, worker, dispatcher and scheduler
  separately, with Postgres and Redis. `gohead standalone` runs them all in one process on
  SQLite ([standalone](../operations/standalone.md)); the site works the same with it.
- **Dynamic zones** (lists of different components) aren't supported: the home page uses
  fixed sections.
- **Spam protection** is a honeypot field and length limits. Add rate limiting or a CAPTCHA
  before opening a real call for papers.
