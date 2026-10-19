# Approvals

An agent with **Require approval** on does not change content itself: each write it makes
becomes a **proposal** that an editor approves, edits or rejects. Use it for agents you are
still tuning, for translations, or for anything readers will see before a person has.

```json
{
  "name": "translator-posts",
  "require_approval": true,
  ...
}
```

In the admin UI, the switch is in the agent form, and templates such as the
[translator](templates.md#translator) turn it on by default.

## What becomes a proposal

Every write made by the built-in content functions:

| Function | Proposal |
|----------|----------|
| `collections.upsert_item` without `item_id` | `create`, with the new item's values |
| `collections.upsert_item` with `item_id`, `collections.patch_item` | `update`, with the item's values after the change |
| the same with a `locale` other than the default | `update` of that translation |
| `collections.delete_item` | `delete` |

The change is validated when the agent proposes it (required fields, types, unique values,
the function's [collection scope](functions.md#limiting-functions-to-collections)): an
invalid change is refused right away and the agent sees the error, as it would without
approval. The function returns `{"status": "pending_approval", "proposal_id": …}`, so the
agent knows nothing was written yet. Functions can pass a `reason`, shown to the reviewer.

## Reviewing

**Approvals** in the admin UI lists pending proposals, with a count in the sidebar. Open one
to see the change field by field against the item as it is now, then:

- **Approve** applies it;
- **Edit and approve** applies your corrected values instead;
- **Reject** discards it, with an optional note.

Who can review: anyone whose role may make the change. A role with `update` on `articles`
reviews updates of articles, `delete` is needed for deletions and `create` for creations,
and changing a published item needs `publish` as for any edit. Viewers do not see
proposals. Each reviewer only sees the collections they may change.

An approved change is made **as the agent, in its run**: the
[history](../guides/content-history.md) shows the agent and its run, reverting the run undoes
it, and the proposal records who approved it. Approved changes emit content events
(webhooks), but never trigger the agent that proposed them.

### Stale proposals

When the item changed after the agent proposed something (a person edited it, another
proposal was approved), the proposal is **stale**: approving it would overwrite that
change, so GoHead refuses with `409` and the UI shows a warning. Review the item, then
reject the proposal or apply it anyway.

If the change can no longer be applied (the item was deleted, a value is no longer valid),
the proposal is marked `failed` with the reason.

## API

```bash
# Pending proposals (status=pending|approved|rejected|failed|all, collection, agent, agentRun)
curl "$GOHEAD/api/proposals?agent=translator-posts" -H "$AUTH"

# One proposal: the change, the item now, the diff, and whether it is stale
curl $GOHEAD/api/proposals/12 -H "$AUTH"

# Approve, optionally with edited values or despite a later change
curl -X POST $GOHEAD/api/proposals/12/approve -H "$AUTH" \
  -d '{"data": {"title": "Bonjour"}, "force": false}'

# Reject
curl -X POST $GOHEAD/api/proposals/12/reject -H "$AUTH" -d '{"note": "Too literal"}'
```

| Status | Meaning |
|--------|---------|
| `pending` | Waiting for a reviewer |
| `approved` | Applied (`reviewedBy`, `decidedAt`) |
| `rejected` | Discarded (`reviewNote`) |
| `failed` | Approved but could not be applied (`error`) |

Approving returns `409` when the proposal was already decided or is stale (send
`"force": true` to apply it anyway), `422` when the change could not be applied, and `403`
when your role may not make it.
