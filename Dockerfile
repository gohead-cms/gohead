# syntax=docker/dockerfile:1.7
#
# GoHead API image. One image runs every role:
#   gohead start | worker | dispatcher | scheduler
#
# CGO is enabled because the SQLite driver (mattn/go-sqlite3) needs it.

ARG GO_VERSION=1.27

FROM golang:${GO_VERSION}-bookworm AS builder
WORKDIR /src

# Dependencies first, for layer caching.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -trimpath \
      -ldflags "-s -w \
        -X github.com/gohead-cms/gohead/cmd.version=${VERSION} \
        -X github.com/gohead-cms/gohead/cmd.commit=${COMMIT} \
        -X github.com/gohead-cms/gohead/cmd.buildDate=${BUILD_DATE}" \
      -o /out/gohead ./main \
 && mkdir -p /out/data

# Distroless glibc base: matches the CGO build, ships CA certs and tzdata,
# no shell, runs as non-root.
FROM gcr.io/distroless/base-debian12:nonroot

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
