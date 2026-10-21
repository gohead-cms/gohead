# Tools

An agent's `tools` say what it may do: read or change content, ask a model about a text,
write to the log. Name a tool with `use`, and limit content tools to collections with `on`:

```json
"tools": [
  { "use": "content.update", "on": ["articles"] },
  { "use": "ai.summarize" }
]
```

That's all a tool needs. GoHead tells the model what the tool is called, what it does and
which arguments it takes, from the catalog below. In the admin UI, **Add a tool** in the agent
editor lists the same catalog.

| Field | | |
|-------|-|-|
| `use` | required | The tool: `content.update`, `ai.summarize`… (or its implementation key, `collections.patch_item`) |
| `on` | content tools | The collections it may use. Empty: every collection, which you rarely want for writes |
| `name` | optional | The name the model sees. Default: the tool's (`update_item`). Set it when the system prompt refers to the tool by another name, or for two tools of the same kind |
| `description` | optional | What the model is told about it. Default: the tool's description, plus the collections in `on` |
| `parameters` | optional | A JSON schema of its arguments. Default: the tool's own. Property names must be arguments the tool reads |

Every tool returns JSON; errors come back as `{"status": "error", "message": "…"}` so the
model can correct itself. `GET /admin/agent-functions` returns the catalog: each tool's
`use`, default `name`, `label`, `description`, `parameters`, and whether it is `scoped` (takes
`on`) and `writes` content.

??? note "Reading agents back, and the long form"
    `GET /admin/agents/{name}` returns both forms in `schema`: `tools` as written (only what
    differs from the catalog), and `functions`, every tool as the model gets it, with its
    name, description, parameters, `impl_key` and `collections`.

    Agents written for earlier versions list `functions` with all of those fields; they keep
    working, and an empty `parameters` schema now gets the tool's own. When a request has
    both `tools` and `functions`, `tools` is used.

## Content tools

| `use` | Name | Arguments | Does |
|-------|------|-----------|------|
| `content.types` | `list_collections` | – | Names of the collections in scope |
| `content.schema` | `get_schema` | `collection_name` | Attributes and types of a collection |
| `content.list` | `list_items` | `collection_name`, `page`, `pageSize` | A page of items |
| `content.save` | `save_item` | `collection_name`, `data` (object), optional `item_id` (string), optional `locale`, optional `reason` | Creates an item, or replaces item `item_id`; with `item_id` and a `locale` other than the default, writes that [translation](../guides/localization.md#writing-translations) |
| `content.update` | `update_item` | `collection_name`, `item_id`, `data` (object), optional `locale`, optional `reason` | Changes the given attributes of an item (or of its translation), keeping the others |
| `content.delete` | `delete_item` | `item_id` (string), optional `collection_name`, optional `reason` | Deletes an item by ID |
| `content.locales` | `list_locales` | – | The content locales, the default first |

Their implementation keys are `collections.list`, `collections.get_schema`,
`collections.list_items`, `collections.upsert_item`, `collections.patch_item`,
`collections.delete_item` and `collections.list_locales`.

In [draft-and-publish](../guides/content-modeling.md#draft-and-publish) collections, items
created by agents are drafts for a human to review and publish, and `content.list` includes
drafts.

Writes are validated like API writes (required fields, types, unique values);
`content.save` with an `item_id` replaces the item's data, `content.update` changes only the
attributes given. Agent writes do not emit content events (no webhooks, no other agents):
sites that show what agents write refresh on their own schedule, or when a person next edits
the entry.

For an agent that [requires approval](approvals.md), writes become proposals instead and
return `{"status": "pending_approval", "proposal_id": …}`; `reason` is shown to the
reviewer. In a [test run](runs-and-limits.md#test-runs) they are validated but not applied
and return `{"status": "success", "dry_run": true, "action": "would_update", …}`.

## Limiting tools to collections

Without `on`, a content tool works on every collection: an agent with `content.delete` can
delete any item. Give each content tool the collections it needs:

```json
{ "use": "content.delete", "on": ["comments"], "name": "delete_spam_comment" }
```

- The check runs in the server, not in the prompt: a call outside the scope returns
  `{"status": "error", "message": "this function is not allowed to access collection '…'"}`
  and changes nothing, whatever the model was told.
- `content.delete` checks the collection the item **actually belongs to**, so an item ID
  from another collection is refused. If the model also passes `collection_name`, it must
  match.
- `content.types` only returns the collections in scope.
- Writes follow relations: a relation value given as an object without an `id` is looked up
  in, or created in, the target collection, so that collection must be in scope too.
  Connecting an existing item by ID (`5` or `{"id": 5}`) only needs the written collection.
- `on` only applies to content tools. Two tools of the same kind can have different scopes:
  give them different names.

!!! tip "Scope every write"
    Give `on` to every `content.save`, `content.update` and `content.delete` an agent has; the
    admin UI warns until you do. A run's [history](runs-and-limits.md#run-history) shows
    refused calls as failed steps.

## AI helpers

These call a model themselves, with the instance-wide [`llm` settings](llm-providers.md#instance-defaults)
(not the agent's own `llm_config`), and return structured JSON.

| `use` | Name | Arguments |
|-------|------|-----------|
| `ai.summarize` | `summarize_text` | `text`, optional `max_length` (words), `style` |
| `ai.tags` | `generate_tags` | `text`, optional `max_tags` |
| `ai.title` | `generate_title` | `text`, optional `max_length`, `style` |
| `ai.rewrite` | `rewrite_text` | `text`, optional `tone`, `style`, `audience` |
| `ai.classify` | `classify_text` | `text`, `categories` (list) |
| `ai.moderate` | `moderate_content` | `text` |
| `ai.sentiment` | `analyze_sentiment` | `text` |
| `ai.score` | `score_quality` | `text`, optional `criteria` |
| `ai.check` | `evaluate_condition` | `text`, `condition` |
| `ai.entities` | `extract_entities` | `text`, optional `entity_types` (list) |
| `ai.extract` | `extract_structured_data` | `text`, `schema` (object) |
| `ai.rank` | `rank_items` | `items` (list), `criteria` |
| `ai.recommend` | `recommend_action` | `context`, `actions` (list), optional `goal` |

Their implementation keys are the `llm.*` functions: `llm.summarize_text`, `llm.generate_tags`…

## System

| `use` | Name | Arguments | Does |
|-------|------|-----------|------|
| `system.log` | `log` | `message`, optional `data` | Writes to the worker log |

## Adding your own

Implementations live in `internal/agent/functions`. Register a new one in an `init`
function and it can be used with its key (`"use": "shop.check_stock"`):

```go
func init() {
    StaticFunctionMap["shop.check_stock"] = checkStock
}

// args is either the raw JSON string from the LLM or a decoded map.
func checkStock(ctx context.Context, args any) (string, error) {
    argMap, err := parseArgs(args)
    if err != nil {
        return `{"status":"error","message":"invalid arguments"}`, nil
    }
    sku, _ := argMap["sku"].(string)
    // ... look it up ...
    return fmt.Sprintf(`{"status":"success","sku":%q,"stock":12}`, sku), nil
}
```

Then describe it in the catalog, `internal/agent/tools`: its `use` name, default name, label,
description and arguments. Without a catalog entry, agents must give the tool's `name`,
`description` and `parameters` themselves.

Return errors the model can act on as JSON with a `nil` Go error; return a Go error only for
failures the run should record.
