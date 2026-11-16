# ADR-006: Standard Technology Stack Baseline

**Status**: Accepted  
**Tanggal**: 2026-08-05  
**Pengambil Keputusan**: Solution Architect, Lead Engineer  
**Terkait**: CONS-013, ASM-013, RISK-015, NFR-013

---

## Konteks
Standardisasi teknologi diperlukan untuk konsistensi kode, efisiensi perawatan, percepatan onboarding engineer, dan pencegahan fragmentasi arsitektur di puluhan repositori layanan.

## Keputusan
Menetapkan tumpukan teknologi inti (*baseline stack*):
- **Backend**: Go 1.22+ (menggunakan `net/http` stdlib, `pgx/v5`, `sqlc`, dan `golang-migrate`).
- **Frontend Core**: Single-Page Application (SPA) / React 19 + Vite + TanStack modern decoupled architecture (ADR-011).
- **Basis Data Transaksional**: PostgreSQL 16+ (*system of record*).
- **Event Streaming Backbone**: Apache Kafka 3.x (mode KRaft) / Redpanda (dev/testing).
- **Orkestrasi**: Docker & Kubernetes.

## Rasional
- Go menawarkan latensi rendah, konsumsi memori sangat kecil, kompilasi cepat, dan konkurensi native (goroutine) yang ideal untuk throughput tinggi.
- `sqlc` menghasilkan kode Go type-safe langsung dari SQL murni tanpa overhead refleksi runtime ORM.

## Konsekuensi
- Menghindari fragmentasi polyglot; setiap perubahan baseline memerlukan persetujuan Change Request (CR) formal.

