# Architecture

```mermaid
flowchart TB
  subgraph Clients
    FE[Front-ends / apps]
    UI[Admin UI<br/>gohead-admin]
    EXT[External systems]
  end
  subgraph GoHead["gohead image"]
    API["API<br/><code>gohead start</code>"]
    DSP["Dispatcher<br/><code>gohead dispatcher</code>"]
    WRK["Worker<br/><code>gohead worker</code>"]
    SCH["Scheduler<br/><code>gohead scheduler</code>"]
  end
  DB[(PostgreSQL / MySQL / SQLite)]
  R[(Redis)]
  LLM[[LLM provider]]

  FE -- REST / GraphQL --> API
  UI -- REST --> API
  EXT -- webhooks --> API
  API --> DB
  API -- events: queue 'events' --> R
  R --> DSP
  DSP -- agent jobs: queue 'agents' --> R
  SCH -- cron jobs --> R
  R --> WRK
  WRK --> DB
  WRK --> LLM
```

| Component | State | Scaling |
|-----------|-------|---------|
| API | stateless (tokens are JWTs) | see [GraphQL reloads](deployment.md#scaling) |
| Dispatcher | stateless | horizontal |
| Worker | stateless, 10 concurrent jobs per process | horizontal |
| Scheduler | in-memory schedule | **one replica** |
| Database | all content, schema, users, agents, agent history | your database's HA |
| Redis | queues (events, agent jobs) | Redis HA / managed Redis |

## Request path

1. Gin middleware: logging, recovery, CORS, Prometheus metrics, OpenTelemetry, JWT auth,
   role check, response envelope.
2. Handlers validate input against the collection schema and write through the storage layer
   (GORM), inside transactions where several rows change.
3. After commit, the storage layer publishes an event to Redis.

## Code map

| Path | Contents |
|------|----------|
| `main/`, `cmd/` | Entry point and the four commands |
| `internal/api` | HTTP handlers and middleware |
| `internal/graphql` | Schema generation, resolvers, mutations |
| `internal/models` | Collections, attributes, items, users, validation |
| `internal/agent` | Dispatcher, worker runner, functions, triggers, job types |
| `pkg/storage` | Database access, queries, events |
| `pkg/auth` | JWT and role permissions |
| `pkg/llm` | Provider adapters (OpenAI, Anthropic, Ollama) |
| `pkg/config`, `pkg/database`, `pkg/migrations` | Configuration, connections, schema migration |
| `gui/` | Admin UI (React + Vite) |
| `deploy/` | Kubernetes manifests and Flux automation |
