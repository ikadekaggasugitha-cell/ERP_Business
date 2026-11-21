# ERP Platform — Development Progress & Roadmap

Status proyek terkini, pelacakan milestone, dan roadmap implementasi sistem ERP enterprise.

---

## 1. Status Milestone Terkini

| Milestone | Deskripsi | Status | Tanggal Verifikasi |
|---|---|---|---|
| **M1: Dokumen Desain & Kebutuhan** | BRD, SAD, DBDD (ERP-DB), API Spec, dan SDD lengkap | ✅ COMPLETED | 2026-08-06 |
| **M2: Database Hardening & Migrasi** | 30 PostgreSQL databases, RLS, role least-privilege, partisi bulanan, seed data | ✅ COMPLETED | 2026-08-07 |
| **M3: Modul Bersama & Arsitektur Layanan** | `services/shared` (`platform`, `httpx`, `eventbus`), outbox relay, CDC | ✅ COMPLETED | 2026-08-09 |
| **M4: Blueprint Microservices (22 Go Microservices + API Gateway)** | Seluruh SVC-001 s.d. SVC-019, SVC-010-CUR (`currency`), SVC-021 (`dimension`), SVC-029 (`localization`), dan `api-gateway` dibangun, sqlc codegen, REST endpoints | ✅ COMPLETED | 2026-11-16 |
| **M5: E2E Integration & Smoke Testing** | Smoke test live **104/104 checks** (`E2E_SMOKE_PASS`) pada compose 25 kontainer healthy — Steps 1–14 termasuk live pertama untuk `13a`–`13g` (currency `:8100`) dan `14a`–`14g` (dimension `:8102` / localization `:8110`), `13g` SSE dengan fix stream-ticket; RBAC 45/45; regresi DB (document-number, optimistic lock, RLS hardened `9f99497`) lulus live | ✅ COMPLETED | 2026-11-16 |
| **M6: Enterprise Architecture Harmonization** | ADR-011 s.d. ADR-015, React 19 + Vite + TanStack SPA architecture, Chrome Extension, Cortex V7 | ✅ COMPLETED | 2026-11-11 |
| **M7: Decoupled Backoffice SPA** | Implementasi React 19 + Vite + TanStack Router + TanStack Query v5 + virtual data tables (`apps/web-spa`) | ✅ COMPLETED | 2026-11-11 |
| **M8: Enterprise Chrome Extension MV3** | Side Panel Quick Approvals, DJP e-Faktur helper, Bank Extractor & Sandbox (`apps/chrome-companion`) | ✅ COMPLETED | 2026-11-11 |
| **M9: Cortex Analytics Data Products** | Dataform 4-lapis (20 SAP DDIC foundations, 4 annotations, 12 star schema products [36 datasets], 6 assertions, 36 declarations di `src/data_modules/custom/`) | ✅ COMPLETED | 2026-11-22 |
| **M9-B: Currency Service & Doc Reconciliation** | Implementasi `currency-service` (:8100), triangulasi USD anchor, standar frontend React 19 + Vite + TanStack | ✅ COMPLETED | 2026-11-14 |
| **M9-C: Full 30-Service Backend Audit Remediation & Extended Services** | Perbaikan seluruh blocker (RLS smoke tenants, payroll `Approved->Posted`, reporting CQRS double-counting + `rpt.coa_cache`, smoke script), perbaikan logika bisnis (partition key `org:agg`, GL/Tax/Treasury/AR/AP/COA/Security/User/Workflow/Sales/Inventory/Manufacturing/HR/Masterdata), 8 layanan extended (`SVC-020`, `SVC-022`..`028`) + 5 downstream consumers + 58 Redpanda topics + 14 DB seeds + otomasi partisi (`deploy/maintain-partitions.sh`) + 4 Dataform extended marts. Verifikasi unit/race/sqlc/vet/compile penuh 32 modul Go, Web SPA, & Dataform lokal hijau; live Compose E2E Step 15 menunggu eksekusi Docker daemon | ✅ COMPLETED (Static/Unit/Race Verified) | 2026-11-22 |
| **M10: Production-Ready Hardening** | Kong OIDC Gateway, Vault secrets, Apicurio Avro registry, HA multi-region, BigQuery deploy | 📅 PLANNED | Target Q3 2027 |

---

## 2. Matriks Status Layanan (30 Go Microservices + `api-gateway`)

| Layanan (SVC) | REST Endpoints | Outbox Event / Topic | Kafka Relay | RLS Enforced | Smoke Test Pass |
|---|---|---|---|---|---|
| `gl-service` (007) | ✅ 201 Created / 200 OK | ✅ `gl.journal.posted` (EOS) | ✅ Active | ✅ Enforced | ✅ Pass |
| `ap-service` (008) | ✅ 201 Created | ✅ `ap.invoice.recorded` | ✅ Active | ✅ Enforced | ✅ Pass |
| `ar-service` (009) | ✅ 201 Created | ✅ `ar.invoice.issued` | ✅ Active | ✅ Enforced | ✅ Pass |
| `coa-service` (006) | ✅ 201 Created | ✅ `coa.account.upserted` | ✅ Active | ✅ Enforced | ✅ Pass |
| `tax-service` (010) | ✅ 201 Created | ✅ `tax.invoice.issued` | ✅ Active | ✅ Enforced | ✅ Pass |
| `currency-service` (010-CUR) | ✅ 201 Created / 200 OK | ✅ `currency.rate.created` / `currency.rate.updated` | ✅ Active | ✅ Enforced | ✅ Pass (Step 13) |
| `treasury-cashbank-service` (011) | ✅ 201 Created | ✅ `treasury.payment.settled` | ✅ Active | ✅ Enforced | ✅ Pass |
| `user-service` (002) | ✅ 201 Created | ✅ `user.account.upserted` | ✅ Active | ✅ Enforced | ✅ Pass |
| `security-service` (003) | ✅ 201 Created | ✅ `security.role.upserted` | ✅ Active | ✅ Enforced | ✅ Pass |
| `masterdata-service` (005) | ✅ 201 Created | ✅ `masterdata.party.upserted` | ✅ Active | ✅ Enforced | ✅ Pass |
| `org-service` (001) | ✅ 201 Created | ✅ `org.unit.upserted` | ✅ Active | ✅ Enforced | ✅ Pass |
| `workflow-service` (004) | ✅ 201 Created | ✅ `workflow.approval.requested` | ✅ Active | ✅ Enforced | ✅ Pass |
| `procurement-service` (012) | ✅ 201 Created / 200 OK | ✅ `procurement.po.approved` / `procurement.goods.received` | ✅ Active | ✅ Enforced | ✅ Pass |
| `inventory-service` (013) | ✅ 201 Created / 200 OK | ✅ `inventory.stock.reserved` / `inventory.goods.issued` / `inventory.reservation.failed` | ✅ Active | ✅ Enforced | ✅ Pass |
| `manufacturing-service` (014) | ✅ 201 Created | ✅ `manufacturing.workorder.completed` | ✅ Active | ✅ Enforced | ✅ Pass |
| `sales-service` (015) | ✅ 201 Created | ✅ `sales.order.confirmed` | ✅ Active | ✅ Enforced | ✅ Pass |
| `hr-service` (016) | ✅ 201 Created | ✅ `hr.employee.updated` | ✅ Active | ✅ Enforced | ✅ Pass |
| `payroll-service` (017) | ✅ 201 Created | ✅ `payroll.run.completed` | ✅ Active | ✅ Enforced | ✅ Pass |
| `dimension-service` (021, :8102) | ✅ Cost/Profit Center (+ cycle guard & assignment), Allocation Rule + Execute | ✅ `dimension.*.upserted`, `dimension.allocation_rule.executed` | ✅ Active | ✅ Enforced | ✅ Pass (Step 14 + Unit/Race) |
| `localization-service` (029, :8110) | ✅ Language, Translation, Bundle, Locale Format, User & Org Preferences | ✅ `localization.language.updated` / `bundle.updated` / `locale_format.updated` | ✅ Active | ✅ Enforced (+ system org seed) | ✅ Pass (Step 14 + Unit/Race) |
| `fixedasset-service` (020, :8101) | ✅ Asset Category, Fixed Asset, Capitalize/Depreciate (SL/DB/UoP)/Dispose | ✅ `fixedasset.asset.acquired` / `depreciation.posted` / `asset.disposed` | ✅ Active | ✅ Enforced | 🧪 Unit/Race Pass (Step 15 wired) |
| `budget-service` (022, :8103) | ✅ Budget, Lines, Policies, Check, Commitments (PO/AP consumer) | ✅ `budget.approved` / `budget.commitment.created` | ✅ Active | ✅ Enforced | 🧪 Unit/Race Pass (Step 15 wired) |
| `financialclose-service` (023, :8104) | ✅ Closing Templates/Runs/Tasks, Period Statuses, Reopening (SoD BR-664) | ✅ `financialclose.period.closed` / `locked` / `reopened` | ✅ Active | ✅ Enforced | 🧪 Unit/Race Pass (Step 15 wired) |
| `consolidation-service` (024, :8105) | ✅ Groups, Members, Elimination Rules, Runs/Eliminate/Adjust/Finalize | ✅ `consolidation.run.finalized` | ✅ Active | ✅ Enforced | 🧪 Unit/Race Pass (Step 15 wired) |
| `project-service` (025, :8106) | ✅ Projects, WBS (+ cycle guard), Budgets, Costs, Progress, Billings, Change Orders | ✅ `project.billing.approved` | ✅ Active | ✅ Enforced | 🧪 Unit/Race Pass (Step 15 wired) |
| `asset-service` (026, :8107) | ✅ Operational Assets, Location History, Movements, Verifications | ✅ `asset.movement.completed` / `asset.verification.recorded` | ✅ Active | ✅ Enforced | 🧪 Unit/Race Pass (Step 15 wired) |
| `maintenance-service` (027, :8108) | ✅ Schedules, Work Orders, Parts, Checklists, Costs | ✅ `maintenance.workorder.parts_requested` / `completed` | ✅ Active | ✅ Enforced | 🧪 Unit/Race Pass (Step 15 wired) |
| `crm-service` (028, :8109) | ✅ Leads, Opportunities, Segments, Interactions, Support Cases, Tasks | ✅ `crm.opportunity.won` | ✅ Active | ✅ Enforced | 🧪 Unit/Race Pass (Step 15 wired) |
| `audit-service` (018) | ✅ 200 OK Query | ➖ Consumer Only (58 Redpanda domain event topics) | ✅ Active | ✅ Enforced | ✅ Pass |
| `reporting-service` (019) | ✅ 200 OK CQRS | ➖ Consumer Only (per-account GL snapshots via `rpt.coa_cache`, sales, inventory) | ✅ Active | ✅ Enforced | ✅ Pass |
| `api-gateway` (GW :8000) | ✅ Auth, Proxy (30 services), SSE | ➖ Reverse Proxy + Kafka SSE Bridge (58 Redpanda domain event topics) | ✅ Active | ✅ Scope-Enforced | ✅ Pass |

---

## 3. Verifikasi & Pengujian Kualitas

- **Smoke Test (`deploy/smoke-test.sh`)**:
  - Baseline Steps 1–14 (**104/104 Live PASSED `E2E_SMOKE_PASS`**, 2026-11-16) pada 25 kontainer compose healthy; diperluas dengan **Step 15** untuk seluruh 8 layanan extended (`:8101`, `:8103`..`:8109`), lifecycle payroll `Approved -> Posted`, serta asersi anti-double-counting per akun pada `reporting-service`. `bash -n deploy/*.sh` dan `docker compose config -q` lulus (eksekusi live Step 15 menunggu run Docker Compose pada mesin host).
- **Go Microservices Verification (32 Modul: `shared`, `api-gateway`, 30 Microservices)**:
  - `sqlc diff` (0 drift di 30 microservices), `go vet ./...`, dan `go test -p 2 -count=1 -race ./...` lulus di seluruh 32 modul Go.
- **Dataform Compilation (`src/data_modules/custom/`)**:
  - Kompilasi lokal `@dataform/cli@3.1.5` lulus (`0` error) menghasilkan **36 datasets** (`20` staging views, `4` annotation views, `12` dimensional/fact tables termasuk `fact_fixed_asset_depreciation`, `fact_budget_vs_actual`, `fact_project_profitability`, `fact_crm_pipeline`), **6 assertions**, dan **36 source declarations**.
- **Benchmark Performa (`db/benchmark/RESULTS.md`)**:
  - Uji beban pada skala 100.000 jurnal entries dan 200.000 baris jurnal.
  - Waktu eksekusi index scan < 1.5ms; partition pruning terbukti aktif mengisolasi scan ke satu partisi bulanan yang relevan.

---

## 4. Langkah Selanjutnya (Immediate Next Steps)

1. **Verifikasi Live Docker Compose E2E (`deploy/smoke-test.sh` Steps 1–15)**:
   - Jalankan `docker compose up -d --build && bash deploy/smoke-test.sh` pada host dengan Docker daemon aktif untuk memverifikasi live seluruh 30 Go microservices + `api-gateway` (Steps 1–15), termasuk 5 consumer lintas domain baru (`gl`, `coa`, `ar`, `inventory`, `sales`).
2. **Ekspansi Workspace Frontend SPA untuk 8 Domain Extended (Opsional Wave 2)**:
   - Tambahkan modul UI React 19 + Vite + TanStack pada `apps/web-spa` untuk Fixed Asset, Budget, Financial Close, Consolidation, Project, Asset, Maintenance, dan CRM.
3. **Verifikasi Runtime Ekstensi Chrome (`apps/chrome-companion`)**:
   - Muat ekstensi via `chrome://extensions` (*Developer mode* → *Load unpacked*) untuk uji UI live dan tambahkan screenshot 1280×800 ke `apps/chrome-companion/CHROMEWEBSTORE.md` sebelum distribusi.
4. **Provisioning BigQuery & Eksekusi Dataform (`src/data_modules/custom`)**:
   - Siapkan dataset `erp_staging`, `sap_ddic_replica`, dan `erp_analytics` di GCP BigQuery ketika kredensial produksi tersedia.
5. **Hardening Infrastruktur Produksi (M10)**:
   - Konfigurasi Kong API Gateway OIDC plugin, HashiCorp Vault secrets, Apicurio Avro Schema Registry, serta integrasi Hardware Web-Bridge (Web Serial/Web HID).
