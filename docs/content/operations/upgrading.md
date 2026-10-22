# Upgrading

GoHead follows [semantic versioning](https://semver.org/). Before 1.0, minor versions
(0.4 → 0.5) can contain breaking changes; each one is listed below and in the
[changelog](../reference/changelog.md). From 1.0, breaking changes to the
[public API](../reference/api-stability.md) only come with a new major version.

## How to upgrade

1. **Back up the database** (and the uploads if they are on local disk). Database
   migrations cannot always be undone; a backup always can.
2. Read the notes for every version between yours and the target, below.
3. You can skip versions: migrations apply in order from wherever your database is.
4. Deploy the new version:

    === "Helm"

        ```bash
        helm upgrade gohead oci://ghcr.io/gohead-cms/charts/gohead --version 0.5.0 \
          --reset-then-reuse-values   # new chart defaults + your values
        ```

        The API pods' `migrate` init container applies the migrations before the new API
        starts; replicas wait for each other on a database lock.

    === "Docker Compose"

        ```bash
        echo "GOHEAD_VERSION=0.5.0" >> .env
        docker compose pull && docker compose up -d
        ```

    === "Binary"

        ```bash
        gohead migrate            # with the new binary and your configuration
        gohead start
        ```

5. Check `gohead migrate status`, the API's `/_health`, and the admin UI.

## Database migrations

The schema is versioned. Each migration runs once, in order, and is recorded in the
`schema_migrations` table. Migrations are serialized by a database lock (PostgreSQL
advisory lock, MySQL `GET_LOCK`), so several replicas or init containers can start at the
same time.

| How | When to use it |
|-----|----------------|
| `auto_migrate: true` *(default)* | The API applies pending migrations when it starts. Fine for a single instance or Compose. |
| `gohead migrate`, then the API with `auto_migrate: false` | Kubernetes and any setup where the schema should change in one controlled step. The Helm chart does this in an init container. With `auto_migrate: false`, the API refuses to start on an outdated schema. |

```bash
gohead migrate status        # what ran, what is pending
gohead migrate               # apply pending migrations
gohead migrate down          # revert the last one, when it can be undone
```

The lock is a session lock: with PgBouncer in transaction pooling mode, point
`gohead migrate` at PostgreSQL directly (or a session-mode pool), or a stuck lock can make
later runs wait for 10 minutes. On SQLite there is no lock: run migrations from one
process (the API itself, or `gohead migrate` while the API is stopped).

`0001_baseline` creates the schema on a new database, and brings a database created by an
earlier version (which migrated automatically from the models) up to date. Later
migrations are explicit, reviewed steps; contributors add them in `pkg/migrations`.

## Version notes

### 0.7.0

- **Migration `0007_connections`** adds the table of [connected apps](../agents/connected-apps.md).
  It runs when the API starts; with `auto_migrate: false`, run `gohead migrate` first.
- **Connected apps' tokens are encrypted with a key derived from `jwt_secret`.** Set
  `jwt_secret` (`GOHEAD_JWT_SECRET`) to the same value in every process (API, workers);
  without it, tokens are stored unencrypted. Changing `jwt_secret` later means entering
  the tokens again.
- **Agents in the short `tools` form.** Agents saved with `functions` keep working, and
  responses now return both `tools` and `functions`. Agents naming a tool that doesn't
  exist are refused when saved.
- **API:** proposals can have `"action": "call"` (a call to a connected app's tool,
  `"collection": "app:<connection>"`). Only admins in a user session see them: clients
  listing proposals with an editor's or an API token's credentials get none. Clients that
  check `action` against a fixed list should accept `call`.

### 0.6.3

No migration and nothing to change. Agents on Anthropic call their functions again: on
0.6.2 and earlier, a run often ended after Claude's first sentence. GoHead can also be
deployed to [Render](render.md) in one click.

### 0.6.2

No migration and nothing to change. The admin UI moves to Chakra UI v3, with the same
screens and behaviour. On a Mac, GoHead also installs with Homebrew
(`brew install gohead-cms/tap/gohead`).

### 0.6.1

A bug fix release, with no migration and nothing to change: public access works again
with rate limiting on (the default). On 0.6.0, anonymous reads of public types answered
`401`; if you turned rate limiting off to work around it, you can turn it back on.

### 0.6.0

!!! warning "Breaking changes"
    - **The admin UI is served by the API.** Binaries and the `gohead` image include it, and
      `gohead start` serves it at `/`. `GET` requests to other paths outside the API (`/api`,
      `/admin`, `/auth`, `/uploads`, `/_health`, `/_metrics`) now return the UI instead of
      `404`. To keep an API-only host (the UI served by `gohead-admin`, say), set
      `ui.enabled: false` (`GOHEAD_UI_ENABLED=false`).
    - **SQLite uses a pure-Go driver.** Existing databases open as they are (same time format)
      and are switched to WAL mode: back up the `-wal` and `-shm` files with the database, or
      use `sqlite3 gohead.db ".backup …"`. Parameters of the former driver in `sqlite://` URLs
      (`_busy_timeout`, `_foreign_keys`, `_loc`…) are ignored: use
      `?_pragma=busy_timeout(5000)`-style parameters.

Migration `0006_jobs` adds the `jobs` table of the [database job queue](standalone.md#how-the-job-queue-works-without-redis).
Nothing changes for installs on Redis: `redis` stays the default queue of `gohead start`,
`worker`, `dispatcher` and `scheduler`.

The `gohead` image is now static (`distroless/static`, no cgo); it still runs as user
`65532` with data in `/data`. An agent is no longer triggered by content events caused by
its own changes.

To move a small install to [`gohead standalone`](standalone.md): stop the API, worker,
dispatcher and scheduler, let Redis's queues drain first, then run `gohead standalone` with
the same configuration and `queue.driver: database`.

Migration `0004_agent_reviews` adds the `proposals` table and two columns (`agents.require_approval`,
`agent_runs.dry_run`). Existing agents keep writing directly. Nothing to do.

Migration `0005_public_role` makes room for [public access](../guides/users-and-roles.md#public-access):
a custom role you named `public` is renamed `public-legacy` (its users and API tokens keep
it). Public access itself is off until you grant something to the new `public` role.

### 0.5.0

!!! warning "Breaking changes"

    - **Self-registration is off by default.** `POST /auth/register` returns `403` unless
      `auth.registration` is `open`, and a registered account is never made admin, even on
      an empty database. Existing accounts are unchanged. If your front-end lets visitors
      sign up, set `GOHEAD_AUTH_REGISTRATION=open`.
    - **The first admin comes from the configuration or the CLI**: set
      `GOHEAD_ADMIN_USERNAME`, `GOHEAD_ADMIN_EMAIL` and `GOHEAD_ADMIN_PASSWORD` (12+
      characters) before the first start, or run `gohead admin create`. Only matters for new
      installs.
    - **PostgreSQL: name columns get a length.** `collections.name`, `components.name`,
      `singletons.name`, `agents.name` and `users.slug` become `varchar(191)`,
      `api_tokens.hash` `varchar(64)` and `media.key` `varchar(255)` (they were `text`). The
      migration fails if an existing value is longer; check first:

        ```sql
        SELECT 'collections', max(length(name)) FROM collections UNION ALL
        SELECT 'components',  max(length(name)) FROM components  UNION ALL
        SELECT 'singletons',  max(length(name)) FROM singletons  UNION ALL
        SELECT 'agents',      max(length(name)) FROM agents      UNION ALL
        SELECT 'users.slug',  max(length(slug)) FROM users;
        ```

    - **MySQL** users of earlier versions: GoHead did not work on MySQL before 0.5 (JSONB
      columns). Start from a new database.

!!! note "Before upgrading"

    - **Content language:** existing content is labelled with the default locale, `en`
      unless `GOHEAD_I18N_DEFAULT_LOCALE` says otherwise. If your content is in another
      language, set it before the first start of 0.5 ([localization](../guides/localization.md)).
    - **MySQL:** use `loc=UTC` in MySQL URLs.

The first start applies three migrations:

- `0001_baseline`: the schema as earlier versions created it.
- `0002_revisions`: [content history](../guides/content-history.md). Existing entries get
  their first revision on their next change.
- `0003_i18n`: [localization](../guides/localization.md).

Also new, with no action needed:

- [custom roles](../guides/users-and-roles.md#custom-roles). Built-in roles keep their rights.
  Publishing now needs the `publish` permission, which `editor` and `admin` have.
- agent run history and limits, and scoped agent functions;
- API tokens and outbound webhooks;
- the OpenAPI document;
- the complete admin UI.

### 0.4.0

!!! warning "Breaking changes"

    - GraphQL: `media` attributes are `Media` objects instead of strings: query
      `cover { url }` instead of `cover`. REST is unchanged.

### 0.3.0

!!! warning "Breaking changes"

    - Every item response is a flat object (`{"id": 1, "title": …}`). Single items,
      create/update responses and singletons used `{id, attributes}`: clients that read
      `data.attributes.x` must read `data.x`.
    - Attribute names `id`, `createdAt`, `updatedAt` and `publishedAt` are reserved.
