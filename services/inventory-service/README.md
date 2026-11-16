# inventory-service — Inventory (SVC-013)

Backend service for `inventory_db`, built from the **ap-service / gl-service**
blueprint (SDD-001). Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9).
The `internal/platform` package is copied verbatim from the reference.

```
services/inventory-service/
├── cmd/server/main.go                    # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql                 # sqlc catalog (plain DDL; source of truth = db/migrations/inventory_db)
│   └── queries/stock.sql                 # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── inventorydb/                       # sqlc-GENERATED — do not edit
│   ├── platform/postgres.go               # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/stock_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspect          | ap-service                             | inventory-service                                      |
| --------------- | -------------------------------------- | ------------------------------------------------------ |
| DB / schema     | `ap_db` / `ap`                         | `inventory_db` / `inv`                                 |
| Principal       | `ap_invoice` + `ap_invoice_line`       | `inv.stock_movement` (partitioned) + `inv.stock_reservation` |
| Partition key   | `invoice_date`                         | `movement_date` (composite PK `id, movement_date`)     |
| Child table     | `ap_invoice_line`                      | none — `stock_movement` has no line table              |
| Transition      | Draft→Recorded (on the aggregate)      | Reserved→Released (on `stock_reservation`; movement has no status) |
| Outbox          | `ap.outbox` (RANGE-partitioned)        | `inv.outbox` (RANGE-partitioned)                       |

`stock_movement` is a partitioned aggregate with **no status column**, so the
optimistic-locked state transition lives on `stock_reservation`
(Reserved→Released). Everything else (RLS via `WithOrgScope`, outbox relay,
migration pipeline, CI) mirrors the reference — see ERP-SDD BAB 2 and
`ap-service/README.md`.

## The migration pipeline

Schema lives in `db/migrations/inventory_db` and is applied with **golang-migrate**:

```bash
make migrate-up        # migrate -path ../../db/migrations/inventory_db -database $DATABASE_URL up
make migrate-version
make migrate-down      # roll back one step
```

Order in a fresh environment: `roles.sql` (once/cluster) → `migrate-up`
(as owner/`erp_migrator`) → `make grants`. Then run the service connecting as
`erp_app_rw` so RLS applies.

## Regenerating queries

Edit `db/queries/*.sql`, then `make sqlc`. `make sqlc-verify` (`sqlc diff`) fails CI
if the committed `internal/inventorydb` is stale. The catalog `db/schema/schema.sql`
is intentionally plain DDL — the authoritative schema (partitioning, generated
`available_qty`, RLS, triggers) is `db/migrations/inventory_db`.
