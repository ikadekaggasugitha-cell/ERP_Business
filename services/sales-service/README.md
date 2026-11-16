# sales-service — Sales (SVC-015)

Backend service for `sales_db`, built from the **ap-service / gl-service**
blueprint (SDD-001). Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9).
The `internal/platform` package is copied verbatim from the reference.

```
services/sales-service/
├── cmd/server/main.go                        # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql                     # sqlc catalog (plain DDL; source of truth = db/migrations/sales_db)
│   └── queries/sales_order.sql               # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── salesdb/                               # sqlc-GENERATED — do not edit
│   ├── platform/postgres.go                   # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/sales_order_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspect          | ap-service                             | sales-service                                          |
| --------------- | -------------------------------------- | ------------------------------------------------------ |
| DB / schema     | `ap_db` / `ap`                         | `sales_db` / `so`                                      |
| Principal       | `ap_invoice` + `ap_invoice_line`       | `so.sales_order` + `so.so_line`                        |
| Partition key   | `invoice_date` (composite PK)          | none (single-column PK on `id`)                        |
| Doc-number      | `invoice_no`                           | `so_no` (UNIQUE `organization_id, so_no`)              |
| Transition      | Draft→Recorded (optimistic lock)       | Draft→Confirmed (optimistic lock)                      |
| Outbox          | `ap.outbox` (RANGE-partitioned)        | `so.outbox` (RANGE-partitioned)                        |

The sales order is the closest structural match to the AP invoice: a plain
(non-partitioned) header + line aggregate with a document number and an
optimistic-locked state transition (Draft→Confirmed). Everything else (RLS via
`WithOrgScope`, outbox relay, migration pipeline, CI) mirrors the reference — see
ERP-SDD BAB 2 and `ap-service/README.md`.

## The migration pipeline

Schema lives in `db/migrations/sales_db` and is applied with **golang-migrate**:

```bash
make migrate-up        # migrate -path ../../db/migrations/sales_db -database $DATABASE_URL up
make migrate-version
make migrate-down      # roll back one step
```

Order in a fresh environment: `roles.sql` (once/cluster) → `migrate-up`
(as owner/`erp_migrator`) → `make grants`. Then run the service connecting as
`erp_app_rw` so RLS applies.

## Regenerating queries

Edit `db/queries/*.sql`, then `make sqlc`. `make sqlc-verify` (`sqlc diff`) fails CI
if the committed `internal/salesdb` is stale. The catalog `db/schema/schema.sql`
is intentionally plain DDL — the authoritative schema (RLS, triggers, partitioned
outbox) is `db/migrations/sales_db`.
