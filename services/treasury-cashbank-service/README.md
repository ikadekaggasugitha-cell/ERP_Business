# treasury-cashbank-service — Treasury, Cash & Bank (SVC-011)

Backend service for `treasury_db`, built from the **ap-service** blueprint (SDD-001).
Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9). The `internal/platform`
package is copied verbatim from the reference.

```
services/treasury-cashbank-service/
├── cmd/server/main.go                  # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql               # sqlc catalog (plain DDL; source of truth = db/migrations/treasury_db)
│   └── queries/payment.sql             # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── treasurydb/                     # sqlc-GENERATED — do not edit
│   ├── platform/postgres.go            # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/payment_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspek           | ap-service                        | treasury-cashbank-service                            |
| --------------- | --------------------------------- | ---------------------------------------------------- |
| DB / skema      | `ap_db` / `ap`                    | `treasury_db` / `trs`                                |
| Aggregate       | `ap_invoice` + `ap_invoice_line`  | `trs.payment` + `trs.payment_allocation`             |
| Partition key   | `invoice_date`                    | none — `trs.payment` is not partitioned              |
| Doc-number      | `invoice_no` registry trigger     | `payment_no` (DB unique `uq_payment_no`, no registry) |
| Transisi status | Draft→Recorded (optimistic lock)  | Draft→Settled (optimistic lock, `SettlePayment`)     |
| Event (EOS)     | `erp.ap.invoice.recorded.v1`      | Payment is Exactly-Once via `correlation_id` + outbox (ADR-005) |
| Outbox          | `ap.outbox` (partitioned, relay)  | `trs.outbox` (partitioned by `created_at`, relay)    |

Everything else (RLS via `WithOrgScope`, outbox relay, migration pipeline, CI) is
identical to the reference — see ERP-SDD BAB 2 and `ap-service/README.md`.

## The migration pipeline

Schema lives in `db/migrations/treasury_db` and is applied with **golang-migrate**:

```bash
make migrate-up        # migrate -path ../../db/migrations/treasury_db -database $DATABASE_URL up
make migrate-version
make migrate-down      # roll back one step
```

The catalog `db/schema/schema.sql` is intentionally plain DDL — the authoritative
schema (partitioned outbox, RLS, triggers) is `db/migrations/treasury_db`.
`make sqlc-verify` (`sqlc diff`) fails CI if the committed `internal/treasurydb` is stale.
