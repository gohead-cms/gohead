# Installation

GoHead is a single binary (`gohead`) with four commands: `start` (the API), `worker`,
`dispatcher` and `scheduler`. Only `start` is required for content management; the other
three run [agents](../agents/index.md) and need Redis.

| Method                         | Best for                          | SQLite | Agents |
|--------------------------------|-----------------------------------|:------:|:------:|
| [Docker Compose](#docker-compose) | Trying everything locally      | yes    | yes    |
| [Container image](#container-image) | Servers, Kubernetes          | yes    | yes    |
| [Release binary](#release-binary)   | Bare-metal with Postgres/MySQL | no¹    | yes    |
| [From source](#from-source)         | Development                  | yes    | yes    |

¹ Release binaries are built without cgo, and the SQLite driver needs cgo. Use PostgreSQL or
MySQL with them, or use the container image.

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

The admin UI image is `ghcr.io/gohead-cms/gohead-admin`; point it at your API with
`GOHEAD_API_URL`. Images are signed, see [verifying images](../operations/security.md#verify-container-images).

## Release binary

Download an archive for your platform from the
[releases page](https://github.com/gohead-cms/gohead/releases), check it against
`checksums.txt`, then:

```bash
tar xzf gohead_*_linux_amd64.tar.gz
./gohead version
GOHEAD_DATABASE_URL="postgres://user:pass@localhost:5432/gohead?sslmode=disable" \
GOHEAD_JWT_SECRET="$(openssl rand -hex 32)" ./gohead start
```

## From source

Requirements: Go 1.25+, a C compiler (for SQLite), Node 22 for the admin UI.

```bash
git clone https://github.com/gohead-cms/gohead.git && cd gohead
make build                       # -> dist/gohead
cp config.sample.yaml config.yaml
./dist/gohead start --config config.yaml
```

Check the server:

```bash
curl localhost:8080/_health
# {"status":"ok"}
```

Next: the [quick start](quick-start.md).
