# Media

Upload images, video, audio and documents, then reference them from your content. Files are
stored on local disk or in any **S3-compatible** bucket (AWS S3, MinIO, Cloudflare R2,
Garage, Ceph…); GoHead keeps a media library with their metadata.

## Upload

`POST /api/media` takes `multipart/form-data`, with one or more files in a field named
`files` (or `file`). Optional `alt` and `caption` fields apply to every file of the request.
Uploading needs the `create` permission.

```bash
curl -X POST $GOHEAD/api/media -H "$AUTH" \
  -F "files=@cover.jpg" -F "alt=Our team at the summit"
```

```json
{
  "data": [{
    "id": 12,
    "url": "/uploads/2026/10/9f86d081884c7d65-cover.jpg",
    "name": "cover.jpg",
    "mime": "image/jpeg",
    "size": 184321,
    "width": 1600,
    "height": 900,
    "hash": "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
    "alt": "Our team at the summit",
    "caption": "",
    "uploadedBy": "alice",
    "createdAt": "2026-10-18T09:00:00Z",
    "updatedAt": "2026-10-18T09:00:00Z"
  }],
  "meta": { "count": 1 }
}
```

- Up to 20 files per request. A request is **atomic**: if one file is refused, none is kept.
- The type is **detected from the content**, not from the file name or the client's
  `Content-Type`. An HTML page renamed `cat.png` is refused.
- `width` and `height` are filled for JPEG, PNG, GIF and WebP images.

| Status | Why |
|--------|-----|
| `413` | a file is larger than `media.max_size_mb` (50 MB by default) |
| `415` | the type is not in `media.allowed_types` |
| `503` | uploads are disabled (see [configuration](#configuration)) |

By default GoHead accepts common images (JPEG, PNG, GIF, WebP, AVIF), video (MP4, WebM,
QuickTime), audio (MP3, Ogg, WAV, AAC, WebM), PDF, text, CSV, Markdown, JSON, ZIP and office
documents (Word, Excel, PowerPoint, OpenDocument). **SVG and HTML are refused**: served from
your domain they can run scripts. Allow them explicitly only if you trust every uploader.

## The media library

| Method | Path | Permission | Does |
|--------|------|------------|------|
| `GET` | `/api/media` | create | List files, newest first: `page`, `pageSize` (≤ 100), `mime=image/`, `search=cover` |
| `GET` | `/api/media/{id}` | create | One file |
| `PUT` | `/api/media/{id}` | update | Change `name`, `alt` or `caption`: `{"data": {"alt": "…"}}` |
| `DELETE` | `/api/media/{id}` | delete | Delete the file and its content |

Browsing the library needs the `create` permission (editors and admins): it includes files
that only drafts use. Viewers reach files through the content that references them.

Deleting a file that content still uses is allowed: those fields then read as `null` (or
drop the file from a list), like a deleted related item.

## Media attributes

A `media` attribute holds an uploaded file:

```bash
curl -X POST $GOHEAD/admin/collections -H "$JSON" -H "$AUTH" -d '{
  "name": "articles", "kind": "collection",
  "attributes": {
    "title":       { "type": "string", "required": true },
    "cover":       { "type": "media", "allowed_types": ["image/*"] },
    "attachments": { "type": "media", "multiple": true, "max": 5 }
  }
}'
```

| Option | Meaning |
|--------|---------|
| `multiple` | the value is a list of files; `min`/`max` bound its length |
| `allowed_types` | MIME patterns the files must match, e.g. `["image/*"]` or `["application/pdf"]` |

Write a file's **ID** (`12`, `"12"` or `{"id": 12}`):

```bash
curl -X POST $GOHEAD/api/collections/articles -H "$JSON" -H "$AUTH" \
  -d '{"data":{"title":"Summit","cover":12,"attachments":[13,14]}}'
```

Reads return the file objects (with `url`), at every `level` except `level=0`, which returns
the IDs. Files inside [components](content-modeling.md#components), singletons and populated
related items are expanded too:

```json
{ "data": { "id": 7, "title": "Summit",
            "cover": { "id": 12, "url": "/uploads/2026/10/9f86…-cover.jpg", "mime": "image/jpeg", "width": 1600, "height": 900, "alt": "…" },
            "attachments": [ { "id": 13, … }, { "id": 14, … } ] } }
```

!!! note "Existing URL values"
    Before uploads existed, `media` attributes held URLs. Any value that is not an ID is still
    accepted as an external URL and returned as is in REST (and `unique` still applies to
    them), unless the attribute sets `allowed_types` (then only uploaded files are accepted).
    In GraphQL the field becomes a `Media` object: query `cover { url }` instead of `cover`.

In GraphQL, media fields are `Media` objects (`id url name mime size width height alt caption`;
a URL-only value has just `url`) and take file IDs as input:

```graphql
mutation { createarticles(input: { title: "Summit", cover: "12" }) { cover { url width } } }
```

## Configuration

| Key | Env | Default |
|-----|-----|---------|
| `media.enabled` | `GOHEAD_MEDIA_ENABLED` | `true` |
| `media.driver` | `GOHEAD_MEDIA_DRIVER` | `local` (`local` or `s3`) |
| `media.max_size_mb` | `GOHEAD_MEDIA_MAX_SIZE_MB` | `50` |
| `media.allowed_types` | `GOHEAD_MEDIA_ALLOWED_TYPES` | *(built-in list above)*; comma-separated patterns |
| `media.public_url` | `GOHEAD_MEDIA_PUBLIC_URL` | *(empty)* base URL files are downloaded from |
| `media.local.path` | `GOHEAD_MEDIA_LOCAL_PATH` | `./uploads` (`/data/uploads` in the container image) |
| `media.s3.endpoint` | `GOHEAD_MEDIA_S3_ENDPOINT` | e.g. `https://s3.eu-west-1.amazonaws.com`, `http://minio:9000` |
| `media.s3.region` | `GOHEAD_MEDIA_S3_REGION` | |
| `media.s3.bucket` | `GOHEAD_MEDIA_S3_BUCKET` | |
| `media.s3.access_key` / `secret_key` | `GOHEAD_MEDIA_S3_ACCESS_KEY` / `_SECRET_KEY` | empty: `AWS_*` variables or the instance/pod role |
| `media.s3.use_ssl` | `GOHEAD_MEDIA_S3_USE_SSL` | `true`; used when `endpoint` has no `http://`/`https://` |
| `media.s3.path_style` | `GOHEAD_MEDIA_S3_PATH_STYLE` | `false`; `true` for MinIO and most self-hosted stores |
| `media.s3.prefix` | `GOHEAD_MEDIA_S3_PREFIX` | key prefix, to share a bucket |
| `media.s3.presign_expiry` | `GOHEAD_MEDIA_S3_PRESIGN_EXPIRY` | `1h` |

### Local disk

The default. Files go to `media.local.path` and the API serves them at `/uploads/<key>`
(publicly, like a bucket URL) with their detected type, `X-Content-Type-Options: nosniff`, a
sandboxing `Content-Security-Policy`, and `Content-Disposition: attachment` for anything that
is not an image, video, audio, PDF or plain text. Range requests work, so videos can seek.

URLs are relative (`/uploads/…`). Set `media.public_url` to your API's public address
(`https://cms.example.com/uploads`) to get absolute URLs, or to a CDN in front of it.

Local disk is only shared by API replicas if the directory is a shared volume. For several
replicas, use S3.

### S3-compatible storage

```yaml
media:
  driver: s3
  public_url: https://media.example.com   # optional
  s3:
    endpoint: https://s3.eu-west-1.amazonaws.com
    region: eu-west-1
    bucket: my-gohead-media
    access_key: …
    secret_key: …
```

The bucket must exist; GoHead checks it at startup and refuses to start if it cannot reach
it. Without `public_url`, file URLs are **presigned** and valid for `presign_expiry`, so the
bucket can stay private (fetch content again rather than storing URLs). With `public_url`
(a public bucket, or a CDN such as CloudFront or Cloudflare in front of it), URLs are
permanent: `public_url` + key.

Objects are stored with their detected `Content-Type`, and types that are not safe to display
inline get `Content-Disposition: attachment`.

The provided `docker-compose.yml` runs MinIO with a public `gohead` bucket, so uploads work
out of the box at `http://localhost:9000/gohead/…`.

!!! note "Changing driver"
    Records keep their keys. To move from local disk to S3, copy the `uploads` directory
    into the bucket (under `prefix`, if set) with the same paths before switching.

!!! tip "Cloudflare R2 and others"
    R2: `endpoint: https://<account>.r2.cloudflarestorage.com`, `region: auto`. MinIO,
    Garage, SeaweedFS and Ceph: set `path_style: true`.
