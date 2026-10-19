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
- **Publishing:** `publish` covers `POST …/publish` and `…/unpublish`, and `"status"` in a create or update body. Without it, items stay drafts, which suits a writer whose work is reviewed before going live.
- **History** and restoring versions need `update` on the type. The [change feed](content-history.md#change-feed) lists only the types the role can update.
- **Related entries** of a type the role can't read are left out of populated relations: they're returned as `null`, or dropped from lists.
- **Reserved for admins:** the admin area stays admin-only, for every custom role. That covers the content model, agents (and reverting their runs), users, tokens, webhooks, locales and roles.

Manage roles as an admin, from a user session, or in the admin UI under **Settings → Roles**:

| Method | Path | Body |
|--------|------|------|
| `GET` | `/admin/roles` | |
| `GET` | `/admin/roles/{name}` | |
| `POST` | `/admin/roles` | `name` (2–32 lowercase letters, digits, `-`, `_`), `description`, `permissions` |
| `PUT` | `/admin/roles/{name}` | `description`, `permissions` (replaced; the name can't change) |
| `DELETE` | `/admin/roles/{name}` | `409` while users or active API tokens still use the role |

Built-in roles can't be changed or deleted. Assign a custom role like any other: `role_name` for [users](#managing-users-admin), `role` for [API tokens](integrations.md#api-tokens). That makes it easy to give a front-end a token limited to the collections it displays. A permission change applies to existing sessions and tokens within a few seconds, with no new login needed.

`GET /auth/me` returns the caller's `permissions` in this format. The built-in roles are expanded into it, so a client can show or hide actions.

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
