# tax-service — Tax Management (SVC-010)

Backend service for `tax_db`, built from the **ap-service** blueprint (SDD-001).
Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9). The `internal/platform`
package is copied verbatim from the reference.

```
services/tax-service/
├── cmd/server/main.go                  # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql               # sqlc catalog (plain DDL; source of truth = db/migrations/tax_db)
│   └── queries/tax.sql                 # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── taxdb/                          # sqlc-GENERATED — do not edit
│   ├── platform/postgres.go            # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/tax_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspek           | ap-service                        | tax-service                                          |
| --------------- | --------------------------------- | ---------------------------------------------------- |
| DB / skema      | `ap_db` / `ap`                    | `tax_db` / `tax`                                     |
| Aggregate       | `ap_invoice` + `ap_invoice_line`  | `tax_transaction` (+ `tax_code`)                     |
| Partition key   | `invoice_date`                    | none — `tax_transaction` is not partitioned          |
| Doc-number      | `invoice_no` registry trigger     | none                                                 |
| Transisi status | Draft→Recorded (optimistic lock)  | none — no base-entity `status` column (`efaktur_status` is a nullable doc flag) |
| Outbox          | `ap.outbox` (partitioned, relay)  | `tax.outbox` (partitioned by `created_at`, relay)    |

Everything else (RLS via `WithOrgScope`, outbox relay, migration pipeline, CI) is
identical to the reference — see ERP-SDD BAB 2 and `ap-service/README.md`.

## The migration pipeline

Schema lives in `db/migrations/tax_db` and is applied with **golang-migrate**:

```bash
make migrate-up        # migrate -path ../../db/migrations/tax_db -database $DATABASE_URL up
make seed              # load tax_db reference data (db/seeds/tax_db/seed.sql)
make migrate-down      # roll back one step
```

The catalog `db/schema/schema.sql` is intentionally plain DDL — the authoritative
schema (partitioned outbox, RLS, triggers) is `db/migrations/tax_db`. `make sqlc-verify`
(`sqlc diff`) fails CI if the committed `internal/taxdb` is stale.
