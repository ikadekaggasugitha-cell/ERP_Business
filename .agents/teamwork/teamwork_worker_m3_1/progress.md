# Progress — Milestone 3 (Currency Service, Doc Reconciliation & Extended Services Roadmap)

Last visited: 2026-11-13T15:56:00Z
Status: COMPLETED

## Steps
- [x] Step 0: Read requirements, PROJECT.md, Survey 3 handoff, and Go domain skill.
- [x] Step 1: Examine existing database migrations (`db/migrations/currency_db/`) and existing service structures (`services/gl-service/`, `services/coa-service/`, `services/shared/`).
- [x] Step 2: Implement `services/currency-service/`:
  - [x] `go.mod`, `Makefile`, `sqlc.yaml`, `README.md`
  - [x] `db/schema/schema.sql` and `db/queries/currency.sql`
  - [x] Run `sqlc generate` producing `internal/currencydb/`
  - [x] `internal/repository/` (RLS WithOrgScope, Outbox relay, Triangulation, Drain worker)
  - [x] `internal/httpapi/` (handlers, models, idempotency, tests)
  - [x] `cmd/server/main.go`
  - [x] `internal/repository/triangulation_test.go`
- [x] Step 3: Run Go build, vet, and unit tests in `services/currency-service/`:
  - `go build ./...`: PASS
  - `go vet ./...`: PASS
  - `go test -count=1 -race -v ./...`: PASS (17 tests)
  - `sqlc diff`: PASS (0 diff)
- [x] Step 4: Wire Integration & Compose:
  - [x] `docker-compose.yml`: added `currency-service` (:8100:8080) and `api-gateway.depends_on`
  - [x] `services/api-gateway/internal/proxy/router.go`: added `/exchange-rate-types` and tested
  - [x] `.github/workflows/currency-service.yml`: created
  - [x] `.github/workflows/go-ci.yml`: added `currency-service`
  - [x] `deploy/smoke-test.sh`: added Step 13
- [x] Step 5: Documentation Reconciliation (Nuxt 3 Eradication):
  - [x] Scan and update all 24 files to replace Nuxt 3 with React 19 + Vite + TanStack
  - [x] Ratify `CONS-013` in `docs/ERP-BRD/BAB 18 Constraints`
  - [x] Verify `grep -rn -i "nuxt" docs/ PROGRESS.md CHANGELOG.md BLUEPRINT.md` returns 0 results
  - [x] Verify `grep -rn -i "vue" docs/ PROGRESS.md CHANGELOG.md BLUEPRINT.md` returns 0 results
- [x] Step 6: Extended Services Roadmap:
  - [x] Create `docs/EXTENDED_SERVICES_ROADMAP.md` (SVC-020..SVC-029)
  - [x] Update `BLUEPRINT.md` Microservices Catalog to 20 active services and link roadmap
- [x] Step 7: Final verification & handoff report generation.

