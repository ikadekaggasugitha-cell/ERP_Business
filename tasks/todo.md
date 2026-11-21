# Todo — Gap Database ERP (rencana: tasks/plan.md)

> Checklist tugas. Tandai `[x]` hanya setelah verifikasi berbukti (command dijalankan & output dilihat).

## Phase 0: Foundation & Housekeeping

- [x] **Task 1 (XS): Rekonsiliasi dokumentasi status**
  - [ ] PROGRESS.md: M5 → 104/104 live pass, next-steps 1–3 dibersihkan/diperbarui
  - [ ] Angka sinkron dengan deploy/README.md & CHANGELOG.md
  - Verifikasi: baca ulang dokumen; tidak ada klaim tanpa bukti
  - Files: `PROGRESS.md`, `CHANGELOG.md`

- [x] **Task 2 (XS): Runner migrasi lokal `db/ops/migrate.sh`**
  - [ ] Apply per-DB / semua-DB, urut nomor; dukung arg satu DB
  - [ ] Dokumentasi singkat di db/README.md
  - Verifikasi: `bash -n db/ops/migrate.sh`; apply ke DB buangan lokal → drop
  - Files: `db/ops/migrate.sh`, `db/README.md`

- [x] **Task 3 (M): RLS smoke generik + CI matrix diperluas**
  - [ ] `db/tests/rls_smoke.sql`: iterasi tabel RLS-enabled, scope A vs B, asersi cross_tenant=0, state-independent
  - [ ] 8 DB extended ditambah ke matrix `.github/workflows/db-regression.yml`
  - Verifikasi: lulus di gl_db & fixedasset_db (DB fresh + tercemar); `bash -n` workflow
  - Files: `db/tests/rls_smoke.sql`, `.github/workflows/db-regression.yml`

- [x] **Task 4 (M): Seeds DB extended**
  - [ ] Inspeksi tabel referensi per schema; tulis seed idempoten yang dibutuhkan
  - [ ] Wire ke `deploy/seeds.sh`
  - Verifikasi: apply 2× tanpa duplikat; smoke tidak rusak
  - Files: `db/seeds/<db>/seed.sql` ×N, `deploy/seeds.sh`

### Checkpoint: Foundation — CI hijau, seeds idempoten, docs akurat

## Phase 1: Sprint 2 — Asset Lifecycle & Budgets

- [x] **Task 5 (L): fixedasset-service domain** — scaffold + sqlc vs DDL `fixedasset_db`, handlers (capitalize/dispose, depreciation SL/DB/UoP `*big.Rat` desimal string), outbox, emisi 3 topik (`acquired`, `depreciation.posted`, `disposed`)
  - Verifikasi: `go vet`+`test -race`+`sqlc diff`; unit test penyusutan SL/DB/UoP; pola outbox patuh
  - Files: `services/fixedasset-service/**`
- [x] **Task 6 (M): fixedasset integrasi** — gateway+RBAC (`fixedasset:*`/`fixedasset.*`), compose :8101, CI, topik DomainTopics/AllTopics + parity test, GL consumer (`fixedasset.*`), smoke Step 15
  - Files: gateway, compose, CI, audit, gl-service, smoke
- [x] **Task 7 (L): budget-service domain** — version/header/line, check-budget, control policies (HardStop/SoftWarning/ApprovalOverride), commitments, konsumsi `procurement.po.approved`/`ap.invoice.recorded`, emisi 2 topik
  - Verifikasi: `go vet`+`test -race`+`sqlc diff`; BR-636..645
  - Files: `services/budget-service/**`
- [x] **Task 8 (M): budget integrasi** — gateway+RBAC, compose :8103, CI, audit/SSE parity, smoke Step 15
  - Files: gateway, compose, CI, audit, smoke

### Checkpoint 1 — vet/test/sqlc diff semua modul lulus (compose live smoke Step 15 menunggu Docker daemon host)

## Phase 2: Sprint 3 — Financial Close & Consolidation

- [x] **Task 9 (L): financialclose domain** — closing template/run/task, period status, period reopening (SoD guard BR-664); emisi 3 topik (`closed`, `locked`, `reopened`)
  - Verifikasi: `go vet`+`test -race`+`sqlc diff`
  - Files: `services/financialclose-service/**`
- [x] **Task 10 (M): financialclose integrasi + konsumen coa** — `coa-service` consumer (`financialclose.period.closed`/`locked`/`reopened` → sinkron `fin.fiscal_period`), gateway, compose :8104, CI, audit, smoke Step 15
  - Files: coa-service, gateway, compose, CI, audit, smoke
- [x] **Task 11 (L): consolidation domain** — group/member, elimination rules, consolidation run (eliminate/adjust/finalize dengan validasi balanced `*big.Rat`), emisi `consolidation.run.finalized`
  - Verifikasi: `go vet`+`test -race`+`sqlc diff`
  - Files: `services/consolidation-service/**`
- [x] **Task 12 (M): consolidation integrasi** — gateway, compose :8105, CI, audit, smoke Step 15
  - Files: gateway, compose, CI, audit, smoke

## Phase 3: Sprint 4 — Projects, Asset Ops, Maintenance, CRM

- [x] **Task 13 (L): project domain** — project/WBS (+ cycle guard)/budget/cost/progress/billing/change order; emisi `project.billing.approved` — Files: `services/project-service/**`
- [x] **Task 14 (M): project integrasi + konsumen AR** — `ar-service` consumer (`project.billing.approved` → auto AR invoice); compose :8106, gateway, CI, audit, smoke Step 15 — Files: ar-service, gateway, compose, CI, audit, smoke
- [x] **Task 15 (S/M): asset domain** — operational asset/location history/movement/verification; emisi `asset.movement.completed` & `asset.verification.recorded` — Files: `services/asset-service/**`
- [x] **Task 16 (M): maintenance domain** — schedule/work order (Open→Assigned→InProgress→Completed→Closed)/parts/checklist/costs; emisi `maintenance.workorder.parts_requested` & `completed` — Files: `services/maintenance-service/**`
- [x] **Task 17 (M): asset+maintenance integrasi + konsumen inventory** — `inventory-service` consumer (`maintenance.workorder.parts_requested` → spare parts issue + `manufacturing.workorder.completed` → FG receipt); compose :8107/:8108, gateway, CI, audit, smoke Step 15 — Files: inventory-service, gateway, compose, CI, audit, smoke
- [x] **Task 18 (S/M): crm domain** — lead/opportunity/customer segment/interaction/support case/activity task; emisi `crm.opportunity.won` — Files: `services/crm-service/**`
- [x] **Task 19 (M): crm integrasi + konsumen sales** — `sales-service` consumer (`crm.opportunity.won` → Draft SO); compose :8109, gateway, CI, audit, smoke Step 15 — Files: sales-service, gateway, compose, CI, audit, smoke

## Phase 4: Sprint 5 — Verification Gate & Analytics

- [x] **Task 20 (M): smoke penuh extended & docs** — port 8101..8110, Step 15 di `deploy/smoke-test.sh` (`bash -n` & `docker compose config -q` lulus), BLUEPRINT/PROGRESS/CHANGELOG = 30 Go microservices + `api-gateway` & 58 topics — Files: `deploy/smoke-test.sh`, docs
- [x] **Task 21 (M): dataform FA/Budget/Project/CRM** — 5 staging declarations baru + `fact_fixed_asset_depreciation`, `fact_budget_vs_actual`, `fact_project_profitability`, `fact_crm_pipeline` + 2 assertions (`assert_fixed_asset_nbv_non_negative`, `assert_project_progress_bounds`) — Verifikasi: offline `dataform compile` 0 error (36 datasets, 6 assertions, 36 declarations) — Files: `src/data_modules/custom/**`
- [x] **Task 22 (M): fitur Sprint 1 ditunda** — `dimension-service` (assignment CRUD, allocation rule PUT/DELETE/execute `*big.Rat`, cycle guard) & `localization-service` (`DELETE` guards, `user-preferences`, `organization-languages`) — Verifikasi: `sqlc diff`, `go vet`, `go test -race` lulus — Files: dimension & localization services
- [x] **Task 23 (M/Ops): automated partition maintenance** — `deploy/maintain-partitions.sh` + `.github/workflows/partition-maintenance.yml` (memanggil `public.ensure_future_partitions` across 30 DBs dengan RLS preserved) + `deploy/README.md` — Verifikasi: `bash -n deploy/*.sh` lulus — Files: `deploy/maintain-partitions.sh`, `.github/workflows/partition-maintenance.yml`, `deploy/README.md`

### Checkpoint 4 (Final) — 32 modul Go `sqlc diff`/`go vet`/`go test -race` hijau; Web SPA & Chrome Extension hijau; Dataform compile 0 error; Compose & scripts valid (live Docker smoke Step 15 siap dijalankan pada host)

## Keputusan terbuka (lihat plan.md §Open Questions)
- [ ] SPA workspace untuk 8 domain: ikut fase atau rencana terpisah?
- [ ] notification/document service: sekarang atau M10?
- [ ] BigQuery credentials untuk dataform run?
- [ ] pg_partman: DB mana yang diotomasi?
- [ ] DDL gap saat membangun: boleh migrasi expand–contract?
