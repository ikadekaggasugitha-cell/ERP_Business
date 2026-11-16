# user-service — User Management (SVC-002)

Backend service for `user_db` (schema `usr`), built from the **ap-service / gl-service**
blueprint (SDD-001). Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9).
The `internal/platform` package is copied verbatim from the reference.
**No passwords are stored** — authentication is delegated to the IdP (SAD BAB 7.3, INT-012).

```
services/user-service/
├── cmd/server/main.go                 # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql              # sqlc catalog (plain DDL; source of truth = db/migrations/user_db)
│   └── queries/user.sql               # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── userdb/                        # sqlc-GENERATED (db.go, models.go, user.sql.go) — do not edit
│   ├── platform/postgres.go           # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/user_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspek           | ap-service                          | user-service                                             |
| --------------- | ----------------------------------- | -------------------------------------------------------- |
| DB / skema      | `ap_db` / `ap`                      | `user_db` / `usr`                                        |
| Aggregate       | `ap_invoice` + `ap_invoice_line`    | `app_user` + `user_org_assignment` (scope grants)        |
| Partition key   | `invoice_date` (composite PK)       | none — principal PK is `(id)`                             |
| Uniqueness      | `invoice_no` (registry trigger)     | `username` / `email` (unique constraints)                |
| Transisi status | Draft→Recorded (optimistic lock)    | Active→Suspended (`SuspendUser`, optimistic lock)        |
| Event (EOS)     | `erp.ap.invoice.recorded.v1`        | user provisioning/assignment events (outbox relay)       |

Everything else (RLS via `WithOrgScope`, outbox relay, migration pipeline, CI) is
identical to the reference — see ERP-SDD BAB 2 and `ap-service/README.md`.

## The migration pipeline

Schema lives in `db/migrations/user_db` (repo-wide set) and is applied with **golang-migrate**:

```bash
make migrate-up        # migrate -path ../../db/migrations/user_db -database $DATABASE_URL up
make migrate-version
make migrate-down      # roll back one step
```

Order in a fresh environment: `roles.sql` (once/cluster) → `migrate-up` (as owner/`erp_migrator`)
→ `make grants`. Then run the service connecting as `erp_app_rw` so RLS applies.

## RLS per request — the core pattern

The pool authenticates as **`erp_app_rw`** (no `BYPASSRLS`), and every scoped call runs
in a transaction that first injects the caller's org scope:

```go
db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
    q := queries.WithTx(tx)
    _, err := q.CreateAppUser(ctx, arg)
    return err
})
// -> BEGIN; SELECT set_config('app.org_scope', '{org-uuids}', true); <queries...>; COMMIT
```

## Regenerating queries

Edit `db/queries/*.sql`, then `make sqlc`. `make sqlc-verify` (`sqlc diff`) fails CI if the
committed `internal/userdb` is stale. The catalog `db/schema/schema.sql` is intentionally plain
DDL — the authoritative schema (partitioned outbox, RLS, triggers, IdP federation) is `db/migrations/user_db`.

## Productionizing notes

- `user_identity` (IdP federation) and the full assignment lifecycle are handled at the service
  boundary; this scaffold covers the principal aggregate + org/role assignments.
- Outbox draining runs without org scope (technical table, no RLS) as a system relay (SDD-022/030).
