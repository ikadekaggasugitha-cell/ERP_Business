# ar-service — Accounts Receivable (SVC-009)

Backend service for `ar_db`, built as a **twin of ap-service** (SDD-001 blueprint).
Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9). The `internal/platform`
package is copied verbatim from the reference.

```
services/ar-service/
├── cmd/server/main.go                  # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql               # sqlc catalog (plain DDL; source of truth = db/migrations/ar_db)
│   └── queries/invoice.sql             # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── ardb/                           # sqlc-GENERATED — do not edit
│   ├── platform/postgres.go            # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/invoice_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspek           | ap-service                              | ar-service                                             |
| --------------- | --------------------------------------- | ------------------------------------------------------ |
| DB / skema      | `ap_db` / `ap`                          | `ar_db` / `ar`                                         |
| Aggregate       | `ap_invoice` + `ap_invoice_line`        | `ar_invoice` + `ar_invoice_line` (+ `ar_receipt_allocation`) |
| Partition key   | `invoice_date`                          | `invoice_date`                                         |
| Doc-number      | `invoice_no` (`ap.tg_register_invoice_no`) | `invoice_no` (`ar.tg_register_invoice_no`)          |
| Statuses        | Draft / Recorded / …                    | Issued / PartiallyPaid / Paid / Overdue / Cancelled (default `Issued`, no Draft) |
| Transisi status | Draft→Recorded                          | Issued→Paid (optimistic lock, `MarkInvoicePaid`)       |
| Outbox          | `ap.outbox` (partitioned, relay)        | `ar.outbox` (partitioned by `created_at`, relay)       |

Everything else (RLS via `WithOrgScope`, outbox relay, migration pipeline, CI) is
identical to the reference — see ERP-SDD BAB 2 and `ap-service/README.md`.

## The migration pipeline

Schema lives in `db/migrations/ar_db` and is applied with **golang-migrate**:

```bash
make migrate-up        # migrate -path ../../db/migrations/ar_db -database $DATABASE_URL up
make migrate-version
make migrate-down      # roll back one step
```

The catalog `db/schema/schema.sql` is intentionally plain DDL — the authoritative
schema (partitioning, RLS, triggers, doc-number registry) is `db/migrations/ar_db`.
`make sqlc-verify` (`sqlc diff`) fails CI if the committed `internal/ardb` is stale.
