#!/usr/bin/env bash
# Apply golang-migrate migrations to ERP databases without docker-compose.
#
# Usage:
#   ./db/ops/migrate.sh <db_name> [target_db]   # apply one migration dir; target_db defaults to <db_name>
#   ./db/ops/migrate.sh all                     # every db/migrations/*_db in name order (fail-fast)
#
# Environment:
#   MIGRATE_DSN_PREFIX  DSN prefix; default postgres://postgres:postgres@localhost:5432/
#                       (matches the docker-compose `db` service credentials)
#
# Requires the golang-migrate CLI (`migrate`) on PATH (brew install golang-migrate).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MIGRATIONS_DIR="$ROOT/db/migrations"
DSN_PREFIX="${MIGRATE_DSN_PREFIX:-postgres://postgres:postgres@localhost:5432/}"

command -v migrate >/dev/null 2>&1 || { echo "error: 'migrate' CLI not on PATH (brew install golang-migrate)" >&2; exit 1; }
[ -d "$MIGRATIONS_DIR" ] || { echo "error: $MIGRATIONS_DIR not found" >&2; exit 1; }

targets=()
target_db=""
case "${1:-}" in
  all)
    for d in "$MIGRATIONS_DIR"/*_db; do [ -d "$d" ] && targets+=("$(basename "$d")"); done
    [ ${#targets[@]} -gt 0 ] || { echo "error: no *_db directories under $MIGRATIONS_DIR" >&2; exit 1; }
    ;;
  "")
    echo "usage: $0 <db_name> [target_db] | all" >&2
    echo "available: $(ls "$MIGRATIONS_DIR" | tr '\n' ' ')" >&2
    exit 2
    ;;
  *)
    [ -d "$MIGRATIONS_DIR/$1" ] || { echo "error: no migration dir '$1' (see usage)" >&2; exit 2; }
    targets+=("$1")
    target_db="${2:-$1}"
    ;;
esac

rc=0
for dir_name in "${targets[@]}"; do
  db="$dir_name"
  if [ -n "$target_db" ] && [ "${#targets[@]}" -eq 1 ]; then
    db="$target_db"
  fi
  echo "== $dir_name -> $db"
  if ! migrate -path "$MIGRATIONS_DIR/$dir_name" -database "${DSN_PREFIX}${db}?sslmode=disable" up; then
    echo "FAILED: $dir_name -> $db" >&2
    rc=1
    break
  fi
done
exit "$rc"
