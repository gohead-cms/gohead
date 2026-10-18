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
| `POST` | `/auth/register` | – | `username`, `email`, `password` |
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
