# gl-service — Reference Repository Layer & Migration Pipeline

Reference implementation of a service's data layer against `gl_db`, and the template for the
other 29 services. Stack (per decision): **pgx v5 + sqlc + golang-migrate**.

```
services/gl-service/
├── cmd/server/main.go                 # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql              # sqlc catalog (plain DDL; source of truth = db/migrations/gl_db)
│   └── queries/journal.sql            # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── gldb/                          # sqlc-GENERATED (db.go, models.go, journal.sql.go) — do not edit
│   ├── platform/postgres.go           # pgxpool + WithOrgScope (RLS boundary)
│   └── repository/journal_repository.go
├── sqlc.yaml   Makefile   go.mod
```

## The migration pipeline

Schema lives in `db/migrations/gl_db` (the repo-wide migration set) and is applied with
**golang-migrate** — verified compatible with the actual tool:

```bash
make migrate-up        # migrate -path ../../db/migrations/gl_db -database $DATABASE_URL up
make migrate-version
make migrate-down      # roll back one step
```

Order in a fresh environment: `roles.sql` (once/cluster) → `migrate-up` (as owner/`erp_migrator`)
→ `make grants` → `make seed`. Expand–contract rules for new migrations: see `db/CHANGELOG.md`.

## RLS per request — the core pattern

The pool authenticates as **`erp_app_rw`** (a non-owner role without `BYPASSRLS`), so RLS is
enforced. Every scoped call runs in a transaction that first injects the caller's org scope:

```go
db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
    q := queries.WithTx(tx)               // sqlc queries bound to the tx
    _, err := q.CreateJournalEntry(ctx, arg)
    return err
})
// -> BEGIN; SELECT set_config('app.org_scope', '{org-uuids}', true); <queries...>; COMMIT
```

`set_config(..., true)` is transaction-local, so the scope never leaks across pooled
connections. Empty scope ⇒ `current_org_scope()` returns NULL ⇒ deny-by-default. The scope
should include the caller's Organization Tree path plus the system org (for shared reference data).

## Regenerating queries

Edit `db/queries/*.sql`, then `make sqlc` (or `docker run --rm -v "$PWD":/src -w /src sqlc/sqlc
generate`). `make sqlc-verify` fails CI if the committed `internal/gldb` is stale. The sqlc
catalog (`db/schema/schema.sql`) is intentionally plain DDL — the authoritative schema (RANGE
partitioning, RLS, triggers, document-number registry) is the migration set.

## Notes for productionizing

- Money columns surface as `pgtype.Numeric`; wrap with a decimal type at the service boundary.
- Outbox draining (`DrainOutbox`/`MarkOutboxPublished`) runs without org scope (technical table,
  no RLS) as a system relay; publish rows to Kafka then mark published within the same tx.
- Replicate this module per service (swap schema/queries/DSN); the `platform` package is shared verbatim.
