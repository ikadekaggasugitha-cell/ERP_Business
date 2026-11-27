#!/usr/bin/env bash
# Runs db/ops/verify_rls.sql against every ERP database. Exit 1 if any database fails.
# Usage: PGHOST=... PGUSER=... PGPASSWORD=... bash deploy/verify-rls.sh [db ...]
set -uo pipefail
cd "$(dirname "$0")/.."

dbs=("$@")
if [ "${#dbs[@]}" -eq 0 ]; then
  # Same list as deploy/bootstrap.sh (single source of truth for the database inventory).
  read -r -a dbs <<<"$(grep -E '^ALL_DBS=' deploy/bootstrap.sh | sed -E 's/^ALL_DBS="(.*)"$/\1/')"
fi

failed=0
for db in "${dbs[@]}"; do
  if out="$(psql -v ON_ERROR_STOP=1 -X -q -d "$db" -f db/ops/verify_rls.sql 2>&1)"; then
    echo "PASS  $db"
  else
    echo "FAIL  $db"
    printf '%s\n' "$out" | sed 's/^/      /'
    failed=$((failed + 1))
  fi
done

if [ "$failed" -ne 0 ]; then
  echo "VERIFY_RLS_FAIL: ${failed} database(s)" >&2
  exit 1
fi
echo "VERIFY_RLS_PASS: ${#dbs[@]} database(s)"
