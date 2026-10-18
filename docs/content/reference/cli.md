# CLI

```text
gohead <command> [--config path]
```

| Command | Description |
|---------|-------------|
| `start` | Run the HTTP API (migrates the database, seeds roles). Graceful shutdown on SIGINT/SIGTERM. |
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
```
