# Content history

Every change to an entry is saved as a **revision**. That covers collection items and singleton content, and every way they change: create, update, publish, unpublish, delete and restore. You can see who changed what, compare any two versions, and restore an old one. That includes entries that were deleted, and everything an agent run changed.

History is for **editors and admins**. Viewers get `403`, because revisions can hold draft content.

## What a revision holds

| Field | Description |
|-------|-------------|
| `version` | `1, 2, 3…` per entry |
| `action` | `create`, `update`, `publish`, `unpublish`, `delete` or `restore` |
| `data` | The entry's stored data after the change, with relations as IDs. For a `delete`, the data the entry had. |
| `status` | `draft` or `published` after the change (draft-and-publish collections) |
| `actorKind`, `actorName` | Who made the change: `user` (username), `token` (API token name), `agent` (agent name) or `system` |
| `agentRunId` | For agents, the [run](../agents/runs-and-limits.md) that made the change |
| `restoredFrom` | For a restore, the version it copied |

A save that changes nothing (same data, same status) doesn't add a revision. The revision is written in the same database transaction as the change itself, so the two can't get out of step.

The newest **100 revisions per entry** are kept. Older ones are pruned as new ones arrive. Change the limit with [`content.revisions_kept`](../getting-started/configuration.md#content-history), where `0` keeps everything. Deleting a collection or a singleton type deletes its history too.

!!! note "Upgrading"
    Entries that existed before 0.5 have no history yet. Their first revision is recorded the next time they change.

## In the admin UI

- **History** opens from the clock icon on a row of the content table, from the edit drawer, or from a singleton's editor. It lists the entry's revisions with who made each one and when. For an agent, it links to the run. Pick a revision to see its changes field by field: against the previous version, or against the current content, which shows what restoring it would change. **Restore this version** makes it current.
- **Recently deleted**, in a collection's header, lists the entries that are still deleted and brings them back.
- **Activity → Content changes** is the change feed across all entries. You can filter it by who made the change (people, API tokens, agents) and by action.
- An agent run's details list the **content changes** it made. Admins can **revert** them (see [below](#reverting-an-agent-run)).

## API

```bash
# History of an item, newest first (page, pageSize up to 100)
curl $GOHEAD/api/collections/articles/42/revisions -H "$AUTH"

# One version, with its data and what changed since the previous version
curl $GOHEAD/api/collections/articles/42/revisions/3 -H "$AUTH"

# ...or against another version, or against the current content
curl "$GOHEAD/api/collections/articles/42/revisions/3?compare=current" -H "$AUTH"

# Make version 3 the current content
curl -X POST $GOHEAD/api/collections/articles/42/revisions/3/restore -H "$AUTH"
```

```json
{
  "data": {
    "id": 9, "entityType": "item", "entityId": 42, "version": 3, "typeName": "articles",
    "action": "update", "status": "draft", "actorKind": "agent", "actorName": "summarizer",
    "agentRunId": 118, "createdAt": "2026-10-19T08:12:03Z",
    "data": { "title": "GoHead 1.0", "summary": "A short summary" },
    "comparedTo": 2,
    "changes": [
      { "field": "summary", "change": "added", "after": "A short summary" }
    ]
  }
}
```

`changes` lists the attributes that differ, sorted by name. Each has a `change` of `added`, `removed` or `changed`, with the `before` and `after` values.

Singletons have the same three routes under `/api/singleton/{name}/revisions`.

### Restoring

A restore copies the old version's data into the entry and records a new `restore` revision. Nothing is rewritten in place, so a restore can itself be undone.

- The data is validated against the **current** content model, as a normal update would be. Attributes the type no longer has are left out and listed in `meta.dropped`. If the old data no longer fits (a required attribute was added since, a unique value is now taken, a related entry is gone), the restore fails with `400` and says why.
- **Restoring a deleted entry brings it back** under the same ID (`meta.undeleted: true`). Use the version of its `delete` revision to get the content it had before it was deleted. In a draft-and-publish collection it comes back as a **draft**.
- Otherwise the entry's publication state is left as it is: restoring an old version of a published article changes its content, and it stays published.
- Restores send the usual `item:updated` webhook and agent events (`item:created` for an entry brought back).

```json
{
  "data": { "id": 42, "title": "GoHead 1.0", "status": "published", "...": "..." },
  "meta": {
    "revision": { "version": 5, "action": "restore", "restoredFrom": 3, "...": "..." },
    "dropped": ["subtitle"],
    "undeleted": false
  }
}
```

### Change feed

`GET /api/revisions` lists revisions across all entries, newest first. It takes these filters:

| Parameter | Description |
|-----------|-------------|
| `collection` or `singleton` | One collection's items, or one singleton |
| `action` | e.g. `delete`, to find deleted entries |
| `actorKind` | `user`, `token`, `agent` or `system` |
| `actor` | A username, token name or agent name |
| `agentRun` | The changes of one agent run |

```bash
# Items deleted from "articles"
curl "$GOHEAD/api/revisions?collection=articles&action=delete" -H "$AUTH"

# Everything agents changed today, page by page
curl "$GOHEAD/api/revisions?actorKind=agent&pageSize=100" -H "$AUTH"
```

## Reverting an agent run

Agent functions such as `collections.upsert_item` and `collections.delete_item` write content as the agent. Each revision they create carries the run's ID. An admin can undo everything a run changed:

```bash
curl -X POST $GOHEAD/admin/agents/summarizer/runs/118/revert -H "$AUTH"
```

The revert works entry by entry:

- An entry the run **created** is deleted. It can still be restored from its history.
- An entry the run **changed or deleted** goes back to the version it had before the run's first change.
- An entry **someone changed after the run** is skipped, so their work isn't lost. Add `?force=true` to revert it anyway.

The response lists what happened to each entry:

```json
{
  "data": [
    { "entityType": "item", "entityId": 42, "typeName": "articles", "result": "reverted", "version": 2 },
    { "entityType": "item", "entityId": 57, "typeName": "articles", "result": "deleted" },
    { "entityType": "item", "entityId": 12, "typeName": "articles", "result": "skipped",
      "reason": "changed after the run (version 6 by user alice)" }
  ],
  "meta": { "counts": { "reverted": 1, "deleted": 1, "skipped": 1 } }
}
```

The revert's own changes are attributed to the admin who asked for it. A run that changed no content returns `409`. An entry can't be reverted if the version from before the run has already been pruned: it's skipped with a reason.

## Not covered

- Changes to the **content model** (collections, attributes, components) and to media files aren't versioned.
- History is available through REST only. GraphQL mutations do record revisions, but there's no GraphQL query for history.
