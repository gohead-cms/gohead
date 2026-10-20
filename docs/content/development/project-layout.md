# Project layout

```text
.
├── main/                 # main.go: calls cmd.Execute()
├── cmd/                  # cobra commands: start, worker, dispatcher, scheduler, version
├── internal/
│   ├── api/handlers      # REST handlers (collections, items, users, agents, webhooks, GraphQL)
│   ├── api/middleware    # auth, roles, CORS, metrics, response envelope
│   ├── graphql           # schema generation, resolvers, mutations
│   ├── models            # collections, attributes, items, users + validation
│   ├── models/agents     # agent definition and validation
│   ├── agent/            # dispatcher, runners (worker loop), functions, triggers, jobs, events
│   └── types             # CMS type → GraphQL type registry
├── pkg/
│   ├── auth              # JWT, role permissions
│   ├── config            # configuration loading (viper)
│   ├── database          # connections (SQLite, PostgreSQL, MySQL)
│   ├── llm               # LLM adapters (langchaingo)
│   ├── migrations, seed  # schema migration, default roles
│   ├── storage           # data access, item queries, events
│   └── testutils         # test database helpers
├── gui/                  # admin UI (React, Vite, Chakra UI)
├── sdk/
│   ├── client/           # @gohead-cms/client: JavaScript client and gohead-client CLI
│   └── create/           # @gohead-cms/create: npm create @gohead-cms
├── templates/            # starter sites (nextjs-blog), copied by @gohead-cms/create
├── package.json          # npm workspaces of sdk/ and templates/
├── deploy/               # Kubernetes (kustomize) and Flux image automation
├── docs/                 # this handbook (MkDocs Material)
├── Dockerfile            # API image
├── docker-compose.yml    # full local stack
└── .goreleaser.yaml      # release binaries
```

## Conventions

- Handlers set `c.Set("response", …)`, `c.Set("status", …)` and optionally `meta`/`details`;
  the `ResponseWrapper` middleware writes the `{data}` / `{error}` envelope.
- Database access goes through `pkg/storage`; multi-row writes use `database.DB.Transaction`.
- Events are published **after** commit.
- New agent functions register themselves in `internal/agent/functions` (`StaticFunctionMap`)
  and accept `any` arguments via `parseArgs`.
