# Templates

Templates are ready-made agents for common jobs. Pick one, answer a few questions about your
content model, review the agent it produces and save it. The result is an ordinary agent:
change its prompt, model or limits like any other.

In the admin UI: **Agents → New agent → Start from a template**.

| Template | Does | Trigger |
|----------|------|---------|
| [Summarizer](#summarizer) | Writes a short summary of a text field into another field | item created or updated |
| [Tagger](#tagger) | Picks tags from the text, optionally from a fixed list | item created or updated |
| [Moderator](#moderator) | Flags abusive, spam or unsafe content for an editor | item created or updated |
| [Translator](#translator) | Translates localized fields into the other locales | item created (optionally updated) |

Each template's agent:

- is triggered by [collection events](triggers.md) of the collection you pick;
- writes with `content.update`, [scoped](functions.md#limiting-tools-to-collections)
  to that collection, so it only changes the fields it is about;
- uses the instance's default model (`llm.provider`, `llm.model`), at most 60 runs an hour;
- can [require approval](approvals.md): on by default for the translator, off for the
  others.

An agent is not triggered by its own changes, so the summarizer updating an item does not
start it again. Try a new agent with a [test run](runs-and-limits.md#test-runs) before
letting it loose.

## Summarizer

| Parameter | |
|-----------|-|
| `collection` | The collection |
| `source` | The text to summarize (`string`, `text` or `richtext`) |
| `target` | Where the summary goes (another text field) |
| `words` | Maximum words (40) |

## Tagger

| Parameter | |
|-----------|-|
| `sources` | The text fields to read |
| `target` | Where the tags go: a `json` field gets a list, a text field a comma-separated list |
| `tags` | Allowed tags, comma-separated. Empty: the agent chooses |
| `max` | Maximum tags (5) |

## Moderator

| Parameter | |
|-----------|-|
| `sources` | The text fields to check |
| `flag` | The field that records the decision: a boolean (`true` when flagged), an enum or a text field |
| `flagged_value`, `ok_value` | The values for an enum or text flag (`flagged`, `ok`); an enum must have them as options |
| `reason` | Optional text field for the reason |

The moderator never edits or deletes content; it only flags it. Filter on the flag in the
admin UI or the API to review flagged items.

## Translator

| Parameter | |
|-----------|-|
| `fields` | The [localized](../guides/localization.md) fields to translate |
| `locales` | Locale codes, comma-separated, not the default locale (the source). Empty: every other locale |
| `on_update` | Also translate when an item is updated (overwrites translations people edited) |

## API

```bash
# Templates and their parameters
curl $GOHEAD/admin/agent-templates -H "$AUTH"

# An agent definition from a template (not saved)
curl -X POST $GOHEAD/admin/agent-templates/summarizer/render -H "$AUTH" \
  -d '{"params": {"collection": "posts", "source": "body", "target": "excerpt", "words": 30}}' \
  | jq .data > agent.json

# Review it, then create the agent
curl -X POST $GOHEAD/admin/agents -H "$AUTH" -d @agent.json
```

Parameters are checked against the content model: a missing or unknown field, a field of
the wrong type or an unknown locale returns `400` with the reason. `name` is optional
(`<template>-<collection>`).
