# ADR-015: Composite Partitioning, Read Replicas & Connection Pooling untuk Jutaan Transaksi

**Status**: Accepted  
**Tanggal**: 2026-11-11  
**Pengambil Keputusan**: Principal Database Architect, Infrastructure Lead, Platform SRE  
**Terkait**: ADR-001 (Microservices & DB-per-service), ADR-002 (Hybrid Multi-Tenancy & RLS), ADR-005 (Selective Exactly-Once), NFR-001 (API Response Time), NFR-002 (Throughput >= 500 tx/sec), NFR-010/011 (Skalabilitas Horizontal & Pertumbuhan Data)

---

## Konteks

Sistem ERP enterprise diwajibkan mampu memproses **jutaan transaksi secara real-time** yang berasal dari ratusan cabang, anak perusahaan, dan unit bisnis di berbagai negara tanpa mengalami degradasi performa:
1. **Ledakan Ukuran Tabel (Table Bloat & Index Degradation)**: Tabel jurnal umum (`gl.journal_line`), mutasi persediaan (`inv.stock_movement`), invoice, dan jejak audit (`audit.audit_log`) bertumbuh jutaan baris setiap bulan. Pada tabel monolitik biasa, indeks B-Tree yang terlalu besar akan melebihi kapasitas RAM buffer cache, menyebabkan I/O disk tinggi dan autovacuum yang memakan waktu berhari-hari.
2. **Kelelahan Koneksi (*Connection Pool Exhaustion*)**: Dengan 19+ microservices yang masing-masing menjalankan banyak replika pod (HPA), ribuan koneksi langsung ke PostgreSQL akan menyebabkan *context switching overhead* dan konsumsi memori masif pada kernel server basis data.
3. **Persaingan I/O antara Transaksi Tulis vs Laporan Baca**: Staf cabang menerbitkan faktur dan jurnal secara bersamaan dengan staf holding yang menarik laporan agregasi triwulanan. Query agregasi berat berisiko mengunci tabel atau menghabiskan I/O bandwidth.
4. **Distribusi Beban Event Streaming (Kafka)**: Topic Kafka bervolume tinggi harus mampu mendistribusikan jutaan pesan per hari secara merata ke puluhan broker tanpa merusak urutan (*in-order processing*) per dokumen akuntansi.

---

## Keputusan

Kami menetapkan arsitektur data & infrastruktur performa tinggi untuk menangani transaksi jutaan baris secara real-time:

1. **Strategi Partisi Komposit (*Composite Partitioning*) & Partition Pruning**:
   - **Partisi Utama (RANGE by Date)**: Tabel bervolume tinggi (`gl.journal_line`, `inv.stock_movement`, `audit.audit_log`, `outbox`) di-partisi bulanan berdasarkan tanggal transaksi (`posting_date`, `movement_date`, `created_at`).
   - **Sub-Partisi (HASH by `organization_id`)**: Untuk tabel hiper-skala (seperti baris jurnal GL), setiap partisi bulanan dibagi lagi menjadi sub-partisi HASH (4 hingga 8 bucket per bulan) untuk mendistribusikan I/O secara merata.
   - **Disiplin Partition Pruning Wajib**: Seluruh query di layer repository (`sqlc`) **wajib** menyertakan predikat `posting_date` (atau rentang tanggal) dan `organization_id` pada klausa `WHERE`. Hal ini memaksa *PostgreSQL Query Planner* hanya men-scan 1 partisi fisik yang relevan (*partition pruning*) dan mengabaikan 99% data historis lainnya.

2. **Otomasi Pemeliharaan Partisi Tanpa Downtime (`pg_partman` & pg_cron)**:
   - Partisi untuk 3 bulan ke depan secara otomatis di-generate di latar belakang melalui fungsi `ensure_future_partitions()`, lengkap dengan penegakan RLS otomatis pada partisi baru.
   - Partisi data historis berumur lebih dari batas retensi aktif (mis. > 2 tahun) diarsipkan secara transparan menggunakan `DETACH PARTITION` (bukan `DROP TABLE`), lalu dipindahkan ke penyimpanan *cold storage* atau analytical lakehouse tanpa mengunci tabel aktif.

3. **Infrastruktur Connection Pooling Berlapis (PgBouncer)**:
   - Setiap klaster PostgreSQL dilindungi oleh pod **PgBouncer** yang berjalan dalam mode **Transaction Pooling**.
   - Setiap pod microservice Go mengelola koneksi lokal via `pgxpool` (maksimum 15–20 koneksi per pod), yang kemudian disalurkan melalui PgBouncer. Arsitektur ini memungkinkan klaster basis data melayani hingga 10.000 koneksi konkuren klien dengan penggunaan memori yang stabil.
   - Penegakan timeout ketat: `statement_timeout = 10s`, `lock_timeout = 3s`, dan `idle_in_transaction_session_timeout = 15s` untuk mencegah query gantung (*zombie locks*).

4. **Pemisahan Jalur Baca & Tulis (*Segregated Read Replicas*)**:
   - Klaster PostgreSQL menggunakan *streaming replication* ke minimal 2 unit **Read Replicas**.
   - Operasi mutasi transaksional (INSERT/UPDATE/DELETE dengan ACID) diarahkan secara eksklusif ke node Primary (`erp_app_rw`).
   - Query laporan, dashboard analitik, dan pencarian daftar transaksi diarahkan ke pool Read Replicas via role `erp_app_ro`, mengisolasi beban baca dari jalur komitmen transaksi finansial.

5. **Kunci Partisi Kafka untuk Event Streaming Berskala Jutaan**:
   - Seluruh event transaksi menggunakan `organization_id` (atau komposit `organization_id:aggregate_id`) sebagai **Partition Key** Kafka.
   - Hal ini menjamin bahwa seluruh transaksi untuk satu cabang/perusahaan tertentu selalu diproses secara berurutan (*guaranteed chronological order*), sementara beban transaksi ratusan cabang lain terdistribusi secara paralel di puluhan partisi broker Kafka.
   - Konsumen Kafka dikonfigurasi dengan *batch processing* dan *parallel asynchronous commit* untuk mencapai throughput pemrosesan > 5.000 event per detik per consumer group.

---

## Rasional

- Mencegah penurunan performa basis data (*graceful linear scaling*) seiring bertambahnya data dari puluhan gigabyte ke puluhan terabyte.
- Menjamin waktu respons API P95 tetap di bawah 2 detik (NFR-001) dan throughput tulis mencapai > 500 transaksi per detik (NFR-002).
- Mencegah insiden *database connection saturation* saat terjadi lonjakan transaksi serentak di akhir bulan atau jam sibuk operasional cabang.

---

## Konsekuensi

### Positif (+)
- Sistem terbukti mampu menangani jutaan transaksi harian tanpa degradasi indeks atau I/O.
- Backup dan restore menjadi jauh lebih cepat karena partisi historis dapat di-backup secara independen.
- Penggunaan sumber daya server sangat efisien berkat PgBouncer dan partition pruning.

### Negatif (−)
- Skema database memerlukan disiplin pemeliharaan partisi berkala (dimitigasi dengan otomasi cron).
- Semua query pencarian data wajib mematuhi konvensi filter tanggal dan scope organisasi agar partition pruning aktif.

