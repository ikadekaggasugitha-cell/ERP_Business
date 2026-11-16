# audit-service — Audit Trail (SVC-018)

Backend service for `audit_db`, built from the **ap-service / gl-service** blueprint
(SDD-001). Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9). The
`internal/platform` package is copied verbatim from the reference.

```
services/audit-service/
├── cmd/server/main.go                 # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql              # sqlc catalog (plain DDL; source of truth = db/migrations/audit_db)
│   └── queries/audit.sql              # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── auditdb/                        # sqlc-GENERATED (db.go, models.go, audit.sql.go) — do not edit
│   ├── platform/postgres.go           # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/audit_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

audit-service is a **wildcard CONSUMER** — the biggest structural delta from the
producer reference:

| Aspek           | ap-service (producer)                        | audit-service (consumer)                               |
| --------------- | -------------------------------------------- | ------------------------------------------------------ |
| DB / skema      | `ap_db` / `ap`                               | `audit_db` / `audit`                                   |
| Aggregate       | `ap_invoice` + `ap_invoice_line`             | `audit_log` (append-only, partitioned by `occurred_at`)|
| Partition key   | `invoice_date`                               | `occurred_at` (composite PK `(id, occurred_at)`)        |
| Transisi status | Draft→Recorded (optimistic lock)             | **none** — no `status`/`row_version`; append-only       |
| **Outbox**      | RANGE-partitioned producer + relay queries   | **none** — consumer produces no events                  |
| Idempotency     | `processed_event` (inbound)                  | `processed_event` via `MarkEventProcessed` (ON CONFLICT DO NOTHING) |

The consumer pattern: `RecordAuditLog` appends a consumed event into the immutable
trail (RLS-scoped), `MarkEventProcessed` de-dups at-least-once delivery on the
technical `processed_event` ledger (no RLS → no scope), and `Get`/`ListByEntity`
serve the trail. There is **no** `DrainOutbox`/`MarkOutboxPublished` and no
status-transition query.

## RLS per request — the core pattern

The pool authenticates as **`erp_app_rw`** (no `BYPASSRLS`); every scoped call runs
in a transaction that first injects the caller's org scope via
`set_config('app.org_scope', ..., true)`. The authoritative schema (RANGE
partitioning, RLS, append-only immutability trigger) lives in
`db/migrations/audit_db`; `db/schema/schema.sql` is intentionally plain DDL for the
sqlc `diff` gate.
