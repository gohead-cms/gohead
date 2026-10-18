#!/usr/bin/env bash
# Runs `go test` with the given arguments and, when it fails, reports the
# failing tests and their output as GitHub annotations, so failures are
# readable on the pull request without opening the job log.
set -uo pipefail

log=$(mktemp)
go test "$@" 2>&1 | tee "$log"
status=${PIPESTATUS[0]}
if [ "$status" -ne 0 ]; then
  # Failure lines with some context, without log noise (JSON log lines,
  # GIN request logs) and passing tests.
  grep -vE '^\{|^\[GIN|^\s*(=== RUN|--- PASS|=== PAUSE|=== CONT|PASS$)|^ok ' "$log" \
    | grep -E -A10 -- '--- FAIL|^FAIL|panic:|_test\.go:[0-9]+:' \
    | head -400 > "$log.fail"
  # Database errors logged by the code under test (MySQL "Error 1234",
  # PostgreSQL "SQLSTATE"), most frequent first.
  grep -oE '(Error [0-9]{4} \([0-9A-Z]{5}\)|SQLSTATE [0-9A-Z]{5}|ERROR: )[^"]{0,220}' "$log" \
    | sort | uniq -c | sort -rn | head -40 > "$log.db"
  if [ -s "$log.db" ]; then
    msg=$(sed -e 's/%/%25/g' "$log.db" | awk 'BEGIN{ORS="%0A"} {print}')
    echo "::error title=database errors::$msg"
  fi
  # Annotations are capped at 10 per step: send the output in chunks.
  split -l 40 "$log.fail" "$log.chunk."
  n=0
  for chunk in "$log".chunk.*; do
    n=$((n + 1)); [ "$n" -gt 9 ] && break
    msg=$(sed -e 's/%/%25/g' "$chunk" | awk 'BEGIN{ORS="%0A"} {print}')
    echo "::error title=go test failures (part $n)::$msg"
  done
fi
exit "$status"
