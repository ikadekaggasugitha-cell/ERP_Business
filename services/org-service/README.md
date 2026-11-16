# org-service — Organization Management (SVC-001)

Backend service for `org_db` (schema `org`), built from the **ap-service / gl-service**
blueprint (SDD-001). Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9).
The `internal/platform` package is copied verbatim from the reference.

```
services/org-service/
├── cmd/server/main.go                     # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql                  # sqlc catalog (plain DDL; source of truth = db/migrations/org_db)
│   └── queries/organization.sql           # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── orgdb/                              # sqlc-GENERATED (db.go, models.go, organization.sql.go) — do not edit
│   ├── platform/postgres.go               # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/organization_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspek           | ap-service                                   | org-service                                          |
| --------------- | -------------------------------------------- | ---------------------------------------------------- |
| DB / skema      | `ap_db` / `ap`                               | `org_db` / `org`                                     |
| Aggregate       | `ap_invoice` + `ap_invoice_line`             | `organization` + `organization_attribute`           |
| Partition key   | `invoice_date` (composite PK)                | none — principal PK is `(id)`                        |
| Uniqueness      | `invoice_no` (registry trigger)              | `code` per scope (`uq_organization_org_code`)        |
| Transisi status | Draft→Recorded (optimistic lock)             | Draft→Active (`ActivateOrganization`, optimistic)    |
| Event (EOS)     | `erp.ap.invoice.recorded.v1`                 | org unit upserted (outbox relay)                     |

Everything else (RLS via `WithOrgScope`, outbox relay, migration pipeline, CI) is
identical to the reference — see ERP-SDD BAB 2 and `ap-service/README.md`.

## The migration pipeline

Schema lives in `db/migrations/org_db` (repo-wide set) and is applied with **golang-migrate**:

```bash
make migrate-up        # migrate -path ../../db/migrations/org_db -database $DATABASE_URL up
make migrate-version
make migrate-down      # roll back one step
```

Order in a fresh environment: `roles.sql` (once/cluster) → `migrate-up` (as owner/`erp_migrator`)
→ `make grants` → `make seed` (org types). Then run the service connecting as `erp_app_rw` so RLS applies.

## RLS per request — the core pattern

The pool authenticates as **`erp_app_rw`** (no `BYPASSRLS`), and every scoped call runs
in a transaction that first injects the caller's org scope:

```go
db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
    q := queries.WithTx(tx)
    _, err := q.CreateOrganization(ctx, arg)
    return err
})
// -> BEGIN; SELECT set_config('app.org_scope', '{org-uuids}', true); <queries...>; COMMIT
```

## Regenerating queries

Edit `db/queries/*.sql`, then `make sqlc`. `make sqlc-verify` (`sqlc diff`) fails CI if the
committed `internal/orgdb` is stale. The catalog `db/schema/schema.sql` is intentionally plain
DDL — the authoritative schema (partitioned outbox, RLS, triggers, closure table) is `db/migrations/org_db`.

## Productionizing notes

- The org tree (`parent_id`, materialized `path`, `organization_closure`) is validated/maintained
  at the service boundary; this scaffold covers the principal aggregate + attributes.
- Outbox draining runs without org scope (technical table, no RLS) as a system relay (SDD-022/030).
