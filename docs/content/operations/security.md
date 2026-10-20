# Security

## Built in

| Area | Behaviour |
|------|-----------|
| Authentication | HS256 JWT, 72 h expiry, `exp` required, algorithm pinned |
| Registration | Off by default; when open, self-registered accounts are viewers. The first admin comes from the configuration or `gohead admin create`, never from the public API |
| Authorization | One role table enforced by REST handlers and GraphQL resolvers |
| Public access | Off by default; when on, anonymous requests read the listed types only (published content, no media library, no types with passwords), with capped relation depth, bounded GraphQL queries and a per-IP rate limit |
| Passwords | bcrypt; never serialized in responses |
| Brute force | `/auth/*` rate limited per IP; failed logins lock a username out per IP; unknown and known usernames answer with the same message and timing; limits shared across replicas through Redis |
| Client IP | `X-Forwarded-For` honoured only from `trusted_proxies` |
| API tokens | 256-bit random, stored as SHA-256 hashes, role-scoped, expiring, revocable; cannot manage tokens or users |
| Outbound webhooks | HMAC-SHA256 signatures with a timestamp; private, loopback and link-local addresses refused by default (checked after DNS resolution, on every redirect) |
| Uploads | Type detected from content; allow-list (no SVG/HTML by default); size limit; random keys; files served with `nosniff`, a sandboxing CSP and `attachment` for non-media types |
| Input | Every write validated against the schema; unknown attributes rejected; filters and sorts checked against the schema and bound as SQL parameters |
| Secrets in responses | Agent API keys and webhook tokens masked |
| Webhooks | Only webhook agents; per-agent token compared in constant time |
| LLM keys | Passed per client, never via shared process environment |
| Containers | Distroless, non-root, read-only root filesystem in the provided manifests |
| Supply chain | Images signed with cosign (keyless), SBOM and provenance attestations, CodeQL, govulncheck, Dependabot |

## Hardening checklist

- Set `GOHEAD_JWT_SECRET` to 32+ random bytes; rotate it to log everyone out.
- Register the admin account before the API is reachable from the internet.
- Restrict `GOHEAD_CORS_ALLOWED_ORIGINS` to your front-ends.
- Open [public access](../guides/users-and-roles.md#public-access) only for the types your
  site shows; review it when you add types.
- Set `GOHEAD_TRUSTED_PROXIES` to your ingress addresses, and nothing broader.
- Give front-ends and CI [API tokens](../guides/integrations.md#api-tokens) with the smallest
  role and an expiry, instead of user passwords; revoke them when unused (`lastUsedAt`).
- Keep `/_metrics` internal.
- Use TLS everywhere: ingress, database (`sslmode=require`), Redis.
- Give agents the minimum set of [functions](../agents/functions.md); be careful with
  `collections.delete_item`.
- Prefer per-agent API keys with spending limits at the provider.

## Verify container images

Images built from this repository are signed by its GitHub Actions workflow:

```bash
cosign verify ghcr.io/gohead-cms/gohead:0.2.0 \
  --certificate-identity-regexp '^https://github\.com/gohead-cms/gohead/\.github/workflows/images\.yml@refs/(heads/master|tags/v.*)$' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

Inspect the SBOM and provenance with `docker buildx imagetools inspect --format '{{ json .SBOM }}' <image>`.
Release archives carry GitHub build attestations: `gh attestation verify gohead_*.tar.gz -R gohead-cms/gohead`.

## Reporting a vulnerability

Please do not open a public issue. Use
[GitHub private vulnerability reporting](https://github.com/gohead-cms/gohead/security/advisories/new)
on the repository.
