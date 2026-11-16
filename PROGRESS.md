# ERP Platform — Development Progress & Roadmap

Status proyek terkini, pelacakan milestone, dan roadmap implementasi sistem ERP enterprise.

---

## 1. Status Milestone Terkini

| Milestone | Deskripsi | Status | Tanggal Verifikasi |
|---|---|---|---|
| **M1: Dokumen Desain & Kebutuhan** | BRD, SAD, DBDD (ERP-DB), API Spec, dan SDD lengkap | ✅ COMPLETED | 2026-08-06 |
| **M2: Database Hardening & Migrasi** | 30 PostgreSQL databases, RLS, role least-privilege, partisi bulanan, seed data | ✅ COMPLETED | 2026-08-07 |
| **M3: Modul Bersama & Arsitektur Layanan** | `services/shared` (`platform`, `httpx`, `eventbus`), outbox relay, CDC | ✅ COMPLETED | 2026-08-09 |
| **M4: Blueprint Microservices (20 Go Microservices + API Gateway)** | Seluruh SVC-001 s.d. SVC-019 + SVC-010-CUR (`currency-service`) dan `api-gateway` dibangun, sqlc codegen, REST endpoints | ✅ COMPLETED | 2026-08-10 |
| **M5: E2E Integration & Smoke Testing** | Baseline live 79/79 checks (`E2E_SMOKE_PASS`) terverifikasi; `deploy/smoke-test.sh` diperluas menjadi 88 checks (Steps 1–13, `13a`–`13g`, sintaks tervalidasi via `bash -n`; menunggu eksekusi live Step 13 di Docker Compose) | 🔄 IN PROGRESS (79/79 Live, 88 Checks Ready) | 2026-11-14 |
| **M6: Enterprise Architecture Harmonization** | ADR-011 s.d. ADR-015, React 19 + Vite + TanStack SPA architecture, Chrome Extension, Cortex V7 | ✅ COMPLETED | 2026-11-11 |
| **M7: Decoupled Backoffice SPA** | Implementasi React 19 + Vite + TanStack Router + TanStack Query v5 + virtual data tables (`apps/web-spa`) | ✅ COMPLETED | 2026-11-11 |
| **M8: Enterprise Chrome Extension MV3** | Side Panel Quick Approvals, DJP e-Faktur helper, Bank Extractor & Sandbox (`apps/chrome-companion`) | ✅ COMPLETED | 2026-11-11 |
| **M9: Cortex Analytics Data Products** | Dataform 4-lapis (20 SAP DDIC foundations, 4 annotations, 8 star schema products [32 datasets], 4 assertions, 31 declarations di `src/data_modules/custom/`) | ✅ COMPLETED | 2026-11-14 |
| **M9-B: Currency Service & Doc Reconciliation** | Implementasi `currency-service` (:8100), triangulasi USD anchor, 20 Go microservices (+ `api-gateway`), 30 PostgreSQL databases, 20 Redpanda domain event topics, standar React 19 + Vite + TanStack | ✅ COMPLETED | 2026-11-14 |
| **M10: Production-Ready Hardening** | Kong OIDC Gateway, Vault secrets, Apicurio Avro registry, HA multi-region, BigQuery deploy | 📅 PLANNED | Target Q3 2027 |

---

## 2. Matriks Status Layanan (20 Go Microservices + `api-gateway`)

| Layanan (SVC) | REST Endpoints | Outbox Event / Topic | Kafka Relay | RLS Enforced | Smoke Test Pass |
|---|---|---|---|---|---|
| `gl-service` (007) | ✅ 201 Created / 200 OK | ✅ `gl.journal.posted` (EOS) | ✅ Active | ✅ Enforced | ✅ Pass |
| `ap-service` (008) | ✅ 201 Created | ✅ `ap.invoice.recorded` | ✅ Active | ✅ Enforced | ✅ Pass |
| `ar-service` (009) | ✅ 201 Created | ✅ `ar.invoice.issued` | ✅ Active | ✅ Enforced | ✅ Pass |
| `coa-service` (006) | ✅ 201 Created | ✅ `coa.account.upserted` | ✅ Active | ✅ Enforced | ✅ Pass |
| `tax-service` (010) | ✅ 201 Created | ✅ `tax.invoice.issued` | ✅ Active | ✅ Enforced | ✅ Pass |
| `currency-service` (010-CUR) | ✅ 201 Created / 200 OK | ✅ `currency.rate.created` / `currency.rate.updated` | ✅ Active | ✅ Enforced | ⏳ Go/Race Pass (Live Pending) |
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
| `dimension-service` (021, :8102) | ✅ Cost/Profit Center CRUD + Allocation Rule (BR total=100) | ✅ `dimension.cost_center.upserted` / `dimension.profit_center.upserted` / `dimension.allocation_rule.upserted` | ✅ Active | ✅ Enforced | ⏳ Go/Race Pass (Live Pending, Step 14) |
| `localization-service` (029, :8110) | ✅ Language, Translation, Bundle (fallback overlay), Locale Format | ✅ `localization.language.updated` / `localization.bundle.updated` / `localization.locale_format.updated` | ✅ Active | ✅ Enforced (+ system org seed) | ⏳ Go/Race Pass (Live Pending, Step 14) |
| `audit-service` (018) | ✅ 200 OK Query | ➖ Consumer Only (20 Redpanda domain event topics) | ✅ Active | ✅ Enforced | ✅ Pass |
| `reporting-service` (019) | ✅ 200 OK CQRS | ➖ Consumer Only (GL snapshots, sales, inventory) | ✅ Active | ✅ Enforced | ✅ Pass |
| `api-gateway` (GW :8000) | ✅ Auth, Proxy, SSE | ➖ Reverse Proxy + Kafka SSE Bridge (20 Redpanda domain event topics) | ✅ Active | ✅ Scope-Enforced | ✅ Pass |

---

## 3. Verifikasi & Pengujian Kualitas

- **Smoke Test (`deploy/smoke-test.sh`)**:
  - **88 checks** end-to-end (Steps 1–13, termasuk `13a`–`13g`) yang menguji pembuatan resource di 20 Go microservices, outbox relay publikasi ke 20 Redpanda domain event topics, pemrosesan consumer audit & reporting CQRS, saga lintas domain (INT-001, INT-002, INT-003), penolakan transaksi tidak seimbang (BR-556), pengujian optimistic lock (409 Conflict), penegakan isolasi Row-Level Security (RLS 404), serta Step 13 (`13a`–`13g`): `:8100/readyz`, `GET /currencies` (`IDR`), pembuatan/upsert kurs `USD -> IDR` `SPOT`, konversi `100.0000 USD -> IDR` (`POST /exchange-rates/convert`), autentikasi JWT ke `$GW/auth/login`, dan akses terautentikasi Bearer token ke `$GW/currencies`, `$GW/exchange-rates/convert`, serta `$GW/events/stream`.
  - Status: **79/79 Live PASSED (`E2E_SMOKE_PASS`)** pada baseline 19 microservice + `api-gateway`; skrip **88 checks** (termasuk Step 13 `13a`–`13g`) tervalidasi sintaks (`bash -n`) dan siap dijalankan ulang saat Docker Desktop VM aktif.
- **Dataform Compilation (`src/data_modules/custom/`)**:
  - Kompilasi lokal `@dataform/cli@3.1.5` lulus (`0` error) menghasilkan **32 datasets** (`20` staging views, `4` annotation views, `8` dimensional/fact tables), **4 assertions** (`assert_fx_rate_coverage`, `assert_gl_balance`, `assert_org_referential_integrity`, `assert_tri_currency_completeness`), dan **31 source declarations**.
- **Benchmark Performa (`db/benchmark/RESULTS.md`)**:
  - Uji beban pada skala 100.000 jurnal entries dan 200.000 baris jurnal.
  - Waktu eksekusi index scan < 1.5ms; partition pruning terbukti aktif mengisolasi scan ke satu partisi bulanan yang relevan.

---

## 4. Langkah Selanjutnya (Immediate Next Steps)

1. **Eksekusi Live Step 13 (`deploy/smoke-test.sh`)**:
   - Restart Docker Desktop VM setelah memberi ruang disk kosong memadai, lalu jalankan `docker compose up -d --build` dan `bash deploy/smoke-test.sh` (88 checks).
2. **Verifikasi Runtime Ekstensi Chrome (`apps/chrome-companion`)**:
   - Unit test `node --test test/companion.test.js` (15/15) telah lulus; muat ekstensi via `chrome://extensions` (*Developer mode* → *Load unpacked*) untuk uji UI live dan tambahkan screenshot 1280×800 ke `apps/chrome-companion/CHROMEWEBSTORE.md` sebelum distribusi.
3. **Verifikasi Browser Live Backoffice SPA (`apps/web-spa`)**:
   - `tsc --noEmit`, `eslint .`, `vite build`, dan `node --experimental-strip-types --test test/*.test.js` (11/11) telah lulus; lakukan uji interaksi browser terhadap gateway live.
3. **Provisioning BigQuery & Eksekusi Dataform (`src/data_modules/custom`)**:
   - Siapkan dataset `erp_staging`, `sap_ddic_replica`, dan `erp_analytics` di GCP BigQuery ketika kredensial produksi tersedia.
4. **Hardening Infrastruktur Produksi (M10)**:
   - Konfigurasi Kong API Gateway OIDC plugin, HashiCorp Vault secrets, Apicurio Avro Schema Registry, serta integrasi Hardware Web-Bridge (Web Serial/Web HID).
