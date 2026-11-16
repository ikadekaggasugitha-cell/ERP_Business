# Runnable Baseline — 20 Go Microservices + API Gateway, 30 PostgreSQL Databases, 20 Redpanda Domain Event Topics

The full backend stack, runnable end-to-end: **18 event producers + 2 consumers (`audit-service`, `reporting-service`) + `api-gateway` (:8000)** on **Postgres 16 (30 PostgreSQL databases)** + **Redpanda (20 Redpanda domain event topics)**, serving the **React 19 + Vite + TanStack** SPA (`apps/web-spa`) and Enterprise Chrome Companion (`apps/chrome-companion`), built against the shared module `services/shared` (`platform` + `httpx` + `eventbus`).

```text
React 19 + Vite + TanStack SPA ──JWT/SSE──▶ api-gateway (:8000)
                                                 │
client / gateway ──REST──▶ 18 producer services ─┴─outbox(same tx)──▶ relay ──▶ Redpanda (20 domain event topics)
                                                                                        │
                                         ┌──────────────────────────────────────────────┼───────────────┐
                                         ▼                                              ▼               ▼
                                   audit-service                               reporting-service   domain sagas
                              (consumes 20 topics →                         (CQRS read models:     (gl, ap, ar,
                               immutable audit_log)                          gl_balance_snapshot)   inv, trs, tax)
```

## Run it

```bash
docker compose up -d --build     # Postgres (30 DBs) + Redpanda (20 topics) + bootstrap/migrate/grants/seeds + 20 Go microservices + api-gateway
bash deploy/smoke-test.sh        # 88 checks (Steps 1–13, 13a–13g); prints E2E_SMOKE_PASS, exits non-zero on any fail
docker compose down -v           # tear down
```

Bring-up order (compose `depends_on`): `db(healthy) → bootstrap(30 DBs + roles + login) → migrate-<db> ×30 → grants(30 DBs) → seeds → 20 Go microservices + api-gateway`.

## Services & ports (20 Go Microservices + `api-gateway`)

| Domain | Service | Host Port | Emits / Role |
|---|---|---|---|
| Gateway | `api-gateway` | 8000 | `/api/v1/auth/login`, JWT & RBAC scope injection, reverse proxy, `/api/v1/stream/events` SSE bridge |
| Financial | `gl` / `ap` / `ar` / `coa` / `tax` / `treasury` / `currency` | 8081 / 8082 / 8084 / 8085 / 8086 / 8087 / 8100 | `gl.journal.posted` · `ap.invoice.recorded` · `ar.invoice.issued` · `coa.account.upserted` · `tax.invoice.issued` · `treasury.payment.settled` · `currency.rate.created` / `currency.rate.updated` |
| Core-platform | `user` / `security` / `masterdata` / `org` / `workflow` | 8088 / 8089 / 8090 / 8091 / 8092 | `user.account.upserted` · `security.role.upserted` · `masterdata.party.upserted` · `org.unit.upserted` · `workflow.approval.requested` |
| Operational | `procurement` / `inventory` / `manufacturing` / `sales` | 8093 / 8094 / 8095 / 8096 | `procurement.po.approved` · `procurement.goods.received` · `inventory.stock.reserved` · `inventory.goods.issued` · `inventory.reservation.failed` · `manufacturing.workorder.completed` · `sales.order.confirmed` |
| HR | `hr` / `payroll` | 8097 / 8098 | `hr.employee.updated` · `payroll.run.completed` |
| Consumers | `audit` (8083) · `reporting` (8099) | 8083 / 8099 | `audit` consumes 20 Redpanda domain event topics · `reporting` CQRS projects `gl.journal.posted` |

## What the smoke test verifies (88 checks across Steps 1–13)

- Every producer's REST create/action returns the expected status (`201`/`200`).
- **Event flow**: all producers emit → transactional outbox → relay → Redpanda (20 domain event topics) → **audit** appends to `audit_log`; **reporting** projects `gl.journal.posted` into `gl_balance_snapshot` (CQRS read model).
- **Cross-domain sagas & fan-out**:
  - `INT-001`: `sales.order.confirmed → inventory (goods.issued) → ar (invoice.issued) → tax (e-Faktur) + gl (auto-journal) → reporting`, plus out-of-stock compensation (`inventory.reservation.failed → sales Cancelled`).
  - `INT-002 / INT-003`: `procurement.goods.received → ap → gl → reporting`, plus `treasury` draft payment scheduling from `ap.invoice.recorded` and `payroll.run.completed`.
  - `masterdata.party.upserted` fan-out cache across `ap_db`, `ar_db`, `inventory_db`, `sales_db`, and `procurement_db`.
- **Idempotency** (replay + concurrent-guard), **optimistic lock** (`409`), **RLS** (`404` cross-org + `401` deny-by-default), **business rule BR-556** (unbalanced journal → `422`), **audit RLS** (org B sees none of org A).
- **Step 13 (`13a`–`13g`) — Currency Service & API Gateway**:
  - `13a`: Direct `:8100/readyz` (`200`).
  - `13b`: Direct `GET $CUR/currencies` (`200` and includes `IDR`).
  - `13c`: Creates/upserts a `USD -> IDR` `SPOT` rate on `$CUR/exchange-rates` (`201` or `200`).
  - `13d`: Verifies `POST $CUR/exchange-rates/convert` converts `100.0000 USD -> IDR` (`200` with `convertedAmount`).
  - `13e`: Authenticates against `POST $GW/auth/login` (`http://localhost:8000/api/v1/auth/login`) to obtain a valid JWT token (`GW_TOKEN`) and tests `GET $GW/currencies` (`200`) with `-H "Authorization: Bearer $GW_TOKEN"`.
  - `13f`: Verifies `POST $GW/exchange-rates/convert` (`200`) via Gateway with `-H "Authorization: Bearer $GW_TOKEN"`.
  - `13g`: Verifies SSE streaming endpoint `$GW/stream/events?ticket=$GW_TICKET` (`200`) with `-H "Authorization: Bearer $GW_TOKEN"`.

## Auth in this stack

- **Via `api-gateway` (`:8000`)**: Authenticate via `POST /api/v1/auth/login` to receive an HS256 JWT `token` and `streamTicket`. The gateway strips client-spoofed internal headers and injects verified `X-Organization-Id`, `X-Org-Scope`, `X-User-Id`, `X-Roles`, and `X-Permissions` headers.
- **Direct internal service ports (`:8081`–`:8100`)**: Accept trusted internal headers in dev mode:
  ```text
  X-Organization-Id: <uuid>[,<uuid>...]   -> app.org_scope (RLS)
  X-User-Id:         <uuid>               -> created_by / actor
  ```

## Code layout — shared module

`services/shared/` (`platform`, `httpx`, `eventbus`) is imported by every service via `replace github.com/sinergiinvestasiproperti/erp/services/shared => ../shared` — one source of truth, no per-service drift.
Image build uses context `./services` + generic `services/Dockerfile` (`--build-arg SERVICE=<name>`).
