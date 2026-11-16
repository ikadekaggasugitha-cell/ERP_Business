# manufacturing-service — Manufacturing (SVC-014)

Backend service for `manufacturing_db`, built from the **ap-service / gl-service**
blueprint (SDD-001). Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9).
The `internal/platform` package is copied verbatim from the reference.

```
services/manufacturing-service/
├── cmd/server/main.go                        # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql                     # sqlc catalog (plain DDL; source of truth = db/migrations/manufacturing_db)
│   └── queries/work_order.sql                # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── manufacturingdb/                       # sqlc-GENERATED — do not edit
│   ├── platform/postgres.go                   # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/work_order_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspect          | ap-service                             | manufacturing-service                                  |
| --------------- | -------------------------------------- | ------------------------------------------------------ |
| DB / schema     | `ap_db` / `ap`                         | `manufacturing_db` / `mfg`                             |
| Principal       | `ap_invoice` + `ap_invoice_line`       | `mfg.work_order`                                       |
| Child aggregate | `ap_invoice_line` (of the invoice)     | `mfg.bill_of_material` + `mfg.bom_line` (BOM; work_order has no line table) |
| Partition key   | `invoice_date`                         | none (single-column PK on `id`)                        |
| Doc-number      | `invoice_no`                           | `wo_no` (UNIQUE `organization_id, wo_no`)              |
| Transition      | Draft→Recorded (optimistic lock)       | Planned→Released (optimistic lock, on work_order)      |
| Outbox          | `ap.outbox` (RANGE-partitioned)        | `mfg.outbox` (RANGE-partitioned)                       |

`work_order` is the principal aggregate carrying the state machine
(Planned→Released→InProgress→Completed→Closed); it has no line table, so the
header/line (CreateWithChildren) pattern is exercised on the **BOM** aggregate
(`bill_of_material` + `bom_line`), which the work order references by `bom_id`.
Everything else (RLS via `WithOrgScope`, outbox relay, migration pipeline, CI)
mirrors the reference — see ERP-SDD BAB 2 and `ap-service/README.md`.

## The migration pipeline

Schema lives in `db/migrations/manufacturing_db` and is applied with **golang-migrate**:

```bash
make migrate-up        # migrate -path ../../db/migrations/manufacturing_db -database $DATABASE_URL up
make migrate-version
make migrate-down      # roll back one step
```

Order in a fresh environment: `roles.sql` (once/cluster) → `migrate-up`
(as owner/`erp_migrator`) → `make grants`. Then run the service connecting as
`erp_app_rw` so RLS applies.

## Regenerating queries

Edit `db/queries/*.sql`, then `make sqlc`. `make sqlc-verify` (`sqlc diff`) fails CI
if the committed `internal/manufacturingdb` is stale. The catalog
`db/schema/schema.sql` is intentionally plain DDL — the authoritative schema (RLS,
triggers, partitioned outbox) is `db/migrations/manufacturing_db`.
