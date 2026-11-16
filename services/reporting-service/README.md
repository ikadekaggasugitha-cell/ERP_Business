# reporting-service — Reporting / Dashboard read model (SVC-019)

Backend service for `reporting_db`, built from the **ap-service / gl-service**
blueprint (SDD-001) but adapted for **CQRS** (ADR-009): this is a **read model**,
not a system of record. Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9).
The `internal/platform` package is copied verbatim (RLS boundary, SDD-001).

```
services/reporting-service/
├── cmd/server/main.go                          # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql                       # sqlc catalog (plain DDL; source of truth = db/migrations/reporting_db)
│   └── queries/gl_balance_snapshot.sql         # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── reportingdb/                            # sqlc-GENERATED — do not edit
│   ├── platform/postgres.go                    # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/snapshot_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service) — CQRS read model

| Aspek            | ap-service (producer)                | reporting-service (read model)                          |
| ---------------- | ------------------------------------ | ------------------------------------------------------- |
| DB / skema       | `ap_db` / `ap`                       | `reporting_db` / `rpt`                                  |
| Role             | System of record (write side)        | Derived, denormalized, **rebuildable** projection (read side) |
| Base Entity      | full (soft-delete, optimistic lock)  | omitted — projections carry only a `refreshed_at` watermark |
| Aggregate        | `ap_invoice` + lines                 | `gl_balance_snapshot` (+ `sales_summary`, `inventory_position`) |
| Write pattern    | INSERT + status transition (optimistic lock) | **upsert-by-event** (`ON CONFLICT` on the natural key) |
| Outbox / producer| yes (RANGE-partitioned)              | **none** — consumer only                                |
| Idempotency      | producer correlation + processed_event | **`MarkEventProcessed :execrows`** (1 = first, 0 = duplicate) |
| Status transition| Draft→Recorded                       | **none**                                                |

## The read-model pattern

The projector consumes cross-domain events and writes projections idempotently:

```go
n, _ := repo.MarkEventProcessed(ctx, arg)   // 0 => already handled, skip (no org scope)
if n == 1 {
    repo.UpsertGLBalance(ctx, scope, arg)   // ON CONFLICT (org, period, account) DO UPDATE (RLS-scoped)
}
```

## Notes

- The catalog retains the UNIQUE key on `gl_balance_snapshot` (beyond the PK) because
  the upsert query targets it in `ON CONFLICT`; other read tables keep only their PK.
- `processed_event` has no `organization_id` / RLS, so `MarkEventProcessed` runs without
  org scope; projection reads/writes are org-scoped via `WithOrgScope`.
- The authoritative schema (RLS) lives in `db/migrations/reporting_db`; `db/schema/schema.sql`
  is a plain DDL catalog for sqlc only. `make sqlc-verify` fails CI if `internal/reportingdb` is stale.
