# Partition Maintenance & Retention (Step 3)

Every partitioned database ships three reusable maintenance functions (migration
`*_partition_hardening`) so partitions are provisioned and archived **without code changes**.

## Partitioned tables

| DB | Table | Key | Granularity |
| -- | ----- | --- | ----------- |
| gl_db | `gl.journal_entry`, **`gl.journal_line`** | `posting_date` | monthly |
| ap_db / ar_db | `ap.ap_invoice` / `ar.ar_invoice` | `invoice_date` | monthly |
| audit_db | `audit.audit_log` | `occurred_at` | monthly |
| inventory_db | `inv.stock_movement` | `movement_date` | monthly |
| all 28 producer DBs | `<schema>.outbox` | `created_at` | monthly |

## Functions (in each partitioned DB, schema `public`)

```sql
-- create one partition (idempotent); auto-enables RLS+policy if the parent has RLS
SELECT public.create_monthly_partition('gl.journal_entry'::regclass, DATE '2027-05-01');

-- ensure partitions exist from the current month through N months ahead
SELECT public.ensure_future_partitions('gl.journal_entry'::regclass, 3);

-- DETACH (archive, not drop) partitions whose start is older than keep_months
SELECT public.detach_old_partitions('gl.journal_entry'::regclass, 24);
```

`create_monthly_partition` derives the name `<table>_yYYYYmMM`, sets the range, and — for
scoped tables (RLS-enabled parent) — enables RLS + the `scope_isolation` policy on the new
partition (closes the REVIEW M1 leak for future partitions). Outbox partitions get no RLS
(technical table), as verified.

## Scheduling (pg_cron)

Run monthly per partitioned DB (extension `pg_cron`, or an external scheduler / k8s CronJob
invoking `psql`). Provision ahead, archive behind:

```sql
-- provision 3 months ahead, 1st of each month
SELECT cron.schedule('gl-partitions-provision', '0 2 1 * *', $$
  SELECT public.ensure_future_partitions('gl.journal_entry'::regclass, 3);
  SELECT public.ensure_future_partitions('gl.journal_line'::regclass, 3);
  SELECT public.ensure_future_partitions('gl.outbox'::regclass, 3);
$$);

-- retention: detach partitions older than policy (then move to cold storage / drop)
SELECT cron.schedule('gl-partitions-archive', '30 2 1 * *', $$
  SELECT public.detach_old_partitions('gl.journal_entry'::regclass, 84);  -- keep 7y (audit/finance)
  SELECT public.detach_old_partitions('gl.outbox'::regclass, 2);          -- outbox: keep 2 months
$$);
```

## Suggested retention (align to compliance / SAD BAB 6.8)

| Table | Keep attached | After detach |
| ----- | ------------- | ------------ |
| `journal_entry` / `journal_line` | 84 months (7y) | archive tablespace / export |
| `audit_log` | ≥ 120 months | cold storage (immutable) |
| `*_invoice` | 84 months | archive |
| `stock_movement` | 24–36 months | archive |
| `outbox` | 2 months (rows purged after publish) | drop |

`detach_old_partitions` DETACHes (preserves data as a standalone table) — operators then move
to a cheap tablespace or `DROP TABLE` after export. Detach never blocks OLTP.

## Verified (PostgreSQL 16)

- `ensure_future_partitions(...,8)` created new partitions; each scoped partition came up with
  RLS + `scope_isolation` policy automatically.
- `journal_line` / `outbox` inserts route to the correct monthly partition; journal_line
  composite FK + `ON DELETE CASCADE` intact.
- `detach_old_partitions(...,3)` detached the out-of-window partitions (data retained).
