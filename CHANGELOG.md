# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- **Extended Services Sprint 1 (`services/dimension-service/`, `services/localization-service/`)**: Implementasi `SVC-021` (:8102; cost center, profit center, allocation rule dengan validasi total persentase tepat 100 via aritmatika rasional) dan `SVC-029` (:8110; language, translation, translation bundle dengan overlay fallback & override org, locale format) di atas skema `dimension_db`/`localization_db` yang sudah ada. Mencakup sqlc, fail-closed RLS (localization menyertakan system org untuk seed global), transactional outbox (6 event topic baru), RBAC `dimension:*`/`localization:*` di gateway, route proxy, sinkronisasi `DomainTopics` ↔ `AllTopics` audit-service, `docker-compose.yml`, matrix `go-ci.yml`, dan Smoke Test Step 14 (`14a`–`14g`, 13 checks; total 101, `bash -n` tervalidasi, live pending).
- **Currency Service Implementation (`services/currency-service/`)**: Microservice ke-20 (`SVC-010-CUR`) pada port 8100 yang terhubung ke `currency_db` (schema `cur`), mencakup master mata uang ISO 4217, dated exchange rates, mesin triangulasi kurs multi-valuta via USD anchor (ADR-014) dengan presisi aritmatika rasional tanpa selisih floating point, transactional outbox relay (`currency.rate.created`, `currency.rate.updated`), sqlc type-safe query generation, fail-closed RLS (`app.org_scope`), serta unit test komprehensif.
- **Extended Services Roadmap (`docs/EXTENDED_SERVICES_ROADMAP.md`)**: Peta jalan arsitektur formal untuk 10 layanan domain lanjutan (`SVC-020` fixed asset hingga `SVC-029` localization) beserta pemetaan database, skema, REST endpoints, asynchronous Kafka events, dan rencana migrasi Phase 6 / Milestone M10.
- **Docker Compose & API Gateway Integration**: Pendaftaran `currency-service` pada port `8100:8080` di `docker-compose.yml`, penambahan `/exchange-rate-types` ke `services/api-gateway/internal/proxy/router.go`, serta workflow CI `.github/workflows/currency-service.yml` dan pembaruan matrix `.github/workflows/go-ci.yml`.
- **Smoke Test Step 13 (`13a`–`13g` in `deploy/smoke-test.sh`)**: Penambahan pengujian E2E lengkap (88 total checks; `bash -n` tervalidasi, siap eksekusi live) yang memverifikasi `:8100/readyz` (`200`), `GET $CUR/currencies` (`200` dan memuat `IDR`), pembuatan/upsert kurs `USD -> IDR` `SPOT` pada `$CUR/exchange-rates` (`201`/`200`), konversi `100.0000 USD -> IDR` via `POST $CUR/exchange-rates/convert` (`200`), autentikasi JWT pada `POST $GW/auth/login`, serta akses terautentikasi Bearer token pada `$GW/currencies` (`200`), `$GW/exchange-rates/convert` (`200`), dan `$GW/stream/events?ticket=$GW_TICKET` (`200`).

### Changed
- **Dokumentasi Terpadu & Ratifikasi Frontend (CONS-013)**: Rekonsiliasi menyeluruh pada `PROGRESS.md`, `BLUEPRINT.md`, `CHANGELOG.md`, dan `deploy/README.md` untuk mencerminkan **30 PostgreSQL databases**, **20 Go microservices** (+ `api-gateway`), **20 Redpanda domain event topics**, dan standar frontend tunggal **React 19 + Vite + TanStack** (0 referensi framework SSR legacy).
- **Microservices Catalog (`BLUEPRINT.md`)**: Pembaruan katalog microservices aktif menjadi 20 Go microservices + `api-gateway` (:8000) dengan menambahkan `SVC-010-CUR` (`currency-service`) dan tautan ke `docs/EXTENDED_SERVICES_ROADMAP.md`.
- **Cortex Framework V7 Dataform Module (`src/data_modules/custom/`)**: Implementasi lengkap dan verifikasi kompilasi lokal (`@dataform/cli@3.1.5`, `0` error) untuk modul analitik 4 lapis (**32 datasets**, **4 assertions**, **31 declarations**): `foundations/` (20 staging views SAP DDIC termasuk `MARD`, `MAKT`, `MARA`, `MBEW`, `MATDOC`, `TVKO`), `annotations/` (4 views: `ann_gl_document`, `ann_sales_order`, `ann_material`, `ann_purchase_order`), `products/` (5 dimensi & 3 tabel fakta berpartisi tri-currency), dan `tests/` (4 assertions: `assert_fx_rate_coverage`, `assert_gl_balance`, `assert_org_referential_integrity`, `assert_tri_currency_completeness`), dengan konfigurasi `workflow_settings.yaml` yang menggunakan `node_modules` lokal tanpa `dataformCoreVersion`.
- **Enterprise Chrome Extension Companion Implementation (`apps/chrome-companion/`)**: Implementasi Manifest V3 lengkap meliputi Service Worker ephemeral (`service-worker.js`, `chrome.alarms` 5 menit, sinkronisasi Kong/workflow), antarmuka Side Panel 3-tab patuh CSP (`sidepanel.html/.css/.js`), content script DJP Online (`tax-helper.js`), ekstraktor mutasi rekening korporasi BCA/BNI/Mandiri (`bank-extractor.js`), parser HTML terisolasi dalam MV3 Sandbox (`sandbox.html/.css/.js`), ikon 16/48/128px, serta panduan distribusi privat `CHROMEWEBSTORE.md`.
- **Decoupled SPA Application Core (`apps/web-spa`)**: Kerangka kerja React 19 + Vite + TanStack Router + TanStack Query v5 dengan Zustand state management, tabel virtualisasi (`@tanstack/react-virtual`), Server-Sent Events (SSE) listener, serta integrasi API Gateway.
- **Decoupled Single-Page Application (SPA) Presentation Architecture** (`ADR-011`): Standar arsitektur antarmuka backoffice transaksional interaktif berbasis React 19 + Vite + TanStack Router, TanStack Query v5, Zustand, virtual scrolling data table (`@tanstack/react-virtual`), dan koneksi real-time Server-Sent Events (SSE) via API Gateway.
- **Enterprise Chrome Extension Companion (Manifest V3)** (`ADR-012`): Modul pendamping browser resmi untuk otomasi pengisian e-Faktur/pajak pemerintah, ekstraksi rekening koran internet banking korporasi, panel persetujuan cepat (*Side Panel Quick Approvals*), dan hardware web-bridge untuk scanner barcode/RFID gudang.
- **Cortex Framework V7 Data Products & SAP DDIC Coexistence** (`ADR-013`): Tata kelola integrasi data analitik enterprise, mencakup replikasi tabel kamus data SAP (BKPF, BSEG, ACDOCA, MARC), aturan namespace kustom (`custom`/`sinergi`), dan model dimensional star schema Dataform (`dim_organization`, `dim_currency`, `dim_fiscal_period`, `fact_gl_transaction`).
- **Tri-Currency & Multi-Country Real-Time Accounting Engine** (`ADR-014`): Mesin pencatatan 3 lapis valuta (Transaction, Functional, Presentation Currency) pada setiap baris jurnal/invoice, mesin triangulasi kurs real-time ber-cache, otomasi selisih kurs bulanan (PSAK 10 / IAS 21), serta dukungan kalender fiskal multi-negara (*dual-period tagging*).
- **High-Throughput Composite Partitioning & Read-Replica Topology** (`ADR-015`): Desain partisi komposit (RANGE tanggal + HASH unit organisasi), otomasi rotasi partisi bulanan (`pg_partman`), PgBouncer transaction pooling untuk 10.000 koneksi klien, pemisahan node Read-Replicas, dan partisi Kafka per `organization_id` untuk jutaan transaksi real-time.
- **Architecture Decision Records (ADR) Repository**: Terstruktur di `docs/adr/` (`ADR-001` s.d. `ADR-015`) dan indeks di `docs/adr/README.md`.
- **Technical Blueprint** (`BLUEPRINT.md`): Dokumentasi arsitektur terpadu, ringkasan tumpukan teknologi, katalog 20 Go microservices (+ `api-gateway`), 30 PostgreSQL databases, 20 Redpanda domain event topics, dan standar variabel lingkungan.
- **Development Roadmap & Progress Tracker** (`PROGRESS.md`): Status verifikasi implementasi layanan dan milestone rilis.
- Pembaruan dokumen **SAD (Software Architecture Document)** pada BAB 1, BAB 3, BAB 4, BAB 5, dan BAB 6 untuk mencerminkan arsitektur SPA modern, Chrome Extension Companion, Cortex Data Products, Tri-Currency, dan penskalaan transaksi berkecepatan tinggi.
- Pembaruan dokumen **BRD (Business Requirement Document)** pada BAB 13 untuk menyelaraskan arsitektur presentasi dan integrasi eksternal.

---

## [1.0.0] — 2026-08-11

### Added
- **20 Go Microservices (+ `api-gateway`) & 30 PostgreSQL Databases**:
  - Financial Services: `gl-service`, `ap-service`, `ar-service`, `coa-service`, `tax-service`, `currency-service`, `treasury-cashbank-service`.
  - Core-Platform Services: `org-service`, `user-service`, `security-service`, `masterdata-service`, `workflow-service`.
  - Operational Services: `procurement-service`, `inventory-service`, `manufacturing-service`, `sales-service`.
  - HR & Payroll Services: `hr-service`, `payroll-service`.
  - Event Consumers & Gateway: `audit-service` (immutable audit trail), `reporting-service` (CQRS read model), dan `api-gateway` (auth, RBAC proxy, SSE stream).
- **Shared Platform Module (`services/shared/`)**:
  - `platform`: Inisialisasi koneksi basis data `pgxpool`, middleware penegakan RLS `WithOrgScope`, dan recovery goroutine.
  - `httpx`: Server HTTP standar, router routing, middleware korelasi request `X-Correlation-Id`, logging terstruktur JSON, dan serialisasi respon error.
  - `eventbus`: Outbox publisher, relay worker, consumer Kafka KRaft / Redpanda dengan penanganan idempotensi dan graceful shutdown.
- **Database Architecture & Hardening (30 PostgreSQL Databases)**:
  - Skema PostgreSQL dengan Row-Level Security (RLS) di seluruh tabel operasional.
  - Role basis data least-privilege: `erp_migrator` (owner), `erp_app_rw`, dan `erp_app_ro`.
  - Partisi bulanan pada `gl.journal_line` dan `<schema>.outbox` di 18 layanan produsen.
  - Framework seed data idempoten per-tenant pada `db/seeds/`.
  - Pengujian performa skala besar di `db/benchmark/` (100k journal entries, 200k lines).
- **End-to-End Verification Pipeline**:
  - Docker Compose multi-service bring-up.
  - Skrip verifikasi menyeluruh `deploy/smoke-test.sh` (79/79 live pass pada baseline awal; diperluas menjadi 88 checks Steps 1–13 siap eksekusi live Docker Compose).
