# GraphQL

`POST /api/graphql` with a bearer token. The schema is generated from your collections and
rebuilt automatically when a collection changes.

```bash
curl -X POST $GOHEAD/api/graphql -H "$JSON" -H "$AUTH" -d '{
  "query": "query Articles($n: Int) { articles(limit: $n) { id title author { name } } }",
  "variables": { "n": 5 },
  "operationName": "Articles"
}'
```

## Queries

Each collection becomes a field named after it, returning a list:

| Argument | Type | Default | |
|----------|------|---------|--|
| `id` | `ID` | | Return just this item (a one-element list). |
| `limit` | `Int` | 25 | 1–100. |
| `offset` | `Int` | 0 | Items to skip. |
| `status` | `String` | `published` | `published`, `draft` or `all`, as in [REST](rest-api.md#read). Drafts need the `update` permission. |
| `locale`, `fallback` | `String`, `Boolean` | default locale, `true` | Collections with [localized attributes](localization.md#graphql) only. |

Every type has `id`, `createdAt` and `updatedAt`; types of
[draft-and-publish](content-modeling.md#draft-and-publish) collections also have `status` and
`publishedAt`. Viewers never see drafts, including through relations.

```graphql
{
  articles(limit: 10, offset: 20) {
    id
    title
    views
    author { id name }     # oneToOne / oneToMany: an object
    tags { id label }      # manyToMany: a list
  }
}
```

### Components

Component attributes are typed objects (lists when repeatable), named `Component_<name>`:

```graphql
{
  pages { title seo { meta_title } faq { question answer } }
}
```

## Mutations

For a collection named `articles`:

| Mutation | Arguments | Returns |
|----------|-----------|---------|
| `createarticles` | `input: articlesInput!` | the created item |
| `updatearticles` | `id: ID!`, `input: articlesUpdateInput!` | the updated item |
| `deletearticles` | `id: ID!` | `Boolean` |
| `publisharticles`, `unpublisharticles` | `id: ID!` | the item (draft-and-publish collections) |

`articlesInput` marks required attributes as non-null; `articlesUpdateInput` makes every
field optional because updates are **partial** (unlike REST `PUT`). In draft-and-publish
collections, create and update also take `status: "draft" | "published"`. Component inputs
are input objects (`Component_seoInput`).

```graphql
mutation Launch($id: ID!) {
  updatearticles(id: $id, status: "published", input: { title: "Launch day" }) { id status publishedAt }
}
```

## Rules

- Permissions are the same as REST: viewers can query, editors and admins can mutate.
  A forbidden operation returns an error such as `access denied: role 'viewer' cannot create items`.
- Writes go through the same validation, nested-relation handling and events as REST,
  so agents react to GraphQL writes too.
- Relation inputs take IDs (`author: "1"`; a list of IDs for manyToMany).
- `media` attributes are [`Media`](media.md#media-attributes) objects and take file IDs as input.
- `date` and `datetime` attributes are strings (`"2026-10-18"`, `"2026-10-18T09:00:00Z"`),
  as in REST.
- Dynamic-zone attributes are not supported yet.

## Errors

The HTTP status is `400` when the GraphQL request fails; the first GraphQL error is in
`error.details`:

```json
{ "error": { "status": 400, "name": "ValidationError", "message": "GraphQL request failed",
             "details": "Cannot query field \"articels\" on type \"Query\". Did you mean \"articles\"?" } }
```

!!! note "Multiple API replicas"
    The schema is rebuilt in the API process that handled the collection change, which then
    tells the other replicas through Redis; they rebuild within milliseconds. If a notification
    is lost (Redis restart, network blip), replicas catch up within 30 seconds. See
    [scaling](../operations/deployment.md#scaling).
