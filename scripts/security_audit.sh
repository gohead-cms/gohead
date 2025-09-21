#!/usr/bin/env bash
# scripts/security_audit.sh
set -euo pipefail

# Resolve repo root (script is in scripts/ or script/, code is one level up)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "[*] Security audit from: ${SCRIPT_DIR}"
echo "[*] Using repo root:     ${REPO_ROOT}"
pushd "${REPO_ROOT}" >/dev/null

# Optional: ensure we don't accidentally use a workspace file outside this repo
go env -w GOWORK=off >/dev/null 2>&1 || true

echo "[+] Go: tidy"
go mod tidy

echo "[+] Go: update minor/patch (best-effort)"
# Use JSON to reliably pull available updates; falls back to simple list if jq missing
if command -v jq >/dev/null 2>&1; then
  # Extract modules with available updates and upgrade them individually
  go list -m -u -json all \
    | jq -r 'select(.Update) | "\(.Path)@\(.Update.Version)"' \
    | xargs -r -n1 -I{} sh -c 'echo "    -> go get {}" && go get {} || true'
else
  # Fallback heuristic (may miss some cases without jq)
  go list -m -u all 2>/dev/null \
    | awk '$3 ~ /^v/ {print $1"@"$3}' \
    | xargs -r -n1 -I{} sh -c 'echo "    -> go get {}" && go get {} || true'
fi

go mod tidy

echo "[+] Build & test"
go build ./...
go test ./... -run . -count=1

echo "[+] govulncheck"
if ! command -v govulncheck >/devnull 2>&1; then
  go install golang.org/x/vuln/cmd/govulncheck@latest
fi
# Continue even if vulnerabilities are found; exit code is not fatal here
govulncheck ./... || true

# --- JavaScript / TypeScript audit (if applicable) ---
if [ -f package.json ]; then
  echo "[+] JS audit"
  # Prefer pnpm if present; otherwise npm
  if command -v corepack >/dev/null 2>&1; then
    corepack enable || true
  fi

  if command -v pnpm >/dev/null 2>&1; then
    echo "    -> Using pnpm"
    pnpm install
    pnpm audit --fix || true
  else
    echo "    -> Using npm"
    if [ -f package-lock.json ]; then
      npm ci || npm install
    else
      npm install
    fi
    npm audit fix || true
  fi
fi

echo "[+] Done. Uncommitted changes:"
git status --porcelain || true

popd >/dev/null
