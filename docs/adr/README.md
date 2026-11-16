# Architecture Decision Records (ADRs)

Dokumen ini mencatat seluruh keputusan arsitektural yang mengikat dan permanen untuk platform Enterprise Resource Planning (ERP). Format mengikuti standar ADR: Status, Konteks, Keputusan, Rasional, Konsekuensi, dan Referensi Kebutuhan (BRD/SAD/NFR).

ADR bersifat **immutable**. Keputusan yang telah berstatus `Accepted` tidak dihapus; jika diganti di masa depan, ADR baru dibuat dengan status `Accepted` dan mereferensikan ADR lama dengan status `Superseded`.

---

## Indeks ADR

| # | Judul | Status | Tanggal | Domain / Aspek |
|---|---|---|---|---|
| [ADR-001](./ADR-001-microservices-db-per-service.md) | Domain-Driven Microservices + Database-per-Service | ✅ Accepted | 2026-08-05 | Arsitektur Sistem |
| [ADR-002](./ADR-002-hybrid-multi-tenancy-rls.md) | Hybrid Multi-Tenancy & Row-Level Security | ✅ Accepted | 2026-08-05 | Tata Kelola Data & Tenancy |
| [ADR-003](./ADR-003-kong-apicurio-registry.md) | Kong API Gateway dan Apicurio Registry | ✅ Accepted | 2026-08-05 | Integrasi & Kontrak API |
| [ADR-004](./ADR-004-self-managed-infrastructure.md) | Infrastruktur Self-Managed pada Kubernetes | ✅ Accepted | 2026-08-05 | Infrastruktur & Kedaulatan Data |
| [ADR-005](./ADR-005-selective-exactly-once-delivery.md) | Selective Exactly-Once Event Delivery | ✅ Accepted | 2026-08-05 | Event Streaming & Konsistensi |
| [ADR-006](./ADR-006-standard-technology-stack-baseline.md) | Standard Technology Stack Baseline | ✅ Accepted | 2026-08-05 | Standarisasi Teknologi |
| [ADR-007](./ADR-007-api-style-rest-and-grpc.md) | API Style: REST (Eksternal) + gRPC (Internal) | ✅ Accepted | 2026-08-05 | Komunikasi Layanan |
| [ADR-008](./ADR-008-transactional-outbox-cdc.md) | Transactional Outbox dengan Change Data Capture (CDC) | ✅ Accepted | 2026-08-05 | Konsistensi Transaksi & Event |
| [ADR-009](./ADR-009-cqrs-read-model.md) | CQRS Read Model untuk Reporting & Dashboard | ✅ Accepted | 2026-08-05 | Performa Baca & Pelaporan |
| [ADR-010](./ADR-010-observability-opentelemetry.md) | Observability Stack Berbasis OpenTelemetry | ✅ Accepted | 2026-08-05 | Observabilitas & Telemetri |
| [ADR-011](./ADR-011-decoupled-spa-architecture.md) | Decoupled Single-Page Application (SPA) untuk Enterprise Backoffice | ✅ Accepted | 2026-11-11 | Frontend & UI Presentation |
| [ADR-012](./ADR-012-enterprise-chrome-extension-companion.md) | Enterprise Chrome Extension Companion (Manifest V3) | ✅ Accepted | 2026-11-11 | Ekstensi Browser & Otomasi Lapangan |
| [ADR-013](./ADR-013-cortex-data-products-sap-coexistence.md) | Cortex Framework Data Products & SAP DDIC Coexistence | ✅ Accepted | 2026-11-11 | Integrasi Analitik & Legacy ERP |
| [ADR-014](./ADR-014-tri-currency-multi-country-engine.md) | Tri-Currency & Multi-Country Real-Time Accounting Engine | ✅ Accepted | 2026-11-11 | Finansial Multi-Valuta & Multi-Negara |
| [ADR-015](./ADR-015-high-throughput-partitioning-sharding.md) | Composite Partitioning, Read Replicas & Connection Pooling untuk Jutaan Transaksi | ✅ Accepted | 2026-11-11 | Skalabilitas & Performa Tinggi |

