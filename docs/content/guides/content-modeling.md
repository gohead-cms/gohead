# Content modeling

All schema endpoints require an **admin** token.

## Collections

```bash
curl -X POST $GOHEAD/admin/collections -H "$JSON" -H "$AUTH" -d '{
  "name": "products",
  "kind": "collection",
  "description": "Items sold in the shop",
  "attributes": {
    "name":     { "type": "string", "required": true, "unique": true },
    "sku":      { "type": "uid", "required": true, "unique": true },
    "price":    { "type": "float", "required": true },
    "stock":    { "type": "int", "min": 0 },
    "status":   { "type": "enum", "options": ["draft", "active", "archived"] },
    "released": { "type": "date" },
    "specs":    { "type": "json" }
  }
}'
```

| Field | Required | Notes |
|-------|----------|-------|
| `name` | yes | Used in URLs (`/api/collections/products`) and as the GraphQL field name. |
| `kind` | yes | `collection`. |
| `description` | no | Free text. |
| `draft_and_publish` | no | `true` enables the [draft/publish workflow](#draft-and-publish). Default `false`. |
| `attributes` | yes | Map of attribute name → definition. |

Attribute names `id`, `createdAt`, `updatedAt` and `publishedAt` are reserved (they are
system fields in every [response](rest-api.md#responses)), and so is `status` in
draft-and-publish collections.

Manage collections with `GET /admin/collections`, `GET|PUT|DELETE /admin/collections/{name}`.
Updating a collection replaces its attribute list: attributes you omit are removed.
Deleting a collection deletes its items, and is refused while another collection has a
relation pointing to it.

## Attribute types

| Type | Value | Options |
|------|-------|---------|
| `string`, `text`, `richtext`, `email` | string | `pattern` (regex), `unique` |
| `password` | string | Stored as given; prefer [users](users-and-roles.md) for accounts. |
| `uid` | string (slug, SKU…) | `unique` |
| `int` / `integer` | whole number | `min`, `max` |
| `float` / `decimal` | number | |
| `bool` / `boolean` | `true` / `false` | |
| `date` | `"YYYY-MM-DD"` | |
| `datetime` | RFC 3339, e.g. `"2026-10-18T09:00:00Z"` | |
| `enum` / `enumeration` | one of `options` | `options` (required) |
| `json` | JSON object | |
| `media` | an uploaded file's ID (see [media](media.md)) | `multiple`, `allowed_types`, `min`/`max` (with `multiple`) |
| `relation` | ID(s) of items in `target` | `relation`, `target` (required) |
| `component` | an object, or a list with `repeatable` | `component` (required), `repeatable`, `min`/`max` (list length) |

Every attribute also accepts `required` (default `false`) and `unique`.

## Relations

```json
"author": { "type": "relation", "relation": "oneToOne", "target": "authors" },
"tags":   { "type": "relation", "relation": "manyToMany", "target": "tags" }
```

| `relation` | Stored value | In responses |
|------------|--------------|--------------|
| `oneToOne`, `oneToMany` | one item ID | an object |
| `manyToMany` | list of item IDs | a list |

When writing an item you can pass an existing ID (`"author": 1`), a list of IDs
(`"tags": [4, 5]`), or a **nested object** that is created in the same transaction:

```json
{ "data": { "title": "Nested", "author": { "name": "Grace Hopper" } } }
```

The target collection must exist before you create the relation.

## Singletons

For content that exists exactly once, such as a home page or global settings:

```bash
curl -X POST $GOHEAD/admin/singleton -H "$JSON" -H "$AUTH" -d '{
  "name": "homepage",
  "attributes": {
    "headline": { "type": "string", "required": true },
    "tagline":  { "type": "text" }
  }
}'

# write and read its single item
curl -X PUT $GOHEAD/api/singleton/homepage -H "$JSON" -H "$AUTH" \
  -d '{"data":{"headline":"Welcome","tagline":"Built with GoHead"}}'
curl $GOHEAD/api/singleton/homepage -H "$AUTH"
```

```json
{ "data": { "id": 1, "headline": "Welcome", "tagline": "Built with GoHead",
            "createdAt": "2026-10-18T09:00:00Z", "updatedAt": "2026-10-18T09:00:00Z" } }
```

Writing needs the `update` permission; reading needs `read`. `level` populates relations
as for items. `POST /admin/singleton` creates or updates (upsert by name). List them with
`GET /admin/singletons`; delete with `DELETE /admin/singleton/{name}`.

## Components

Reusable attribute groups:

```bash
curl -X POST $GOHEAD/admin/components -H "$JSON" -H "$AUTH" -d '{
  "name": "seo",
  "attributes": {
    "meta_title":       { "type": "string", "required": true },
    "meta_description": { "type": "text" }
  }
}'
```

Use it from a collection (or a singleton) with `"type": "component"`. Add
`"repeatable": true` for a list of them; `min` and `max` then bound the list length:

```bash
curl -X POST $GOHEAD/admin/collections -H "$JSON" -H "$AUTH" -d '{
  "name": "pages",
  "kind": "collection",
  "attributes": {
    "title": { "type": "string", "required": true },
    "seo":   { "type": "component", "component": "seo", "required": true },
    "faq":   { "type": "component", "component": "question", "repeatable": true, "max": 20 }
  }
}'
```

Item values are plain objects, validated field by field with the component's rules
(types, `required`, `pattern`, `min`/`max`, `options`); unknown fields are rejected:

```json
{ "data": {
    "title": "Pricing",
    "seo": { "meta_title": "Pricing - Acme" },
    "faq": [ { "question": "Is there a free plan?", "answer": "Yes." } ]
} }
```

They come back exactly like that in REST responses, and as typed objects in
[GraphQL](graphql.md#components). Components can contain other components (up to five levels
deep), but not relations: put relations on the collection.

Manage components with `GET /admin/components` and `GET|PUT|DELETE /admin/components/{name}`.
A component that is still used by a collection, singleton or other component cannot be
deleted or renamed (`409`, with the list of attributes using it).

## Draft and publish

With `"draft_and_publish": true`, a collection's items have a status:

- **New items are drafts** unless created with `"status": "published"`.
- **Readers only see published items.** Viewers never see drafts, in lists, by ID, or inside
  populated relations. Lists return published items by default for every role; editors and
  admins add `status=draft` or `status=all`, and get drafts when fetching an item by ID.
- Items gain two response fields: `status` (`draft` or `published`) and `publishedAt`.

```bash
# create a draft, then publish it
curl -X POST $GOHEAD/api/collections/articles -H "$JSON" -H "$AUTH" -d '{"data":{"title":"Soon"}}'
curl -X POST $GOHEAD/api/collections/articles/7/publish -H "$AUTH"
curl -X POST $GOHEAD/api/collections/articles/7/unpublish -H "$AUTH"

# or set the status while writing
curl -X PUT $GOHEAD/api/collections/articles/7 -H "$JSON" -H "$AUTH" \
  -d '{"data":{"title":"Now"},"status":"published"}'
```

Publishing needs the `update` permission. Publishing and unpublishing emit `item:published`
and `item:unpublished` [events](../agents/triggers.md), for example to rebuild a static site.

An item has a single version: editing a published item changes what readers see right
away. To prepare changes privately, unpublish first, or keep drafts in a separate item.

Turning the option on for an existing collection publishes its existing items, so nothing
disappears. Turning it off makes every item visible again (drafts become published). An
update that does not mention `draft_and_publish` keeps the current setting; the option is
also accepted as `options.draftAndPublish`, the shape `GET /admin/collections/{name}` returns.

Items created inline through a [nested relation](#relations) are drafts too: publish them
separately, or readers will see the relation as empty.

## Naming tips

- Use lower-case plural names for collections (`articles`); GraphQL fields and mutations
  use the name as-is (`articles`, `createarticles`).
- Attribute names should be valid identifiers (`meta_title`, not `meta title`) so they can be
  used in GraphQL and in [filters](rest-api.md#filtering-and-sorting).
