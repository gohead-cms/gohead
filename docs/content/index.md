---
hide:
  - navigation
---

<div class="hero" markdown>

![GoHead](assets/pic/gohead_logo.png)

# GoHead

An event-driven headless CMS, written in Go, with an AI agent engine built in.
Model your content, get REST and GraphQL APIs instantly, and let agents react to every change.

[Quick start](getting-started/quick-start.md){ .md-button .md-button--primary }
[Install](getting-started/installation.md){ .md-button }

</div>

## Why GoHead

<div class="grid cards" markdown>

-   :material-database-cog: **Schema in, API out**

    ---

    Define collections, singletons and reusable components over HTTP. REST endpoints and a
    GraphQL schema appear immediately, with no code generation and no restart.

-   :material-robot-outline: **Agents that react to content**

    ---

    Attach LLM agents to collection events, cron schedules or webhooks. They read and write
    content through typed functions, with OpenAI, Anthropic or Ollama.

-   :material-shield-lock-outline: **Safe by default**

    ---

    Role-based access enforced the same way in REST and GraphQL, validated writes,
    atomic nested creates, masked secrets, signed container images.

-   :material-rocket-launch-outline: **Ships as one binary**

    ---

    One Go binary runs the API, worker, dispatcher and scheduler. SQLite for a laptop,
    PostgreSQL or MySQL in production, Redis for jobs. Kubernetes and Flux manifests included.

</div>

## Five lines to a working API

```bash
docker compose up -d                                   # API, workers, admin UI, Postgres, Redis
curl -X POST localhost:8080/auth/register -H 'Content-Type: application/json' \
  -d '{"username":"admin","email":"admin@example.com","password":"change-me-please"}'
# log in, create a collection, start writing content:
```

Then follow the [quick start](getting-started/quick-start.md): it takes about ten minutes.

## Where to go next

| You want to…                               | Read                                                   |
|--------------------------------------------|--------------------------------------------------------|
| Try it locally                             | [Quick start](getting-started/quick-start.md)          |
| Understand how the pieces fit              | [Concepts](guides/concepts.md)                         |
| Design your content types                  | [Content modeling](guides/content-modeling.md)         |
| Query and write content                    | [REST API](guides/rest-api.md), [GraphQL](guides/graphql.md) |
| Automate with AI                           | [Agents](agents/index.md)                              |
| Run it in production                       | [Deployment](operations/deployment.md)                 |
| Contribute                                 | [Contributing](development/contributing.md)            |
