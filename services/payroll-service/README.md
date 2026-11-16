# payroll-service — Payroll Management (SVC-017)

Backend service for `payroll_db`, built from the **ap-service / gl-service**
blueprint (SDD-001). Stack: **pgx v5 + sqlc + golang-migrate** (ERP-SDD BAB 2–9).
The `internal/platform` package is copied verbatim (RLS boundary, SDD-001).

```
services/payroll-service/
├── cmd/server/main.go                       # entrypoint: pool (app role) + /healthz + wiring
├── db/
│   ├── schema/schema.sql                    # sqlc catalog (plain DDL; source of truth = db/migrations/payroll_db)
│   └── queries/payroll_run.sql              # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── payrolldb/                           # sqlc-GENERATED — do not edit
│   ├── platform/postgres.go                 # pgxpool + WithOrgScope (RLS boundary) — SHARED verbatim
│   └── repository/payroll_run_repository.go
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Delta from the reference (ap-service)

| Aspek           | ap-service                          | payroll-service                                      |
| --------------- | ----------------------------------- | ---------------------------------------------------- |
| DB / skema      | `ap_db` / `ap`                      | `payroll_db` / `pay`                                 |
| Aggregate       | `ap_invoice` + `ap_invoice_line`    | `payroll_run` + `payroll_result` (per-employee)      |
| Partition key   | `invoice_date` (aggregate)          | none — `payroll_run` is not partitioned              |
| PII             | —                                   | gross_amount / net_amount (encrypted at app layer, BAB 7.4) |
| Transisi status | Draft→Recorded                      | Draft→Calculated (`CalculatePayrollRun`, optimistic lock); full lifecycle Draft/Calculated/Approved/Posted/Paid |
| Outbox / EOS    | RANGE-partitioned, PK (id,created_at) | RANGE-partitioned, PK (id,created_at); GL posting is Exactly-Once via correlation_id (ADR-005) |

RLS via `WithOrgScope`, the partitioned outbox relay, and the migration pipeline
are identical to the reference — see ERP-SDD BAB 2 and `ap-service/README.md`.

## Notes

- Sensitive amounts are encrypted at the application layer; the DB stores ciphertext.
  Do NOT log decrypted amounts.
- Payroll posting to GL is Exactly-Once (ADR-005): `correlation_id` on the run anchors
  the outbox event `payroll.run.completed`; the relay drains without org scope.
- The authoritative schema (RLS, triggers, partitioned outbox) lives in
  `db/migrations/payroll_db`; `db/schema/schema.sql` is a plain DDL catalog for sqlc only.
