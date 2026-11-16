# ap-service — Accounts Payable (SVC-008)

Backend service for `ap_db`, built from the **gl-service** blueprint (SDD-001). Stack:
**pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9). This is the first replicated
service after the reference; the `internal/platform` package is copied verbatim.

```
services/ap-service/
├── cmd/server/main.go                 # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql              # sqlc catalog (plain DDL; source of truth = db/migrations/ap_db)
│   └── queries/invoice.sql            # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── apdb/                          # sqlc-GENERATED (db.go, models.go, invoice.sql.go) — do not edit
│   ├── platform/postgres.go           # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/invoice_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (gl-service)

| Aspek           | gl-service                         | ap-service                                          |
| --------------- | ---------------------------------- | --------------------------------------------------- |
| DB / skema      | `gl_db` / `gl`                     | `ap_db` / `ap`                                       |
| Aggregate       | `journal_entry` + `journal_line`   | `ap_invoice` + `ap_invoice_line` (+ `ap_payment_allocation`) |
| Partition key   | `posting_date`                     | `invoice_date`                                      |
| Doc-number      | `journal_no` (registry trigger)    | `invoice_no` (`ap.tg_register_invoice_no`)          |
| Transisi status | Draft→Posted (`RecordInvoice`↔`PostJournalEntry`) | Draft→Recorded (optimistic lock)     |
| Event (EOS)     | `erp.gl.journal.posted.v1`         | `erp.ap.invoice.recorded.v1`                        |

Everything else (RLS via `WithOrgScope`, outbox relay, migration pipeline, CI) is
identical to the reference — see ERP-SDD BAB 2 and `gl-service/README.md`.

## The migration pipeline

Schema lives in `db/migrations/ap_db` (repo-wide set) and is applied with **golang-migrate**:

```bash
make migrate-up        # migrate -path ../../db/migrations/ap_db -database $DATABASE_URL up
make migrate-version
make migrate-down      # roll back one step
```

Order in a fresh environment: `roles.sql` (once/cluster) → `migrate-up` (as owner/`erp_migrator`)
→ `make grants`. Then run the service connecting as `erp_app_rw` so RLS applies.

## RLS per request — the core pattern

Identical to gl-service: the pool authenticates as **`erp_app_rw`** (no `BYPASSRLS`),
and every scoped call runs in a transaction that first injects the caller's org scope:

```go
db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
    q := queries.WithTx(tx)
    _, err := q.CreateInvoice(ctx, arg)
    return err
})
// -> BEGIN; SELECT set_config('app.org_scope', '{org-uuids}', true); <queries...>; COMMIT
```

## Regenerating queries

Edit `db/queries/*.sql`, then `make sqlc`. `make sqlc-verify` (`sqlc diff`) fails CI if the
committed `internal/apdb` is stale. The catalog `db/schema/schema.sql` is intentionally plain
DDL — the authoritative schema (partitioning, RLS, triggers, doc-number registry) is `db/migrations/ap_db`.

## Productionizing notes

- Money columns surface as `pgtype.Numeric`; wrap with a decimal type at the service boundary (SDD-006).
- Outbox draining runs without org scope (technical table, no RLS) as a system relay (SDD-022/030).
- `ap-invoice` is an EOS producer of `erp.ap.invoice.recorded.v1` (ADR-005); Idempotency-Key mandatory (SDD-030).
