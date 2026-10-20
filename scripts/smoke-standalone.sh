#!/usr/bin/env bash
# Smoke test of `gohead standalone`: one process, no Redis, SQLite.
#
#   scripts/smoke-standalone.sh ./dist/gohead
#
# Starts the binary in a temporary directory, then checks the API, the
# embedded admin UI, and that a content change reaches a webhook through
# the database job queue (event → dispatcher → delivery). Prints the
# process's memory use. Needs curl and python3.
set -euo pipefail

BIN=$(realpath "${1:-./dist/gohead}")
PORT=${PORT:-18080}
HOOK_PORT=${HOOK_PORT:-18081}
API="http://127.0.0.1:$PORT"
WORK=$(mktemp -d)
cleanup() {
  [ -n "${GOHEAD_PID:-}" ] && kill "$GOHEAD_PID" 2>/dev/null && wait "$GOHEAD_PID" 2>/dev/null || true
  [ -n "${HOOK_PID:-}" ] && kill "$HOOK_PID" 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT
fail() { echo "FAIL: $*" >&2; echo "--- gohead log" >&2; tail -50 "$WORK/gohead.log" >&2; exit 1; }

# A webhook receiver writing what it gets.
cat > "$WORK/hook.py" <<'EOF'
import http.server, sys
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        body = self.rfile.read(int(self.headers.get("Content-Length", 0)))
        open(sys.argv[2], "ab").write(body + b"\n")
        self.send_response(200); self.end_headers()
    def log_message(self, *a): pass
http.server.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), H).serve_forever()
EOF
python3 "$WORK/hook.py" "$HOOK_PORT" "$WORK/hooks.jsonl" &
HOOK_PID=$!

cd "$WORK"
GOHEAD_SERVER_PORT=$PORT \
GOHEAD_LOG_LEVEL=info \
GOHEAD_ADMIN_USERNAME=admin GOHEAD_ADMIN_EMAIL=admin@example.com GOHEAD_ADMIN_PASSWORD=smoke-password-123 \
GOHEAD_WEBHOOKS_ALLOW_PRIVATE_NETWORKS=true \
GOHEAD_QUEUE_POLL_INTERVAL=500ms \
  "$BIN" standalone --data ./data > gohead.log 2>&1 &
GOHEAD_PID=$!

for _ in $(seq 1 60); do curl -fs "$API/_health" > /dev/null && break; sleep 0.5; done
curl -fs "$API/_health" > /dev/null || fail "the API did not start"
grep -q '"driver":"database"' gohead.log || fail "the database job queue is not used"
if grep -q 'Redis unreachable' gohead.log; then fail "standalone tried Redis"; fi
[ -f data/gohead.db ] || fail "no database in the data directory"
[ -f data/jwt_secret ] || fail "no JWT secret saved"

# Embedded UI (when built in).
if grep -q "Admin UI served" gohead.log; then
  curl -fs "$API/" | grep -q '<div id="root">' || fail "/ is not the admin UI"
  curl -fs "$API/contrib" | grep -q '<div id="root">' || fail "UI routes do not serve the UI"
  curl -fs "$API/config.js" | grep -q 'window.location.origin' || fail "config.js"
  echo "ok  admin UI"
else
  echo "--  admin UI not in this build"
fi

json() { python3 -c "import json,sys; d=json.load(sys.stdin); print($1)"; }
TOKEN=$(curl -fs -X POST "$API/auth/login" -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"smoke-password-123"}' | json 'd["data"]["token"]')
auth=(-H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json')

curl -fs "${auth[@]}" -X POST "$API/admin/collections" \
  -d '{"name":"notes","kind":"collection","attributes":{"title":{"type":"string","required":true}}}' > /dev/null || fail "create collection"
curl -fs "${auth[@]}" -X POST "$API/admin/webhooks" \
  -d "{\"name\":\"smoke\",\"url\":\"http://127.0.0.1:$HOOK_PORT/hook\",\"events\":[\"item:created\"]}" > /dev/null || fail "create webhook"
curl -fs "${auth[@]}" -X POST "$API/api/collections/notes" -d '{"data":{"title":"hello from standalone"}}' > /dev/null || fail "create item"
echo "ok  API"

for _ in $(seq 1 40); do grep -q "hello from standalone" hooks.jsonl 2>/dev/null && break; sleep 0.5; done
grep -q "hello from standalone" hooks.jsonl 2>/dev/null || fail "the webhook was not delivered"
echo "ok  webhook delivered through the database queue"

# Memory after the work, and when idle.
sleep 2
rss=$(awk '/VmRSS/ {print $2}' "/proc/$GOHEAD_PID/status" 2>/dev/null || echo "?")
hwm=$(awk '/VmHWM/ {print $2}' "/proc/$GOHEAD_PID/status" 2>/dev/null || echo "?")
echo "ok  memory: ${rss} kB resident (peak ${hwm} kB)"

kill -TERM "$GOHEAD_PID"
wait "$GOHEAD_PID" 2>/dev/null || true
GOHEAD_PID=
grep -q "Shutting down server" gohead.log || fail "no clean shutdown"
echo "ok  clean shutdown"
