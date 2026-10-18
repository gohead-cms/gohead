# Users & roles

## Roles

| Role | Content: read | Content: create / update / delete | Schema, users, agents (`/admin/*`) |
|------|:---:|:---:|:---:|
| `admin`  | ✓ | ✓ | ✓ |
| `editor` | ✓ | ✓ |   |
| `viewer` | ✓ |   |   |

The three roles are created automatically at startup. The same table is enforced by the REST
API and by GraphQL resolvers.

## Registration and the first admin

`POST /auth/register` is public, but the caller **cannot choose a role**:

- the first account created on an empty instance becomes `admin`;
- every later account becomes `viewer`.

```bash
curl -X POST $GOHEAD/auth/register -H "$JSON" \
  -d '{"username":"alice","email":"alice@example.com","password":"at-least-6-chars"}'
```

!!! tip "Lock the first account"
    Register the admin account right after the first deployment, before the URL is public.

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
