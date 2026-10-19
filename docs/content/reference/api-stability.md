# API stability

GoHead follows [semantic versioning](https://semver.org/). From **1.0**, everything listed
as *stable* below keeps working across minor and patch versions; breaking it needs a new
major version. Before 1.0, minor versions may break it, always with an entry in the
[upgrade guide](../operations/upgrading.md).

## Stable

| Surface | What is covered |
|---------|-----------------|
| REST API | Every path, method, parameter, request body and response in `/api/openapi.json`, including `/auth/*` and `/admin/*`; the response envelope (`data`, `meta`, `error`); status codes |
| GraphQL | Every type, field, argument and mutation generated from the content model, and their naming rules |
| Content events and webhooks | Event names (`item:created`…), the webhook JSON payload and signature headers |
| Configuration | Keys and `GOHEAD_*` variables in the [configuration reference](../getting-started/configuration.md) |
| CLI | Commands and flags in the [CLI reference](cli.md) |
| Helm chart | Values documented in `values.yaml` |

Adding things is not a breaking change: new endpoints, optional parameters, new fields in
responses, new enum values in responses, new event names. Clients should ignore fields they
don't know.

## Not covered

- The database schema: use the API, not the tables. It changes through
  [migrations](../operations/upgrading.md#database-migrations).
- Log messages and their fields, metric names (changes are noted in the changelog).
- The admin UI's internals, and Go packages (GoHead is an application, not a library).
- Agent behaviour that depends on the model (prompts are yours; results vary).

## Deprecation

A stable feature that will go away is first **deprecated** in a minor version: marked
`deprecated` in OpenAPI and GraphQL, noted in the changelog, and kept working until the next
major version.

## How it is enforced

`api/openapi.json` and `api/graphql.json` in the repository are snapshots of the API
generated from a fixed content model that uses every attribute type. CI fails when the code
and the snapshots disagree, so every API change shows up in review, and runs
[oasdiff](https://github.com/oasdiff/oasdiff) and
[GraphQL Inspector](https://the-guild.dev/graphql/inspector) against the snapshots of the
last release to catch breaking changes. See [CI/CD](../operations/ci-cd.md#the-api-contract).
