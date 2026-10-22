#!/usr/bin/env bash
# Runs a command and, when it fails, reports its output as GitHub
# annotations (titled with the first argument), so the failure is readable
# on the pull request without opening the job log.
#
#   scripts/annotate-failure.sh "go vet" go vet ./...
set -uo pipefail

title=$1
shift
log=$(mktemp)
"$@" 2>&1 | tee "$log"
status=${PIPESTATUS[0]}
if [ "$status" -ne 0 ]; then
  head -400 "$log" > "$log.head"
  # Annotations are capped at 10 per step: send the output in chunks.
  split -l 40 "$log.head" "$log.chunk."
  n=0
  for chunk in "$log".chunk.*; do
    n=$((n + 1)); [ "$n" -gt 9 ] && break
    msg=$(sed -e 's/%/%25/g' -e 's/\r//g' "$chunk" | awk 'BEGIN{ORS="%0A"} {print}')
    echo "::error title=$title (part $n)::$msg"
  done
fi
exit "$status"
