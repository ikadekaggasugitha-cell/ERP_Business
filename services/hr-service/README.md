# hr-service — Human Resource Management (SVC-016)

Backend service for `hr_db`, built from the **ap-service / gl-service** blueprint
(SDD-001). Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9). The
`internal/platform` package is copied verbatim (RLS boundary, SDD-001).

```
services/hr-service/
├── cmd/server/main.go                    # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql                 # sqlc catalog (plain DDL; source of truth = db/migrations/hr_db)
│   └── queries/employee.sql              # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── hrdb/                             # sqlc-GENERATED — do not edit
│   ├── platform/postgres.go              # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/employee_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspek           | ap-service                          | hr-service                                            |
| --------------- | ----------------------------------- | ----------------------------------------------------- |
| DB / skema      | `ap_db` / `ap`                      | `hr_db` / `hr`                                         |
| Aggregate       | `ap_invoice` + `ap_invoice_line`    | `employee` (+ `position` + `employment` contract)     |
| Partition key   | `invoice_date` (aggregate)          | none — `employee` is not partitioned                  |
| PII             | —                                   | full_name / national_id / date_of_birth / email / amount (encrypted at app layer, BAB 7.4) |
| Transisi status | Draft→Recorded                      | Active→Terminated (`TerminateEmployee`, optimistic lock) |
| Outbox          | RANGE-partitioned, PK (id,created_at) | RANGE-partitioned, PK (id,created_at)                |

RLS via `WithOrgScope`, the partitioned outbox relay, and the migration pipeline
are identical to the reference — see ERP-SDD BAB 2 and `ap-service/README.md`.

## Notes

- PII columns are encrypted at the application layer; the DB stores ciphertext and
  types round-trip it. PII access is audited (audit_db). Do NOT log decrypted PII.
- The authoritative schema (RLS, triggers, partitioned outbox) lives in
  `db/migrations/hr_db`; `db/schema/schema.sql` is a plain DDL catalog for sqlc only.
- `make sqlc-verify` (`sqlc diff`) fails CI if the committed `internal/hrdb` is stale.
