#!/usr/bin/env bash
# Non-destructive backup verification: restore a dump into <db>_verify and
# compare row counts of every base table against the live DB. Fails on mismatch.
# Usage: ./verify_restore.sh <db_name> <dump_file>
set -euo pipefail
DB="${1:?db}"; DUMP="${2:?dump}"; V="${DB}_verify"

psql -v ON_ERROR_STOP=1 -c "DROP DATABASE IF EXISTS $V" >/dev/null
psql -v ON_ERROR_STOP=1 -c "CREATE DATABASE $V" >/dev/null
pg_restore --no-owner -j 4 -d "$V" "$DUMP" >/dev/null 2>&1 || true

# compare per-table counts (exclude partitions; compare parents/regular tables)
SQL="SELECT format('%s.%s', schemaname, relname) FROM pg_stat_user_tables
     WHERE relname !~ '_y2[0-9]{3}m[0-9]{2}\$' ORDER BY 1"
fail=0
while read -r t; do
  [ -z "$t" ] && continue
  a=$(psql -tAqc "SELECT count(*) FROM $t" "$DB")
  b=$(psql -tAqc "SELECT count(*) FROM $t" "$V")
  if [ "$a" != "$b" ]; then echo "  MISMATCH $t: live=$a restored=$b"; fail=1; fi
done < <(psql -tAqc "$SQL" "$DB")

psql -v ON_ERROR_STOP=1 -c "DROP DATABASE $V" >/dev/null
[ $fail -eq 0 ] && echo "VERIFY $DB: PASS (all table counts match)" || { echo "VERIFY $DB: FAIL"; exit 1; }
