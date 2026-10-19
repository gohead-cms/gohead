# Admin UI

The admin UI is a React single-page app in `gui/`. It talks to the API from the browser.

## Run it

=== "Docker Compose"

    Started automatically on <http://localhost:3000>.

=== "Container"

    ```bash
    docker run -d -p 3000:8080 \
      -e GOHEAD_API_URL=https://api.example.com \
      ghcr.io/gohead-cms/gohead-admin:latest
    ```

=== "Development"

    ```bash
    cd gui
    npm ci
    VITE_API_BASE_URL=http://localhost:8080 npm run dev
    ```

`GOHEAD_API_URL` is read when the container starts (served as `/config.js`), so one image
works in every environment. The API must allow the UI's origin in
[`cors.allowed_origins`](../getting-started/configuration.md#cors).

## Signing in

Sign in with an account created by an admin, the
[first administrator](users-and-roles.md#registration-and-the-first-admin), or
self-registration when it is open. What you see depends on your role:

| Role | Sections |
|------|----------|
| viewer | Dashboard, content (read) |
| editor | Dashboard, content editor, media library |
| admin | Everything below |

An expired or revoked session signs you out.

## Sections

| Page | Path | What you can do |
|------|------|-----------------|
| Dashboard | `/dashboard` | Counts of content, media, users, agents and webhooks; agent runs and webhook deliveries of the last 24 hours; recent changes; quick actions |
| Content editor | `/contrib` | Browse, create, edit, publish and delete entries of every collection; edit single types; a full-page editor for long-form content |
| Media library | `/media` | Upload by button or drag and drop, search and filter, edit name, alt text and caption, copy URLs, delete |
| Types | `/model` | Create, edit and delete collections, components and single types: attributes of every type with their options (required, unique, enum values, ranges, relations, media, components), draft and publish |
| Schema designer | `/workspace` | The saved content model as a graph: collections, relations and the agents listening to them |
| Agents | `/agents` | Create and edit agents (model, prompt, trigger, functions from the built-in catalog with collection scopes, limits), run one now, inspect its runs step by step |
| Activity | `/activity` | Runs of every agent with status, tokens, cost and duration, and usage against limits |
| Users | `/settings/users` | Create accounts, change roles, reset passwords. Admins can't demote or delete themselves or the last admin |
| API tokens | `/settings/tokens` | Create role-scoped tokens (the secret is shown once) and revoke them |
| Webhooks | `/settings/webhooks` | Outbound webhooks: events, collections, format, headers, signing secret, test sends and the delivery log |

Everything the UI does goes through the public API, so anything you can click you can
also script; the API reference is linked from the account menu.

## Limits

- Attribute order in the type editor is not stored (the API keeps attributes by name).
- Collections and single types cannot be renamed; components can while unused.
- Viewers cannot browse the media library (listing media needs the editor role).
