# Deploy to Render

[Render](https://render.com) runs GoHead from one click, in your own Render account:

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy?repo=https://github.com/gohead-cms/gohead)

The button reads the repository's [`render.yaml`](https://github.com/gohead-cms/gohead/blob/master/render.yaml)
and creates one web service running [`gohead standalone`](standalone.md): the API, the admin
UI, agents, webhooks and scheduled triggers, in one process on SQLite. A persistent disk keeps
the database, the uploaded files and the generated secrets.

| | |
|-|-|
| Service | One web service, the smallest paid instance (0.5 CPU, 512 MB) |
| Disk | 1 GB at `/data`, for `gohead.db`, `uploads/` and `jwt_secret` |
| Cost | About $7.25 a month: $7 for the instance, $0.25 for the disk ([Render pricing](https://render.com/pricing)) |
| Region | Oregon, Render's default (see [another region](#another-region)) |

Render's free instances have no persistent disk: everything would be lost on each restart, so
the Blueprint uses the smallest paid one.

## Deploy

1. Click **Deploy to Render** and sign in to Render.
2. Fill in the two values Render asks for:
    - `GOHEAD_ADMIN_EMAIL`: the first administrator's email.
    - `GOHEAD_ADMIN_PASSWORD`: their password, **12 characters or more** (GoHead refuses to
      start with a shorter one; the log says so).

    The username is `admin`. `GOHEAD_JWT_SECRET` is generated for you.
3. Click **Deploy Blueprint**. The first deploy takes a few minutes.
4. Open the service's URL (`https://gohead-xxxx.onrender.com`) and sign in as `admin`.

The administrator variables are only read on the first start, on an empty database: changing
them later doesn't change the account. Change the password in the admin UI instead.

## Connect a website

Use the service's URL as `GOHEAD_URL` in your site (for the
[starters](../frontend/index.md), in `.env.local` locally and in your host's settings in
production), then push the content model and the content:

```bash
GOHEAD_URL=https://gohead-xxxx.onrender.com npm run gohead:setup
```

A [custom domain](https://render.com/docs/custom-domains) such as `cms.example.com` works too;
Render provides its TLS certificate.

## Updates

The service runs the latest **0.7.x** release, and is not rebuilt when GoHead changes. To
update, open the service on Render and choose **Manual Deploy → Clear build cache & deploy**:
Render pulls the latest 0.7.x image. Read the [upgrade notes](upgrading.md) first. A service
deployed from your own copy of the repository keeps the version of its Dockerfile: set the
environment variable `GOHEAD_VERSION` to `0.7` (Render passes it to the Docker build) and
deploy. `edge` runs the latest commit of `master`, as the
[public demo](demo.md) does.

A deploy restarts the service: with a disk attached, Render stops the running instance before
starting the new one, so the API is unavailable for a short while.

## How it works

`render.yaml` builds [`deploy/render/Dockerfile`](https://github.com/gohead-cms/gohead/blob/master/deploy/render/Dockerfile),
which is GoHead's released image with two changes: it runs `gohead standalone`, and it runs as
root. Render mounts persistent disks owned by root, and the image's own user could not write to
it. The image has no shell or package manager.

| Setting | Value | Why |
|---------|-------|-----|
| `PORT` | `8080` | Render sends traffic to `$PORT`; GoHead listens on 8080 |
| Health check | `/_health` | Render waits for it before switching traffic to a new deploy |
| `GOHEAD_TRUSTED_PROXIES` | private address ranges | Requests come through Render's proxy: rate limits use the visitor's IP from `X-Forwarded-For` |
| `autoDeployTrigger` | `off` | Your service changes when you deploy it, not when GoHead's repository does |

Render can change a Blueprint's services when its file changes ("Blueprint sync"). To be sure
nothing changes unless you decide it, deploy from your own copy (below), or turn off auto-sync
in the Blueprint's settings.

## Your own copy

Fork [gohead-cms/gohead](https://github.com/gohead-cms/gohead) (or copy `render.yaml` and
`deploy/render/` into a repository of yours), edit them, and deploy with
`https://render.com/deploy?repo=<your repository URL>`.

### Another region

Add `region` to the service in `render.yaml`: `oregon`, `ohio`, `virginia`, `frankfurt` or
`singapore`. Render can't move a service to another region later.

### Agents

Add the `GOHEAD_LLM_*` variables to the service (in `render.yaml`, or in the service's
**Environment** settings): `GOHEAD_LLM_PROVIDER` (`openai`, `anthropic`, `ollama`),
`GOHEAD_LLM_MODEL` and `GOHEAD_LLM_API_KEY`. See [agents](../agents/index.md).

### More traffic or more files

- **Uploads**: images and files are served by GoHead from the disk, and count toward your
  Render bandwidth. With many visitors, store them in S3-compatible storage with a CDN
  (`GOHEAD_MEDIA_DRIVER=s3`, see [configuration](../getting-started/configuration.md)).
- **Database**: SQLite on the disk suits a site with a small team of editors. For more, use
  [Render Postgres](https://render.com/docs/postgresql) with `GOHEAD_DATABASE_URL`.
- **Scaling out**: a service with a disk runs on one instance. To run several API replicas,
  move to PostgreSQL, S3 and Redis, and see [deployment](deployment.md#scaling).

## Backups

Render takes a snapshot of the disk every 24 hours and keeps it for at least seven days;
restore one from the service's **Disk** page. A restore replaces the whole disk: the
database, the uploads and the secrets, as they were at that time.

The image has no shell, so Render's **Shell** tab and `scp` don't work on this service. For
backups you keep elsewhere, move the data out of the disk: the database to
[Render Postgres](https://render.com/docs/postgresql) (then `pg_dump`, or Render's own
Postgres backups) and the uploads to S3-compatible storage.
