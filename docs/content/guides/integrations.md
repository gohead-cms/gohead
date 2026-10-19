# Integrations

Three ways to connect GoHead to the rest of your stack: **API tokens** for front-ends and CI,
an **OpenAPI** description for typed clients, and **outbound webhooks** that push content
events to Slack, Zapier or a site-rebuild hook.

## API tokens

User JWTs expire after 72 hours. For a website, a build pipeline or an integration, create a
long-lived **API token** instead:

```bash
curl -X POST $GOHEAD/admin/tokens -H "$JSON" -H "$AUTH" \
  -d '{"name": "website", "role": "viewer", "expires_in_days": 365}'
```

```json
{
  "data": {
    "id": 3, "name": "website", "role": "viewer", "prefix": "ght_9f2c1a",
    "status": "active", "expiresAt": "2027-10-18T09:00:00Z", "lastUsedAt": null,
    "token": "ght_9f2c1a…"
  },
  "meta": { "warning": "Store this token now: it cannot be shown again." }
}
```

Use it like a JWT: `Authorization: Bearer ght_…`, for REST, GraphQL and uploads.

| Field | Meaning |
|-------|---------|
| `name`, `description` | what the token is for |
| `role` | `viewer` (default, read-only), `editor` (content writes) or `admin` (also schema) |
| `expires_in_days` or `expires_at` | optional expiry (1–3650 days, or an RFC 3339 date); none means it never expires |

- **The token is shown once.** GoHead stores only its SHA-256 hash; a lost token is
  replaced, not recovered.
- **Revoke** with `DELETE /admin/tokens/{id}`: the token stops working immediately. The record
  stays (name, creator, last use) for auditing.
- `GET /admin/tokens` lists tokens with their `status` (`active`, `expired`, `revoked`) and
  `lastUsedAt` (updated at most once a minute).
- Token management (and `/admin/users`) needs an admin **user session**: an API token, even an
  admin one, cannot create other tokens or users, so a leaked token cannot mint credentials.
- Tokens start with `ght_`, so secret scanners and log filters can recognize them. Requests
  made with a token are attributed to `token:<name>` (usernames cannot contain `:`).

!!! tip "Pick the smallest role"
    A public website only needs `viewer`: it sees published content only. Give `editor` to
    importers and CI jobs that write content, and `admin` only to pipelines that manage the
    schema.

## OpenAPI and Swagger UI

GoHead describes its REST API as an **OpenAPI 3.1** document, generated from your
collections, components and singletons, the same way the GraphQL schema is. Every
collection gets a typed schema (`articles`) and an input schema (`articlesInput`), and every
route its parameters (`level`, `status`, `filters`, `sort`…).

| URL | What |
|-----|------|
| `/api/openapi.json` | the document (needs a token unless `openapi.public` is on) |
| `/api/docs` | Swagger UI; it asks for a token and keeps it in the browser tab |

Generate a typed client:

```bash
curl -s $GOHEAD/api/openapi.json -H "Authorization: Bearer $TOKEN" > openapi.json
npx openapi-typescript openapi.json -o gohead.d.ts          # TypeScript types
npx @openapitools/openapi-generator-cli generate -i openapi.json -g go -o ./goheadclient
```

The document always reflects the current content model; regenerate clients after schema
changes (in CI, with a `viewer` token).

| Key | Env | Default |
|-----|-----|---------|
| `openapi.enabled` | `GOHEAD_OPENAPI_ENABLED` | `true` |
| `openapi.public` | `GOHEAD_OPENAPI_PUBLIC` | `false`: the document describes your content model |

The Swagger UI page loads `swagger-ui-dist` from jsDelivr with subresource integrity.

## Outbound webhooks

A webhook sends content events to a URL, with no agent or code to write: post to Slack,
trigger a Zapier/Make/n8n workflow, or call a Netlify/Vercel/Cloudflare deploy hook to rebuild
a static site when content is published.

```bash
curl -X POST $GOHEAD/admin/webhooks -H "$JSON" -H "$AUTH" -d '{
  "name": "rebuild site",
  "url": "https://api.netlify.com/build_hooks/abc123",
  "events": ["item:published", "item:unpublished", "item:deleted"],
  "collections": ["articles", "pages"]
}'
```

| Field | Meaning |
|-------|---------|
| `url` | `http(s)` URL to `POST` to |
| `events` | any of `item:created`, `item:updated`, `item:deleted`, `item:published`, `item:unpublished`; empty or `["*"]` for all |
| `collections` | collection names; empty for all |
| `format` | `json` (default) or `slack` |
| `headers` | extra headers, e.g. `{"Authorization": "Bearer …"}`; values are never shown again |
| `enabled` | default `true` |
| `signed` | default `true`: requests are signed with a secret (see below) |

The response contains the signing **secret** once (`whsec_…`). `PUT /admin/webhooks/{id}`
changes any field (fields left out are kept; `"rotate_secret": true` issues a new secret),
`DELETE` removes the webhook.

### Payload

```http
POST /build_hooks/abc123
Content-Type: application/json
User-Agent: GoHead-Webhooks/1
X-Gohead-Event: item:published
X-Gohead-Delivery: 5f1c…-3
X-Gohead-Timestamp: 1760778000
X-Gohead-Signature: sha256=9b1d…

{
  "id": "5f1c…-3",
  "event": "item:published",
  "collection": "articles",
  "occurredAt": "2026-10-18T09:00:00Z",
  "data": { "id": 12, "title": "Launch day", "body": "…" }
}
```

With `"format": "slack"` the body is `{"text": "GoHead: \"Launch day\" (#12) published in articles"}`,
which Slack incoming webhooks (and Discord's `/slack` endpoints, Mattermost…) accept.

`X-Gohead-Delivery` is the same on every retry of one event: use it to ignore duplicates.
`data` is the item as stored, except `password` attributes, which are never sent. Items with
[translations](localization.md) also have `"translations": {"fr": {…}}`, filtered the same way.

### Verify the signature

The signature is an HMAC-SHA256 of `<timestamp>.<raw body>` with the secret:

```js
import crypto from "node:crypto";

function verify(req, rawBody, secret) {
  const ts = req.headers["x-gohead-timestamp"];
  const expected = "sha256=" + crypto.createHmac("sha256", secret).update(`${ts}.${rawBody}`).digest("hex");
  const given = req.headers["x-gohead-signature"] || "";
  const fresh = Math.abs(Date.now() / 1000 - Number(ts)) < 300; // reject replays
  return fresh && given.length === expected.length &&
    crypto.timingSafeEqual(Buffer.from(given), Buffer.from(expected));
}
```

### Delivery, retries and logs

- Events go through the queue: the `dispatcher` turns each event into one delivery per
  matching webhook and a `worker` sends it. Both run in the provided Docker Compose and
  Kubernetes setups; webhooks need Redis, like agents.
- A `2xx` answer is a success. Network errors, timeouts, `5xx`, `408` and `429` are retried
  with exponential backoff (`webhooks.max_retries`, default 8, over several hours). Other
  `4xx` answers are not retried, and neither are redirects: GoHead does not follow them (they
  would carry your custom headers to another host), so use the final URL.
- `GET /admin/webhooks/{id}/deliveries` shows the last 100 attempts (status, error, duration);
  the webhook itself shows `lastStatus`, `lastError` and `lastDeliveryAt`.
- `POST /admin/webhooks/{id}/test` sends a `webhook:test` event immediately and returns the
  result, to check a URL, headers and signature.
- Drafts: `item:created` and `item:updated` fire for drafts too. Subscribe to
  `item:published` / `item:unpublished` to react only to what readers see.

### Network safety

Webhook URLs are resolved and checked at each connection, so DNS tricks do not help. Private
and loopback addresses (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `100.64.0.0/10`,
`127.0.0.0/8`, `fc00::/7`, …) are refused unless `webhooks.allow_private_networks` is on (for
an internal rebuild service or local testing). Always refused: link-local addresses (including
the cloud metadata endpoint `169.254.169.254`), the AWS IPv6 and Alibaba metadata addresses,
reserved ranges, and NAT64/6to4 addresses that embed a forbidden IPv4 address. Proxies from the
environment are not used for webhooks. Delivery errors are reported in generic terms
("could not connect", "did not answer in time") so the test endpoint cannot be used to map
internal networks.

| Key | Env | Default |
|-----|-----|---------|
| `webhooks.allow_private_networks` | `GOHEAD_WEBHOOKS_ALLOW_PRIVATE_NETWORKS` | `false` |
| `webhooks.timeout` | `GOHEAD_WEBHOOKS_TIMEOUT` | `10s` per attempt |
| `webhooks.max_retries` | `GOHEAD_WEBHOOKS_MAX_RETRIES` | `8` |

!!! note "Webhooks or agents?"
    Webhooks forward events as they are. Use an [agent](../agents/index.md) when something
    should *think* about the content (summarize, tag, translate) or write back to GoHead.
