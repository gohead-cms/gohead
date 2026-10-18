#!/usr/bin/env bash
# Runs `go test` with the given arguments and, when it fails, reports the
# failing tests and their output as GitHub annotations, so failures are
# readable on the pull request without opening the job log.
set -uo pipefail

log=$(mktemp)
go test "$@" 2>&1 | tee "$log"
status=${PIPESTATUS[0]}
if [ "$status" -ne 0 ]; then
  # Failure lines with a little context, without passing tests' noise.
  grep -E -A12 -- '--- FAIL|^FAIL|panic:|_test\.go:[0-9]+:|cannot|undefined' "$log" \
    | grep -vE '^\s*(=== RUN|--- PASS|=== PAUSE|=== CONT|PASS$|ok )' \
    | grep -vE '^\{"level"|^\[GIN|\[[0-9.]+ms\]' \
    | head -400 > "$log.fail"
  # Annotations are capped at 10 per step: send the output in chunks.
  split -l 40 "$log.fail" "$log.chunk."
  n=0
  for chunk in "$log".chunk.*; do
    n=$((n + 1)); [ "$n" -gt 10 ] && break
    msg=$(sed -e 's/%/%25/g' "$chunk" | awk 'BEGIN{ORS="%0A"} {print}')
    echo "::error title=go test failures (part $n)::$msg"
  done
fi
exit "$status"
