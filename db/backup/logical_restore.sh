#!/usr/bin/env bash
# Restore one service database from a custom-format dump.
# Usage: ./logical_restore.sh <db_name> <dump_file> [target_db]
set -euo pipefail
DB="${1:?db name}"; DUMP="${2:?dump file}"; TARGET="${3:-$DB}"

echo "[$(date -u +%FT%TZ)] restoring $DUMP -> $TARGET"
psql -v ON_ERROR_STOP=1 -tAc "SELECT 1 FROM pg_database WHERE datname='$TARGET'" | grep -q 1 \
  || psql -v ON_ERROR_STOP=1 -c "CREATE DATABASE $TARGET"
# --clean --if-exists makes restore idempotent onto an existing DB; -j parallelises
pg_restore --no-owner --clean --if-exists -j 4 -d "$TARGET" "$DUMP"
echo "[$(date -u +%FT%TZ)] restore complete."
