# Connected apps

Connected apps give agents tools from other services: open an issue on GitHub, send an
email with Gmail, add a row to a Google Sheet, post to Slack. Each connection is an
[MCP](https://modelcontextprotocol.io) server (Model Context Protocol, the standard way to
offer tools to AI models). GoHead lists the server's tools, and agents add them like
built-in tools:

```json
"tools": [
  { "use": "content.list", "on": ["bug_reports"] },
  { "use": "github.create_issue" }
]
```

Calls of tools that may change something in the app wait for an admin's
[approval](#approvals) unless the agent says otherwise.

## Connect an app

In the admin UI, open **AI → Connected apps**, click **Connect an app** and choose:

| Service | What agents get | Token |
|---------|-----------------|-------|
| **GitHub** | Issues, pull requests, repositories and code search, with [GitHub's MCP server](https://github.com/github/github-mcp-server) | A [fine-grained personal access token](https://github.com/settings/personal-access-tokens/new), limited to the repositories and permissions the agents need |
| **Zapier** | Gmail, Google Drive, Google Sheets, Slack, Notion and thousands of other apps, through the actions you enable in [Zapier MCP](https://docs.zapier.com/mcp/get-started/connect/other) | At mcp.zapier.com: **Add MCP server**, choose **Other** as the AI agent, add the actions agents may use, then **Settings → API token → Generate token** |
| **Other MCP server** | The tools of any server that speaks MCP over HTTP (Streamable HTTP transport): your own, or a service's | A bearer token, a token in another header, or none |

GoHead connects to the server, lists its tools and shows them on the connection: their
name, what they do and whether they only **read** or may **write**. If the server can't be
reached or refuses the token, the connection is saved with the error; fix it with **Edit**,
or click **Refresh** once the server is back. Refresh too when the server's tools change
(for Zapier, after you add or remove actions).

The **prefix** names the connection's tools: with `github`, the tool `create_issue` is
`github.create_issue` for agents and `github_create_issue` for the model. It cannot be
changed later, since agents use it.

!!! tip "Gmail, Drive, Notion"
    Google and Notion run their own MCP servers, but they sign in with OAuth, which GoHead
    does not do yet: connect these through Zapier for now.

## Give the tools to an agent

In the agent editor, **Add a tool** lists the connected apps' tools under **Connected
apps**, by app; the search box finds one among many. In JSON, name the tool with `use`:

```json
{ "use": "github.create_issue" }
{ "use": "github.search_issues", "name": "find_issue" }
{ "use": "zapier.gmail_send_email", "approval": false }
```

`name`, `description` and `parameters` override what the server says, as for
[built-in tools](functions.md). `on` does not apply: the app decides what the token may
reach.

Tell the agent in its system prompt when to use the tool, for example: *"For each new bug
report, search GitHub for a similar issue with github_search_issues. If there is none,
open one with github_create_issue, with the report's title and a link to it."*

The tool returns the app's answer to the model as
`{"status": "success", "result": …}` (or `"status": "error"`), with JSON answers kept as
JSON. A tool the connection no longer offers, or a deleted connection, answers with an error.

## Approvals

MCP servers say which of their tools only read. Every other tool may change something in
the app (send, create, delete…), so by default its calls are not made: each becomes a
proposal on the [Approvals](approvals.md) page, and the agent is told
`{"status": "pending_approval", "proposal_id": …}`.

| Tool | Agent without approval | `"approval": false` | `"approval": true` | Agent that [requires approval](approvals.md) |
|------|------|------|------|------|
| Reads | called | called | proposal | called |
| May write | **proposal** | called | proposal | proposal |

An admin reviews the call: which tool, of which app, with which arguments. **Approve and
call** makes it and records what the tool answered; **Edit and call** makes it with
corrected arguments; **Reject** discards it. A call that fails (the app refused it, the
connection is gone) marks the proposal `failed` with the error. Only admins see and review
calls: editors review content changes only.

In a [test run](runs-and-limits.md#test-runs), tools that may write, and tools set to
`"approval": true`, are never called: they return
`{"status": "success", "dry_run": true, "action": "would_call", …}`. Other tools that only
read are called, so the run sees real data.

!!! warning "What a token can do, an agent can do"
    An agent may call its tools with any arguments the model chooses, including what a
    page or an email it reads asks it to do. Give each connection a token limited to what
    agents need, keep approval on for tools that send or delete, and give each agent only
    the tools its task needs.

## Security

- **Tokens are write-only.** The API never returns them (`"hasToken": true`). Changing a
  connection's URL or how the token is sent needs the token again, so a stored token never
  goes anywhere else. Some servers take a secret in the URL's query string instead: the API
  hides its values, but they are not encrypted, so prefer a token when the server accepts
  one.
- **Encrypted at rest** (AES-256-GCM) with a key derived from `jwt_secret`. Set
  `jwt_secret` (`GOHEAD_JWT_SECRET`) to the same value in every process; without it, each
  process would use a different key, so tokens are stored unencrypted and the admin UI says
  so. Changing `jwt_secret` makes stored tokens unreadable: enter them again.
  [`gohead standalone`](../operations/standalone.md) saves the secret it generates, so its
  tokens are encrypted.
- **Private addresses are refused**, as for [webhooks](../guides/integrations.md): a
  connection cannot reach `127.0.0.1`, `10.0.0.0/8` and other private ranges unless
  `connections.allow_private_networks` is on, and never cloud metadata addresses. Redirects
  are not followed.
- Only admins manage connections and review calls, and only from a signed-in session.
  [API tokens](../guides/integrations.md#api-tokens), even with the admin role, cannot manage
  connections, see or approve calls, or save an agent whose app tool has
  `"approval": false`; they can still give agents app tools that wait for approval.

| Key | Env | Default |
|-----|-----|---------|
| `connections.allow_private_networks` | `GOHEAD_CONNECTIONS_ALLOW_PRIVATE_NETWORKS` | `false` |
| `connections.timeout` | `GOHEAD_CONNECTIONS_TIMEOUT` | `30s` (one listing or call, at most `90s`) |

## API

```bash
# The services the admin UI offers (meta.encrypted: tokens are encrypted)
curl $GOHEAD/admin/connection-presets -H "$AUTH"

# Connect: from a preset...
curl -X POST $GOHEAD/admin/connections -H "$AUTH" -H "$JSON" \
  -d '{"preset": "github", "token": "github_pat_…"}'

# ...or any server: authType none, bearer (Authorization: Bearer) or header (headerName)
curl -X POST $GOHEAD/admin/connections -H "$AUTH" -H "$JSON" -d '{
  "name": "tracker", "label": "Issue tracker",
  "url": "https://mcp.example.com/mcp", "authType": "header", "headerName": "X-API-Key",
  "token": "…"
}'

curl $GOHEAD/admin/connections -H "$AUTH"                  # list, with each one's tools
curl -X POST $GOHEAD/admin/connections/github/refresh -H "$AUTH"   # list the tools again
curl -X PUT $GOHEAD/admin/connections/github -H "$AUTH" -H "$JSON" -d '{"token": "…"}'
curl -X DELETE $GOHEAD/admin/connections/github -H "$AUTH"
```

Creating, updating and refreshing return the connection with its `tools` and `lastError`;
when the tools could not be listed, `meta.warning` says why. `GET /admin/agent-functions`
lists the connected apps' tools with the built-in ones (`"group": "apps"`, `app`,
`appLabel`). Deleting a connection rejects its calls waiting for approval.

Call proposals have `"action": "call"`, `"collection": "app:<connection>"` and
`"data": {"tool": …, "arguments": {…}}`; `GET /api/proposals/{id}` adds `call` with the
tool's description. Approve with `{"arguments": {…}}` to make the call with other
arguments; the approved proposal keeps the tool's answer in `data.result`.
