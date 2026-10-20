# CI/CD & releases

All automation lives in `.github/workflows`.

| Workflow | Trigger | Does |
|----------|---------|------|
| `ci.yml` | push to `master`, pull requests | gofmt, `go mod tidy` check, `go vet`, build, tests with `-race` + coverage, database tests on PostgreSQL 17 and MySQL 8.4, `govulncheck`, admin UI lint (no warnings allowed) and build, Playwright smoke test of the admin UI against the branch's API, JavaScript client unit tests and tests against the branch's API, the Next.js starter set up, linted and built against it, Helm chart lint + kubeconform + install on kind with `helm test`, docs build (`--strict`) |
| `images.yml` | push to `master`, tags `v*.*.*`, PRs touching code | Builds `gohead` and `gohead-admin` images; pushes, signs (cosign keyless) and attests (SBOM + provenance) on `master` and tags; PRs build `amd64` only, without pushing |
| `release.yml` | tags `v*.*.*` | GoReleaser: binaries for Linux/macOS/Windows, archives, checksums, SBOMs, changelog, optional Homebrew cask; build provenance attestation; Helm chart pushed to `oci://ghcr.io/gohead-cms/charts/gohead` with the same version |
| `npm.yml` | tags `v*.*.*`, manual | Publishes `@gohead-cms/client` and `@gohead-cms/create` to npm with the tag's version and provenance (`next` dist-tag for pre-releases); packs without publishing when `NPM_TOKEN` is missing |
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

1. Make sure `master` is green.
2. In `CHANGELOG.md`, rename `[Unreleased]` to the version and date, and check that every
   breaking change has an entry in the [upgrade guide](upgrading.md).
3. Set `version` and `appVersion` in `charts/gohead/Chart.yaml` to the version (the release
   workflow overrides them in the published chart, but the repository should match).
4. Tag and push:

    ```bash
    git tag -a v0.5.0 -m "v0.5.0"
    git push origin v0.5.0
    ```

5. `release.yml` publishes the GitHub release and the Helm chart; `images.yml` publishes
   the images `0.5.0`, `0.5` and `latest`, and warns if they cannot be pulled anonymously;
   `npm.yml` publishes the npm packages with the same version.
6. Install the release from scratch (Compose or Helm) and upgrade a copy of a previous
   install before announcing it.
7. Flux promotes production on its next image scan.

### The API contract

`api/openapi.json` and `api/graphql.json` are snapshots of the public API, generated from a
fixed content model (`internal/contract`). The **API contract** CI job checks that they
match the code and compares them with the snapshots of the last release:

- a change to the API makes the job fail until the snapshots are regenerated
  (`make api-contract`, or add the label **update-api-contract** to the PR and CI commits
  them), so every API change is visible in review;
- a breaking change since the last release (removed endpoint or field, new required
  parameter, narrowed type…) fails the job, found by `oasdiff` and `graphql-inspector`.
  After 1.0, breaking changes wait for the next major version; label the PR
  **breaking-change** to acknowledge one deliberately.

See [API stability](../reference/api-stability.md).

Release notes are generated from commit messages: use conventional commits
(`feat:`, `fix:`, `fix(security):`, `docs:`…), see [contributing](../development/contributing.md).

## One-time repository setup

| Setting | Why |
|---------|-----|
| **Settings → Pages → Source: GitHub Actions** | lets `docs.yml` publish the handbook |
| **Settings → Actions → General → Workflow permissions: read and write** *(or keep read-only; workflows request what they need)* | |
| Package visibility for `gohead`, `gohead-admin` and `charts/gohead` on GHCR: **public** | anonymous `docker pull` and `helm install` |
| Secret `HOMEBREW_TAP_TOKEN` *(optional)* + repository `gohead-cms/homebrew-tap` | publishes `brew install gohead-cms/tap/gohead` |
| npm organization `gohead-cms` + secret `NPM_TOKEN` (an automation token) | publishes `@gohead-cms/client` and `@gohead-cms/create` |
| Branch protection on `master` requiring the CI jobs | keeps `master` deployable |

## Running the checks locally

```bash
make lint            # gofmt + go vet
make test-race       # Go tests with the race detector
make gui             # build the admin UI
cd gui && npm run lint
make test-e2e        # admin UI smoke tests (API on :8080, empty database)
make helm-lint       # Helm chart
make docs            # serve the handbook on :8000
npm ci && npm test   # JavaScript client and create package (repository root)
```
