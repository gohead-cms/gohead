# Contributing to GoHead

Thank you for helping! Bug reports, docs, tests and features are all welcome.

The full guide (setup, tests, PostgreSQL tests, admin UI, docs, conventions) is in the
handbook: **[Contributing](https://gohead-cms.github.io/gohead/development/contributing/)**.

## In short

```bash
git clone https://github.com/<you>/gohead.git && cd gohead
git remote add upstream https://github.com/gohead-cms/gohead.git
git checkout -b feat/short-description

make lint test-race     # gofmt, go vet, tests with the race detector
make up                 # full stack (API, workers, admin UI, Postgres, Redis)
make docs               # handbook on http://localhost:8000
```

- Branch from `master` and open pull requests against `master`; CI must be green.
- Add a test for every bug fix, and update the docs and `CHANGELOG.md` (*Unreleased*) for
  user-visible changes.
- Use [conventional commits](https://www.conventionalcommits.org/): `feat:`, `fix:`,
  `fix(security):`, `docs:`, `test:`, `refactor:`, `ci:`, `chore:`. Release notes are
  generated from them.

## Reporting issues

Search the [issues](https://github.com/gohead-cms/gohead/issues) first, then include your
GoHead version (`gohead version`), database, steps to reproduce and logs.

Security vulnerabilities: please report them privately through
[GitHub security advisories](https://github.com/gohead-cms/gohead/security/advisories/new),
not in a public issue.

By contributing you agree that your contributions are licensed under the [MIT License](LICENSE.md).
