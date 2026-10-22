#!/usr/bin/env bash
# Builds and starts the demo's conference site on Render (render.yaml in
# this folder), from the repository root: the starter and the JavaScript
# client of this commit, with GoHead's addresses from the Blueprint.
#
#   bash deploy/demo/site.sh build|start
set -euo pipefail

# GoHead over the private network, and its public address for files.
if [ -z "${GOHEAD_URL:-}" ] && [ -n "${GOHEAD_HOSTPORT:-}" ]; then
  export GOHEAD_URL="http://${GOHEAD_HOSTPORT}"
fi
if [ -z "${GOHEAD_PUBLIC_URL:-}" ] && [ -n "${GOHEAD_PUBLIC_HOST:-}" ]; then
  export GOHEAD_PUBLIC_URL="https://${GOHEAD_PUBLIC_HOST}"
fi
# This site's own address: canonical links, the sitemap, RSS.
if [ -z "${NEXT_PUBLIC_SITE_URL:-}" ] && [ -n "${RENDER_EXTERNAL_URL:-}" ]; then
  export NEXT_PUBLIC_SITE_URL="${RENDER_EXTERNAL_URL}"
fi

case "${1:-}" in
  build)
    npm ci --no-audit --no-fund
    npm run build --workspace @gohead-cms/client
    npm run build --workspace gohead-nextjs-conference
    ;;
  start)
    exec npm run start --workspace gohead-nextjs-conference -- --port "${PORT:-3000}"
    ;;
  *)
    echo "usage: $0 build|start" >&2
    exit 2
    ;;
esac
