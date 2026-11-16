# Rekomendasi Arsitektur: Membangun ERP Enterprise Berbasis Microservices

Dokumen ini menyajikan rekomendasi arsitektur komprehensif, justifikasi teknis, strategi mitigasi integritas transaksi (ACID), model konkurensi tinggi, serta panduan tata kelola pemeliharaan jangka panjang untuk pembangunan platform **Enterprise Resource Planning (ERP)** berbasis **Microservices**.

---

## 1. Eksekutif Ringkasan & Dilema Klasik Arsitektur ERP

Membangun sistem ERP tingkat enterprise selalu berhadapan dengan dilema arsitektur fundamental:
- **Monolith Terpusat**: Memberikan kemudahan transaksi lokal ACID dalam satu basis data (`BEGIN ... COMMIT`), namun memiliki kelemahan fatal pada skala enterprise:
  - *Lock contention* dan *bottleneck* I/O parah saat ribuan cabang melakukan posting transaksi bersamaan.
  - *Blast radius* tanpa batas: kegagalan modul penggajian atau laporan berat di akhir bulan dapat mengunci tabel dan melumpuhkan operasional kasir/penjualan.
  - *Release paralysis*: perubahan kecil pada modul HR memerlukan pengujian regresi dan redeploy seluruh monolit ERP.
- **Microservices Tradisional Tanpa Tata Kelola**: Menawarkan skalabilitas independen, namun berisiko tinggi merusak integritas finansial (*data inconsistency*, fenomena *distributed monolith*, dan hilangnya jejak audit).

**Rekomendasi Utama:**  
Platform ERP Enterprise direkomendasikan dibangun dengan **Domain-Driven Microservices + Database-per-Service**, dengan **4 Pilar Disiplin Arsitektur** yang menjamin:
1. **Integritas Transaksi Finansial (ACID)** tetap terjaga 100% tanpa menggunakan Two-Phase Commit (2PC) yang lambat.
2. **Konkurensi Sangat Tinggi** melayani ribuan transaksi per detik lintas cabang dan negara.
3. **Modularitas Kuat** dengan batas *bounded context* yang divalidasi oleh compiler.
4. **Kemudahan Pemeliharaan Jangka Panjang (Maintainability)** berbasis kontrak eksplisit dan isolasi dependensi.

---

## 2. Empat Pilar Arsitektur Microservices ERP Enterprise

```text
┌─────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                 4 PILAR ARSITEKTUR ERP ENTERPRISE                               │
├───────────────────────────────┬─────────────────────────────────┬───────────────────────────────┤
│ 1. INTEGRITAS DATA & ACID     │ 2. KONKURANSI & PERFORMA TINGGI │ 3. MODULARITAS & DDD BOUNDARY │
│ • Local ACID di dalam Service │ • Go 1.23+ Stateless Concurrency│ • 19 Bounded Contexts (DDD)   │
│ • Transactional Outbox + CDC  │ • PgBouncer Transaction Pooling │ • Clean / Hexagonal Layout    │
│ • Choreographed Saga + Reverse│ • Composite Partitioning        │ • Database-per-Service        │
│ • Selective Exactly-Once (EOS)│ • CQRS Read Model Projections   │ • Kontrak OpenAPI & Avro      │
├───────────────────────────────┴─────────────────────────────────┴───────────────────────────────┤
│ 4. TATA KELOLA & PEMELIHARAAN JANGKA PANJANG (MAINTAINABILITY)                                  │
│ • GitOps & Pipeline CI/CD Independen per Service (Expand–Contract DB Migrations)                │
│ • Observabilitas Terdistribusi (OpenTelemetry Tracing, Prometheus SLA/SLO, Audit Forensik)      │
│ • Single Source of Truth Dokumentasi (BLUEPRINT.md, ADR-001..015, Keep-a-Changelog)            │
└─────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Pilar 1: Menjaga Integritas Transaksi (ACID) pada Lingkungan Terdistribusi

Tantangan terbesar microservices pada ERP adalah transaksi yang melibatkan banyak modul (misalnya alur *Order-to-Cash*: Sales → Inventory → AR → GL). Rekomendasi solusi:

### 3.1 Local ACID Consistency (Database-per-Service)
- Setiap transaksi lokal di dalam satu layanan (misal: validasi balance jurnal di `gl-service` atau pencatatan invoice di `ap-service`) dijalankan dengan **transaksi ACID penuh** di PostgreSQL (`SERIALIZABLE` atau `READ COMMITTED` + *row locking*).
- Dilarang keras melakukan *distributed transaction coordinator* 2PC (Two-Phase Commit / XA) karena menciptakan *locking* terdistribusi yang mematikan performa sistem.

### 3.2 Pola Transactional Outbox & CDC (Menghilangkan Dual-Write)
- Untuk menerbitkan event ke modul lain, layanan menulis perubahan data domain dan record event ke tabel `<schema>.outbox` dalam **SATU transaksi PostgreSQL lokal yang sama** (atomik).
- Relay worker / CDC (Change Data Capture) membaca WAL atau tabel outbox dan mempublikasikannya ke Kafka.
- **Jaminan**: Tidak ada data tersimpan tanpa event terkirim, dan tidak ada event terkirim tanpa data tersimpan (*zero dual-write discrepancy*).

### 3.3 Alur Transaksi Terdistribusi Berbasis Saga (Choreography)
- Alur lintas layanan dijalankan sebagai **Saga**: rangkaian transaksi lokal yang saling memicu via event.
- Setiap langkah maju memiliki **Compensating Action** (aksi pembalik otomatis). Contoh:
  1. `sales-service`: Simpan pesanan (*Sales Order Confirmed*).
  2. `inventory-service`: Coba reservasi stok. Jika stok habis, terbitkan `inventory.reservation.failed`.
  3. `sales-service`: Mendengar event kegagalan dan secara otomatis menjalankan transaksi kompensasi: mengubah status pesanan menjadi *Cancelled / On Hold*.

### 3.4 Selective Exactly-Once Semantics (ADR-005)
- **Exactly-Once Semantics (EOS)** diberlakukan khusus pada titik mutasi finansial kritikal:
  - `General Ledger` (Journal Posting)
  - `Treasury` (Payment Settlement)
  - `Inventory` (Stock Reservation & Issue)
  - `AR / AP` (Invoice Recognition)
- Menggunakan Kafka Transaksional + tabel pencatatan deduplikasi (`processed_event`) pada konsumen untuk menjamin transaksi tidak pernah diproses ganda meskipun terjadi *network retry*.

---

## 4. Pilar 2: Model Konkurensi & Throughput Tinggi (High Concurrency)

Untuk menangani jutaan transaksi harian dari ratusan cabang tanpa degradasi latensi (SLA P95 < 2 detik):

### 4.1 Bahasa Pemrograman & Runtime Ringan (Go 1.23+)
- Backend ditulis dalam **Go** menggunakan *idiomatic lightweight stack* (`net/http` stdlib + `pgx/v5` + `sqlc`).
- Go memanfaatkan konkurensi native berbasis **Goroutines** (konsumsi memori hanya ~2 KB per goroutine vs 1–2 MB per thread pada Java/JVM), mampu menangani puluhan ribu koneksi HTTP/gRPC konkuren dengan CPU minimal.

### 4.2 Connection Pooling Berlapis (PgBouncer)
- Pod microservices Go terhubung ke basis data melalui **PgBouncer** dalam mode **Transaction Pooling**.
- Setiap pod membatasi koneksi lokal (`pgxpool`) maksimal 15–20 koneksi, sementara PgBouncer mengelolanya menjadi ribuan transaksi konkuren ke PostgreSQL engine tanpa kehabisan *database connections* (*connection saturation*).

### 4.3 Composite Partitioning & Mandatory Partition Pruning
- Tabel transaksi hiper-skala (`gl.journal_line`, `inv.stock_movement`, `audit.audit_log`, `outbox`) dipartisi secara komposit:
  - **RANGE Bulanan** berdasarkan tanggal (`posting_date`).
  - **HASH Sub-partition** berdasarkan `organization_id`.
- **Aturan Pruning Wajib**: Seluruh query data transaksional wajib memfilter rentang tanggal dan `organization_id`, memaksa PostgreSQL mengeksekusi *partition pruning* (hanya men-scan 1 partisi fisik yang relevan) dan mengabaikan puluhan juta baris historis lainnya.

### 4.4 Segregasi Beban Baca/Tulis (Read Replicas & CQRS)
- Operasi tulis (INSERT/UPDATE/DELETE) diarahkan ke node Primary (`erp_app_rw`).
- Query pencarian daftar transaksi dan laporan operasional diarahkan ke node **Read Replicas** (`erp_app_ro`).
- Dashboard agregasi eksekutif dilayani oleh model baca terdenormalisasi **CQRS** di `reporting-service` (ADR-009) dan lakehouse analitik **Cortex Framework V7** (ADR-013), membebaskan klaster OLTP dari query agregasi berat.

---

## 5. Pilar 3: Modularitas Berkelanjutan (Domain-Driven Design)

Mencegah microservices berubah menjadi *"distributed spaghetti"* melalui pemisahan domain yang disiplin:

### 5.1 Dekomposisi 4 Subdomain & 19 Bounded Contexts
1. **Financial Management Domain**: `gl-service`, `ap-service`, `ar-service`, `coa-service`, `tax-service`, `treasury-cashbank-service`.
2. **Core-Platform Domain**: `org-service` (closure table), `user-service`, `security-service` (RBAC), `workflow-service` (approval engine), `masterdata-service` (Vendor/Customer single source of truth).
3. **Operational Domain**: `procurement-service`, `inventory-service`, `manufacturing-service`, `sales-service`.
4. **Human Resources Domain**: `hr-service`, `payroll-service`.
5. **Cross-Cutting Consumers**: `audit-service` (append-only forensic log), `reporting-service` (CQRS projections).

### 5.2 Standar Paket Internal (Clean / Hexagonal Layout)
Setiap layanan mematuhi struktur paket Go yang seragam (SDD-004):
- `cmd/server/`: Titik masuk (*entrypoint*) aplikasi, inisialisasi dependensi, dan graceful shutdown.
- `internal/platform/`: Penanganan koneksi database, RLS session context injection, dan outbox relay.
- `internal/repository/`: Logika akses data yang di-generate via `sqlc` (zero SQL string di luar file SQL).
- `internal/httpapi/`: HTTP handler, deserialisasi JSON, dan pemetaan error domain ke status HTTP.
- Penggunaan folder `internal/` memastikan paket domain tidak dapat diimpor sembarangan oleh layanan lain; batasan domain ditegakkan oleh compiler Go.

---

## 6. Pilar 4: Kemudahan Pemeliharaan Jangka Panjang (Maintainability)

### 6.1 Tata Kelola Kontrak API & Skema Event
- **API Kontrak (REST OpenAPI 3.1)**: Dipublikasikan dan diagregasi di Kong Gateway. Kompatibilitas divalidasi di CI pipeline sebelum rilis.
- **Event Schema (Avro/Protobuf di Apicurio)**: Menegakkan aturan kompatibilitas `BACKWARD`. Penambahan field bersifat opsional; perubahan breaking dilarang pada versi mayor yang sama.

### 6.2 Strategi Migrasi Basis Data Tanpa Downtime (Expand–Contract)
Migrasi skema PostgreSQL dikelola via `golang-migrate` mengikuti pola 3 tahap:
1. **Expand**: Menambahkan tabel/kolom baru tanpa menghapus kolom lama. Versi aplikasi lama dan baru dapat berjalan berdampingan.
2. **Migrate**: Memindahkan/menulis data ke struktur baru.
3. **Contract**: Menghapus kolom lama setelah seluruh pod layanan dimigrasikan ke versi terbaru.

### 6.3 Dokumentasi Hidup (*Living Documentation*)
Dokumentasi teknis diperlakukan sebagai kode (*documentation-as-code*):
- **[`BLUEPRINT.md`](file:///Users/aggasugitha/Documents/erp.business/BLUEPRINT.md)**: Gambaran arsitektur terpadu, katalog 19 layanan, dan env vars.
- **[`docs/adr/`](file:///Users/aggasugitha/Documents/erp.business/docs/adr)**: Catatan keputusan permanen yang tidak boleh dihapus (`ADR-001` s.d. `ADR-015`).
- **[`CHANGELOG.md`](file:///Users/aggasugitha/Documents/erp.business/CHANGELOG.md)** & **[`PROGRESS.md`](file:///Users/aggasugitha/Documents/erp.business/PROGRESS.md)**: Pelacakan perubahan berkala dan status pengujian.

---

## 7. Kesimpulan & Rekomendasi Eksekusi

Membangun ERP Enterprise berbasis **Microservices** adalah keputusan yang tepat dan terbukti untuk skala ratusan cabang, multi-negara, dan jutaan transaksi, **dengan syarat mutlak** menerapkan arsitektur yang telah distandarkan pada repositori ini:

1. **Jadikan Database-per-Service sebagai hukum mutlak**: Tidak boleh ada *cross-database join*.
2. **Tegakkan Transactional Outbox + Saga**: Menghilangkan kegagalan *dual-write* dan menjamin konsistensi data lintas domain.
3. **Posisikan PostgreSQL sebagai System of Record Tunggal**: Apache Kafka berfungsi sebagai *event backbone*, bukan tempat penyimpanan data utama.
4. **Gunakan Decoupled SPA & Chrome Extension**: Memberikan antarmuka kerja berkecepatan tinggi tanpa membebani server backend dengan beban rendering HTML.
5. **Gunakan Cortex Framework Data Products**: Mengisolasi query analitik dan laporan konsolidasi holding ke BigQuery agar transaksi harian tetap berjalan dengan latensi sub-detik.

