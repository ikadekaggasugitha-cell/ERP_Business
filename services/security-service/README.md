# security-service — Security Management (SVC-003)

Backend service for `security_db` (schema `sec`), built from the **ap-service / gl-service**
blueprint (SDD-001). Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9).
The `internal/platform` package is copied verbatim from the reference. Implements RBAC +
Scope-Based Access (BRD 9.4, BR-031..040).

```
services/security-service/
├── cmd/server/main.go                 # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql              # sqlc catalog (plain DDL; source of truth = db/migrations/security_db)
│   └── queries/role.sql               # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── securitydb/                    # sqlc-GENERATED (db.go, models.go, role.sql.go) — do not edit
│   ├── platform/postgres.go           # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/role_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspek           | ap-service                          | security-service                                            |
| --------------- | ----------------------------------- | ----------------------------------------------------------- |
| DB / skema      | `ap_db` / `ap`                      | `security_db` / `sec`                                       |
| Aggregate       | `ap_invoice` + `ap_invoice_line`    | `role` + `role_permission` (RBAC grants)                    |
| Partition key   | `invoice_date` (composite PK)       | none — principal PK is `(id)`                               |
| Uniqueness      | `invoice_no` (registry trigger)     | `code` per scope (`uq_role_org_code`)                       |
| Transisi status | Draft→Recorded (optimistic lock)    | **none** — `sec.role` has no `status` column (see below)    |
| List filter     | by `status`                         | by `is_system` (system vs. tenant-defined roles)            |
| Event (EOS)     | `erp.ap.invoice.recorded.v1`        | role/permission change events (outbox relay)                |

Everything else (RLS via `WithOrgScope`, outbox relay, migration pipeline, CI) is
identical to the reference — see ERP-SDD BAB 2 and `ap-service/README.md`.

## Schema anomaly — no status transition

Unlike the AP invoice (Draft→Recorded) or organization (Draft→Active) aggregates, the
`sec.role` table carries `row_version` for optimistic locking but has **no `status`
column** — RBAC roles are configuration, not lifecycle-managed documents. Per the
blueprint rule ("emit a transition only when the principal has BOTH `status` and
`row_version`"), this service therefore ships Create / Get / List + the outbox relay,
and no `:execrows` status-transition query.

## The migration pipeline

Schema lives in `db/migrations/security_db` (repo-wide set) and is applied with **golang-migrate**:

```bash
make migrate-up        # migrate -path ../../db/migrations/security_db -database $DATABASE_URL up
make migrate-version
make migrate-down      # roll back one step
```

Order in a fresh environment: `roles.sql` (once/cluster) → `migrate-up` (as owner/`erp_migrator`)
→ `make grants` → `make seed` (permissions + system roles). Then run the service connecting as `erp_app_rw` so RLS applies.

## RLS per request — the core pattern

The pool authenticates as **`erp_app_rw`** (no `BYPASSRLS`), and every scoped call runs
in a transaction that first injects the caller's org scope:

```go
db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
    q := queries.WithTx(tx)
    _, err := q.CreateRole(ctx, arg)
    return err
})
// -> BEGIN; SELECT set_config('app.org_scope', '{org-uuids}', true); <queries...>; COMMIT
```

## Regenerating queries

Edit `db/queries/*.sql`, then `make sqlc`. `make sqlc-verify` (`sqlc diff`) fails CI if the
committed `internal/securitydb` is stale. The catalog `db/schema/schema.sql` is intentionally plain
DDL — the authoritative schema (partitioned outbox, RLS, triggers, scope-based access) is `db/migrations/security_db`.

## Productionizing notes

- `permission`, `role_scope` and the full role→scope model are handled at the service boundary;
  this scaffold covers the principal role aggregate + its permission grants.
- Outbox draining runs without org scope (technical table, no RLS) as a system relay (SDD-022/030).
