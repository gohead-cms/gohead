# Localization

GoHead stores content in several languages **field by field**. You mark the attributes that differ by language (a title, a body, a slug) as **localized**. Everything else is **shared**: prices, dates, relations, flags. An entry has a single ID in every language. Its shared values are written once, and only its localized values are translated.

| | Default locale | Other locales |
|---|---|---|
| Localized attributes | the entry's main data | a translation per locale; a missing value falls back to the default locale |
| Shared attributes | the entry's main data | the same values, read-only |
| Relations | shared | shared |
| Draft / published | per entry | per entry |
| History | every change, translations included | |

## Locales

Every instance has one **default locale**, `en` unless `i18n.default_locale` says otherwise. Existing content is in the default locale. Add the other locales your site needs:

```bash
# List (any signed-in user)
curl $GOHEAD/api/locales -H "$AUTH"

# Add (admin). Codes follow BCP 47: en, fr, de, pt-BR, zh-Hans…
curl -X POST $GOHEAD/admin/locales -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"code": "fr", "name": "Français"}'

# Rename
curl -X PUT $GOHEAD/admin/locales/fr -H "$AUTH" -d '{"name": "French"}'

# Delete. If entries have French content, add ?purge=true to delete it too.
curl -X DELETE "$GOHEAD/admin/locales/fr" -H "$AUTH"
```

In the admin UI, locales are under **Settings → Locales**.

!!! warning "Choose the default locale before writing content"
    The default locale is the language of every entry's main data. Making another locale the default (`PUT /admin/locales/de {"isDefault": true}`) only **relabels** that data, so it's refused while any entry has a translation in the new default. Use it to fix the label: if your existing content is actually German, add `de` and make it the default before translating anything. On a new instance, set `GOHEAD_I18N_DEFAULT_LOCALE` before the first start.

## Localized attributes

Add `"localized": true` to an attribute. Any type can be localized except relations, which are shared by every locale. That includes components (the whole component value differs by locale) and media (a different image per language).

```json
{
  "name": "articles",
  "attributes": {
    "title":  { "type": "string", "required": true, "localized": true },
    "slug":   { "type": "uid", "unique": true, "localized": true },
    "body":   { "type": "richtext", "localized": true },
    "cover":  { "type": "media", "localized": true },
    "price":  { "type": "float" },
    "author": { "type": "relation", "relation": "oneToOne", "target": "authors" }
  }
}
```

- **Required** applies to the default locale. A translation can leave a value out, and readers then get the default locale's value.
- **Unique** applies within each locale: two articles can't share a French slug, but the French and English slugs of one article can be the same.
- In a type with localized attributes, `locale` and `locales` are reserved attribute names.
- Turning `localized` off on an attribute stops serving its translations. The values stay stored, and turning `localized` back on brings them back.

## Reading in a locale

Add `?locale=` to any read: lists, single entries and singletons.

```bash
curl "$GOHEAD/api/collections/articles/42?locale=fr" -H "$AUTH"
```

```json
{
  "data": {
    "id": 42,
    "title": "Bonjour le monde",
    "body": "…",
    "price": 9.5,
    "locale": "fr",
    "locales": ["en", "fr"],
    "createdAt": "…", "updatedAt": "…"
  }
}
```

- **Fallback:** a localized value with no translation is read from the default locale. Add `fallback=false` to leave it out instead, for example to find what still needs translating.
- **`locale` and `locales`:** every response of a type with localized attributes includes `locale` (the locale of the values) and `locales` (the default locale, then the locales that have a translation).
- **Default:** without `?locale=`, you get the default locale.
- **Unknown locales** return `400`.
- **Filters and sorts** use the requested locale: `?locale=fr&filters[slug]=bonjour-le-monde` finds the article by its French slug, and `sort=title` sorts by French title. With fallback, untranslated entries are matched and sorted on their default value.
- **Relations** are populated in the same locale.

## Writing translations

Create entries in the default locale, as before. Then write each locale with `PUT ?locale=`. The body holds **localized attributes only**, and it replaces that locale's values:

```bash
curl -X PUT "$GOHEAD/api/collections/articles/42?locale=fr" -H "$AUTH" \
  -H 'Content-Type: application/json' \
  -d '{"data": {"title": "Bonjour le monde", "slug": "bonjour-le-monde", "body": "…"}}'
```

- Shared attributes in the body are refused (`400 … is not localized: edit it in the default locale`).
- A `null` value removes that value from the translation, so it falls back to the default locale. `{"data": {}}` removes the whole translation.
- Values are validated like any write: types, enum options, media types, components, unique values within the locale.
- `"status"` in the body publishes or unpublishes the entry, the same as in the default locale.
- `POST` with `?locale=` other than the default returns `400`: create the entry first.
- Singletons work the same way: `PUT /api/singleton/homepage?locale=fr`, once the singleton has content in the default locale.

## GraphQL

Collections with localized attributes take `locale` and `fallback` arguments, and their types have `locale` and `locales` fields. Update mutations take `locale`. Updates are partial in GraphQL, so the input is merged over that locale's existing values.

```graphql
{
  articles(locale: "fr", limit: 10) {
    title
    locale
    author { name bio }   # read in French too
  }
}

mutation {
  updatearticles(id: "42", locale: "fr", input: { title: "Bonjour le monde" }) { title locales }
}
```

## History, webhooks and agents

- **History:** every revision keeps the translations. Diffs name translated values `<attribute>@<locale>`, for example `title@fr`, and restoring a version restores its translations too. See [content history](content-history.md).
- **Webhooks:** webhook payloads add a `translations` object (`{"fr": {"title": "…"}}`) next to `data`. `password` attributes are left out, as in `data`.
- **Agents:**
    - `collections.upsert_item` takes a `locale`: with `item_id`, it writes that locale's translation. That makes it easy to build a translation agent, for example one triggered on `item:created` that translates new articles into every locale.
    - `collections.list_locales` lists the locales.
    - `collections.list_items` returns each item's `translations`.

```json
{
  "name": "translator",
  "system_prompt": "An article was created. Translate its localized fields (title, body) into each non-default locale returned by list_locales, and save each translation with save_translation, passing item_id and locale. Keep the meaning and tone; do not translate product names.",
  "max_turns": 8,
  "llm_config": { "provider": "openai", "model": "gpt-4o-mini" },
  "trigger": {
    "type": "collection_event",
    "event_trigger": { "collection": "articles", "events": ["item:created"] }
  },
  "functions": [
    {
      "name": "list_locales",
      "description": "List the content locales",
      "impl_key": "collections.list_locales",
      "parameters": { "type": "object", "properties": {} }
    },
    {
      "name": "save_translation",
      "description": "Save an article's values in one locale",
      "impl_key": "collections.upsert_item",
      "collections": ["articles"],
      "parameters": {
        "type": "object",
        "properties": {
          "collection_name": { "type": "string" },
          "item_id": { "type": "string" },
          "locale": { "type": "string" },
          "data": { "type": "object" }
        },
        "required": ["collection_name", "item_id", "locale", "data"]
      }
    }
  ]
}
```

Each translation an agent writes is a revision tied to the agent's run, so it can be [reverted](content-history.md#reverting-an-agent-run) like any other agent change.

## In the admin UI

- **Settings → Locales** manages the locales.
- In the type editor, each attribute has a **Localized** switch. It isn't available on relations.
- The entry editor has a locale switcher.
    - In another locale, only localized attributes can be edited, and each one shows its default-locale value as a reference.
    - Shared attributes are edited in the default locale.
    - **Remove translation** deletes that locale's values.
- The content list can be viewed in any locale, and it shows which locales each entry has.

## Not covered

- **Fields only:** entries are translated attribute by attribute. A locale can't have its own entries, or entries missing from other locales. To publish something in one language only, use a shared attribute, for example a `languages` enum, and filter on it.
- **No per-locale publishing:** publishing applies to the entry in every locale.
- **Components:** attributes inside components can't be localized one by one. Localize the component attribute instead.
