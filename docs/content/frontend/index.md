# Build a website on GoHead

GoHead stores and serves your content; your front-end, in any framework, reads it over
[REST](../guides/rest-api.md) or [GraphQL](../guides/graphql.md). For JavaScript and
TypeScript front-ends, GoHead ships:

<div class="grid cards" markdown>

-   :material-language-typescript: **[The JavaScript client](sdk.md)**

    ---

    `@gohead-cms/client`: typed reads and writes, filters, locales, media, GraphQL and
    webhook verification, for Node, browsers and edge runtimes.

-   :material-file-code-outline: **[Content models as code](model-as-code.md)**

    ---

    Keep collections, singletons and public access in a JSON file next to your front-end,
    generate TypeScript types, push the model and sample content from the command line.

-   :material-rocket-launch-outline: **[Starter templates](nextjs.md)**

    ---

    `npm create @gohead-cms@latest`: a Next.js blog with its content model, sample content,
    two languages and pages that update when you publish.

-   :material-map-marker-path: **[Feature tour](feature-tour.md)**

    ---

    A conference site that uses every GoHead feature, from visitor forms and draft previews
    to seven agents, with a guide to each one.

</div>

## Ten minutes to a website

```bash
npm create @gohead-cms@latest my-site      # Next.js blog, with its content model
cd my-site
docker compose up -d                        # GoHead on http://localhost:8080
npm run gohead:setup                        # content model, sample content, types
npm run dev                                 # http://localhost:3000
```

Open the admin UI (`http://localhost:8080`), edit an article, and refresh the site.

!!! tip "Without Docker, on a Mac"
    Run GoHead with [Homebrew](../getting-started/installation.md#homebrew) instead of
    `docker compose up -d`, in a second terminal from the site's folder:

    ```bash
    brew install gohead-cms/tap/gohead
    GOHEAD_ADMIN_USERNAME=admin GOHEAD_ADMIN_EMAIL=admin@example.com \
    GOHEAD_ADMIN_PASSWORD=gohead-local-password GOHEAD_WEBHOOKS_ALLOW_PRIVATE_NETWORKS=true \
      gohead standalone --data .gohead
    ```

    The account is the one the starter's `.env.local` signs in with, and the data stays in
    `.gohead/` (git-ignored). The admin UI is on `http://localhost:8080` too. Point the
    revalidation webhook at `http://localhost:3000/api/revalidate`: GoHead runs on your
    machine, not in a container, so `host.docker.internal` is not needed.

## Reading content: public access or a token

| | Public access | API token |
|-|---------------|-----------|
| What | Anonymous visitors read the types you open in the [public role](../guides/users-and-roles.md#public-access) | A server sends a [token](../guides/integrations.md#api-tokens) with the role you chose |
| Use for | Browsers, mobile apps, static builds, server rendering | Private content, drafts (preview), writes, large static builds |
| Limits | Published content only, 2 levels of relations, rate limited per IP | Those of the token's role |

The starters use public access for published content, so the site works without a secret,
and accept an optional `GOHEAD_TOKEN` for private instances.

!!! warning "Never ship a token to the browser"
    Anything in client-side code is public. Read from server code (server components, route
    handlers, build scripts) when you need a token, and keep it out of `NEXT_PUBLIC_*`
    variables.

## Other frameworks

The client runs wherever `fetch` does: Astro, Nuxt, SvelteKit, Remix, Express, Bun, Deno,
Cloudflare Workers. In other languages, use the [REST API](../guides/rest-api.md), the
[OpenAPI document](../guides/integrations.md#openapi-and-swagger-ui) to generate a client,
or [GraphQL](../guides/graphql.md).
