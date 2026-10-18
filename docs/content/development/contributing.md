# Contributing

Thanks for helping! Bugs, docs, tests and features are all welcome.

## Setup

Requirements: Go 1.26+, a C compiler (SQLite), Node 22 (admin UI), Docker (optional),
Python 3.12 (docs, optional).

```bash
git clone https://github.com/<you>/gohead.git && cd gohead
git remote add upstream https://github.com/gohead-cms/gohead.git
make build && cp config.sample.yaml config.yaml && ./dist/gohead start
```

Run everything (API, workers, UI, Postgres, Redis) from your checkout:

```bash
make up        # docker compose up -d --build
```

## Tests

```bash
make test-race                                   # all Go tests, SQLite in memory
GOHEAD_TEST_DATABASE_URL=postgres://gohead:gohead@localhost:5432/gohead?sslmode=disable \
  make test-db                                   # DB tests against PostgreSQL
GOHEAD_TEST_DATABASE_URL='mysql://root:gohead@tcp(127.0.0.1:3306)/gohead?charset=utf8mb4&parseTime=True&loc=UTC' \
  make test-db                                   # ... or MySQL
make lint                                        # gofmt + go vet
```

Tests that touch the database use `pkg/testutils.SetupTestServer()`, which gives you a fresh
database with the default roles. With `GOHEAD_TEST_DATABASE_URL` set, that database is
**wiped** before each test (PostgreSQL: the `public` schema; MySQL: the database is dropped
and re-created): use a throwaway database.

Throwaway servers:

```bash
docker run -d --rm -p 5432:5432 -e POSTGRES_USER=gohead -e POSTGRES_PASSWORD=gohead postgres:17-alpine
docker run -d --rm -p 3306:3306 -e MYSQL_ROOT_PASSWORD=gohead -e MYSQL_DATABASE=gohead mysql:8.4
```

Code that builds SQL by hand must work on all three databases: use the helpers in
`pkg/validation` (`JSONPath`, `JSONTextExpr`) for JSON fields, give indexed string columns a
`size`, and quote nothing yourself (`trigger` and `key` are reserved words in MySQL).

Every bug fix should come with a test that fails without it.

## Admin UI

```bash
cd gui && npm ci && npm run dev    # http://localhost:5173, talks to http://localhost:8080
npm run lint                       # must pass with no warnings (CI blocks on it)
```

Data loading goes through `useApiQuery` (`gui/src/shared/hooks`), which only sets state
when a request settles; don't set state synchronously in effects (the React hooks lint rules
reject it).

The Playwright smoke tests in `gui/e2e` drive a real API. Start one on an empty database,
then run them:

```bash
GOHEAD_DATABASE_URL=sqlite://e2e.db GOHEAD_RATE_LIMIT_ENABLED=false make run &
cd gui && npx playwright install chromium && npm run test:e2e
```

They register the first user (who becomes admin), so never point them at a database you
care about.

## Docs

```bash
make docs                          # http://localhost:8000, live reload
```

Pages live in `docs/content`; add new ones to `nav` in `docs/mkdocs.yml`. CI builds with
`--strict`, so broken links fail the build.

## Pull requests

1. Branch from `master`: `git checkout -b feat/short-description`.
2. Keep changes focused; update docs and `CHANGELOG.md` (*Unreleased*) for user-visible changes.
3. Make sure `make lint test-race` passes.
4. Open the PR against `master`. CI must be green.

## Commit messages

[Conventional commits](https://www.conventionalcommits.org/): `<type>(<scope>): <summary>`.

| Type | Use for |
|------|---------|
| `feat` | new feature |
| `fix` | bug fix (`fix(security)` for security fixes) |
| `docs` | documentation only |
| `test` | tests only |
| `refactor` | no behaviour change |
| `ci`, `chore` | build, tooling |

Release notes are generated from these prefixes.

## Reporting issues

Search [existing issues](https://github.com/gohead-cms/gohead/issues) first. Include the
GoHead version (`gohead version`), your database, steps to reproduce and relevant logs.
Security issues: see [Security](../operations/security.md#reporting-a-vulnerability).
