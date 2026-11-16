# workflow-service — Workflow Management (SVC-004)

Backend service for `workflow_db`, built from the **ap-service / gl-service** blueprint
(SDD-001). Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9). The
`internal/platform` package is copied verbatim from the reference.

```
services/workflow-service/
├── cmd/server/main.go                 # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql              # sqlc catalog (plain DDL; source of truth = db/migrations/workflow_db)
│   └── queries/workflow.sql           # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── workflowdb/                     # sqlc-GENERATED (db.go, models.go, workflow.sql.go) — do not edit
│   ├── platform/postgres.go           # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/workflow_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspek           | ap-service                                   | workflow-service                                       |
| --------------- | -------------------------------------------- | ------------------------------------------------------ |
| DB / skema      | `ap_db` / `ap`                               | `workflow_db` / `wf`                                   |
| Aggregate       | `ap_invoice` + `ap_invoice_line`             | `workflow_instance` + `workflow_task`                  |
| Partition key   | `invoice_date` (aggregate partitioned)       | none (aggregate not partitioned; only outbox is)       |
| Doc-number      | `invoice_no` (registry trigger)              | none                                                   |
| Transisi status | Draft→Recorded (optimistic lock)             | Pending→Approved (optimistic lock)                     |
| Outbox          | RANGE-partitioned by `created_at`, PK (id,created_at) | identical — RANGE-partitioned producer         |

Everything else (RLS via `WithOrgScope`, outbox relay, migration pipeline, CI) is
identical to the reference — see ERP-SDD BAB 2 and `ap-service/README.md`.

## RLS per request — the core pattern

The pool authenticates as **`erp_app_rw`** (no `BYPASSRLS`), and every scoped call
runs in a transaction that first injects the caller's org scope via
`set_config('app.org_scope', ..., true)`. The authoritative schema (RLS, triggers,
partitioned outbox) lives in `db/migrations/workflow_db`; `db/schema/schema.sql` is
intentionally plain DDL for the sqlc `diff` gate.
