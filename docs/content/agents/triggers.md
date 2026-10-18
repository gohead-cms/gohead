# Triggers

`trigger.type` decides when an agent runs.

| Type | Runs when | Extra fields |
|------|-----------|--------------|
| `collection_event` | an item is created, updated, deleted, published or unpublished | `event_trigger.collection`, `event_trigger.events` |
| `webhook` | `POST /agents/webhook/{id}` with the right token | `webhook_token` |
| `cron` | on a schedule | `cron` |
| `manual` | never on its own (definition only) | |

## Collection events

```json
"trigger": {
  "type": "collection_event",
  "event_trigger": { "collection": "articles", "events": ["item:created", "item:updated"] }
}
```

Events: `item:created`, `item:updated`, `item:deleted`, and for
[draft-and-publish](../guides/content-modeling.md#draft-and-publish) collections
`item:published` and `item:unpublished` (when an item becomes visible to readers, or stops
being visible). They are emitted for REST and GraphQL writes. Writes made by agents (in the worker) do **not** emit events, so agents cannot
trigger each other in loops. The agent's first message contains the event type, the
collection, the item ID and the item data.

## Webhook

```json
"trigger": { "type": "webhook", "webhook_token": "a-long-random-string" }
```

Call it with the agent's numeric ID (shown when you create or list agents):

```bash
curl -X POST $GOHEAD/agents/webhook/2 \
  -H "Content-Type: application/json" \
  -H "Webhook-Token: a-long-random-string" \
  -d '{"event":"order.paid","order_id":123}'
```

| Response | Meaning |
|----------|---------|
| `202` `{"data":{"status":"Job accepted for processing"}}` | queued |
| `401` | wrong or missing token, or the agent is not a webhook agent |
| `404` | no agent with this ID |
| `500` | the job could not be queued (is Redis up?) |

The JSON body is passed to the agent in its first message. Tokens are compared in constant
time; generate them with `openssl rand -hex 32`.

## Cron

```json
"trigger": { "type": "cron", "cron": "0 7 * * 1-5" }
```

Standard five-field cron expressions, evaluated in **UTC**. The scheduler re-reads cron agents
every minute, so new or changed schedules apply without a restart. Run exactly one scheduler,
or each tick runs once per scheduler.

## Manual

`manual` agents are stored but never started automatically. Use them to keep a definition
around, or switch their trigger later with `PUT /admin/agents/{name}`.
