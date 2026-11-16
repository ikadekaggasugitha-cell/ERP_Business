# DB Operations — Roles, Grants & Runbook

Production hardening that makes RLS actually enforced (REVIEW O1) and applies least privilege.

## Role model

| Role           | Login | Owns objects | RLS applies | Purpose                                  |
| -------------- | ----- | ------------ | ----------- | ---------------------------------------- |
| `erp_migrator` | no*   | **yes**      | (owner→n/a) | Runs migrations; owns schema/tables.     |
| `erp_app_rw`   | no*   | no           | **yes**     | Application read/write.                  |
| `erp_app_ro`   | no*   | no           | **yes**     | Reporting / read-only.                   |

\* Group roles (NOLOGIN). Per-environment **login** roles are created and granted membership:
```sql
CREATE ROLE gl_app        LOGIN PASSWORD :'pw' IN ROLE erp_app_rw;
CREATE ROLE reporting_app LOGIN PASSWORD :'pw' IN ROLE erp_app_ro;
```
None have `SUPERUSER` or `BYPASSRLS` → RLS is enforced for the app (owner/superuser bypass RLS, which is why the app must **not** be the owner).

## Apply order

```bash
# 1) once per cluster
psql -U postgres -f db/ops/roles.sql
# 2) migrations as the owner (erp_migrator in prod; superuser in dev)
migrate -path db/migrations/gl_db -database "$GL_DSN" up
# 3) grants, once per database (re-run after adding schemas/tables)
for db in <all dbs>; do psql -U postgres -d "$db" -f db/ops/apply_grants.sql; done
```

## Per-request scope (RLS)

The application sets the caller's organization scope per transaction before running queries:
```sql
SET LOCAL app.org_scope = '{<uuid>,<uuid>,...}';  -- Organization Tree path
```
Unset ⇒ `current_org_scope()` returns NULL ⇒ policy denies all rows (deny-by-default).

## Grant strategy for partitioned tables

`apply_grants.sql` grants on **parent/regular tables only, not child partitions**, so the app
touches partitioned tables only via the parent. Verified: as `erp_app_rw`, `SELECT` via the
parent is RLS-isolated, while a direct `SELECT` on a partition is **permission denied**.
Per-partition RLS (migration `000003`) is the second layer.

## Verified (PostgreSQL 16)

- `erp_app_rw`/`erp_app_ro`/`erp_migrator`: `super=false, bypassrls=false`.
- `erp_app_rw` scope=orgA ⇒ parent visible=1 (isolated); direct partition ⇒ permission denied.
- `erp_app_ro` ⇒ INSERT denied (read-only).

## Related runbook items (from REVIEW O2–O4)

- **O2** reporting projector: connect with a scope covering the orgs it projects (or a dedicated writer role).
- **O3** partition provisioning: schedule `public.ensure_future_partitions()` (see `db/ops/partition_maintenance.sql`) via pg_cron; it calls `enable_partition_rls()` for each new partition.
- **O4** expand–contract: additive columns nullable→backfill→constraint; large indexes via `CREATE INDEX CONCURRENTLY` (separate, non-transactional migration).
