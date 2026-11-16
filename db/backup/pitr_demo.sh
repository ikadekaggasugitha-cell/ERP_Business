#!/usr/bin/env bash
# Point-in-Time Recovery demo/validation (VERIFIED on PostgreSQL 16).
# Reproduces: base backup -> activity -> disaster -> recover to a target time on
# a SECOND instance, proving pre-target data survives and the disaster is undone.
#
# Prerequisite: primary started with WAL archiving, e.g.:
#   postgres -c wal_level=replica -c archive_mode=on \
#     -c "archive_command=test ! -f $ARCHIVE/%f && cp %p $ARCHIVE/%f" \
#     -c max_wal_senders=10 -c archive_timeout=60
# Run as the postgres OS user. Adjust paths/ports as needed.
set -euo pipefail
ARCHIVE="${ARCHIVE:-/var/lib/postgresql/archive}"
RESTORE="${RESTORE:-/var/lib/postgresql/restore}"
PORT="${PORT:-5433}"
DB="${DB:-pitrdb}"

echo "1) base backup (before the activity we want to recover)"
rm -rf "$RESTORE"
pg_basebackup -h /var/run/postgresql -U postgres -D "$RESTORE" -Fp -Xstream -c fast

echo "2) post-backup activity + capture recovery target + disaster"
psql -U postgres -c "CREATE DATABASE $DB" 2>/dev/null || true
psql -U postgres -d "$DB" -c "CREATE TABLE IF NOT EXISTS m2(id int primary key, note text);"
psql -U postgres -d "$DB" -c "INSERT INTO m2 VALUES (1,'A-before-target');"
TARGET="$(psql -U postgres -d "$DB" -tAc 'SELECT now();')"
echo "   recovery_target_time = $TARGET"
psql -U postgres -d "$DB" -c "SELECT pg_sleep(1); INSERT INTO m2 VALUES (2,'B-after-target'); DROP TABLE m2;"
psql -U postgres -tAc "SELECT pg_switch_wal();" >/dev/null   # flush WAL up to now into archive
sleep 2

echo "3) configure the restore data dir for PITR"
touch "$RESTORE/recovery.signal"
cat >> "$RESTORE/postgresql.auto.conf" <<CONF
restore_command = 'cp $ARCHIVE/%f %p'
recovery_target_time = '$TARGET'
recovery_target_action = 'promote'
CONF

echo "4) start the recovery instance on port $PORT"
pg_ctl -D "$RESTORE" -l /tmp/pitr_restore.log -o "-p $PORT" -w start
sleep 3
grep -iE 'point-in-time|consistent|recovery stopping|redo done|promot' /tmp/pitr_restore.log | tail -6

echo "5) verify: pre-target row present, post-target row + disaster absent"
psql -U postgres -p "$PORT" -d "$DB" -tAc \
  "SELECT 'm2_restored='||(to_regclass('public.m2') IS NOT NULL)::text||' rows='||COALESCE(string_agg(id||':'||note,', ' ORDER BY id),'(none)') FROM m2;"

# Verified result: m2_restored=true rows=1:A-before-target
