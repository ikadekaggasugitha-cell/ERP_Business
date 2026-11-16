# Backup, Restore & Disaster Recovery

Two complementary layers, both **validated on PostgreSQL 16** (see "Verified" below).

| Layer | Tool | Protects against | RPO | RTO |
| ----- | ---- | ---------------- | --- | --- |
| Logical, per-DB | `pg_dump -Fc` / `pg_restore` | Accidental DROP of a single service DB, migration gone wrong, table export | last nightly dump (≤ 24h) | minutes/DB |
| Physical + WAL (PITR) | `pg_basebackup` + WAL archiving | Node loss, corruption, "recover to 5 minutes ago" | ≤ `archive_timeout` (60s) — near-0 with streaming replica | base-restore + WAL replay (target ≤ 30 min) |

Database-per-service means each DB is backed up/restored independently — a bad deploy in one
service can be rolled back without touching the others.

## Scripts

| Script | Purpose |
| ------ | ------- |
| `logical_backup.sh`  | Dump all 30 DBs (custom format) + global roles; checksummed. |
| `logical_restore.sh` | `<db> <dump> [target]` — createdb + parallel `pg_restore --clean`. |
| `verify_restore.sh`  | `<db> <dump>` — restore into `<db>_verify`, compare every table's row count, fail on mismatch (backup integrity gate). |
| `pitr_demo.sh`       | Reproducible PITR walkthrough (base backup → disaster → recover to target). |

## Production WAL archiving (PITR prerequisite)

```conf
wal_level = replica
archive_mode = on
archive_command = 'test ! -f /wal_archive/%f && cp %p /wal_archive/%f'   # or S3/wal-g/pgBackRest
archive_timeout = 60          # bounds RPO to 60s of WAL
max_wal_senders = 10
```
Prefer **pgBackRest** or **wal-g** in production (compression, S3, incremental base backups,
retention, and one-command PITR) — the scripts here show the native mechanics they wrap.
Add a streaming **standby** (physical replication) for near-zero RPO + fast failover.

## Restore procedures

**Single DB (logical):**
```bash
./logical_restore.sh gl_db /var/backups/erp/<ts>/gl_db.dump
# then re-apply grants + seeds if restoring to a fresh cluster:
psql -d gl_db -f db/ops/apply_grants.sql
```

**Point-in-time (physical):** see `pitr_demo.sh` — base backup + `recovery_target_time` +
`restore_command` reading the WAL archive, promoted on recovery.

## Cadence & retention (suggested)

- Base backup: nightly (full) + WAL archived continuously.
- Logical dumps: nightly per DB, `verify_restore.sh` on a sample DB each run.
- Retention: 7–14 days hot, 90 days offsite; align WAL retention to the oldest base backup you keep.
- **Quarterly DR drill**: restore to an isolated environment and run the regression tests
  (`db/tests/`) against it.

## Verified (PostgreSQL 16, this repo)

- **Logical:** dumped `gl_db` (500 journal entries + 500 registry rows), dropped the database,
  restored from the `-Fc` dump → **counts matched and all 13 partitions preserved**.
- **PITR:** base backup → `INSERT A` → capture target → `INSERT B` → `DROP TABLE` (disaster) →
  recovered a second instance to the target time. Log: *"recovery stopping before commit … time
  08:13:13"*; recovered DB had **A present, B and the DROP absent** (`m2_restored=true,
  rows=1:A-before-target`).
