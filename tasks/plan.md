# Implementation Plan: Gap Analisis & Pengembangan Database ERP

> Task list target: `tasks/todo.md` (default markdown checklist; tidak ada tracker eksternal).
> Sumber: audit artefak DB seluruh repo per 2026-11-17. Spec acuan: `docs/EXTENDED_SERVICES_ROADMAP.md` (Sprint 2–5 diratifikasi), `db/REVIEW.md`, guardrail AGENTS.md.

## Overview

Layer database fisik sehat dan lengkap: 30/30 DB bermigrasi & hardened (M2 ✅), RLS di semua DB, outbox+trigger 29/30 (`reporting_db` CQRS-derived memang tanpa partisi/trigger), remediasi review M1–N4 selesai. Gap tersisa ada di lapisan atas dan sekelilingnya: 8 layanan extended belum dibangun di atas DB yang sudah bermigrasi, seeds hanya 6/30 DB, regresi hanya 3 skrip (CI hanya gl_db/org_db), pg_partman 2/30, analytics M9 belum dieksekusi, dan docs stale.

## Architecture Decisions

- **Tanpa migrasi DDL baru** untuk 8 service — dibangun di atas migrasi existing; gap DDL → expand–contract terpisah (keputusan pemilik; lihat Open Questions).
- **Pola Sprint 1 sebagai template**: sqlc pgx/v5, outbox transaksional patuh (`pubErr + break + commit`), RLS fail-closed, `ScopeWithSystemOrg` bila seed global.
- **Katalog topik pariter**: event baru masuk `DomainTopics` (gateway) + `AllTopics` (audit) dengan test pariter, per-integrasi.
- **Regresi state-independent** (pelajaran `9f99497`): asersi per-identitas test, bukan hitungan global.
- **Verifikasi berbukti**: skill `erp-local-verification` §1–§3 tanpa Docker; compose hanya di checkpoint.

## Gap Summary

| # | Gap | Bukti |
|---|-----|-------|
| G1 | 8 layanan extended belum dibangun (DB sudah ada) | migrasi `db/migrations/{fixedasset,budget,financialclose,consolidation,project,asset,maintenance,crm}_db/` lengkap; `services/` 22 layanan |
| G2 | Fitur Sprint 1 ditunda (alloc execute, statutory rules) | roadmap §5.1 |
| G3 | Seeds 6/30 DB | `db/seeds/` hanya 6 subdir |
| G4 | Regresi 3 skrip; CI hanya gl_db/org_db | `db/tests/`, `.github/workflows/db-regression.yml` |
| G5 | pg_partman 2/30; housekeeping O3 runbook | scan migrasi; `db/ops/partition_maintenance.md` |
| G6 | BigQuery + `dataform run` belum dieksekusi; model FA/Budget/Project belum ada | PROGRESS §4.3; roadmap §5.5 |
| G7 | PROGRESS.md stale (M5 79/79 padahal 104/104 live terverifikasi 2026-11-16) | PROGRESS §1 & §4 |
| G8 | notification/document service butuh desain DB baru di luar 30 DB | roadmap §4 |

## Task List

### Phase 0: Foundation & Housekeeping

### Task 1: Rekonsiliasi dokumentasi status

**Description:** PROGRESS.md masih mencatat M5 "79/79 live" padahal smoke 104/104 live telah terverifikasi (2026-11-16, termasuk Step 13 currency dan Step 14 dimension/localization), next-steps §4 butir 1–3 kedaluwarsa, dan angka test web-spa berubah (battery Phase 0 sesi ini: tsc/eslint/vite hijau, vitest 67/67, node 17/17, companion 15/15).

**Acceptance criteria:**
- [ ] M5 status ✅ COMPLETED dengan tanggal verifikasi live & angka 104/104
- [ ] Baris matrix currency/dimension/localization dari "⏳ Live Pending" → "✅ Pass"
- [ ] §3 bullet smoke & §4 next-steps mencerminkan kondisi terkini; butir yang selesai dihapus/di-tandai selesai
- [ ] Angka 104 konsisten di PROGRESS/deploy README/CHANGELOG

**Verification:**
- [ ] `grep -n "104" PROGRESS.md deploy/README.md CHANGELOG.md` — angka pariter
- [ ] Review manual diff: tidak ada klaim tanpa bukti

**Dependencies:** None. **Files:** `PROGRESS.md`, `CHANGELOG.md`. **Estimated scope:** XS (1–2 file).

### Task 2: Runner migrasi lokal `db/ops/migrate.sh`

**Description:** Tidak ada shell runner untuk migrasi di luar docker-compose one-shot containers; developer tanpa compose tidak bisa mengapply migrasi. CLI `migrate` (golang-migrate) sudah terpasang di Homebrew.

**Acceptance criteria:**
- [ ] `db/ops/migrate.sh <db_name>` apply up satu DB; `db/ops/migrate.sh all` apply semua DB urut nama
- [ ] URL default `postgres://postgres:postgres@localhost:5432` (paritas compose), override via `MIGRATE_DSN_PREFIX`
- [ ] Dokumentasi singkat di `db/README.md`

**Verification:**
- [ ] `bash -n db/ops/migrate.sh`
- [ ] Apply ke DB buangan (scratch) `fixedasset_db` di PG16 compose → sukses; `schema_migrations` terisi 3 versi; drop DB buangan setelahnya

**Dependencies:** None. **Files:** `db/ops/migrate.sh`, `db/README.md`. **Estimated scope:** XS.

### Task 3: RLS smoke generik lintas-DB + perluasan CI

**Description:** `rls_regression.sql` spesifik gl/org; 8 DB extended (RLS-nya sudah ada di migrasi) tidak ter-regresi sama sekali. Buat smoke generik yang bisa dijalankan pada DB mana pun, lalu daftarkan ke CI.

**Acceptance criteria:**
- [ ] `db/tests/rls_smoke.sql`: (a) setiap tabel RLS-enabled punya policy `scope_isolation`; (b) empty scope → 0 baris terlihat (fail-closed) pada setiap tabel RLS-enabled berkorelasi `organization_id`; (c) cross-tenant: dengan scope org terdata, 0 baris org lain terlihat — state-independent (lulus di DB fresh & tercemar), DB kosong → lulus dengan catatan
- [ ] `.github/workflows/db-regression.yml`: job baru menjalankan `rls_smoke.sql` pada 8 DB extended (migrasi diapply dulu)
- [ ] Script RAISE EXCEPTION saat gagal → `psql -v ON_ERROR_STOP=1` exit non-zero

**Verification:**
- [ ] RED: pada DB scratch, buat tabel RLS-enabled TANPA policy → script GAGAL dengan pesan jelas
- [ ] GREEN: script LULUS di `gl_db` (tercemar smoke, PG16 compose) dan `fixedasset_db` scratch fresh
- [ ] YAML valid; sintaks SQL dicoba via psql

**Dependencies:** Task 2 (runner untuk provisioning scratch, opsional). **Files:** `db/tests/rls_smoke.sql`, `.github/workflows/db-regression.yml`. **Estimated scope:** M.

### Task 4: Seeds DB extended

**Description:** Hanya 6/30 DB punya seed. Inspeksi schema extended, identifikasi tabel referensi yang dibutuhkan sebelum CRUD layanan (mis. `fa.asset_category`, `crm.opportunity_stage`), tulis seed idempoten, wire ke `deploy/seeds.sh`.

**Acceptance criteria:**
- [ ] Keputusan per-DB terdokumentasi (butuh seed / tidak, dengan alasan) termasuk `dimension_db`
- [ ] Seed baru idempoten (`ON CONFLICT`), UUID deterministik, mengikuti gaya seed existing
- [ ] `deploy/seeds.sh` memuat DB baru

**Verification:**
- [ ] Apply 2× pada DB scratch → tanpa duplikat (jumlah baris stabil)
- [ ] `bash -n deploy/seeds.sh`

**Dependencies:** Task 2 (runner, opsional). **Files:** `db/seeds/<db>/seed.sql` ×N, `deploy/seeds.sh`. **Estimated scope:** M.

### Checkpoint: Foundation — CI hijau, seeds idempoten, docs akurat

### Phase 1: Sprint 2 — Asset Lifecycle & Budgets (SVC-020, SVC-022)

### Task 5: fixedasset-service domain (SVC-020)

**Description:** Scaffold modul Go di atas migrasi `fixedasset_db` yang sudah ada: ekstrak snapshot katalog `db/schema/schema.sql`, author query sqlc, handlers (asset-category CRUD, fixed-asset CRUD, capitalize, dispose, transfer, depreciation-runs Straight-Line/Declining-Balance/Units-of-Production dengan aritmetika desimal string), outbox relay patuh, konsumsi `erp.ap.invoice.recorded.v1` → kandidat aset, emisi `fa.asset.created/capitalized/disposed` + `fa.depreciation.posted`.

**Acceptance criteria:**
- [ ] `go vet ./...`, `go test -p 2 ./...`, `sqlc diff` hijau
- [ ] Unit test perhitungan penyusutan 3 metode (kasus tepat desimal, bukan float)
- [ ] Outbox drain mengikuti bentuk patuh 20-check; RLS scope per request; If-Match/rowVersion di endpoint mutasi
- [ ] Endpoint sesuai §3.1 roadmap

**Verification:** `go -C services/fixedasset-service vet ./... && go -C services/fixedasset-service test -p 2 -count=1 ./...` + sqlc diff. **Dependencies:** Phase 0. **Files:** `services/fixedasset-service/**`. **Estimated scope:** L (pola repeatable Sprint 1, satu sesi fokus).

### Task 6: fixedasset-service integrasi

**Description:** Gateway proxy+RBAC (`fa:read/write`, `fixedasset.read/write` dua format), compose `:8101`, workflow CI + matrix, 4 topik ke `DomainTopics`/`AllTopics` + test pariter, smoke step (readyz, CRUD, RLS 404, posting depreciation → jurnal GL bila dirancang).

**Acceptance criteria:**
- [ ] hub_test/consumer_test hijau dengan topik baru (panjang + isi + anti-duplikat)
- [ ] `bash -n deploy/smoke-test.sh`; rolePermissions dua format
- [ ] Smoke step live di compose

**Verification:** `go -C services/api-gateway test ... hub_test` + `go -C services/audit-service test ...` + smoke. **Dependencies:** Task 5. **Files:** `services/api-gateway/**`, `services/audit-service/**`, `docker-compose.yml`, `.github/workflows/*`, `deploy/smoke-test.sh`. **Estimated scope:** M.

### Task 7: budget-service domain (SVC-022)

**Description:** budget_version/header/line, check-commitment (high-throughput, idempoten, hard/soft), utilization-report, konsumsi `procurement.pr.created` (soft), `procurement.po.approved` (hard), `ap.invoice.recorded` (aktual), emisi `budget.version.approved`, `budget.threshold.exceeded`. Batas BR-636..645.

**Acceptance criteria:** vet/test/sqlc hijau; unit test commitment hard/soft & konversi aktual; outbox patuh. **Verification:** go test + sqlc. **Dependencies:** Phase 0. **Files:** `services/budget-service/**`. **Estimated scope:** L.

### Task 8: budget-service integrasi + hook procurement

**Description:** Wiring integrasi ala Task 6 + hook commitment-control ke workflow PR/PO procurement-service + smoke step.

**Acceptance criteria:** PO approved me-reserve hard commitment (smoke live); tests gateway/audit hijau. **Verification:** smoke + go test. **Dependencies:** Task 7. **Files:** `services/procurement-service/**`, gateway, compose, CI, audit, smoke. **Estimated scope:** M.

### Checkpoint 1 — vet/test semua modul; compose up; smoke 15a–15b live; CI hijau

### Phase 2: Sprint 3 — Financial Close & Consolidation (SVC-023, SVC-024)

### Task 9: financialclose-service domain (SVC-023)

**Description:** closing_task/run, fx_revaluation_run (PSAK 10/IAS 21 via kurs CLOSING `currency-service`), accrual reversal, period lock orchestrator, emisi 3 topik.

**Acceptance criteria:** vet/test/sqlc hijau; unit test revaluasi selisih kurs desimal eksak. **Verification:** go test + sqlc. **Dependencies:** Phase 1. **Files:** `services/financialclose-service/**`. **Estimated scope:** L.

### Task 10: financialclose integrasi + konsumen coa

**Description:** Wiring integrasi + konsumen `fcl.period.locked` di coa-service (transisi periode → Locked) + smoke step.

**Acceptance criteria:** lock live; coa menolak posting periode locked. **Verification:** smoke + go test. **Dependencies:** Task 9. **Files:** coa-service, gateway, compose, CI, audit, smoke. **Estimated scope:** M.

### Task 11: consolidation-service domain (SVC-024)

**Description:** consolidation_group/member, elimination_rule, consolidation_run (eliminasi IC + minor interest + CTA), laporan balance sheet konsolidasi.

**Acceptance criteria:** vet/test/sqlc hijau; unit test eliminasi & CTA desimal eksak. **Verification:** go test + sqlc. **Dependencies:** Phase 1. **Files:** `services/consolidation-service/**`. **Estimated scope:** L.

### Task 12: consolidation integrasi + smoke

**Description:** Wiring integrasi + smoke step. **Acceptance criteria:** smoke live; tests hijau. **Verification:** smoke + go test. **Dependencies:** Task 11. **Files:** gateway, compose, CI, audit, smoke. **Estimated scope:** M.

### Checkpoint 2 — smoke 15–16 live; regresi db aman; docs diperbarui

### Phase 3: Sprint 4 — Projects, Asset Ops, Maintenance, CRM (SVC-025/026/027/028)

### Task 13: project-service domain (SVC-025)

**Description:** project, WBS, milestone, project_budget, timesheet, billing schedule; emisi `project.created/milestone.achieved/cost.posted`.

**Acceptance criteria:** vet/test/sqlc hijau; outbox patuh. **Verification:** go test + sqlc. **Dependencies:** Phase 2. **Files:** `services/project-service/**`. **Estimated scope:** L.

### Task 14: project integrasi + konsumen AR

**Description:** Wiring integrasi + konsumen milestone→invoice draft di ar-service + smoke step. **Acceptance criteria:** saga live. **Verification:** smoke + go test. **Dependencies:** Task 13. **Files:** ar-service, gateway, compose, CI, audit, smoke. **Estimated scope:** M.

### Task 15: asset-service domain (SVC-026)

**Description:** equipment, asset_tag, custody_transfer, location_history (barcode/RFID via companion), emisi 2 topik. **Acceptance criteria:** vet/test/sqlc hijau. **Verification:** go test. **Dependencies:** Phase 2. **Files:** `services/asset-service/**`. **Estimated scope:** S/M.

### Task 16: maintenance-service domain (SVC-027)

**Description:** maintenance_plan, work_order(+task), part_requirement, downtime_log/MTBF, emisi 3 topik. **Acceptance criteria:** vet/test/sqlc hijau; unit test MTBF. **Verification:** go test. **Dependencies:** Phase 2. **Files:** `services/maintenance-service/**`. **Estimated scope:** M.

### Task 17: asset+maintenance integrasi + konsumen inventory

**Description:** Wiring integrasi + konsumen `maintenance.sparepart.reserved` di inventory-service + smoke step. **Acceptance criteria:** saga live. **Verification:** smoke + go test. **Dependencies:** Task 15, 16. **Files:** inventory-service, gateway, compose, CI, audit, smoke. **Estimated scope:** M.

### Task 18: crm-service domain (SVC-028)

**Description:** lead, opportunity(+stage), quotation, customer_activity, convert-to-order; emisi `crm.lead.qualified`, `crm.opportunity.won`. **Acceptance criteria:** vet/test/sqlc hijau. **Verification:** go test. **Dependencies:** Phase 2. **Files:** `services/crm-service/**`. **Estimated scope:** S/M.

### Task 19: crm integrasi + konsumen sales

**Description:** Wiring integrasi + konsumen `crm.opportunity.won` → sales order di sales-service + smoke step. **Acceptance criteria:** saga live. **Verification:** smoke + go test. **Dependencies:** Task 18. **Files:** sales-service, gateway, compose, CI, audit, smoke. **Estimated scope:** M.

### Checkpoint 3 — 4 saga lintas-domain live (fa→gl, project→ar, mnt→inv, crm→sales)

### Phase 4: Sprint 5 — Verification Gate & Analytics

### Task 20: Smoke penuh extended

**Description:** Step smoke untuk port 8101..8109 (8102/8110 sudah ada Step 14), saga INT-004..007, perbarui total checks di docs. **Acceptance criteria:** `bash -n`; live pass; angka docs pariter. **Verification:** smoke + grep docs. **Dependencies:** Phase 3. **Files:** `deploy/smoke-test.sh`, docs. **Estimated scope:** M.

### Task 21: Dataform FA/Budget/Project

**Description:** Deklarasi source baru (ERP DB), products `fact_fixed_asset`, `fact_budget_commitment`, `fact_project_cost`, assertion terkait. **Acceptance criteria:** `dataform compile` 0 error. **Verification:** compile lokal. **Dependencies:** Phase 3. **Files:** `src/data_modules/custom/**`. **Estimated scope:** M.

### Task 22: Fitur Sprint 1 ditunda

**Description:** Spike DDL aktual `cc.*`/`loc.*` vs katalog §3.1/§3.10, lalu implement `POST /allocations/execute` (dimension) & statutory rules (localization) sesuai DDL riil. **Acceptance criteria:** unit test alokasi total=100; overlay statutory; catatan deviasi katalog → roadmap. **Verification:** go test. **Dependencies:** None (bisa kapan saja). **Files:** dimension & localization services. **Estimated scope:** M.

### Task 23: pg_partman rollout (Ops O3)

**Description:** partman ke DB panas (ap, ar, inventory, treasury) + job housekeeping yang memanggil `enable_partition_rls()`; dokumentasi runbook. **Acceptance criteria:** partisi bulan depan terbentuk otomatis di DB uji; RLS aktif di partisi baru. **Verification:** psql di DB uji. **Dependencies:** None. **Files:** `db/migrations/<db>_db/00000N_*`, `db/ops/partition_maintenance.md`. **Estimated scope:** M.

### Checkpoint 4 (Final) — smoke live pass penuh; CI 30 DB; BLUEPRINT/PROGRESS/CHANGELOG = 28 layanan; semua regresi hijau

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| DDL aktual ≠ katalog roadmap (sudah terjadi di Sprint 1) | High | Spike baca DDL migrasi sebelum menulis handler; kriteria AC tiap task: query tervalidasi vs DDL riil |
| Katalog topik membesar (40 → ±57) | Med | Update dua daftar + test atomik per integrasi (pola `6e42a2b`) |
| Regresi bergantung state DB | Med | Semua skrip baru state-independent |
| Disk host kritis / Docker VM | Med | Kerja utama Docker-free; compose hanya checkpoint |
| Outbox baru melanggar pola patuh | High | Salin bentuk `pubErr + break + commit`; tambahkan ke daftar cek compliance |
| PG lokal 14 vs CI 16 | Low | CI postgres:16 = gate; lokal hanya smoke awal |

## Open Questions (keputusan pemilik)

1. SPA workspace untuk 8 domain baru — ikut fase atau rencana terpisah?
2. notification/document service — sekarang atau M10?
3. Kredensial BigQuery untuk provision + `dataform run`?
4. pg_partman — DB mana yang diotomasi sekarang?
5. DDL gap ditemukan saat membangun — boleh migrasi expand–contract, atau tunda fitur?
