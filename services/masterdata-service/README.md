# masterdata-service — Master Data Management (SVC-005)

Backend service for `masterdata_db`, built from the **ap-service / gl-service**
blueprint (SDD-001). Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9).
The `internal/platform` package is copied verbatim from the reference.

```
services/masterdata-service/
├── cmd/server/main.go                 # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql              # sqlc catalog (plain DDL; source of truth = db/migrations/masterdata_db)
│   └── queries/party.sql              # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── masterdatadb/                   # sqlc-GENERATED (db.go, models.go, party.sql.go) — do not edit
│   ├── platform/postgres.go           # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/party_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspek           | ap-service                                   | masterdata-service                                     |
| --------------- | -------------------------------------------- | ------------------------------------------------------ |
| DB / skema      | `ap_db` / `ap`                               | `masterdata_db` / `md`                                 |
| Aggregate       | `ap_invoice` + `ap_invoice_line`             | `party` + `party_address`                              |
| Partition key   | `invoice_date` (aggregate partitioned)       | none (aggregate not partitioned; only outbox is)       |
| Doc-number      | `invoice_no` (registry trigger)              | none                                                   |
| Transisi status | Draft→Recorded (optimistic lock)             | **none** — `party` has `row_version` but no `status`   |
| Outbox          | RANGE-partitioned by `created_at`, PK (id,created_at) | identical — RANGE-partitioned producer         |

The one structural delta worth calling out: `md.party` is a config/master table
with `row_version` for optimistic locking but **no `status` column**, so there is
no state-transition query — only `Create`/`Get`/`List`. Everything else (RLS via
`WithOrgScope`, outbox relay, migration pipeline, CI) is identical to the reference.

## RLS per request — the core pattern

The pool authenticates as **`erp_app_rw`** (no `BYPASSRLS`), and every scoped call
runs in a transaction that first injects the caller's org scope via
`set_config('app.org_scope', ..., true)`. The authoritative schema (RLS, triggers,
partitioned outbox) lives in `db/migrations/masterdata_db`; `db/schema/schema.sql`
is intentionally plain DDL for the sqlc `diff` gate.
