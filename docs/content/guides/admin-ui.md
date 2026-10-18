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

## What you can do

| Page | Path | Purpose |
|------|------|---------|
| Dashboard | `/dashboard` | Overview of collections and agents |
| Collections | `/collections` | Collection schemas and their relations as a graph |
| Workspace | `/workspace` | Visual canvas of collections and agents |
| Contributions | `/contrib` | Browse and create content items |
| Automation | `/automation` | Agents and their triggers |
| Settings | `/settings` | Session and display settings |

Sign in with any account; admin pages need the `admin` role.
