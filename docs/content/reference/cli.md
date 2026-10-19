# CLI

```text
gohead <command> [--config path]
```

| Command | Description |
|---------|-------------|
| `start` | Run the HTTP API: applies pending migrations (unless `auto_migrate` is false), seeds roles, creates the configured first admin. Graceful shutdown on SIGINT/SIGTERM. |
| `migrate` | Apply pending [database migrations](../operations/upgrading.md#database-migrations) and exit. |
| `migrate status` | List migrations and when they ran; exits non-zero if some are pending. |
| `migrate down [--steps N]` | Revert the last N migrations (only those that can be undone). |
| `admin create --username U --email E --password-stdin` | Create an administrator (password: 12+ characters, from stdin). |
| `admin set-password --username U --password-stdin` | Replace a user's password, e.g. to recover the admin account. |
| `worker` | Run agent jobs from the `agents` queue (10 concurrent jobs). |
| `dispatcher` | Turn content events from the `events` queue into agent jobs. |
| `scheduler` | Enqueue cron-triggered agent runs; reloads schedules every minute. Run one instance. |
| `version` | Print version, commit and build date. |

All long-running commands accept `-c, --config` (default `config.yaml`, optional) and read
`GOHEAD_*` environment variables, see [configuration](../getting-started/configuration.md).

```bash
gohead start --config /etc/gohead/config.yaml
GOHEAD_LOG_LEVEL=debug gohead worker
gohead version
gohead migrate status
echo "$PASSWORD" | gohead admin create --username alice --email alice@example.com --password-stdin
```
