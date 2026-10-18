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
| `attributes` | yes | Map of attribute name → definition. |

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
| `media` | string (URL or path) | |
| `relation` | ID(s) of items in `target` | `relation`, `target` (required) |
| `component` | *see below* | `component` (component name) |

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
{ "data": { "id": 1, "attributes": { "headline": "Welcome", "tagline": "Built with GoHead" } } }
```

`POST /admin/singleton` creates or updates (upsert by name). List them with
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

Manage with `GET /admin/components`, `GET|PUT|DELETE /admin/components/{name}`.

!!! warning "Current limitation"
    Components can be defined and referenced from a collection (`"type": "component",
    "component": "seo"`), but item values for component attributes are not validated or
    stored yet, and component fields are not exposed in GraphQL. Until then, model reusable
    groups as a `json` attribute or as a related collection.

## Naming tips

- Use lower-case plural names for collections (`articles`); GraphQL fields and mutations
  use the name as-is (`articles`, `createarticles`).
- Attribute names should be valid identifiers (`meta_title`, not `meta title`) so they can be
  used in GraphQL and in [filters](rest-api.md#filtering-and-sorting).
