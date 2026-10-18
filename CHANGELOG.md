# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

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