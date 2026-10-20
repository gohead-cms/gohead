# syntax=docker/dockerfile:1.7
#
# GoHead image. One image runs every role:
#   gohead standalone | start | worker | dispatcher | scheduler
#
# Pure Go (SQLite included, no cgo), with the admin UI embedded: `gohead
# standalone` serves the API and the UI on :8080 with no other service.

ARG GO_VERSION=1.27
ARG NODE_VERSION=26

# Admin UI (the same for every platform).
FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS ui
WORKDIR /gui
COPY gui/package.json gui/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY gui/ ./
RUN npm run build && rm -f dist/config.js \
 && find dist -type f \( -name '*.js' -o -name '*.css' -o -name '*.html' -o -name '*.svg' \) -size +1k -exec gzip -9 -k {} \;

# Binary, cross-compiled on the build platform.
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bookworm AS builder
WORKDIR /src

# Dependencies first, for layer caching.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN find internal/webui/static -mindepth 1 ! -name .keep -exec rm -rf {} +
COPY --from=ui /gui/dist/ internal/webui/static/

ARG TARGETOS TARGETARCH TARGETVARIANT
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} \
    go build -trimpath \
      -ldflags "-s -w \
        -X github.com/gohead-cms/gohead/cmd.version=${VERSION} \
        -X github.com/gohead-cms/gohead/cmd.commit=${COMMIT} \
        -X github.com/gohead-cms/gohead/cmd.buildDate=${BUILD_DATE}" \
      -o /out/gohead ./main \
 && mkdir -p /out/data

# Distroless static base: CA certs and tzdata, no shell, runs as non-root.
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="GoHead" \
      org.opencontainers.image.description="Event-driven headless CMS with an AI agent engine" \
      org.opencontainers.image.source="https://github.com/gohead-cms/gohead" \
      org.opencontainers.image.licenses="MIT"

WORKDIR /app
COPY --from=builder /out/gohead /usr/local/bin/gohead
COPY --from=builder --chown=65532:65532 /out/data /data

# Writable location for a SQLite file when no external database is used,
# and for uploads with the local media driver.
ENV GOHEAD_DATABASE_URL=sqlite:///data/gohead.db \
    GOHEAD_MEDIA_LOCAL_PATH=/data/uploads
VOLUME ["/data"]

EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/gohead"]
CMD ["start", "--config", "/app/config.yaml"]
