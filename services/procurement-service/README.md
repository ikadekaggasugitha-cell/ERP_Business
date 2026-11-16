# procurement-service — Procurement (SVC-012)

Backend service for `procurement_db`, built from the **ap-service / gl-service**
blueprint (SDD-001). Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9).
The `internal/platform` package is copied verbatim from the reference.

```
services/procurement-service/
├── cmd/server/main.go                        # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql                     # sqlc catalog (plain DDL; source of truth = db/migrations/procurement_db)
│   └── queries/purchase_order.sql            # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── procurementdb/                         # sqlc-GENERATED — do not edit
│   ├── platform/postgres.go                   # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/purchase_order_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspect          | ap-service                             | procurement-service                                   |
| --------------- | -------------------------------------- | ----------------------------------------------------- |
| DB / schemas    | `ap_db` / `ap`                         | `procurement_db` / `pr`, `po`, `gr` (multi-schema)    |
| Principal       | `ap_invoice` + `ap_invoice_line`       | `po.purchase_order` + `po.po_line`                    |
| Partition key   | `invoice_date` (composite PK)          | none (single-column PK on `id`)                       |
| Doc-number      | `invoice_no`                           | `po_no` (UNIQUE `organization_id, po_no`)             |
| Transition      | Draft→Recorded (optimistic lock)       | Draft→Approved (optimistic lock)                      |
| Outbox          | `ap.outbox` (RANGE-partitioned)        | `pr.outbox` (RANGE-partitioned, technical schema `pr`)|

`purchase_order` is a plain (non-partitioned) aggregate, so `Get` keys on `id`
only and the child `po_line` needs no partition-date propagation. The outbox and
`processed_event` technical tables live in schema `pr`. Everything else (RLS via
`WithOrgScope`, outbox relay, migration pipeline, CI) mirrors the reference — see
ERP-SDD BAB 2 and `ap-service/README.md`.

## The migration pipeline

Schema lives in `db/migrations/procurement_db` and is applied with **golang-migrate**:

```bash
make migrate-up        # migrate -path ../../db/migrations/procurement_db -database $DATABASE_URL up
make migrate-version
make migrate-down      # roll back one step
```

Order in a fresh environment: `roles.sql` (once/cluster) → `migrate-up`
(as owner/`erp_migrator`) → `make grants`. Then run the service connecting as
`erp_app_rw` so RLS applies.

## Regenerating queries

Edit `db/queries/*.sql`, then `make sqlc`. `make sqlc-verify` (`sqlc diff`) fails CI
if the committed `internal/procurementdb` is stale. The catalog `db/schema/schema.sql`
is intentionally plain DDL — the authoritative schema (partitioning, RLS, triggers,
doc-number uniqueness) is `db/migrations/procurement_db`.
