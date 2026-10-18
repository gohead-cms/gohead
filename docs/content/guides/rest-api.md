# REST API

Content lives under `/api/collections/{collection}`. All calls need
`Authorization: Bearer <token>`; writes need the `create`, `update` or `delete` permission of
your [role](users-and-roles.md).

| Method | Path | Does |
|--------|------|------|
| `GET` | `/api/collections/{c}` | List items (filter, sort, paginate) |
| `POST` | `/api/collections/{c}` | Create one item or a batch |
| `GET` | `/api/collections/{c}/{id}` | Get one item |
| `PUT` | `/api/collections/{c}/{id}` | Replace an item's data |
| `DELETE` | `/api/collections/{c}/{id}` | Delete an item |
| `GET` / `PUT` | `/api/singleton/{name}` | Read / write a singleton's item |

## Responses

Successful responses wrap the payload in `data` (and sometimes `meta`). Errors always use
the same envelope:

```json
{ "error": { "status": 404, "name": "NotFoundError", "message": "Collection not found", "details": null } }
```

| Status | `name` | Typical cause |
|--------|--------|---------------|
| 400 | `ValidationError` | invalid JSON, failed validation, bad filter |
| 401 | `UnauthorizedError` | missing or invalid token |
| 403 | `ForbiddenError` | role lacks the permission |
| 404 | `NotFoundError` | unknown collection or item |
| 409 | `ConflictError` | duplicate unique value (users) |
| 500 | `InternalServerError` | unexpected server error (see logs) |

## Create

One item:

```bash
curl -X POST $GOHEAD/api/collections/articles -H "$JSON" -H "$AUTH" \
  -d '{"data":{"title":"Hello GoHead","views":42,"author":1}}'
```

```json
{ "data": { "id": 2, "attributes": { "title": "Hello GoHead", "views": 42, "author": { "data": { "id": 1 } } } }, "meta": {} }
```

A batch is an array of `{"data": …}` objects. It is **atomic**: one invalid item rejects the
whole batch, and the error tells you which one:

```json
{
  "error": {
    "status": 400, "name": "ValidationError",
    "message": "Validation failed for an item in the batch",
    "details": { "item_index": 1, "error": "missing required attribute: 'title'" }
  }
}
```

Unknown attributes are rejected, required attributes are enforced, `unique` attributes are
checked, and nested relation objects are created in the same transaction
(see [relations](content-modeling.md#relations)).

## Read

```bash
curl "$GOHEAD/api/collections/articles/2?level=1" -H "$AUTH"
```

```json
{ "data": { "id": 2, "attributes": { "title": "Hello GoHead", "author": { "id": 1, "name": "Ada Lovelace" } } }, "meta": {} }
```

`level` controls how deep relations are populated: `1` (default) to `5`.

!!! note "Two response shapes"
    Single-item responses use `{id, attributes}`; list responses return flat objects
    (`{id, title, …}`). Both are stable; they will be unified in a future major version.

## Update

`PUT` **replaces** the item's data: send every attribute you want to keep. Validation is
the same as on create.

```bash
curl -X PUT $GOHEAD/api/collections/articles/2 -H "$JSON" -H "$AUTH" \
  -d '{"data":{"title":"Hello GoHead (updated)","views":43,"author":1}}'
```

## Delete

```bash
curl -X DELETE $GOHEAD/api/collections/articles/2 -H "$AUTH"
# {"data":null,"meta":{}}
```

## Filtering and sorting

```bash
curl -G $GOHEAD/api/collections/articles -H "$AUTH" \
  --data-urlencode 'filters[published]=true' \
  --data-urlencode 'filters[views][$gte]=40' \
  --data-urlencode 'filters[title][$contains]=go' \
  --data-urlencode 'sort=views:desc,title' \
  --data-urlencode 'page=1' --data-urlencode 'pageSize=20'
```

| Parameter | Meaning |
|-----------|---------|
| `filters[<field>][<op>]=<value>` | Condition on a field. Several filters are combined with AND. |
| `filters[<field>]=<value>` | Shorthand for `$eq`. |
| `sort=<field>[:asc\|:desc],…` | One or more sort keys; ties are broken by `id`. |
| `page`, `pageSize` | 1-based page, 1–100 items per page (default 10). |

| Operator | Meaning | Applies to |
|----------|---------|------------|
| `$eq`, `$ne` | equal, not equal | all filterable types |
| `$lt`, `$lte`, `$gt`, `$gte` | comparisons | numbers, text, dates (ISO strings compare correctly) |
| `$contains` | substring match | text types |

**Filterable fields:** `id`, `created_at`, `updated_at`, and attributes of type `string`,
`text`, `richtext`, `email`, `uid`, `enum`, `int`, `float`, `decimal`, `bool`, `date`,
`datetime`, `time`. Numbers compare numerically; booleans support `$eq`/`$ne` only.
Unknown fields, relations, JSON and media attributes are rejected with `400`, as are
more than 20 filters per request.

Response:

```json
{
  "data": [ { "id": 3, "title": "Filtering 101", "views": 250 }, { "id": 2, "title": "Hello GoHead", "views": 42 } ],
  "meta": { "pagination": { "page": 1, "pageSize": 10, "pageCount": 1, "total": 2 } }
}
```

## Webhooks

`POST /agents/webhook/{agentId}` starts an agent whose trigger type is `webhook`. It is
authenticated with the agent's token instead of a user token:

```bash
curl -X POST $GOHEAD/agents/webhook/2 -H "$JSON" -H "Webhook-Token: s3cret-token" \
  -d '{"event":"order.paid","order_id":123}'
# 202 {"data":{"status":"Job accepted for processing"}}
```

See [agent triggers](../agents/triggers.md#webhook).
