# Standalone and Raspberry Pi

`gohead standalone` runs all of GoHead in one process, from one file: the API, the admin UI,
and the background jobs (content events, agents, webhooks, cron triggers). It needs no
Redis, no reverse proxy and no separate admin container; data goes to SQLite. It suits small
sites, personal projects, home servers and single-board computers such as a Raspberry Pi.

| | Standalone | Full deployment |
|---|---|---|
| Processes | 1 | API, worker, dispatcher, scheduler, admin UI |
| Services | none | Redis, PostgreSQL or MySQL |
| Database | SQLite (PostgreSQL and MySQL also work) | PostgreSQL or MySQL |
| Job queue | in the database | Redis |
| Memory | about 45 MB resident | about 30 MB per process, plus Redis and the database |
| Scaling | one process per database | API, workers and dispatchers scale out |

## Run it

Download the archive for your platform from the
[releases page](https://github.com/gohead-cms/gohead/releases) (on a Raspberry Pi:
`linux_arm64` with a 64-bit OS, `linux_armv7` with a 32-bit OS, `linux_armv6` on a Pi Zero or
Pi 1), then:

```bash
tar xzf gohead_*_linux_arm64.tar.gz
GOHEAD_ADMIN_USERNAME=admin \
GOHEAD_ADMIN_EMAIL=you@example.com \
GOHEAD_ADMIN_PASSWORD='choose-a-long-password' \
  ./gohead standalone
```

Open <http://raspberrypi.local:8080> (or the machine's address) and sign in. The API is on the
same address: `/api/…`, `/admin/…`, `/auth/…`.

The administrator variables are only needed for the first start. Everything else is kept in
the data directory (`./data`, or `--data <dir>`, or `GOHEAD_DATA_DIR`). GoHead creates the
database readable by its user only; keep the directory private too (`chmod 750`):

| File | Contents |
|------|----------|
| `gohead.db` (+ `-wal`, `-shm`) | the SQLite database: content, model, users, agents, queued jobs |
| `uploads/` | media files |
| `jwt_secret` | the key that signs sessions, generated at first start (keep it private) |

Settings you configure yourself win: `GOHEAD_DATABASE_URL`, `GOHEAD_MEDIA_LOCAL_PATH` and
`GOHEAD_JWT_SECRET` (or the same keys in `config.yaml`) replace the defaults above, so
standalone also runs on PostgreSQL or with S3 media.

With Docker, the image runs standalone too, with its data in the `/data` volume:

```bash
docker run -d --name gohead -p 8080:8080 -v gohead-data:/data \
  -e GOHEAD_ADMIN_USERNAME=admin -e GOHEAD_ADMIN_EMAIL=you@example.com \
  -e GOHEAD_ADMIN_PASSWORD='choose-a-long-password' \
  ghcr.io/gohead-cms/gohead:latest standalone
```

## Run it as a service

A systemd unit keeps it running and restarts it after a reboot:

```ini title="/etc/systemd/system/gohead.service"
[Unit]
Description=GoHead
After=network-online.target
Wants=network-online.target

[Service]
User=gohead
WorkingDirectory=/var/lib/gohead
ExecStart=/usr/local/bin/gohead standalone --data /var/lib/gohead
Environment=GOHEAD_LOG_LEVEL=warn
# Optional: a soft memory ceiling for the Go runtime.
Environment=GOMEMLIMIT=96MiB
Restart=on-failure
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=/var/lib/gohead

[Install]
WantedBy=multi-user.target
```

```bash
sudo useradd --system --home /var/lib/gohead --create-home gohead
sudo install -m 755 gohead /usr/local/bin/gohead
sudo systemctl daemon-reload && sudo systemctl enable --now gohead
journalctl -u gohead -f
```

For HTTPS, put Caddy or nginx in front (Caddy gets certificates by itself:
`caddy reverse-proxy --from cms.example.com --to localhost:8080`), and set
`GOHEAD_TRUSTED_PROXIES=127.0.0.1` so rate limits see the real client addresses.

## Resources

Measured on Linux after creating a collection, an entry and a webhook delivery (the CI runs
the same check on x86-64 and ARM64 hardware): about **45 MB** resident memory, and about **0.1 % of one CPU core** when idle
(the job queue checks the database once a second). The binary is about 55 MB, the admin UI
included.

To use less:

- `GOMEMLIMIT=64MiB` makes the Go runtime collect garbage sooner.
- `queue.poll_interval` (`GOHEAD_QUEUE_POLL_INTERVAL`, default `1s`): jobs queued by the
  process itself start at once; the interval only delays retries. `5s` is fine on a Pi.
- `queue.concurrency` (default `4` in standalone): jobs run at once. `1` or `2` on a Pi Zero.
- `GOHEAD_LOG_LEVEL=warn` writes less to the SD card.
- Keep the data directory on an SSD or a good SD card; SQLite runs in WAL mode.

## How the job queue works without Redis

Collection events, agent runs and webhook deliveries are rows of the `jobs` table. Workers in
the process claim them with a conditional update, so each job runs once; a job queued by the
process starts immediately, other processes' jobs are found by polling. Failed jobs are retried
with the same backoff as with Redis; a job whose process stopped is picked up again within
about 80 seconds (its lease is renewed every 20 seconds and lasts one minute). Agent runs
have their own workers and events theirs, so long agent runs never delay webhooks and event
dispatching for long. Finished jobs are deleted (webhook deliveries are kept for their retention period to
avoid duplicates); jobs that failed for good are kept 7 days.

The database queue also works with separate processes: set `queue.driver: database`
(`GOHEAD_QUEUE_DRIVER=database`) for `gohead start`, `worker`, `dispatcher` and `scheduler`,
on PostgreSQL or MySQL. Leases use each machine's clock: keep the machines synchronized
(NTP). Without Redis, rate limits are counted per process and content-model
changes are not broadcast between API processes, so run **one API process** per database.
Configure `redis.address` to share them: standalone uses Redis for those when it is set,
while keeping the database queue.

## Backups

Stop GoHead (or use SQLite's online backup) and copy the data directory:

```bash
sqlite3 data/gohead.db ".backup 'backup/gohead.db'"
rsync -a data/uploads/ backup/uploads/
```

Keep `jwt_secret` with your backups, or every session ends after a restore.

## Moving to a full deployment

Standalone and the full deployment use the same database schema. To grow, move the data to
PostgreSQL, set `GOHEAD_DATABASE_URL`, and start the processes separately (see
[Deployment](deployment.md)). Queued jobs in the `jobs` table are only read by the database
queue: let it drain before switching `queue.driver` to `redis`.
