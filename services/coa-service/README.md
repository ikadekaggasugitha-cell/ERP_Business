# coa-service — Financial Org, Fiscal Calendar & Chart of Accounts (SVC-006)

Backend service for `coa_db`, built from the **ap-service / gl-service** blueprint
(SDD-001). Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9). The
`internal/platform` package is copied verbatim from the reference.

```
services/coa-service/
├── cmd/server/main.go                  # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql               # sqlc catalog (plain DDL; source of truth = db/migrations/coa_db)
│   └── queries/account.sql             # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── coadb/                          # sqlc-GENERATED — do not edit
│   ├── platform/postgres.go            # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/account_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspek           | ap-service                             | coa-service                                           |
| --------------- | -------------------------------------- | ----------------------------------------------------- |
| DB / skema      | `ap_db` / `ap`                         | `coa_db` / `fin` + `coa` (two schemas)                |
| Aggregate       | `ap_invoice` + `ap_invoice_line`       | `coa.account` (+ `coa.posting_rule`, `fin.financial_organization`) |
| Partition key   | `invoice_date` (aggregate partitioned) | none — `coa.account` is not partitioned               |
| Doc-number      | `invoice_no` registry trigger          | none                                                  |
| Transisi status | Draft→Recorded (optimistic lock)       | none — `coa.account` has no `status` column           |
| Outbox          | `ap.outbox` (partitioned, relay)       | `coa.outbox` (partitioned by `created_at`, relay)     |

Everything else (RLS via `WithOrgScope`, outbox relay, migration pipeline, CI) is
identical to the reference — see ERP-SDD BAB 2 and `ap-service/README.md`.

## The migration pipeline

Schema lives in `db/migrations/coa_db` and is applied with **golang-migrate**:

```bash
make migrate-up        # migrate -path ../../db/migrations/coa_db -database $DATABASE_URL up
make seed              # load coa_db reference data (db/seeds/coa_db/seed.sql)
make migrate-down      # roll back one step
```

The catalog `db/schema/schema.sql` is intentionally plain DDL — the authoritative
schema (partitioned outbox, RLS, triggers) is `db/migrations/coa_db`. `make sqlc-verify`
(`sqlc diff`) fails CI if the committed `internal/coadb` is stale.
