# Functions

Functions are the tools an agent's LLM can call. Each entry in an agent's `functions` list
maps a **name the LLM sees** to a **built-in implementation** (`impl_key`) and describes its
arguments with a JSON schema:

```json
{
  "name": "save_summary",
  "description": "Store a summary item",
  "impl_key": "collections.upsert_item",
  "parameters": {
    "type": "object",
    "properties": {
      "collection_name": { "type": "string" },
      "data": { "type": "object" }
    },
    "required": ["collection_name", "data"]
  }
}
```

`name` must be unique within the agent; `impl_key` must be one of the keys below.
The schema's property names must match the arguments the implementation reads.
Every function returns JSON; errors come back as `{"status": "error", "message": "…"}` so
the LLM can correct itself.

## Content functions

| `impl_key` | Arguments | Does |
|------------|-----------|------|
| `collections.list` | – | Names of all collections |
| `collections.get_schema` | `collection_name` | Attributes and types of a collection |
| `collections.list_items` | `collection_name`, `page`, `pageSize` | A page of items |
| `collections.upsert_item` | `collection_name`, `data` (object), optional `item_id` (string), optional `locale` | Creates an item, or replaces item `item_id`; with `item_id` and a `locale` other than the default, writes that [translation](../guides/localization.md#writing-translations) |
| `collections.list_locales` | – | The content locales, the default first |
| `collections.delete_item` | `item_id` (string), optional `collection_name` | Deletes an item by ID |
| `system.log` | `message`, optional `data` | Writes to the worker log |

In [draft-and-publish](../guides/content-modeling.md#draft-and-publish) collections, items
created by agents are drafts for a human to review and publish, and `collections.list_items`
includes drafts.

Creates through `collections.upsert_item` are validated like API writes (required fields,
types, unique values), and so are updates with `item_id`, which replace the item's data.
Agent writes do not emit content events.

## Limiting functions to collections

Without a scope, a `collections.*` function works on every collection: an agent with
`collections.delete_item` can delete any item. Add `collections` to restrict it:

```json
{
  "name": "delete_spam_comment",
  "description": "Delete a comment flagged as spam",
  "impl_key": "collections.delete_item",
  "collections": ["comments"],
  "parameters": {
    "type": "object",
    "properties": { "item_id": { "type": "string" } },
    "required": ["item_id"]
  }
}
```

- The check runs in the server, not in the prompt: a call outside the scope returns
  `{"status": "error", "message": "this function is not allowed to access collection '…'"}`
  and changes nothing, whatever the model was told.
- `collections.delete_item` checks the collection the item **actually belongs to**, so an
  item ID from another collection is refused. If the model also passes `collection_name`, it
  must match.
- `collections.list` only returns the collections in scope.
- Writes follow relations: a relation value given as an object without an `id` is looked up
  in, or created in, the target collection, so that collection must be in scope too.
  Connecting an existing item by ID (`5` or `{"id": 5}`) only needs the written collection.
- `collections` is only accepted on `collections.*` functions. Scope each function on its
  own: two functions can share an `impl_key` with different scopes.

!!! tip "Scope every write"
    Give `collections` to every `collections.upsert_item` and `collections.delete_item` an
    agent has. A run's [history](runs-and-limits.md#run-history) shows refused calls as
    failed steps.

## LLM helper functions

These call an LLM themselves using the instance-wide [`llm` settings](llm-providers.md#instance-defaults)
(not the agent's own `llm_config`) and return structured JSON.

| `impl_key` | Arguments |
|------------|-----------|
| `llm.analyze_sentiment` | `text` |
| `llm.classify_text` | `text`, `categories` (list) |
| `llm.moderate_content` | `text` |
| `llm.score_quality` | `text`, optional `criteria` |
| `llm.summarize_text` | `text`, optional `max_length` (words), `style` |
| `llm.rewrite_text` | `text`, optional `tone`, `style`, `audience` |
| `llm.generate_tags` | `text`, optional `max_tags` |
| `llm.generate_title` | `text`, optional `max_length`, `style` |
| `llm.evaluate_condition` | `text`, `condition` |
| `llm.rank_items` | `items` (list), `criteria` |
| `llm.recommend_action` | `context`, `actions` (list), optional `goal` |
| `llm.extract_entities` | `text`, optional `entity_types` (list) |
| `llm.extract_structured_data` | `text`, `schema` (object) |

## Adding your own

Implementations live in `internal/agent/functions`. Register a new one in an `init`
function and it becomes available as an `impl_key`:

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

Return errors the LLM can act on as JSON with a `nil` Go error; return a Go error only for
failures the run should record.
