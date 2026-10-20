# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- **Public access**: requests without a token act with the new `public` role. It grants nothing by default; give it `read` on the collections, singletons and media your site shows (`PUT /admin/roles/public`, or **Settings → Roles → Public**) and browsers and apps read published content without a token. It never writes or sees drafts; the content model, GraphQL introspection and the admin area still need a token. Anonymous requests are rate limited per IP (`rate_limit.public_per_minute`, 600).
- **Agent approvals**: an agent with `require_approval` proposes its content changes instead of making them. Editors review proposals in **Approvals** (field diff, approve, edit and approve, reject) or with `/api/proposals`; whoever may make a change may approve it. Approved changes are applied as the agent in its run (history, run revert), and stale proposals are flagged before they overwrite a later edit. Migration `0004_agent_reviews`.
- **Test runs**: `POST /admin/agents/{name}/test` and a **Test run** button run an agent with a message or a simulated collection event on an existing item; content writes are validated and reported (`would_update`…) but not applied. Runs started from the UI are recorded as `queued` right away and the response carries the run ID.
- **Agent templates**: summarizer, tagger, moderator and translator, rendered from a few parameters checked against the content model (`/admin/agent-templates`), in the admin UI under **New agent**.
- `collections.patch_item` agent function (changes some attributes, keeps the others) and a `reason` argument on content writes.

### Changed
- An agent is no longer triggered by content events caused by its own changes.

## [0.5.0] - 2026-10-19

### Breaking changes
- **Self-registration is off by default** (`auth.registration: disabled`): `POST /auth/register` returns `403` unless set to `open`, and it never creates an admin, even on an empty database. See the [upgrade guide](https://gohead-cms.github.io/gohead/operations/upgrading/).
- **The first administrator** is created from `admin.username` / `admin.email` / `admin.password` (`GOHEAD_ADMIN_*`, password 12+ characters) when the API starts on a database without users, or with `gohead admin create`.

### Added
- **Custom roles with per-type permissions**: besides `admin`, `editor` and `viewer`, roles grant `read`, `create`, `update`, `delete` and `publish` per collection (`singleton:<name>` for singletons, `*` for every other type) and `read`/`create`/`update`/`delete` on the media library. They're managed under `/admin/roles` and in **Settings → Roles**, and assigned to users and API tokens like built-in roles. The REST API, GraphQL, the schema endpoints, populated relations, content history and the change feed all enforce them. `GET /auth/me` returns the caller's permissions.
- **Localization**: locales (`/api/locales`, managed under `/admin/locales`, default from `i18n.default_locale`) and attributes marked `"localized": true`, which keep a value per locale while the rest of the entry is shared. Read any entry, list or singleton with `?locale=` (falling back to the default locale unless `fallback=false`; filters and sorts use the locale), and write translations with `PUT ?locale=`. Unique localized values are unique per locale. GraphQL takes `locale` and `fallback`, and update mutations take `locale`. Translations are part of the history, webhook payloads (`translations`) and agent functions (`collections.upsert_item` with `locale`, `collections.list_locales`). Migration `0003_i18n`. In the admin UI: Settings → Locales, a Localized switch per attribute, and a locale switcher in the entry editor.
- **Content history**: every change to an item or a singleton (create, update, publish, unpublish, delete, restore) is recorded as a revision with the data, who made it (user, API token or agent with its run) and when, in the same transaction (migration `0002_revisions`). Compare versions field by field, restore any version (including deleted items, under the same ID), browse a change feed (`GET /api/revisions`), and revert everything an agent run changed (`POST /admin/agents/{name}/runs/{run}/revert`, skipping entries changed since unless forced). `content.revisions_kept` (100) per entry are kept. In the admin UI: history with diffs and restore on every entry, recently deleted entries, content changes in Activity, and run revert.
- **Versioned database migrations** recorded in `schema_migrations` and serialized by a database lock, so replicas starting together no longer race: `gohead migrate`, `migrate status`, `migrate down`. The API applies them at startup unless `auto_migrate` is false (then it refuses to start on an outdated schema); the Helm chart runs them in an init container.
- `gohead admin create` and `gohead admin set-password` (passwords from standard input only).
- **API contract**: `api/openapi.json` and `api/graphql.json` snapshot the public API; CI fails when they drift from the code and reports breaking changes against the last release (oasdiff, GraphQL Inspector). See *API stability*.
- Docs: upgrade guide with version notes, API stability policy, release checklist. Image builds warn when a package cannot be pulled anonymously.
- Helm: `bootstrapAdmin` values and a `migrate` init container (`migrations.initContainer`).
- **A complete admin UI**: a real dashboard; a content model editor for collections, components and single types (every attribute type and option); the schema designer shows the saved model; content editing for single types; a media library (drag-and-drop upload, metadata, search); agents can be created, edited, deleted and run from the UI, with a function catalog and collection scopes; an activity page across agents; users, API tokens and webhooks (with test sends and delivery logs) under Settings. Navigation follows the signed-in user's role, and an expired session signs out.
- API: `GET /auth/me`; `GET /admin/overview` (dashboard counts and recent activity); `GET /admin/runs` (runs of every agent); `GET /admin/agent-functions` (built-in functions with descriptions and argument schemas); `POST /admin/agents/{name}/run` (run an agent now); `GET /api/schema/collections|singletons|components[/{name}]`, the content model read-only for every signed-in user.
- **API tokens**: long-lived, role-scoped (`viewer`, `editor`, `admin`), optionally expiring and revocable keys (`ght_…`) for front-ends, CI and integrations, managed under `/admin/tokens`. Only a SHA-256 hash is stored; last use is recorded. Tokens cannot manage tokens or users.
- **OpenAPI 3.1** description generated from the content model (collections, components, singletons, media, auth, GraphQL) at `/api/openapi.json`, with Swagger UI at `/api/docs`. Behind a token unless `openapi.public` is on.
- **Outbound webhooks** on content events (`item:created|updated|deleted|published|unpublished`), filtered by event and collection, in JSON or Slack format, with custom headers, HMAC-SHA256 signatures, retries with backoff, a delivery log and a test endpoint (`/admin/webhooks`). Private and link-local addresses are refused by default.
- **Agent run history**: every run records its trigger, input, each model and function call (arguments, results, errors, tokens, duration), output, status (`succeeded`, `failed`, `limited`), tokens and cost. Runs are saved as they progress, the newest `agents.runs_kept` (200) per agent are kept, and they are available at `/admin/agents/{name}/runs`, `/runs/{id}` and `/usage`.
- Admin UI: an agent page (**AI Management → Agents**) with an overview (model, prices, trigger, limits, function scopes) and a **Runs** tab: usage against limits, a filterable run list and a step-by-step view of each run.
- **Collection-scoped functions**: `collections.*` functions accept `"collections": [...]`. Calls outside the scope are refused by the server; `collections.delete_item` checks the collection the item actually belongs to, and `collections.list` only shows collections in scope.
- **Per-agent limits**: `max_runs_per_hour`, `max_runs_per_day`, `max_tokens_per_run`, `max_tokens_per_day` and `max_cost_per_day`, over rolling windows, shared by all workers. Runs over a limit are skipped or stopped, recorded as `limited` and not retried.
- Token usage is read from OpenAI-compatible, Anthropic and Ollama responses, including calls made by `llm.*` helper functions. Costs use `input_cost_per_million` / `output_cost_per_million` set in an agent's `llm_config` or in the `llm` defaults (no built-in price list).
- **Helm chart** (`oci://ghcr.io/gohead-cms/charts/gohead`, released with every version): API with ingress, autoscaling, disruption budget and ServiceMonitor, worker, dispatcher, single scheduler, admin UI, optional bundled Redis; secrets from values or an existing Secret.
- **MySQL support is tested**: CI runs the database test suite on MySQL 8.4 as well as PostgreSQL 17 (and SQLite).
- Playwright smoke test of the admin UI (login, schema designer, content creation, agents) against the real API in CI.

### Changed
- Publishing and unpublishing require the new `publish` permission instead of `update`, and so does changing a published item in a draft-and-publish collection. Built-in editors and admins have both, so nothing changes for them.
- A user's role is read from the account on each request, so changing it no longer waits for the user's token to expire.
- Deleting a singleton's content removes it for good (it was soft-deleted, which blocked creating new content); its revisions keep the old content. `storage.UpdateItem`, which saved unvalidated data, was removed.
- MySQL: JSON columns are created as `JSON` (they used PostgreSQL's `JSONB`, which MySQL rejects), and unique text columns (collection, component, singleton, agent and user slug names, token hashes, media keys) have a length. On PostgreSQL these columns become `varchar(191)` (`varchar(64)` and `varchar(255)` for token hashes and media keys) instead of `text`, changed by the automatic migration at startup: it fails if an existing value is longer, so check first, e.g. `SELECT max(length(name)) FROM collections` (and `components`, `singletons`, `agents`, `users.slug`).
- Use `loc=UTC` in MySQL URLs (the examples did not), so times are stored in UTC like on the other databases.
- The admin UI lint (ESLint, React hooks rules) is clean and blocks CI.
- The JWT secret that `docker-compose.yml` falls back to is treated as unset (a random secret is generated and a warning logged), and the quick start generates the secret and the admin password.
- `/admin/users` requires a user session (not an API token).
- Usernames cannot contain `:` (reserved for `token:<name>` attribution).
- Deleting an agent deletes its run history and conversation.

### Fixed
- `GET /api/singleton/{name}?status=all` lets editors and admins see links to draft entries, so saving a singleton from the admin UI no longer drops them.
- `GET /admin/agents` caps `pageSize` at 100.
- `/auth/me` is no longer rate limited together with login.
- `POST /admin/agents` no longer logs the whole request, which could include an LLM API key.
- The dashboard's recent entries only use a title, name, label or slug as the entry title, never other fields.
- Editing a collection through the API's own schema dropped `unique`, enum `options`, `min`/`max`, `pattern` and the description (they were not returned, and an update replaces the definition); relation targets returned as `api::x.x` are accepted back.
- Admins can no longer demote or delete their own account or the last admin.
- Two agent functions may use the same implementation with different scopes, as documented; renaming an agent to a taken name returns `409` instead of `500`; agent list filters only accept known fields.
- Single type admin endpoints returned objects as error messages.
- Removed admin UI pages that only showed mock data (dashboard) or saved nothing (schema designer edits, relation drawing).
- MySQL: updating a collection, component or singleton failed (`Incorrect datetime value`), looking up related items by unique fields used PostgreSQL-only SQL, and finding the agents of a collection event failed (`trigger` is a reserved word).
- Admin UI: the workspace's Edit action on a collection node opened nothing; the relation dialog sent `oneToMny` and labelled many-to-many as many-to-one; hooks were called conditionally in the sidebar and the collection list.
- Re-creating an agent with the name of a deleted one restored the old configuration instead of the new one.
- The admin UI's **Agents** link led to an empty page.

## [0.4.0] - 2026-10-18

### Breaking changes
- GraphQL: `media` attributes are now `Media` objects instead of strings. Query `cover { url }` instead of `cover`. REST still returns URL values as they were stored.

### Added
- **Media uploads**: `POST /api/media` (multipart, several files, atomic) stores files on local disk or in any S3-compatible bucket (AWS S3, MinIO, Cloudflare R2, Garage…), with a media library (`GET/PUT/DELETE /api/media/{id}`, list with `mime` and `search` filters). Types are detected from the content and checked against an allow-list (no SVG/HTML by default), sizes are limited (`media.max_size_mb`), and image dimensions, SHA-256 hash, alt text and caption are recorded.
- `media` attributes reference uploaded files by ID (`multiple`, `allowed_types`, `min`/`max`) and are returned as file objects with their URL, including inside components, singletons and related items; GraphQL exposes them as `Media` objects. Existing URL values keep working.
- Local files are served at `/uploads/<key>` with safe headers (`nosniff`, sandboxing CSP, attachments for non-media types) and range requests; S3 files use `media.public_url` (public bucket or CDN) or presigned URLs.
- `docker-compose.yml` runs MinIO with a ready-to-use public bucket; the container image stores local uploads in `/data/uploads`.
- Admin UI: upload fields for media attributes.

## [0.3.0] - 2026-10-18

### Breaking changes
- **One response shape for items.** Every item response (lists, single items, create, update, populated relations, singletons) is a flat object: the attributes at the top level next to `id`, `createdAt` and `updatedAt`. Single items, create/update responses and singletons used `{id, attributes}`, and create responses nested relations as `{data: {id}}`. Clients that read `data.attributes.x` must read `data.x`.
- Attribute names `id`, `createdAt`, `updatedAt` and `publishedAt` are reserved for new and updated collections.
- `level=0` is now accepted (no relation population); create and update responses populate relations one level deep, like reads.

### Added
- **Component values in items**: collections and singletons can use components (`"type": "component", "component": "seo"`), single or `repeatable` (with `min`/`max` list length). Values are validated field by field, can nest components, and are exposed in GraphQL as typed objects and input objects. Components that are in use cannot be deleted or renamed (`409`).
- **Draft and publish**, opt-in per collection (`"draft_and_publish": true`): items start as drafts, `status`/`publishedAt` appear in responses, `POST /api/collections/{c}/{id}/publish|unpublish` and a `"status"` field on writes change state, lists return published items unless an editor asks for `status=draft|all`, and viewers never see drafts (also in relations and GraphQL). New `item:published` / `item:unpublished` events. Enabling it on an existing collection publishes the existing items.
- GraphQL: `createdAt`/`updatedAt` on every type, `status` argument on queries, `publish<Name>`/`unpublish<Name>` mutations, list inputs for many-to-many relations.
- Sorting and filtering on `createdAt`, `updatedAt` and `publishedAt`.
- Admin UI: edits update the entry instead of creating a new one, drafts are listed with a status badge and a publish toggle.

### Fixed
- One-to-many relations were missing from single-item and create responses.
- GraphQL: `date`/`datetime` attributes are strings, as stored (they were declared `DateTime`, so values read back as `null` and plain dates were rejected).
- Updating an item with a `unique` attribute failed when the value did not change; soft-deleted items no longer block unique values; numbers are compared by value (`1000000` was not seen as a duplicate of `1e+06`), and names with dots or dashes work on MySQL/SQLite.
- REST `PUT` did not emit `item:updated`.
- Relations nested deeper than one level were not populated, and a deleted related item made the whole read fail.
- Viewers could write singleton content; singleton writes now need the `update` permission.
- Agent item updates (`collections.upsert_item` with `item_id`) are validated like API writes.
- Item list errors use the standard error envelope.

## [0.2.0] - 2026-10-18

### Security
- **Registration privilege escalation**: `POST /auth/register` no longer accepts a client-chosen role. The first account on an empty instance becomes `admin`; later ones are `viewer`.
- **GraphQL authorization bypass**: queries and mutations now enforce role permissions and item validation, like the REST API.
- **JWT**: replaced the unmaintained `dgrijalva/jwt-go` with `golang-jwt/jwt/v5`; only HS256 is accepted and `exp` is required. The sample/empty `jwt_secret` is replaced by a random per-run secret.
- **Agent webhooks**: only `webhook`-triggered agents with a token can be invoked, and the token is compared in constant time.
- **Secrets in responses**: password hashes are never serialized; agent API keys and webhook tokens are masked.
- **Per-agent LLM keys**: keys are passed to each client directly instead of through process-wide environment variables.
- Login no longer logs issued tokens.
- **Brute-force protection**: `/auth/login` and `/auth/register` are rate limited per client IP (20/min), and 5 failed logins for a username from one IP lock it out for 15 minutes (`429` + `Retry-After`). Unknown usernames take as long as known ones. Configurable under `rate_limit.*`.
- `X-Forwarded-For` is only trusted from `trusted_proxies` (previously from anyone).
- Collection list `filter`/`sort` parameters are restricted to known columns (they were interpolated into SQL).
- **Dependencies**: upgraded to fix 15 reachable vulnerabilities reported by govulncheck (golang.org/x/net, x/text, OpenTelemetry, gRPC, pgx, quic-go). Go 1.26 is now the minimum; builds use Go 1.27.

### Changed
- Telemetry is off by default.
- The Docker image entrypoint is `gohead`; pass the command (`start`, `worker`, …).

### Added
- **Multi-replica API**: collection changes are broadcast over Redis so every API replica rebuilds its GraphQL schema; replicas poll a version counter to catch up after missed messages or Redis outages. Rate-limit counters are shared through Redis too, with an in-memory fallback.
- Admin user management: `/admin/users` (create, list, get, update, delete).
- `GET /admin/components` and `GET /admin/singletons`.
- Filtering (`filters[field][$op]=value`) and multi-key sorting (`sort=field:desc`) on item lists, validated against the collection schema.
- GraphQL `variables` / `operationName`, partial update inputs, list types for to-many relations, and hot reload of the schema when collections change.
- `base_url` option for LLM providers.
- `GOHEAD_TEST_DATABASE_URL` to run the test suite against PostgreSQL.
- Graceful shutdown on SIGINT/SIGTERM.
- **Handbook** (MkDocs Material, published to GitHub Pages) and a new README.
- **Delivery pipeline**: CI (lint, race tests, PostgreSQL tests, govulncheck, UI and docs builds), multi-arch signed images on GHCR with SBOM/provenance, GoReleaser v2 releases, CodeQL, Dependabot.
- **Deployment**: Kubernetes manifests (kustomize, staging/production overlays, External Secrets) and Flux image automation; full-stack `docker-compose.yml`; admin UI image configurable at runtime (`GOHEAD_API_URL`).
- Configuration works from environment variables alone; `redis.password`/`redis.db` and the `llm` block are honoured; `gohead version` reports the build.
- The scheduler reloads cron agents every minute.

### Fixed
- Item creation leaked an open database transaction on every request; single and bulk creation are now atomic and events are published after commit.
- Bulk creation stored unvalidated data instead of the processed data.
- GraphQL list pagination passed the offset as a page number.
- Tracing was shut down as soon as the server started; spans are now exported and flushed on exit.
- Agent webhooks crashed because the queue client was never wired into the handler.
- Singleton delete and attribute updates used a non-existent column.
- Restoring a soft-deleted singleton restored another entity's attributes.
- Singleton item updates ignored the singleton's attributes during validation.
- Deleting a component left its attributes behind.
- Roles were duplicated on every startup.
- `JSONMap` could not be scanned from drivers that return strings.
- The GraphQL schema crashed on component/dynamiczone attributes and unknown relation targets.
- `POST /admin/agents` redirected because of a trailing slash.
- Relation population depth is capped (`level` ≤ 5).
- Middleware errors now use the same `{"error": {...}}` envelope as handlers.
- The test suite passes again (7 of 12 packages were failing), including under `-race`.
- Enum attributes could not be saved; components could not be created (tables not migrated).
- `float`, `decimal`, `integer`, `enumeration` and `password` values were rejected.
- Cron triggers never ran an agent; agent functions failed on string arguments from the runner.
- GraphQL relation fields resolved to `null`; relation IDs showed as `0` after nested creates.
- Agent history replayed to the LLM is bounded (last 40 messages).
- The default CORS headers did not allow `Authorization`, so the admin UI failed from another origin.
- The Docker image did not build, and the admin UI did not type-check.

## [0.1.0] - 2025-01-05
### Added
- **User Role Management**: Introduced roles (`admin`, `editor`, `viewer`) with distinct permissions.
- **API Endpoints**: Implemented `/auth/register` and `/auth/login` routes.
- **Test Utilities**: Added helper methods for mocking JSON POST requests and initializing test servers.
- **Slug Utility**: Added a utility for generating URL-friendly slugs, including support for accented characters.

### Changed
- **Refactored Role Handling**: Centralized role fetching and validation in storage layer.
- **Improved Logging**: Enhanced request and error logging with structured logs for debugging.

### Fixed
- **Database Migrations**: Resolved issues with `AutoMigrate` not applying schema changes during tests.
- **Test Failures**: Fixed inconsistencies in test cases for duplicate entries and invalid roles.

### Legend:
- **Added**: For new features.
- **Changed**: For changes in existing functionality.
- **Fixed**: For any bug fixes.
- **Deprecated**: For soon-to-be removed features.
- **Removed**: For now-removed features.