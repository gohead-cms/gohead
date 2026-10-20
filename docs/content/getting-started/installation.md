# Installation

GoHead is a single binary (`gohead`), the admin UI included. It runs in one of two ways:

- **Standalone**: `gohead standalone` runs everything in one process (API, admin UI, agents,
  webhooks) with SQLite and no other service. See
  [Standalone and Raspberry Pi](../operations/standalone.md).
- **Separate processes**: `start` (the API and admin UI), `worker`, `dispatcher` and
  `scheduler`, with Redis and PostgreSQL or MySQL, to scale out. Only `start` is required for
  content management; the other three run [agents](../agents/index.md) and webhooks.

| Method                         | Best for                          |
|--------------------------------|-----------------------------------|
| [Release binary](#release-binary)   | Standalone, small servers, Raspberry Pi |
| [Docker Compose](#docker-compose) | Trying the full stack locally  |
| [Container image](#container-image) | Servers, Kubernetes          |
| [From source](#from-source)         | Development                  |

## Docker Compose

The repository ships a `docker-compose.yml` with the API, the three background processes,
the admin UI, PostgreSQL and Redis.

```bash
git clone https://github.com/gohead-cms/gohead.git
cd gohead
echo "GOHEAD_JWT_SECRET=$(openssl rand -hex 32)" > .env
docker compose up -d            # published images
# or: docker compose up -d --build   (build from your checkout)
```

| Service    | URL / role                          |
|------------|-------------------------------------|
| `api`      | <http://localhost:8080>             |
| `admin`    | <http://localhost:3000> (admin UI)  |
| `worker`, `dispatcher`, `scheduler` | background agents |
| `postgres`, `redis` | data stores                |

Set `OPENAI_API_KEY` (or `ANTHROPIC_API_KEY`) in `.env` to run agents.

## Container image

Images are published to GitHub Container Registry for `linux/amd64` and `linux/arm64`:

```bash
docker run -d --name gohead -p 8080:8080 \
  -e GOHEAD_JWT_SECRET="$(openssl rand -hex 32)" \
  -v gohead-data:/data \
  ghcr.io/gohead-cms/gohead:latest
```

Without `GOHEAD_DATABASE_URL`, the image stores a SQLite database in `/data`. Tags:

| Tag                       | Meaning                                    |
|---------------------------|--------------------------------------------|
| `latest`, `X.Y.Z`, `X.Y`  | Releases                                   |
| `edge`                    | Latest `master` build                      |
| `master-<sha>-<timestamp>`| Every `master` build (used by Flux staging)|

The image serves the admin UI at `/` too. To serve it separately, the admin UI image is
`ghcr.io/gohead-cms/gohead-admin`; point it at your API with `GOHEAD_API_URL`. Run
`… gohead:latest standalone` for a single container with agents and webhooks (no Redis); it
keeps its data in `/data`. Images are signed, see [verifying images](../operations/security.md#verify-container-images).

## Release binary

Download an archive for your platform from the
[releases page](https://github.com/gohead-cms/gohead/releases) (Linux and macOS on x86-64 and
ARM64, Windows on x86-64, Linux on ARMv7 and ARMv6), check it against `checksums.txt`, then:

```bash
tar xzf gohead_*_linux_amd64.tar.gz
./gohead version
GOHEAD_ADMIN_USERNAME=admin GOHEAD_ADMIN_EMAIL=you@example.com \
GOHEAD_ADMIN_PASSWORD='choose-a-long-password' ./gohead standalone
```

Open <http://localhost:8080>: the admin UI and the API share the address. Data goes to
`./data`. To run the processes separately on PostgreSQL instead:

```bash
GOHEAD_DATABASE_URL="postgres://user:pass@localhost:5432/gohead?sslmode=disable" \
GOHEAD_JWT_SECRET="$(openssl rand -hex 32)" ./gohead start
```

## From source

Requirements: Go 1.26+ and Node 22 (for the admin UI). No C compiler: SQLite is pure Go.

```bash
git clone https://github.com/gohead-cms/gohead.git && cd gohead
make ui build                    # admin UI, then -> dist/gohead
cp config.sample.yaml config.yaml
./dist/gohead start --config config.yaml
```

`make build` alone builds a binary without the admin UI (run the UI with `npm run dev` in
`gui/`).

Check the server:

```bash
curl localhost:8080/_health
# {"status":"ok"}
```

Next: the [quick start](quick-start.md).
