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
| `content.save` without `item_id` | `create`, with the new item's values |
| `content.save` with `item_id`, `content.update` | `update`, with the item's values after the change |
| the same with a `locale` other than the default | `update` of that translation |
| `content.delete` | `delete` |

The change is validated when the agent proposes it (required fields, types, unique values,
the function's [collection scope](functions.md#limiting-tools-to-collections)): an
invalid change is refused right away and the agent sees the error, as it would without
approval. The function returns `{"status": "pending_approval", "proposal_id": …}`, so the
agent knows nothing was written yet. Functions can pass a `reason`, shown to the reviewer.

Calls to [connected apps](connected-apps.md) can need approval too: by default, every call
of a tool that may change something in the app (send an email, open an issue) becomes a
`call` proposal, whether or not the agent requires approval. Admins review those; see
[Connected apps → Approvals](connected-apps.md#approvals).

## Reviewing

**Approvals** in the admin UI lists pending proposals, with a count in the sidebar. Open one
to see the change field by field against the item as it is now, then:

- **Approve** applies it;
- **Edit and approve** applies your corrected values instead;
- **Reject** discards it, with an optional note.

Who can review: anyone whose role may make the change. A role with `update` on `articles`
reviews updates of articles, `delete` is needed for deletions and `create` for creations,
and changing a published item needs `publish` as for any edit. Each reviewer only sees the
proposals they could apply; viewers see none.

An approved change is made **as the agent, in its run**: the
[history](../guides/content-history.md) shows the agent and its run, reverting the run undoes
it, and the proposal records who approved it. A change you edited before approving is
yours: the history shows you, not the agent. Approved changes emit content events
(webhooks), but never trigger the agent that proposed them.

### Stale proposals

When the item changed after the agent proposed something (a person edited it, another
proposal was approved), the proposal is **stale** and GoHead refuses to approve it with
`409`; the UI shows a warning. The proposal then shows what applying it **now** would do:
the item as it is, with only the attributes the agent changed. Reject it, or apply it
anyway: changes made since to other attributes are kept.

If the change can no longer be applied (the item or its collection was deleted, a locale
removed, a value is no longer valid), applying it marks the proposal `failed` with the
reason. Deleting a collection fails its pending proposals.

## API

```bash
# Pending proposals (status=pending|approved|rejected|failed|all, collection, agent, agentRun)
curl "$GOHEAD/api/proposals?agent=translator-posts" -H "$AUTH"

# One proposal: the item now (current), what approving writes (proposed),
# the diff between the two, and whether it is stale
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

Lists mark pending proposals whose item changed with `"stale": true`. Approving returns
`409` when the proposal was already decided (`details.code: "decided"`) or is stale
(`details.code: "stale"`; send `"force": true` to apply it anyway), `422` when the change could not be applied (the
proposal is then `failed`), `403` when the change needs a permission you lack (`publish`
for a published item, `create` for related items it would create) and `404` for proposals
your role may not apply. Password values are masked; a masked value sent back in `data`
keeps the proposed one.
