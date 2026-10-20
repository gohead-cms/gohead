# Troubleshooting & FAQ

## The server does not start

| Log message | Fix |
|-------------|-----|
| `docker compose pull`: `Head "https://ghcr.io/v2/gohead-cms/…": unauthorized` | The images are private on GHCR. Maintainers: set the `gohead`, `gohead-admin` and `charts/gohead` packages to public (package settings → *Change visibility*). Meanwhile, build locally: `docker compose up -d --build`. |
| `pull access denied for minio/minio` | MinIO no longer publishes images. Update `docker-compose.yml` (uploads now use local disk by default), or point `GOHEAD_MEDIA_S3_*` at another S3-compatible store. |
| `failed to connect to database` | Check `GOHEAD_DATABASE_URL` and that the database accepts connections. |
| `Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo` | You are using a release binary with SQLite. Use PostgreSQL/MySQL, the container image, or `make build`. |
| `failed to read config file` | The file exists but is invalid YAML. A missing file is fine. |
| `jwt_secret is not set …` (warning) | Set `GOHEAD_JWT_SECRET`; otherwise tokens break on restart. |
| `media storage: cannot reach bucket …` / `bucket … does not exist` | Check `GOHEAD_MEDIA_S3_*`: endpoint, credentials, and that the bucket exists. Use `path_style: true` for MinIO and similar. |
| `Media uploads are disabled: the local media directory is not writable` (warning) | Point `GOHEAD_MEDIA_LOCAL_PATH` at a writable volume, or use the S3 driver. |

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

The schema is rebuilt in the API replica that handled the change and broadcast to the others
through Redis.

- With several replicas, check that they all use the same Redis (`GOHEAD_REDIS_ADDRESS`,
  `GOHEAD_REDIS_DB`). Logs show `Schema sync started` at startup and
  `Schema reloaded from another replica's change` when a broadcast arrives.
- After a Redis outage, replicas catch up within 30 seconds.
- Without Redis, only the replica that handled the change has the new schema: restart the API
  deployment, or run a single replica. See [scaling](deployment.md#scaling).

## 429 Too Many Requests on login

`/auth/*` is rate limited per client IP, and repeated failed logins lock that username out from
that IP for 15 minutes (see [rate limiting](../getting-started/configuration.md#rate-limiting)).
Wait for the `Retry-After` delay. If everyone is limited at once, the API probably sees the
ingress's IP for every user: set `GOHEAD_TRUSTED_PROXIES`.

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

## Uploads fail or files do not display

- `415`: the detected type is not allowed. The type comes from the file's content, so a
  renamed file keeps its real type. Extend `media.allowed_types` if needed.
- `413`: raise `media.max_size_mb`, and the body size limit of your ingress
  (`nginx.ingress.kubernetes.io/proxy-body-size`).
- Images break in the front-end with the local driver: URLs are relative (`/uploads/…`); set
  `GOHEAD_MEDIA_PUBLIC_URL` to the API's public URL.
- S3 URLs stop working after an hour: without `media.public_url` they are presigned. Fetch
  content again, or serve the bucket publicly / through a CDN and set `media.public_url`.

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
