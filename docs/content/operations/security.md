# Security

## Built in

| Area | Behaviour |
|------|-----------|
| Authentication | HS256 JWT, 72 h expiry, `exp` required, algorithm pinned |
| Registration | First account becomes admin; later self-registrations are viewers |
| Authorization | One role table enforced by REST handlers and GraphQL resolvers |
| Passwords | bcrypt; never serialized in responses |
| Brute force | `/auth/*` rate limited per IP; failed logins lock a username out per IP; unknown and known usernames answer with the same message and timing; limits shared across replicas through Redis |
| Client IP | `X-Forwarded-For` honoured only from `trusted_proxies` |
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
- Set `GOHEAD_TRUSTED_PROXIES` to your ingress addresses, and nothing broader.
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
