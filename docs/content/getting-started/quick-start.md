# Quick start

In ten minutes you will create an admin account, model two content types, write content,
query it with filters and GraphQL, and see where agents plug in. Every response below was
captured from a real server.

!!! tip "Prerequisites"
    A running API on `localhost:8080` ([installation](installation.md)), `curl`, and
    optionally `jq`. Commands use these shell variables:

    ```bash
    export GOHEAD=http://localhost:8080
    export JSON='Content-Type: application/json'
    ```

## 1. Create the first account

The first account registered on an empty instance becomes the **admin**. Every later
self-registration gets the read-only **viewer** role.

```bash
curl -s -X POST $GOHEAD/auth/register -H "$JSON" \
  -d '{"username":"admin","email":"admin@example.com","password":"change-me-please"}'
```

```json
{ "data": "User registered successfully" }
```

Log in and keep the token:

```bash
export TOKEN=$(curl -s -X POST $GOHEAD/auth/login -H "$JSON" \
  -d '{"username":"admin","password":"change-me-please"}' | jq -r .data.token)
export AUTH="Authorization: Bearer $TOKEN"
```

## 2. Model your content

A **collection** is a content type with typed attributes. Create `authors`, then `articles`
with a relation to it:

```bash
curl -s -X POST $GOHEAD/admin/collections -H "$JSON" -H "$AUTH" -d '{
  "name": "authors", "kind": "collection",
  "attributes": { "name": { "type": "string", "required": true }, "bio": { "type": "text" } }
}'

curl -s -X POST $GOHEAD/admin/collections -H "$JSON" -H "$AUTH" -d '{
  "name": "articles", "kind": "collection",
  "attributes": {
    "title":     { "type": "string", "required": true },
    "body":      { "type": "richtext" },
    "views":     { "type": "int" },
    "published": { "type": "bool" },
    "category":  { "type": "enum", "options": ["news", "tutorial", "opinion"] },
    "author":    { "type": "relation", "relation": "oneToOne", "target": "authors" }
  }
}'
```

```json
{
  "data": {
    "id": 2,
    "uid": "api::articles.articles",
    "schema": {
      "collectionName": "articles",
      "info": { "displayName": "Articles", "pluralName": "articles", "singularName": "article" },
      "attributes": {
        "author":   { "type": "relation", "relation": "oneToOne", "target": "api::authors.authors", "required": false },
        "title":    { "type": "string", "required": true },
        "...": "..."
      }
    }
  },
  "meta": { "message": "Collection created successfully" }
}
```

The REST endpoints and GraphQL fields for both collections exist now, no restart needed.

## 3. Write content

```bash
curl -s -X POST $GOHEAD/api/collections/authors -H "$JSON" -H "$AUTH" \
  -d '{"data":{"name":"Ada Lovelace","bio":"First programmer."}}'

curl -s -X POST $GOHEAD/api/collections/articles -H "$JSON" -H "$AUTH" \
  -d '{"data":{"title":"Hello GoHead","body":"<p>First post</p>","views":42,
               "published":true,"category":"news","author":1}}'
```

```json
{
  "data": {
    "id": 2,
    "title": "Hello GoHead",
    "body": "<p>First post</p>",
    "views": 42,
    "published": true,
    "category": "news",
    "author": { "id": 1, "name": "Ada Lovelace", "bio": "First programmer.",
                "createdAt": "2026-10-18T09:00:00Z", "updatedAt": "2026-10-18T09:00:00Z" },
    "createdAt": "2026-10-18T09:00:01Z",
    "updatedAt": "2026-10-18T09:00:01Z"
  }
}
```

Every item has this flat shape, in lists and single reads alike. Here `published` is an
ordinary attribute; for a real review workflow, enable
[draft and publish](../guides/content-modeling.md#draft-and-publish) on the collection.

!!! note "Item IDs are global"
    Items of every collection share one ID sequence, which is why the first article is `2`.

Send an **array** to create several items in one transaction. If any item is invalid,
nothing is stored:

```bash
curl -s -X POST $GOHEAD/api/collections/articles -H "$JSON" -H "$AUTH" -d '[
  {"data":{"title":"Filtering 101","views":250,"published":true,"category":"tutorial"}},
  {"data":{"title":"Draft ideas","views":3,"published":false,"category":"opinion"}}
]'
```

A relation can also be created inline. GoHead creates the author and links it in the same
transaction:

```bash
curl -s -X POST $GOHEAD/api/collections/articles -H "$JSON" -H "$AUTH" \
  -d '{"data":{"title":"Nested","author":{"name":"Grace Hopper"}}}'
```

Validation errors are explicit:

```json
{
  "error": {
    "status": 400,
    "name": "ValidationError",
    "message": "validation failed for attribute 'category': attribute 'category' must be one of [news tutorial opinion]",
    "details": null
  }
}
```

## 4. Query with filters

Published articles with at least 40 views, most viewed first:

```bash
curl -s -G $GOHEAD/api/collections/articles -H "$AUTH" \
  --data-urlencode 'filters[published]=true' \
  --data-urlencode 'filters[views][$gte]=40' \
  --data-urlencode 'sort=views:desc'
```

```json
{
  "data": [
    { "id": 3, "title": "Filtering 101", "views": 250, "published": true, "category": "tutorial",
      "author": null, "createdAt": "…", "updatedAt": "…" },
    { "id": 2, "title": "Hello GoHead", "views": 42, "published": true, "category": "news",
      "body": "<p>First post</p>", "createdAt": "…", "updatedAt": "…",
      "author": { "id": 1, "name": "Ada Lovelace", "bio": "First programmer.", "createdAt": "…", "updatedAt": "…" } }
  ],
  "meta": { "pagination": { "page": 1, "pageSize": 10, "pageCount": 1, "total": 2 } }
}
```

Relations are populated one level deep by default (`?level=` up to 5). See the
[REST API guide](../guides/rest-api.md) for every operator.

## 5. Use GraphQL

```bash
curl -s -X POST $GOHEAD/api/graphql -H "$JSON" -H "$AUTH" \
  -d '{"query":"{ articles(limit: 2) { id title views author { name } } }"}'
```

```json
{
  "data": {
    "articles": [
      { "id": "2", "title": "Hello GoHead", "views": 42, "author": { "name": "Ada Lovelace" } },
      { "id": "3", "title": "Filtering 101", "views": 250, "author": null }
    ]
  }
}
```

Mutations work the same way and accept variables:

```bash
curl -s -X POST $GOHEAD/api/graphql -H "$JSON" -H "$AUTH" -d '{
  "query": "mutation($t: String!) { createarticles(input: {title: $t, views: 7}) { id title } }",
  "variables": { "t": "Via GraphQL" }
}'
```

## 6. Add an agent

Agents run in the `worker` process (Docker Compose starts it for you). This one is called
every time an article is created:

```bash
curl -s -X POST $GOHEAD/admin/agents -H "$JSON" -H "$AUTH" -d '{
  "name": "summarizer",
  "system_prompt": "Summarize new articles in two sentences.",
  "max_turns": 4,
  "llm_config": { "provider": "openai", "model": "gpt-4o-mini" },
  "memory": { "type": "in-memory", "session_scope": "user" },
  "trigger": { "type": "collection_event",
               "event_trigger": { "collection": "articles", "events": ["item:created"] } }
}'
```

Read the [agents guide](../agents/index.md) to give it functions so it can write back to
your content.

## Next steps

- Invite users and assign roles: [Users & roles](../guides/users-and-roles.md)
- Explore the schema options: [Content modeling](../guides/content-modeling.md)
- Go to production: [Deployment](../operations/deployment.md)
