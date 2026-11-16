# Enterprise Resource Planning (ERP) — Technical Blueprint

Dokumen ini merupakan cetak biru teknis resmi (*Technical Blueprint*) dari platform Enterprise Resource Planning (ERP) multi-organisasi, multi-valuta, dan multi-negara. Dokumen ini menjadi sumber kebenaran teknis (*single source of truth*) untuk arsitektur sistem, tumpukan teknologi, titik masuk, skema basis data, katalog layanan, dan variabel lingkungan.

---

## 1. Architecture Overview

Platform ERP mengadopsi arsitektur **Domain-Driven Microservices** yang terdesentralisasi dengan **database-per-service** (**30 PostgreSQL databases**, **20 Go microservices** + `api-gateway`, ADR-001), backbone **event streaming Apache Kafka / Redpanda** (**20 Redpanda domain event topics**) dengan semantik *Selective Exactly-Once* (ADR-005), **API Gateway** (ADR-003), antarmuka **Decoupled Single-Page Application (SPA)** berbasis **React 19 + Vite + TanStack** (ADR-011), **Enterprise Chrome Extension Companion** (ADR-012), dan lapisan analitik **Cortex Framework Data Products** (ADR-013).

```text
                               ┌────────────────────────────────────────────────────────┐
                               │                    CLIENT LAYER                        │
                               │  ┌───────────────────────┐  ┌────────────────────────┐ │
                               │  │ Decoupled Backoffice  │  │ Enterprise Companion   │ │
                               │  │ SPA (React 19, Vite,  │  │ Chrome Extension (MV3) │ │
                               │  │ TanStack Router/Query)│  │ (SidePanel, Tax, Bank) │ │
                               │  └───────────┬───────────┘  └───────────┬────────────┘ │
                               └──────────────┼──────────────────────────┼──────────────┘
                                              ▼ HTTPS / REST / SSE       ▼ HTTPS / REST
                               ┌────────────────────────────────────────────────────────┐
                               │             API GATEWAY & SECURITY LAYER               │
                               │  Go API Gateway / Kong (:8000, JWT/OIDC, RBAC, SSE)    │
                               └──────────────────────────┬─────────────────────────────┘
                                                          ▼ REST / mTLS
     ┌────────────────────────────────────────────────────┴──────────────────────────────────────────────────────┐
     │                                 20 GO MICROSERVICES BACKEND (Go 1.23+)                                    │
     │  ┌───────────────────────┐  ┌───────────────────────┐  ┌───────────────────────┐  ┌────────────────────┐  │
     │  │   Financial Domain    │  │  Core-Platform Domain │  │   Operational Domain  │  │  HR/Payroll Domain │  │
     │  │  gl, ap, ar, coa, tax,│  │  user, security, org, │  │ procurement, inventory│  │    hr, payroll     │  │
     │  │  treasury, currency   │  │  masterdata, workflow │  │  manufacturing, sales │  │                    │  │
     │  └───────────┬───────────┘  └───────────┬───────────┘  └───────────┬───────────┘  └─────────┬──────────┘  │
     └──────────────┼──────────────────────────┼──────────────────────────┼────────────────────────┼─────────────┘
                    │ (Transactional Outbox)   │ (Transactional Outbox)   │ (Outbox)               │ (Outbox)
                    ▼                          ▼                          ▼                        ▼
     ┌───────────────────────────────────────────────────────────────────────────────────────────────────────────┐
     │                            EVENT STREAMING BACKBONE (Apache Kafka / Redpanda)                             │
     │  Topics: 20 Redpanda domain event topics (Apicurio Registry: Avro/Protobuf Schema Governance)             │
     │  Delivery: Selective Exactly-Once (EOS: GL, Payment, Stock, Invoice) | At-Least-Once + Idempotency        │
     └──────────────────────────────┬──────────────────────────────────────────┬─────────────────────────────────┘
                                    │ Subscribe                                │ Subscribe
                                    ▼                                          ▼
     ┌──────────────────────────────────────────────────┐      ┌─────────────────────────────────────────────────┐
     │          READ MODEL & AUDIT CONSUMERS            │      │        ANALYTIC DATA LAKEHOUSE (CORTEX V7)      │
     │  audit-service (immutable audit_log)             │      │  Replicated SAP DDIC Tables (BKPF, BSEG, ACDOCA)│
     │  reporting-service (CQRS gl_balance_snapshot)    │      │  Dataform Models: 32 Datasets & 4 Assertions    │
     └──────────────────────────────────────────────────┘      └─────────────────────────────────────────────────┘
```

---

## 2. Tech Stack

| Lapisan / Komponen | Teknologi | Versi | Peran & Alasan Pemilihan |
|---|---|---|---|
| **Backend Services** | Go (20 Go microservices + `api-gateway`) | 1.23+ | Latensi sangat rendah, konsumsi memori minimal, konkurensi native (goroutine) untuk ribuan TPS. |
| **HTTP Routing (Go)** | `net/http` stdlib (`http.ServeMux`) | Go 1.22+ | Standar library ringan tanpa ketergantungan framework berat. |
| **Database Driver** | `pgx/v5` (`pgxpool`) | v5.6+ | Driver PostgreSQL native dengan performa tertinggi dan koneksi pooling efisien. |
| **Database Query Codegen** | `sqlc` | v1.27+ | Menghasilkan kode Go type-safe langsung dari SQL murni tanpa refleksi runtime. |
| **Database Migration** | `golang-migrate` | v4.18+ | Migrasi skema berversi deklaratif berbasis expand–contract. |
| **Basis Data Relasional** | PostgreSQL (30 PostgreSQL databases) | 16+ | System of record tunggal dengan Row-Level Security (RLS) dan composite partitioning. |
| **Connection Pooler** | PgBouncer | 1.23+ | Transaction pooling untuk melayani hingga 10.000 koneksi klien konkuren. |
| **Event Broker** | Apache Kafka (KRaft) / Redpanda | 3.8+ | Streaming backbone (20 Redpanda domain event topics) dengan jaminan order per partition key (`organization_id`). |
| **Schema Governance** | Apicurio Registry | v2.5+ | Registrasi skema Avro/Protobuf dengan kompatibilitas `BACKWARD`. |
| **API Gateway** | Go `api-gateway` (:8000) / Kong Gateway | 3.6+ | JWT/OIDC token enforcement, RBAC scope injection, SSE event streaming, routing path-based `/api/v1`. |
| **Frontend Presentation** | Decoupled SPA (React 19 + Vite + TanStack Router & TanStack Query v5) | Latest | Antarmuka transaksi backoffice interaktif kaya data (Zustand & Tailwind CSS v4). |
| **Browser Companion** | Chrome Extension | Manifest V3 | Otomasi pengisian portal pajak (e-Faktur), rekonsiliasi bank, dan quick approvals. |
| **Enterprise Analytics** | Cortex Framework V7 + Dataform | V7 / v3.1.5 | Integrasi tabel SAP DDIC, star-schema reporting (32 datasets, 4 assertions), dan multi-country consolidation. |
| **Observability** | OpenTelemetry + Prometheus + Grafana | Latest | Tracing terdistribusi (`X-Correlation-Id`), metrik latensi/SLA, dan log JSON terpusat. |
| **Container & Orchestration**| Docker + Kubernetes (Self-Managed) | K8s 1.30+ | Operator-based deployment (Strimzi untuk Kafka, CloudNativePG untuk PostgreSQL). |

---

## 3. Entry Points

| URL / Host | Komponen / Aplikasi | Tujuan & Audiens | Protokol / Auth |
|---|---|---|---|
| `app.erp.example` | Decoupled SPA Backoffice (`React 19 + Vite + TanStack`) | Antarmuka utama staf akuntansi, purchasing, warehouse, dan HR. | HTTPS / OIDC Bearer Token |
| `api.erp.example/api/v1` | `api-gateway` (:8000) / Kong API Gateway | Endpoint publik/klien tunggal untuk seluruh 20 Go microservices & SSE stream. | HTTPS / OAuth2 JWT Scope |
| `admin.erp.example` | Platform Admin Portal | Konfigurasi struktur organisasi, tenancy, role permission, dan fitur. | HTTPS / Multi-Factor Auth |
| `chrome-extension://<id>` | Enterprise Companion Extension | Panel persetujuan cepat (Side Panel), bridge scanner gudang, dan e-Faktur helper. | Chrome MV3 / PKCE Token |
| `reporting.erp.example` | Analytics & BI Dashboard | Dashboard eksekutif bervolume tinggi berbasis Cortex Data Products. | HTTPS / OIDC Role-Based |
| `internal://<svc>:8080` | Klaster Privat Kubernetes | Komunikasi antar-layanan internal (read-only atau gRPC sync). | mTLS / Private Mesh |

---

## 4. Microservices Catalog (20 Go Microservices + `api-gateway`)

| ID | Layanan | Port Host | Database | Peran Utama & Event Kunci yang Diproduksi / Dikonsumsi |
|---|---|---|---|---|
| **GW-000** | `api-gateway` | 8000 | *(stateless)* | Autentikasi JWT (`/api/v1/auth/login`), RBAC & `X-Org-Scope` header injection, reverse proxy ke 20 Go microservices, SSE stream bridge (`/api/v1/stream/events`). |
| **SVC-001** | `org-service` | 8091 | `org_db` | Struktur hierarki pohon organisasi closure table; emit `org.unit.upserted`. |
| **SVC-002** | `user-service` | 8088 | `user_db` | Manajemen pengguna, profil, dan penugasan unit organisasi; emit `user.account.upserted`. |
| **SVC-003** | `security-service` | 8089 | `security_db` | Role-Based Access Control (RBAC), permission matrix, dan scope; emit `security.role.upserted`. |
| **SVC-004** | `workflow-service` | 8092 | `workflow_db` | Mesin approval multi-level konfiguratif; emit `workflow.approval.requested`. |
| **SVC-005** | `masterdata-service` | 8090 | `masterdata_db` | Single Source of Truth rekanan (Vendor/Customer); emit `masterdata.party.upserted`. |
| **SVC-006** | `coa-service` | 8085 | `coa_db` | Master Chart of Accounts, akun hierarkis, mapping akun; emit `coa.account.upserted`. |
| **SVC-007** | `gl-service` | 8081 | `gl_db` | Muara buku besar, validasi balance jurnal (BR-556), EOS; emit `gl.journal.posted`. |
| **SVC-008** | `ap-service` | 8082 | `ap_db` | Faktur pembelian, verifikasi 3-way matching; emit `ap.invoice.recorded`. |
| **SVC-009** | `ar-service` | 8084 | `ar_db` | Faktur penjualan, kartu piutang pelanggan; emit `ar.invoice.issued`. |
| **SVC-010** | `tax-service` | 8086 | `tax_db` | Kode pajak, perhitungan PPN/PPh, integrasi e-Faktur; emit `tax.invoice.issued`. |
| **SVC-010-CUR** | `currency-service` | 8100 | `currency_db` | Master mata uang ISO 4217, dated exchange rates, mesin triangulasi kurs via anchor USD (ADR-014); emit `currency.rate.created`, `currency.rate.updated`. |
| **SVC-011** | `treasury-cashbank-service` | 8087 | `treasury_db` | Rekening bank, kas kecil, settlement pembayaran; emit `treasury.payment.settled`. |
| **SVC-012** | `procurement-service` | 8093 | `procurement_db` | Purchase Requisition (PR), Purchase Order (PO); emit `procurement.po.approved`, `procurement.goods.received`. |
| **SVC-013** | `inventory-service` | 8094 | `inventory_db` | Mutasi stok barang, reservasi stok real-time; emit `inventory.stock.reserved`, `inventory.goods.issued`, `inventory.reservation.failed`. |
| **SVC-014** | `manufacturing-service` | 8095 | `manufacturing_db` | Bill of Materials (BOM), Work Order produksi; emit `manufacturing.workorder.completed`. |
| **SVC-015** | `sales-service` | 8096 | `sales_db` | Sales Quotation, Sales Order konfirmasi; emit `sales.order.confirmed`. |
| **SVC-016** | `hr-service` | 8097 | `hr_db` | Data karyawan, departemen, cuti dan kehadiran; emit `hr.employee.updated`. |
| **SVC-017** | `payroll-service` | 8098 | `payroll_db` | Komponen gaji, kalkulasi batch payroll; emit `payroll.run.completed`. |
| **SVC-018** | `audit-service` | 8083 | `audit_db` | Konsumen 20 Redpanda domain event topics → log audit immutable & query penelusuran forensik. |
| **SVC-019** | `reporting-service` | 8099 | `reporting_db` | Konsumen CQRS → proyeksi read model denormalisasi (`gl_balance_snapshot`, `sales_summary`, `inventory_position`). |

### 4.1 Extended Services Roadmap (SVC-020 .. SVC-029)
Untuk peta jalan 10 layanan domain lanjutan (`SVC-020` fixed asset hingga `SVC-029` localization) beserta integrasi Kafka outbox dan migrasi Phase 6, lihat dokumen resmi: [docs/EXTENDED_SERVICES_ROADMAP.md](docs/EXTENDED_SERVICES_ROADMAP.md).

---

## 5. Database Schema & Partitioning Summary

Platform mengelola **30 PostgreSQL databases** terisolasi (20 database layanan aktif + 10 database domain lanjutan yang telah dimigrasikan dan di-hardening). Standar partisi dan keamanan:
- **Tabel Hiper-Skala Berpartisi Bulanan (RANGE by Date)**:
  - `gl.journal_entry` & `gl.journal_line` (kunci: `posting_date`)
  - `inv.stock_movement` (kunci: `movement_date`)
  - `audit.audit_log` (kunci: `occurred_at`)
  - `<schema>.outbox` pada seluruh 18 service produsen (kunci: `created_at`)
- **Row-Level Security (RLS)**:
  - Seluruh tabel operasional dan master di 30 PostgreSQL databases menerapkan RLS berbasis `app.org_scope`.
  - Akses non-owner dilarang bypass RLS; data di luar izin scope otomatis menghasilkan `404 Not Found`.
- **Indeks Wajib**:
  - `UNIQUE (organization_id, code)` pada semua tabel master.
  - Partial Index `idx_outbox_unpublished` (`WHERE published_at IS NULL`) untuk mempercepat polling outbox relay.
  - BRIN Index pada kolom timestamp data time-series historis.

---

## 6. Environment Variables (Standard per Service Pod)

```bash
# --- Konfigurasi Basis Data ---
DATABASE_URL=postgres://erp_app_rw:<secret>@pgbouncer.prod:6432/<db_name>?sslmode=require
DATABASE_RO_URL=postgres://erp_app_ro:<secret>@pgbouncer-ro.prod:6432/<db_name>?sslmode=require
DATABASE_MAX_OPEN_CONNS=20
DATABASE_MAX_IDLE_CONNS=5
DATABASE_CONN_MAX_LIFETIME=30m

# --- HTTP & Server ---
HTTP_ADDR=:8080
READ_TIMEOUT=10s
WRITE_TIMEOUT=15s
IDLE_TIMEOUT=60s

# --- Kafka Event Backbone ---
KAFKA_BROKERS=kafka-0.kafka-headless.prod:9092,kafka-1.kafka-headless.prod:9092,kafka-2.kafka-headless.prod:9092
KAFKA_CLIENT_ID=<service-name>
KAFKA_GROUP_ID=<service-name>-group
KAFKA_ENABLE_IDEMPOTENCE=true
KAFKA_TRANSACTIONAL_ID=<service-name>-tx  # khusus EOS producer

# --- Schema Registry & Gateway ---
APICURIO_URL=http://apicurio.prod:8080/apis/registry/v2
KONG_ADMIN_URL=http://kong-admin.prod:8001

# --- Observabilitas & Keamanan ---
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector.monitoring:4317
OTEL_SERVICE_NAME=<service-name>
LOG_LEVEL=info
LOG_FORMAT=json
```
