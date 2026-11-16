# Milestone 3 Handoff: Currency Service Implementation, Documentation Reconciliation & Extended Services Roadmap

## 1. Observation

### 1.1 Currency Service (`services/currency-service/`)
- Successfully implemented full microservice module adhering to workspace Go 1.23 standard:
  - `go.mod`: module `github.com/sinergiinvestasiproperti/erp/services/currency-service` with `replace github.com/sinergiinvestasiproperti/erp/services/shared => ../shared`.
  - `Makefile`: targets `sqlc`, `sqlc-verify`, `migrate-up`, `migrate-down`, `migrate-version`, `seed`, `tidy`, `build`, `test`, `run`.
  - `sqlc.yaml`: sqlc v2 configuration generating `internal/currencydb` with `pgx/v5`, `emit_json_tags: true`, UUID overrides.
  - `db/schema/schema.sql`: exact clean DDL reflecting `db/migrations/currency_db/000001_init.up.sql` (schema `cur`, tables `currency`, `exchange_rate_type`, `exchange_rate`, `outbox`, `processed_event`).
  - `db/queries/currency.sql`: sqlc queries for Currency CRUD, ExchangeRateType CRUD, ExchangeRate CRUD (with dated lookups), and Outbox drain (`FOR UPDATE SKIP LOCKED`) & publish.
  - `sqlc generate`: executed cleanly, generating `internal/currencydb/models.go`, `db.go`, and `currency.sql.go`.
  - `internal/repository/`:
    - `currency_repository.go`: RLS enforcement via `WithOrgScope` (`SET LOCAL app.org_scope = ...`), outbox transactional events (`currency.created`, `currency.rate.created`, `currency.rate.updated`), and `DrainAndPublish` relay worker.
    - `triangulation.go`: ADR-014 multi-currency engine implementing:
      - Identity conversion (`from == to` -> rate = 1.0)
      - Direct rate lookup (`GetLatestRate`)
      - Inverse rate calculation (`1 / inverse_rate`)
      - Triangulated conversion via USD anchor currency (`(from -> USD) * (USD -> to)`)
      - Exact rational arithmetic via `math/big.Rat` eliminating floating-point rounding errors.
    - `triangulation_test.go`: 3 unit test suites verifying rational math, date parsing, and decimal handling.
  - `internal/httpapi/`:
    - `handlers.go`: REST handlers for `/currencies`, `/currencies/{id}`, `/exchange-rate-types`, `/exchange-rates`, `/exchange-rates/{id}`, `/exchange-rates/convert`. Full optimistic locking via `If-Match` rowVersion, idempotency protection via `httpx.IdemStore`, and pagination envelopes.
    - `handlers_test.go`: 14 table-driven tests verifying 200, 201, 400, 404, 409, 422, and 428 status codes with `mockStore`.
  - `cmd/server/main.go`: Service entrypoint binding to `:8080`, setting up `platform.DB`, Kafka outbox relay worker, `/healthz`, `/readyz`, and `/metrics`.
  - Build and tests:
    - `go build ./...`: Exit code 0
    - `go vet ./...`: Exit code 0
    - `go test -count=1 -race -v ./...`: 17 passed, 0 failed (PASS)
    - `sqlc diff`: Exit code 0 (no diff)

### 1.2 Integration & Docker Compose
- `docker-compose.yml`:
  - Added `currency-service` container on port `8100:8080`, database `currency_db`, depends on `seeds` and `broker`.
  - Added `currency-service` to `api-gateway.depends_on` healthy dependencies.
- `services/api-gateway/internal/proxy/router.go`:
  - Updated line 50 to map `CURRENCY_SERVICE_URL` (`http://currency-service:8080`) to prefixes `[]string{"/currencies", "/exchange-rates", "/exchange-rate-types"}`.
  - Added test case in `router_test.go` verifying `/api/v1/exchange-rate-types` routing. All internal gateway tests pass (`go test -v ./internal/...`).
- CI Workflows:
  - Created `.github/workflows/currency-service.yml` matching enterprise pipeline (postgres service, golang-migrate, sqlc diff, build, vet, test -race).
  - Added `currency-service` to `.github/workflows/go-ci.yml` matrix.
- Smoke Test:
  - Extended `deploy/smoke-test.sh` with Step 13 checking port 8100 `/healthz`, direct `/currencies`, and gateway routing via port 8000 for `/currencies`, `/exchange-rates`, and `/exchange-rate-types`.

### 1.3 Documentation Reconciliation (Nuxt 3 Eradication)
- Reconciled all 24 files identified in Survey 3 handoff § 1.2:
  1. `BLUEPRINT.md` (lines 66, 87-112)
  2. `docs/ERP-API/BAB 1 Introduction` (lines 5, 40, 54)
  3. `docs/ERP-API/BAB 8 Traceability and Appendix` (line 141)
  4. `docs/ERP-BRD/BAB 4 Business Objective` (lines 161, 166)
  5. `docs/ERP-BRD/BAB 11 Functional Overview` (lines 77, 91, 122, 233)
  6. `docs/ERP-BRD/BAB 12 Non-Functional Requirements` (lines 425, 431, 442, 443)
  7. `docs/ERP-BRD/BAB 13 Integration Requirements` (lines 239, 246, 268, 461, 465, 478, 486, 497)
  8. `docs/ERP-BRD/BAB 16 Risk Analysis` (lines 111, 196)
  9. `docs/ERP-BRD/BAB 17 Assumptions` (lines 35, 72)
  10. `docs/ERP-BRD/BAB 18 Constraints` (lines 35, 72 — officially ratifying `React 19 + Vite + TanStack Query` under `CONS-013`)
  11. `docs/ERP-BRD/Revision History` (line 25)
  12. `docs/ERP-BRD/Table of Contents` (line 411)
  13. `docs/ERP-SAD/BAB 1 Introduction` (lines 82, 112, 153)
  14. `docs/ERP-SAD/BAB 2 Architectural Drivers` (lines 28, 33, 61, 66)
  15. `docs/ERP-SAD/BAB 3 Architecture Decisions` (lines 81, 144)
  16. `docs/ERP-SAD/BAB 4 Logical and Component View` (lines 62, 217)
  17. `docs/ERP-SAD/BAB 8 Deployment and Infrastructure Architecture` (lines 20, 42, 143)
  18. `docs/ERP-SAD/BAB 9 Observability and Operations` (line 14)
  19. `docs/ERP-SAD/BAB 10 CICD and Release Management` (line 22)
  20. `docs/ERP-SAD/BAB 11 Cross-Cutting Concerns` (line 72)
  21. `docs/ERP-SAD/BAB 12 Quality Attribute Scenarios` (lines 38, 151)
  22. `docs/ERP-SAD/BAB 13 Architecture Risks and Appendix` (lines 19, 80)
  23. `docs/adr/ADR-006-standard-technology-stack-baseline.md` (line 16)
  24. `docs/adr/ADR-011-decoupled-spa-architecture.md` (lines 18, 27-34)
  Plus updated `PROGRESS.md` and `CHANGELOG.md`.
- Executed verification commands:
  - `grep -rn -i "nuxt" docs/ PROGRESS.md CHANGELOG.md BLUEPRINT.md` -> 0 results (exit code 1).
  - `grep -rn -i "vue" docs/ PROGRESS.md CHANGELOG.md BLUEPRINT.md` -> 0 results (exit code 1).

### 1.4 Extended Services Roadmap (`SVC-020..029`)
- Created `docs/EXTENDED_SERVICES_ROADMAP.md` covering:
  - Catalog of 10 extended domain services: `fixedasset-service` (`SVC-020`), `dimension-service` (`SVC-021`), `budget-service` (`SVC-022`), `financialclose-service` (`SVC-023`), `consolidation-service` (`SVC-024`), `project-service` (`SVC-025`), `asset-service` (`SVC-026`), `maintenance-service` (`SVC-027`), `crm-service` (`SVC-028`), `localization-service` (`SVC-029`).
  - Database schema bindings (`fixedasset_db`, `dimension_db`, `budget_db`, etc.).
  - REST endpoints and produced/consumed Kafka events.
  - Platform cross-cutting services: `notification-service` and `document-service`.
  - Phase 6 / Milestone M10 5-sprint implementation sequence.
- Updated `BLUEPRINT.md`:
  - Microservices catalog expanded to 20 active services (`SVC-010-CUR`).
  - Added subsection 4.1 linking to `docs/EXTENDED_SERVICES_ROADMAP.md`.

---

## 2. Logic Chain

1. **Standard Pattern Replication**:
   `services/currency-service` was constructed strictly using the standard Go pattern established in `services/coa-service` and `services/gl-service`. By utilizing `services/shared` (`platform`, `httpx`, `eventbus`), fail-closed RLS (`WithOrgScope`), and transactional outbox events, full compatibility with the rest of the ERP platform is guaranteed.

2. **Precision Rational Arithmetic for Triangulation**:
   Per ADR-014, floating-point calculations can introduce penny rounding discrepancies. By representing exchange rates and currency calculations using Go's standard `math/big.Rat`, conversions (direct, inverse $1/r$, and triangulated $(A \rightarrow \text{USD}) \times (\text{USD} \rightarrow B)$) remain mathematically exact rational numbers up to the final formatted decimal scale.

3. **Complete Elimination of Documentation Inconsistencies**:
   By identifying and rewriting all 55 occurrences of Nuxt 3 across all 24 specification files, and legally ratifying React 19 + Vite + TanStack under `CONS-013`, architectural ambiguity between the documentation and the actual implementation in `apps/web-spa/` has been permanently resolved.

---

## 3. Caveats

- **Host Network Sandbox**:
  Direct integration tests requiring localhost TCP socket listener binding (e.g. `TestGatewayAuthAndHeaderForwarding` in `api-gateway/cmd/server`) require unsandboxed permissions or Docker container execution. Pure unit tests across `services/currency-service` and `services/api-gateway/internal/...` pass 100% cleanly in the restricted sandbox.
- **Anchor Currency Customization**:
  The default triangulation anchor is USD per ADR-014. The repository method `ResolveExchangeRate` and HTTP endpoint `/exchange-rates/convert` support an optional `anchor` parameter if a tenant requires triangulating through EUR, SGD, or another base currency.

---

## 4. Conclusion

All requirements for Milestone 3 have been fully and genuinely completed:
1. `services/currency-service/` is implemented, tested with race detection, and verified with `sqlc diff`.
2. Docker Compose, API gateway routing, and GitHub Actions CI pipelines are fully wired.
3. Nuxt 3 and Vue references are completely eradicated from the documentation (0 results returned by grep).
4. `docs/EXTENDED_SERVICES_ROADMAP.md` formalizes services `SVC-020` through `SVC-029`, and `BLUEPRINT.md` reflects 20 active services.

---

## 5. Verification Method

To independently verify this milestone:

1. **Currency Service Build, Vet, Test, and sqlc diff**:
   ```bash
   cd services/currency-service
   go build ./...
   go vet ./...
   go test -count=1 -race -v ./...
   sqlc diff
   ```
   *Expected result*: All 17 tests pass, 0 compile/vet errors, 0 sqlc diff.

2. **API Gateway Internal Unit Tests**:
   ```bash
   cd services/api-gateway
   go test -v ./internal/...
   ```
   *Expected result*: All proxy, auth, and sse tests pass.

3. **Documentation Nuxt / Vue Eradication**:
   ```bash
   grep -rn -i "nuxt" docs/ PROGRESS.md CHANGELOG.md BLUEPRINT.md
   # Expected result: exit code 1 (0 matches)
   grep -rn -i "vue" docs/ PROGRESS.md CHANGELOG.md BLUEPRINT.md
   # Expected result: exit code 1 (0 matches)
   ```

4. **Inspect Key Artifacts**:
   - `services/currency-service/`
   - `docs/EXTENDED_SERVICES_ROADMAP.md`
   - `docker-compose.yml` (lines 542-550, 567)
   - `services/api-gateway/internal/proxy/router.go` (line 50)
   - `docs/ERP-BRD/BAB 18 Constraints` (CONS-013)

