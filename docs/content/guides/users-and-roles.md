# Users & roles

## Roles

Three roles are built in and created at startup:

| Role | Content | Publish | Media library | Admin area (`/admin/*`: content model, agents, users, tokens, settings) |
|------|---------|:---:|---------------|:---:|
| `admin`  | read, create, update, delete | ✓ | ✓ | ✓ |
| `editor` | read, create, update, delete | ✓ | ✓ |   |
| `viewer` | read (published content) |   |   |   |

You can add **custom roles** with permissions per content type: for example, a writer who drafts articles but can't publish, a reviewer who publishes but doesn't edit, or a team that only sees its own collections. The REST API and GraphQL enforce the same rules.

### Custom roles

A role's permissions list actions by **content type**:

```json
{
  "name": "writer",
  "description": "Drafts articles and news; publishing is left to editors",
  "permissions": {
    "content": {
      "articles": ["create", "update"],
      "news": ["create", "update", "delete"],
      "singleton:homepage": ["read"],
      "*": ["read"]
    },
    "media": ["read", "create"]
  }
}
```

| Key | Means | Actions |
|-----|-------|---------|
| a collection name | items of that collection | `read`, `create`, `update`, `delete`, `publish` |
| `singleton:<name>` | a singleton's content | `read`, `update` |
| `*` | every type not listed | as above |
| `media` | the media library | `read` (browse), `create` (upload), `update`, `delete` |

- **Unlisted types:** a type that isn't listed uses `*`. If `*` isn't listed either, the role can't reach that type: it gets `403`, and the type doesn't appear in `/api/schema/*`.
- **Read is implied:** any other action includes `read`, and the server adds it.
- **Drafts** are visible to roles that can `update` or `publish` the type, both directly and through `status=all`. Other roles see published content only.
- **Publishing:** `publish` covers `POST …/publish` and `…/unpublish`, and `"status"` in a create or update body. In [draft-and-publish](content-modeling.md#draft-and-publish) collections it is also needed to change an item that is **published**, because that changes live content. Without `publish`, a role writes and edits drafts only, which suits a writer whose work is reviewed before going live. It can still delete published items if it has `delete`.
- **No review step without draft and publish:** in collections without draft and publish, every item is live, so `create` and `update` publish directly.
- **History:** reading history and restoring versions need `update` on the type. Restoring also needs `create` when it brings back a deleted item, and `publish` when the item is published or would come back published. The [change feed](content-history.md#change-feed) lists only the types the role can update.
- **Related entries** of a type the role can't read are left out of populated relations: they're returned as `null`, or dropped from lists, including bare IDs at `level=0`. Drafts of a related type are only shown to roles that can see drafts of that type.
- **Linking by ID:** relation and media values are IDs. A role that can write a type can link any existing item or file by ID, so a missing ID is reported in the validation error. The linked content stays hidden when the role can't read it, except media, which is shown with the content that uses it. Keep content that must stay private out of reach of roles that can write the types linking to it.
- **Reserved for admins:** the admin area stays admin-only, for every custom role. That covers the content model, agents (and reverting their runs), users, tokens, webhooks, locales and roles.

Manage roles as an admin, from a user session, or in the admin UI under **Settings → Roles**:

| Method | Path | Body |
|--------|------|------|
| `GET` | `/admin/roles` | |
| `GET` | `/admin/roles/{name}` | |
| `POST` | `/admin/roles` | `name` (2–32 lowercase letters, digits, `-`, `_`), `description`, `permissions` |
| `PUT` | `/admin/roles/{name}` | `description`, `permissions` (replaced; the name can't change) |
| `DELETE` | `/admin/roles/{name}` | `409` while users or active API tokens still use the role |

Built-in roles can't be changed or deleted. Assign a custom role like any other: `role_name` for [users](#managing-users-admin), `role` for [API tokens](integrations.md#api-tokens). That makes it easy to give a front-end a token limited to the collections it displays. A permission change, or giving a user another role, applies to existing sessions and tokens within a few seconds, with no new login needed.

`GET /auth/me` returns the caller's `permissions` in this format. The built-in roles are expanded into it, so a client can show or hide actions.

### Public access

Requests **without a token** act with the `public` role. It grants nothing at first, so
anonymous requests get `401`. Grant it `read` on the types your website or app shows, and
browsers, mobile apps and static site builds can read them without a token:

```bash
curl -X PUT $GOHEAD/admin/roles/public -H "$JSON" -H "$AUTH" -d '{
  "description": "Website visitors",
  "permissions": {
    "content": { "articles": ["read"], "categories": ["read"], "singleton:homepage": ["read"] },
    "media": ["read"]
  }
}'

curl $GOHEAD/api/collections/articles   # no Authorization header
```

In the admin UI: **Settings → Roles → Public**.

- **Read only, published only.** The role can only hold `read`; visitors never see drafts,
  history or proposals, and never write.
- **What stays private:** related entries of types the role can't read are left out, as for
  any role. The content model (`/api/schema/*`, `/api/openapi.json` unless
  `openapi.public` is on), GraphQL introspection and the admin area need a token. GraphQL
  queries work anonymously for the granted types.
- **Rate limit:** anonymous requests are limited per client IP
  ([`rate_limit.public_per_minute`](../getting-started/configuration.md#rate-limiting), 600 a minute).
- A request that sends a token always acts as that token, even with public access on, and
  a forbidden anonymous request gets `401`, so a client knows a token may help.
- The `public` role can't be deleted or given to users and API tokens.

Anything granted here is readable by anyone on the internet. For content that only your
own servers should read, keep public access off and use an [API token](integrations.md#api-tokens).

## Registration and the first admin

The first administrator never comes from the public API. Create it either way:

- **Configuration**: set `admin.username`, `admin.email` and `admin.password`
  (`GOHEAD_ADMIN_USERNAME`, `GOHEAD_ADMIN_EMAIL`, `GOHEAD_ADMIN_PASSWORD`). The API creates
  that account at startup when the database has no users, and ignores the settings
  afterwards, so they can stay in place (or be removed).
- **Command line**, at any time:

    ```bash
    echo "$PASSWORD" | gohead admin create --username alice --email alice@example.com --password-stdin
    echo "$PASSWORD" | gohead admin set-password --username alice --password-stdin   # recovery
    ```

Administrator passwords need at least 12 characters.

Other accounts are created by an admin (`POST /admin/users`, or the admin UI). Public
self-registration is controlled by `auth.registration` (`GOHEAD_AUTH_REGISTRATION`):

| Value | `POST /auth/register` |
|-------|-----------------------|
| `disabled` *(default)* | `403` |
| `open` | creates a **viewer** account; the caller cannot choose a role |

```bash
curl -X POST $GOHEAD/auth/register -H "$JSON" \
  -d '{"username":"alice","email":"alice@example.com","password":"at-least-6-chars"}'
```

## Logging in

```bash
curl -X POST $GOHEAD/auth/login -H "$JSON" -d '{"username":"alice","password":"at-least-6-chars"}'
# {"data":{"token":"eyJhbGciOiJIUzI1NiIs..."}}
```

Send the token as `Authorization: Bearer <token>`. Tokens are HS256 JWTs that expire after
**72 hours** (for front-ends and CI, use a long-lived [API token](integrations.md#api-tokens)); they are signed with [`jwt_secret`](../getting-started/configuration.md#server).
Changing the secret invalidates every token.

A wrong username or password returns `401` with the same message either way. After 5 failed
attempts for a username from one IP, that IP gets `429` with a `Retry-After` header for 15
minutes; a successful login clears the count. Limits are
[configurable](../getting-started/configuration.md#rate-limiting).

## Managing users (admin)

| Method | Path | Body |
|--------|------|------|
| `GET` | `/admin/users` | |
| `GET` | `/admin/users/{id}` | |
| `POST` | `/admin/users` | `username`, `email`, `password`, optional `role_name` (default `viewer`) |
| `PUT` | `/admin/users/{id}` | any of `username`, `email`, `password`, `role_name` |
| `DELETE` | `/admin/users/{id}` | |

```bash
# invite an editor
curl -X POST $GOHEAD/admin/users -H "$JSON" -H "$AUTH" \
  -d '{"username":"bob","email":"bob@example.com","password":"temporary-pass","role_name":"editor"}'

# promote alice
curl -X PUT $GOHEAD/admin/users/2 -H "$JSON" -H "$AUTH" -d '{"role_name":"editor"}'
```

Rules:

- passwords are hashed with bcrypt and never returned;
- `PUT` rejects any field not listed above;
- duplicate usernames or emails return `409`;
- an admin cannot delete their own account.

Role changes apply at the user's next login (the role is embedded in the token).
