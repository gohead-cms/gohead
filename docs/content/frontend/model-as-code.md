# Content models as code

The `gohead-client` command line (in `@gohead-cms/client`) generates TypeScript types, keeps
your content model in a file you commit, and loads sample content. Run it with `npx`:

```bash
npx @gohead-cms/client <command>
```

| Command | Does |
|---------|------|
| `types [--out gohead.d.ts] [--model <file>]` | TypeScript types of the content model, from the instance or a model file |
| `model pull [--out gohead/model.json]` | Writes the instance's content model to a file |
| `model push [<file>] [--dry-run] [--allow-removals] [--allow-publish-drafts]` | Applies a model file (default `gohead/model.json`) |
| `seed [<file>] [--force]` | Uploads media and creates sample entries (default `gohead/seed.json`) |
| `token create --name <name> [--role viewer] [--days <n>]` | Creates an API token and prints it |

**Connection:** `--url` (`GOHEAD_URL`, default `http://localhost:8080`), and `--token`
(`GOHEAD_TOKEN`) or `--username` and `--password` (`GOHEAD_USERNAME`, `GOHEAD_PASSWORD`).
`.env.local` and `.env` in the current directory are read. `model push` and `token create`
need an **admin user** (API tokens can't manage roles or tokens); `types`, `model pull` and
`seed` work with an admin API token too.

## The model file

```json
{
  "$schema": "https://gohead-cms.github.io/gohead/schemas/model.json",
  "locales": [
    { "code": "en", "name": "English", "default": true },
    { "code": "fr", "name": "Français" }
  ],
  "components": {
    "seo": { "attributes": { "meta_title": { "type": "string" }, "meta_description": { "type": "text" } } }
  },
  "collections": {
    "articles": {
      "description": "Blog posts",
      "draftAndPublish": true,
      "attributes": {
        "title": { "type": "string", "required": true, "localized": true },
        "slug": { "type": "uid", "required": true, "unique": true },
        "body": { "type": "richtext", "localized": true },
        "cover": { "type": "media", "allowed_types": ["image/*"] },
        "author": { "type": "relation", "relation": "oneToOne", "target": "authors" },
        "seo": { "type": "component", "component": "seo" }
      }
    },
    "authors": { "attributes": { "name": { "type": "string", "required": true } } }
  },
  "singletons": {
    "homepage": { "attributes": { "headline": { "type": "string", "localized": true } } }
  },
  "public": {
    "content": { "articles": ["read"], "authors": ["read"], "singleton:homepage": ["read"] }
  }
}
```

Attributes take the same settings as the [content model API](../guides/content-modeling.md).
`public` sets what anonymous visitors may read ([public access](../guides/users-and-roles.md#public-access)).
The `$schema` line gives editors completion and validation.

### Pushing

```bash
npx @gohead-cms/client model push --dry-run   # show what would change
npx @gohead-cms/client model push
```

```text
Applying gohead/model.json to http://localhost:8080:
  create locale fr
  create component seo (+meta_title, +meta_description)
  create collection authors (+name)
  create collection articles (+title, +slug, +body, +cover, +author, +seo)
  update public access
Done (5 changes).
```

- Locales and components first, then collections (relations between new collections,
  cycles included, are handled), singletons and public access.
- Types that exist only on the server are **never deleted**.
- Removing an attribute from a type is refused unless you pass `--allow-removals`: its
  values stay in the database but leave the API.
- Public access is **added to**: types the server opens that the model doesn't list stay
  open (the plan says so) unless you pass `--allow-removals`.
- Turning `draftAndPublish` off publishes every draft of the collection, so it is refused
  unless you pass `--allow-publish-drafts`. A collection without `draftAndPublish` in the
  file has it off.
- Pushing the same file again changes nothing, so it fits CI and deployment scripts.

### Pulling

`model pull` writes the instance's model in the same format, sorted, so changes made in the
admin UI show up as a readable diff.

## TypeScript types

```bash
npx @gohead-cms/client types --out src/gohead.d.ts
# or offline, from the file:
npx @gohead-cms/client types --model gohead/model.json --out src/gohead.d.ts
```

For each collection, singleton and component, the file declares the entry as read
(`Article`) and as written (`ArticleInput`), and registers them with the client:

```ts
export interface Article {
  title: string;                        // required
  slug: string;
  body?: string | null;
  cover?: Media | number | null;        // a file (an ID at level 0)
  author?: Entry<Author> | number | null; // a relation: populated, or an ID below the read's level
  seo?: Seo | null;
}

declare module "@gohead-cms/client" {
  interface Collections {
    articles: { attributes: Article; input: ArticleInput };
  }
}
```

| Attribute type | Read as | Written as |
|----------------|---------|------------|
| `string`, `text`, `richtext`, `email`, `uid`, `date`, `datetime`, `time` | `string` | `string` |
| `int`, `float`, `decimal` | `number` | `number` |
| `bool` | `boolean` | `boolean` |
| `enum` | a union of its options | the same |
| `json` | `JsonValue` | `JsonValue` |
| `media` | `Media \| number` (a list if `multiple`) | file ID(s) |
| `relation` | `Entry<Target> \| number` (a list for `manyToMany`) | ID(s) or objects to create |
| `component` | the component's interface (a list if `repeatable`) | the same |
| `password` | not returned | `string` |

## Sample content

A seed file creates entries in order, with references between them:

```json
{
  "media": {
    "ada": { "file": "media/ada.jpg", "alt": "Ada" }
  },
  "collections": {
    "authors": [
      { "$id": "ada", "name": "Ada Lovelace", "avatar": "@media:ada" }
    ],
    "articles": [
      {
        "$id": "hello",
        "$status": "published",
        "title": "Hello",
        "slug": "hello",
        "author": "@ref:authors/ada",
        "$translations": { "fr": { "title": "Bonjour" } }
      }
    ]
  },
  "singletons": {
    "homepage": { "headline": "Welcome", "$translations": { "fr": { "headline": "Bienvenue" } } }
  }
}
```

| Key | |
|-----|-|
| `@media:<key>` | the ID of a file declared under `media` (paths are relative to the seed file) |
| `@ref:<collection>/<$id>` | the ID of an entry created earlier in the file |
| `$status` | `published` or `draft` in draft-and-publish collections |
| `$translations` | localized values per locale |

Collections that already have entries, and singletons that have content, are skipped, so
running it twice doesn't duplicate anything; `--force` seeds them anyway. References to
entries of a skipped collection are found by their `slug`. Every reference is checked
before anything is uploaded or created.

## In CI

Apply the model on deploy, and check that generated types are up to date:

```yaml
- run: npx @gohead-cms/client model push gohead/model.json
  env:
    GOHEAD_URL: ${{ vars.GOHEAD_URL }}
    GOHEAD_USERNAME: ${{ secrets.GOHEAD_USERNAME }}
    GOHEAD_PASSWORD: ${{ secrets.GOHEAD_PASSWORD }}
- run: npx @gohead-cms/client types --model gohead/model.json --out src/gohead.d.ts && git diff --exit-code src/gohead.d.ts
```
