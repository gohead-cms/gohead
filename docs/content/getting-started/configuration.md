# Configuration

GoHead reads an optional YAML file (`--config`, default `config.yaml`) and then
environment variables, which win. A missing file is not an error, so containers can be
configured entirely with environment variables.

**Environment variable names:** prefix `GOHEAD_`, upper case, dots become underscores.
`redis.address` → `GOHEAD_REDIS_ADDRESS`. List values (CORS) are comma-separated.

A fully commented [`config.sample.yaml`](https://github.com/gohead-cms/gohead/blob/master/config.sample.yaml)
lives at the root of the repository.

## Reference

### Server

| Key | Env | Default | Description |
|-----|-----|---------|-------------|
| `server_port` | `GOHEAD_SERVER_PORT` | `8080` | HTTP port of the API. |
| `mode` | `GOHEAD_MODE` | `production` | Gin mode: `development`, `production`, `test`. |
| `log_level` | `GOHEAD_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error`. Logs are JSON on stdout. |
| `jwt_secret` | `GOHEAD_JWT_SECRET` | *(random)* | Signs login tokens (HS256). See below. |
| `telemetry_enabled` | `GOHEAD_TELEMETRY_ENABLED` | `false` | Export OpenTelemetry traces over OTLP/gRPC. |

!!! warning "Set `jwt_secret`"
    If it is empty or left at the sample value `your-secret-key`, GoHead generates a random
    secret at startup and logs a warning. That is safe, but every token becomes invalid when
    the process restarts, and multiple API replicas will reject each other's tokens.
    Use `openssl rand -hex 32`.

### Database

| Key | Env | Default |
|-----|-----|---------|
| `database_url` | `GOHEAD_DATABASE_URL` | `sqlite://gohead-cms.db` (`sqlite:///data/gohead.db` in the image) |

Supported URLs:

=== "PostgreSQL"

    ```
    postgres://user:password@host:5432/gohead?sslmode=require
    ```

=== "MySQL"

    ```
    mysql://user:password@tcp(host:3306)/gohead?charset=utf8mb4&parseTime=True&loc=Local
    ```

=== "SQLite"

    ```
    sqlite://path/to/gohead.db
    ```

    Needs a cgo build (container image or `make build`). One writer at a time: fine for
    development and small sites.

The schema is migrated automatically when the API starts.

### Redis

| Key | Env | Default |
|-----|-----|---------|
| `redis.address` | `GOHEAD_REDIS_ADDRESS` | `localhost:6379` |
| `redis.password` | `GOHEAD_REDIS_PASSWORD` | *(empty)* |
| `redis.db` | `GOHEAD_REDIS_DB` | `0` |

Redis carries content events and agent jobs, and lets API replicas cooperate: GraphQL schema
changes are broadcast to every replica, and rate limits are shared. The API keeps working
without it (events are dropped with an error log, limits are counted per process), but agents
will not run and you should keep a single API replica.

### Media

Uploads and file storage (local disk or S3-compatible) are configured under `media.*`; see
[Media → Configuration](../guides/media.md#configuration).

### Rate limiting

| Key | Env | Default |
|-----|-----|---------|
| `rate_limit.enabled` | `GOHEAD_RATE_LIMIT_ENABLED` | `true` |
| `rate_limit.auth_per_minute` | `GOHEAD_RATE_LIMIT_AUTH_PER_MINUTE` | `20` |
| `rate_limit.login_failures` | `GOHEAD_RATE_LIMIT_LOGIN_FAILURES` | `5` |
| `rate_limit.login_window` | `GOHEAD_RATE_LIMIT_LOGIN_WINDOW` | `15m` |
| `trusted_proxies` | `GOHEAD_TRUSTED_PROXIES` | *(empty)* |

- `auth_per_minute` caps requests to `/auth/login` and `/auth/register` per client IP.
- After `login_failures` failed logins for one username from one IP, further attempts from that
  IP are refused for `login_window` (other IPs, and a successful login, are unaffected). Set it
  to `0` to disable the lockout.
- Refused requests get `429 Too Many Requests` with a `Retry-After` header.
- Counters live in Redis, so the budget is shared by all API replicas. If Redis is down, each
  replica falls back to its own in-memory counters.
- `trusted_proxies` lists the IPs or CIDRs of your load balancer / ingress (comma-separated in
  the environment variable). Only requests from those addresses may set the client IP through
  `X-Forwarded-For`; when it is empty the header is ignored. Behind an ingress, set it, or every
  user shares the ingress's IP and its budget.

### CORS

| Key | Env | Default |
|-----|-----|---------|
| `cors.allowed_origins` | `GOHEAD_CORS_ALLOWED_ORIGINS` | `*` |
| `cors.allowed_methods` | `GOHEAD_CORS_ALLOWED_METHODS` | `GET,POST,PUT,DELETE,OPTIONS` |
| `cors.allowed_headers` | `GOHEAD_CORS_ALLOWED_HEADERS` | `Origin,Accept,Content-Type,Authorization,X-Requested-With,Webhook-Token` |
| `cors.allow_credentials` | `GOHEAD_CORS_ALLOW_CREDENTIALS` | `true` |
| `cors.max_age` | `GOHEAD_CORS_MAX_AGE` | `86400` |

In production, list your front-end origins explicitly instead of `*`.

### LLM defaults

Instance-wide defaults for [agents](../agents/llm-providers.md). Each agent can override them in
its own `llm_config`.

| Key | Env | Default |
|-----|-----|---------|
| `llm.provider` | `GOHEAD_LLM_PROVIDER` | `openai` |
| `llm.model` | `GOHEAD_LLM_MODEL` | `gpt-4o` |
| `llm.api_key` | `GOHEAD_LLM_API_KEY` | `$OPENAI_API_KEY` |
| `llm.base_url` | `GOHEAD_LLM_BASE_URL` | *(provider default)* |

Provider keys can also come from `OPENAI_API_KEY` / `ANTHROPIC_API_KEY`.

### Tracing

When `telemetry_enabled` is true, traces go to the OTLP endpoint given by the standard
OpenTelemetry variables, for example `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317`.

## Admin UI

The admin UI container reads one variable:

| Env | Description |
|-----|-------------|
| `GOHEAD_API_URL` | Public URL of the API as seen from the browser, e.g. `https://api.example.com`. |

## Example: production environment

```bash
GOHEAD_MODE=production
GOHEAD_LOG_LEVEL=info
GOHEAD_DATABASE_URL=postgres://gohead:…@db.internal:5432/gohead?sslmode=require
GOHEAD_JWT_SECRET=…64 hex chars…
GOHEAD_REDIS_ADDRESS=redis.internal:6379
GOHEAD_REDIS_PASSWORD=…
GOHEAD_CORS_ALLOWED_ORIGINS=https://admin.example.com,https://www.example.com
OPENAI_API_KEY=…
```
