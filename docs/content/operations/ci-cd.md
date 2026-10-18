# CI/CD & releases

All automation lives in `.github/workflows`.

| Workflow | Trigger | Does |
|----------|---------|------|
| `ci.yml` | push to `master`, pull requests | gofmt, `go mod tidy` check, `go vet`, build, tests with `-race` + coverage, tests on PostgreSQL, `govulncheck`, admin UI build (lint advisory), docs build (`--strict`) |
| `images.yml` | push to `master`, tags `v*.*.*`, PRs touching code | Builds `gohead` and `gohead-admin` images; pushes, signs (cosign keyless) and attests (SBOM + provenance) on `master` and tags; PRs build `amd64` only, without pushing |
| `release.yml` | tags `v*.*.*` | GoReleaser: binaries for Linux/macOS/Windows, archives, checksums, SBOMs, changelog, optional Homebrew cask; build provenance attestation |
| `docs.yml` | push to `master` touching docs | Builds this handbook and deploys it to GitHub Pages |
| `codeql.yml` | push, PRs, weekly | CodeQL for Go and TypeScript |

Dependabot keeps Go modules, npm packages, Actions, base images and docs dependencies up to
date (`.github/dependabot.yml`).

## Image tags

| Event | Tags |
|-------|------|
| push to `master` | `edge`, `master-<sha7>-<unix-timestamp>` |
| tag `v1.4.2` | `1.4.2`, `1.4`, `1`, `latest` |
| tag `v0.3.0` | `0.3.0`, `0.3`, `latest` (no major tag for 0.x) |
| tag `v1.5.0-rc.1` | `1.5.0-rc.1` only |

The timestamped `master-…` tag is what Flux's staging policy follows; semver tags feed
production. See [deployment](deployment.md#how-releases-flow).

## Cutting a release

1. Make sure `master` is green and `CHANGELOG.md` has an entry for the version.
2. Tag and push:

    ```bash
    git tag -a v0.2.0 -m "v0.2.0"
    git push origin v0.2.0
    ```

3. `release.yml` publishes the GitHub release; `images.yml` publishes `0.2.0` and `latest`.
4. Flux promotes production on its next image scan.

Release notes are generated from commit messages: use conventional commits
(`feat:`, `fix:`, `fix(security):`, `docs:`…), see [contributing](../development/contributing.md).

## One-time repository setup

| Setting | Why |
|---------|-----|
| **Settings → Pages → Source: GitHub Actions** | lets `docs.yml` publish the handbook |
| **Settings → Actions → General → Workflow permissions: read and write** *(or keep read-only; workflows request what they need)* | |
| Package visibility for `gohead` and `gohead-admin` on GHCR: **public** | anonymous `docker pull` |
| Secret `HOMEBREW_TAP_TOKEN` *(optional)* + repository `gohead-cms/homebrew-tap` | publishes `brew install gohead-cms/tap/gohead` |
| Branch protection on `master` requiring the CI jobs | keeps `master` deployable |

## Running the checks locally

```bash
make lint            # gofmt + go vet
make test-race       # Go tests with the race detector
make gui             # build the admin UI
make docs            # serve the handbook on :8000
```
