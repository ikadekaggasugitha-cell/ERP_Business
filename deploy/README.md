# Runnable Baseline — 30 Go Microservices + API Gateway, 30 PostgreSQL Databases, 58 Redpanda Domain Event Topics

The full backend stack, runnable end-to-end: **28 event producers + 2 consumers (`audit-service`, `reporting-service`) + `api-gateway` (:8000)** on **Postgres 16 (30 PostgreSQL databases)** + **Redpanda (58 Redpanda domain event topics)**, serving the **React 19 + Vite + TanStack** SPA (`apps/web-spa`) and Enterprise Chrome Companion (`apps/chrome-companion`), built against the shared module `services/shared` (`platform` + `httpx` + `eventbus`).

```text
React 19 + Vite + TanStack SPA ──JWT/SSE──▶ api-gateway (:8000)
                                                 │
client / gateway ──REST──▶ 28 producer services ─┴─outbox(same tx)──▶ relay ──▶ Redpanda (58 domain event topics)
                                                                                        │
                                         ┌──────────────────────────────────────────────┼───────────────┐
                                         ▼                                              ▼               ▼
                                   audit-service                               reporting-service   domain sagas
                              (consumes 58 topics →                         (CQRS read models:     (gl, coa, ap, ar,
                               immutable audit_log)                          gl_balance_snapshot)   inv, sales, trs, tax, bdg)
```

## Run it

```bash
docker compose up -d --build     # Postgres (30 DBs) + Redpanda (58 topics) + bootstrap/migrate/grants/seeds + 30 Go microservices + api-gateway
bash deploy/smoke-test.sh        # Steps 1–15 (core + currency + dimension/localization + 8 extended services); prints E2E_SMOKE_PASS, exits non-zero on any fail
bash deploy/maintain-partitions.sh # Pre-creates +6 months of monthly partitions across all partitioned tables in all 30 DBs
docker compose down -v           # tear down
```

Bring-up order (compose `depends_on`): `db(healthy) → bootstrap(30 DBs + roles + login) → migrate-<db> ×30 → grants(30 DBs + public.idempotency_key) → seeds (15 DBs incl. user_db) → 30 Go microservices → api-gateway`.

## Production Deployment Manifests (`deploy/k8s/`, `deploy/kong/`, `deploy/observability/`)

- **Kubernetes / Kustomize (`deploy/k8s/`)**:
  - `base/`: `namespace.yaml` (restricted PSS), `networkpolicies.yaml` (default-deny + explicit ingress/egress), `postgres-cnpg.yaml` (CloudNativePG HA cluster + PgBouncer `rw`/`ro` poolers), `kafka-strimzi.yaml` (KRaft cluster + topics), `external-secrets.yaml` (Vault KV v2 `ClusterSecretStore` + `ExternalSecret` for HMAC, JWT, OIDC, DB, PII keyring), `api-gateway.yaml`, `web-spa.yaml`, `microservices.yaml` (all 30 microservices running non-root `10001:10001`, `readOnlyRootFilesystem: true`, `/healthz` + `/readyz` probes, HPA, PDB), and `maintenance-cronjobs.yaml` (hourly `erp-idempotency-prune` + daily `erp-backup-verify`).
  - `overlays/staging` & `overlays/prod`: environment-specific namespace & replica overlays (`kubectl kustomize deploy/k8s/overlays/prod`).
- **Debezium CDC & BigQuery Sink (`deploy/cdc/`)**: PostgreSQL `pgoutput` source connectors (`debezium-postgres-connectors.json`) + BigQuery `erp_staging` sink (`bigquery-sink-erp-staging.json`) feeding Cortex Dataform (50 actions: 43 datasets, 7 assertions, 42 declarations).
- **Kong Gateway 3.x (`deploy/kong/kong.yml`)**: Declarative `_format_version: "3.0"` config with strict CORS, Redis-backed rate limiting, security response headers (HSTS/CSP/X-Frame-Options), `/openapi.yaml`, and SSE-safe `/api/v1/stream/events` (`response_buffering: false`).
- **Observability (`deploy/observability/`)**: `prometheus.yml` (scrapes `api-gateway:8000` + all 30 microservices `/metrics`), `otel-collector.yaml` (OTLP gRPC/HTTP -> Tempo + Prometheus), `alert-rules.yml` (5xx >1%, P99 >500ms, outbox lag, DB pool saturation, CrashLoopBackOff), and `grafana-dashboard-erp.json`.

## Services & ports (30 Go Microservices + `api-gateway`)

| Domain | Service | Host Port | Emits / Role |
|---|---|---|---|
| Gateway | `api-gateway` | 8000 | `/api/v1/auth/login`, JWT & RBAC scope injection, reverse proxy, `/api/v1/stream/events` SSE bridge |
| Financial | `gl` / `ap` / `ar` / `coa` / `tax` / `treasury` / `currency` | 8081 / 8082 / 8084 / 8085 / 8086 / 8087 / 8100 | `gl.journal.posted`, `gl.journal.reversed` · `ap.invoice.recorded` · `ar.invoice.issued`, `ar.invoice.paid` · `coa.account.upserted`, `coa.period.updated` · `tax.invoice.issued` · `treasury.payment.settled` · `currency.rate.created` / `updated` |
| Core-platform | `user` / `security` / `masterdata` / `org` / `workflow` | 8088 / 8089 / 8090 / 8091 / 8092 | `user.account.upserted` · `security.role.upserted` · `masterdata.party.upserted`, `masterdata.item.upserted` · `org.unit.upserted` · `workflow.approval.requested` / `completed` |
| Extended Financial | `fixedasset` / `dimension` / `budget` / `financialclose` / `consolidation` | 8101 / 8102 / 8103 / 8104 / 8105 | `fixedasset.asset.acquired` / `depreciation.posted` / `disposed` · `dimension.*` (incl. `allocation_rule.executed`) · `budget.approved` / `commitment.created` · `financialclose.period.closed` / `locked` / `reopened` · `consolidation.run.finalized` |
| Extended Operational | `project` / `asset` / `maintenance` / `crm` / `localization` | 8106 / 8107 / 8108 / 8109 / 8110 | `project.billing.approved` · `asset.movement.completed` / `verification.recorded` · `maintenance.workorder.parts_requested` / `completed` · `crm.opportunity.won` · `localization.*` |
| Operational | `procurement` / `inventory` / `manufacturing` / `sales` | 8093 / 8094 / 8095 / 8096 | `procurement.po.approved` / `goods.received` · `inventory.stock.reserved` / `goods.issued` / `reservation.failed` · `manufacturing.workorder.released` / `completed` · `sales.order.confirmed` / `delivery.completed` |
| HR | `hr` / `payroll` | 8097 / 8098 | `hr.employee.updated` · `payroll.run.calculated` / `approved` / `posted` / `completed` |
| Consumers | `audit` (8083) · `reporting` (8099) | 8083 / 8099 | `audit` consumes 58 Redpanda domain event topics · `reporting` CQRS projects `gl.journal.posted` (per account via `rpt.coa_cache`) & consumes `coa.account.upserted` |

## Automated Partition Maintenance

Run `deploy/maintain-partitions.sh` (or schedule via `.github/workflows/partition-maintenance.yml` / Kubernetes `CronJob` on the 1st of each month) to pre-create `+MONTHS_AHEAD` (default `6`) monthly partitions for every `pg_partitioned_table` across all 30 databases while preserving RLS policies:

```bash
PGHOST=localhost POSTGRES_USER=postgres POSTGRES_PASSWORD=postgres MONTHS_AHEAD=6 bash deploy/maintain-partitions.sh
# Dry run:
DRY_RUN=true bash deploy/maintain-partitions.sh
```

## What the smoke test verifies (Steps 1–15)

- Every producer's REST create/action returns the expected status (`201`/`200`).
- **Event flow**: all producers emit → transactional outbox → relay → Redpanda (58 domain event topics) → **audit** appends to `audit_log`; **reporting** projects `gl.journal.posted` into `gl_balance_snapshot` (CQRS read model).
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
- **Step 14 (`14a`–`14g`) — Dimension & Localization Services (SVC-021, SVC-029)**:
  - `14a`: Direct `:8102/readyz` (dimension) and `:8110/readyz` (localization) (`200`).
  - `14b`: Creates a cost center + profit center; idempotent replay (`201`).
  - `14c`: RLS — org B cannot read org A's cost center (`404`).
  - `14d`: Allocation rule business rule — target percentages must total exactly 100 (`422` invalid, `201` valid).
  - `14e`: Seeded languages visible via system org, translation creation (`201`), bundle fallback overlay contains the new key.
  - `14f`: Gateway routing `:8000/cost-centers` and `:8000/translation-bundles/id` (`200`).
  - `14g`: Audit consumed `dimension.cost_center.upserted` and `localization.bundle.updated`.

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
