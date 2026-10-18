# Troubleshooting & FAQ

## The server does not start

| Log message | Fix |
|-------------|-----|
| `failed to connect to database` | Check `GOHEAD_DATABASE_URL` and that the database accepts connections. |
| `Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo` | You are using a release binary with SQLite. Use PostgreSQL/MySQL, the container image, or `make build`. |
| `failed to read config file` | The file exists but is invalid YAML. A missing file is fine. |
| `jwt_secret is not set …` (warning) | Set `GOHEAD_JWT_SECRET`; otherwise tokens break on restart. |

## Every request returns 401

- The header must be `Authorization: Bearer <token>`.
- Tokens expire after 72 hours, and become invalid when `jwt_secret` changes or when the API
  restarts without a configured secret.
- With several API replicas, all must share the same `GOHEAD_JWT_SECRET`.

## 403 on a write

Your role lacks the permission. Viewers can only read; schema routes (`/admin/*`) need admin.
An admin can promote you with `PUT /admin/users/{id}` `{"role_name":"editor"}`; log in again
afterwards.

## A new collection is missing from GraphQL

The schema is rebuilt in the API replica that handled the change. With several replicas,
restart the API deployment. See [scaling](deployment.md#scaling).

## The admin UI cannot reach the API

- `GOHEAD_API_URL` must be the URL the **browser** uses, not an internal service name.
- The UI's origin must be in `GOHEAD_CORS_ALLOWED_ORIGINS`.

## Agents never run

1. Are the `worker` and `dispatcher` (and `scheduler` for cron) running, with the same
   database and Redis as the API?
2. API logs show `Failed to enqueue collection event`? Redis is unreachable from the API.
3. Worker logs show `could not create LLM adapter`? Missing API key: set it in the agent's
   `llm_config.api_key` or `OPENAI_API_KEY`/`ANTHROPIC_API_KEY` on the worker.
4. The event list must match exactly (`item:created`, not `created`), and the collection name
   too.
5. Agent writes do not emit events, by design.

## Filtering returns 400

The field is unknown or not filterable (relations, JSON, media), the operator does not apply to
the type (`$contains` on a number, `$gt` on a boolean), or a number filter has a non-numeric
value. The error message says which.

## FAQ

**Is GoHead open source?** Yes, MIT-licensed.

**Which database should I use?** PostgreSQL in production. MySQL works. SQLite is for
development and small single-instance sites.

**Can I run without Redis?** Yes, for pure content management. Events are dropped (logged) and
agents do not run.

**How do I reset an admin password?** Another admin can `PUT /admin/users/{id}`
`{"password": "…"}`. If no admin is left, update the user's role directly in the database.

**Where are files/media stored?** `media` attributes hold URLs or paths; uploads are not
handled by GoHead yet.
