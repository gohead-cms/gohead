# API endpoints

Base URL: your API, e.g. `http://localhost:8080`. **Auth** column: `–` public,
`token` any authenticated user (content permissions per [role](../guides/users-and-roles.md)),
`admin` admin role, `webhook` agent token in `Webhook-Token`.

## System

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/_health` | – | Liveness: `{"status":"ok"}` |
| `GET` | `/_metrics` | – | Prometheus metrics (keep internal) |

## Authentication

| Method | Path | Auth | Body |
|--------|------|------|------|
| `POST` | `/auth/register` | – (only when `auth.registration` is `open`) | `username`, `email`, `password` → viewer account |
| `POST` | `/auth/login` | – | `username`, `password` → `{"data":{"token":…}}` |

## Content

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/collections/{collection}` | token | List items. [Filters, sort, pagination](../guides/rest-api.md#filtering-and-sorting) |
| `POST` | `/api/collections/{collection}` | token | Create one item (`{"data":…}`) or a batch (array) |
| `GET` | `/api/collections/{collection}/{id}` | token | Get an item; `?level=0..5`, `?status=` |
| `PUT` | `/api/collections/{collection}/{id}` | token | Replace an item's data (optional `"status"`) |
| `DELETE` | `/api/collections/{collection}/{id}` | token | Delete an item |
| `POST` | `/api/collections/{collection}/{id}/publish` | token (update) | Publish a draft |
| `POST` | `/api/collections/{collection}/{id}/unpublish` | token (update) | Unpublish an item |
| `GET` | `/api/collections/{collection}/{id}/revisions` | token (update) | [History](../guides/content-history.md) of an item, newest first |
| `GET` | `/api/collections/{collection}/{id}/revisions/{version}` | token (update) | One version with its data and diff; `?compare=<version>\|current` |
| `POST` | `/api/collections/{collection}/{id}/revisions/{version}/restore` | token (update) | Make a version current (brings back a deleted item) |
| `GET` | `/api/singleton/{name}/revisions` | token (update) | History of a singleton (also `/{version}` and `/{version}/restore`) |
| `GET` | `/api/locales` | token | Content [locales](../guides/localization.md), the default first. Reads take `?locale=` and `?fallback=`, and `PUT ?locale=` writes a translation |
| `GET` | `/api/revisions` | token (update) | Change feed across entries; `collection`, `singleton`, `action`, `actorKind`, `actor`, `agentRun` |
| `POST` | `/api/media` | token (create) | Upload files (`multipart/form-data`, field `files`) |
| `GET` | `/api/media` | token (create) | List the media library (`page`, `pageSize`, `mime`, `search`) |
| `GET`, `PUT`, `DELETE` | `/api/media/{id}` | token (create / update / delete) | One file; `PUT` changes `name`, `alt`, `caption` |
| `GET` | `/uploads/{key}` | public | Files of the local media driver |
| `GET` | `/api/openapi.json` | token (or public with `openapi.public`) | OpenAPI 3.1 description of the API |
| `GET` | `/api/docs` | public page | Swagger UI |
| `GET` | `/api/singleton/{name}` | token | Read a singleton's item |
| `POST`, `PUT` | `/api/singleton/{name}` | token | Write a singleton's item |
| `POST` | `/api/graphql` | token | [GraphQL](../guides/graphql.md): `query`, `variables`, `operationName` |

## Schema

| Method | Path | Auth |
|--------|------|------|
| `GET`, `POST` | `/admin/collections` | admin |
| `GET`, `PUT`, `DELETE` | `/admin/collections/{name}` | admin |
| `GET` | `/admin/singletons` | admin |
| `POST` | `/admin/singleton` | admin |
| `GET`, `PUT`, `DELETE` | `/admin/singleton/{name}` | admin |
| `GET`, `POST` | `/admin/components` | admin |
| `GET`, `PUT`, `DELETE` | `/admin/components/{name}` | admin |

`GET /admin/collections` also accepts `filter` (JSON object of column → value), `sort`
(`["name","DESC"]`) and `range` (`[0,9]`), react-admin style, and returns a `Content-Range`
header.

## Roles

| Method | Path | Auth | Body |
|--------|------|------|------|
| `GET` | `/admin/roles`, `/admin/roles/{name}` | admin | |
| `POST` | `/admin/roles` | admin | `name`, `description`, `permissions` ([custom roles](../guides/users-and-roles.md#custom-roles)) |
| `PUT` | `/admin/roles/{name}` | admin | `description`, `permissions` |
| `DELETE` | `/admin/roles/{name}` | admin | refused while the role is in use |

## Locales

| Method | Path | Auth | Body |
|--------|------|------|------|
| `POST` | `/admin/locales` | admin | `code` (BCP 47), `name` |
| `PUT` | `/admin/locales/{code}` | admin | `name`, and/or `isDefault: true` (refused while the locale has translations) |
| `DELETE` | `/admin/locales/{code}` | admin | `?purge=true` also deletes its translations (required when there are some) |

## Users

| Method | Path | Auth | Body |
|--------|------|------|------|
| `GET`, `POST` | `/admin/users` | admin | `username`, `email`, `password`, `role_name` |
| `GET`, `PUT`, `DELETE` | `/admin/users/{id}` | admin | `username`, `email`, `password`, `role_name` |

## Agents

| Method | Path | Auth |
|--------|------|------|
| `GET`, `POST` | `/admin/agents` | admin |
| `GET`, `PUT`, `DELETE` | `/admin/agents/{name}` | admin |
| `GET` | `/admin/agents/{name}/runs` | admin |
| `GET` | `/admin/agents/{name}/runs/{id}` | admin |
| `POST` | `/admin/agents/{name}/runs/{id}/revert` | admin ([revert a run's content changes](../guides/content-history.md#reverting-an-agent-run); `?force=true`) |
| `GET` | `/admin/agents/{name}/usage` | admin |
| `POST` | `/agents/webhook/{id}` | webhook |

## Response envelope

```json
{ "data": … , "meta": { … } }
{ "error": { "status": 400, "name": "ValidationError", "message": "…", "details": … } }
```

## Integrations (admin)

| Method | Path | Does |
|--------|------|------|
| `POST`, `GET` | `/admin/tokens` | Create (returns the secret once) / list [API tokens](../guides/integrations.md#api-tokens); user session only |
| `GET`, `DELETE` | `/admin/tokens/{id}` | One token / revoke it; user session only |
| `POST`, `GET` | `/admin/webhooks` | Create / list [outbound webhooks](../guides/integrations.md#outbound-webhooks) |
| `GET`, `PUT`, `DELETE` | `/admin/webhooks/{id}` | One webhook |
| `GET` | `/admin/webhooks/{id}/deliveries` | Last delivery attempts |
| `POST` | `/admin/webhooks/{id}/test` | Send a test event now |
