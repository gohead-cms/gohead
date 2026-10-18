# Contributing

Thanks for helping! Bugs, docs, tests and features are all welcome.

## Setup

Requirements: Go 1.25+, a C compiler (SQLite), Node 22 (admin UI), Docker (optional),
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
  make test-postgres                             # DB tests against PostgreSQL
make lint                                        # gofmt + go vet
```

Tests that touch the database use `pkg/testutils.SetupTestServer()`, which gives you a fresh
database with the default roles. With `GOHEAD_TEST_DATABASE_URL` set, the `public` schema of
that database is **wiped** before each test: use a throwaway database.

Every bug fix should come with a test that fails without it.

## Admin UI

```bash
cd gui && npm ci && npm run dev    # http://localhost:5173, talks to http://localhost:8080
```

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
