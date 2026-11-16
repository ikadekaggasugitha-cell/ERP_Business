# ADR-013: Cortex Framework Data Products & SAP DDIC Coexistence

**Status**: Accepted  
**Tanggal**: 2026-11-11  
**Pengambil Keputusan**: Chief Enterprise Architect, Lead Data Architect, Head of ERP Delivery  
**Terkait**: ADR-001 (Microservices & DB-per-service), ADR-009 (CQRS Read Model), CONS-006 (Legacy Coexistence), CONS-014 (PostgreSQL System of Record), NFR-002 (Transaction Throughput), NFR-011 (Organizational & Transactional Growth)

---

## Konteks

Pada level korporasi multi-anak perusahaan dan multi-negara, ERP baru tidak langsung menggantikan seluruh sistem lama dalam satu malam. Sebagian anak perusahaan atau divisi manufaktur besar mungkin masih menggunakan **SAP (ECC 6.0 atau S/4HANA)** atau aplikasi legacy selama masa transisi bertahun-tahun (CONS-006).

Selain itu, kebutuhan pelaporan analitik holding (seperti analisis tren penjualan 5 tahunan lintas cabang, profitabilitas produk global, konsolidasi multi-perusahaan) membutuhkan agregasi jutaan baris data historis. Menjalankan query analitik seberat ini langsung ke basis data transaksional operasional (`gl_db`, `sales_db`, `inventory_db`) akan menghabiskan *connection pool*, mengunci I/O disk, dan melanggar SLA waktu respons API transaksional (NFR-001 & NFR-002).

Dibutuhkan arsitektur produk data analitik tingkat enterprise yang mampu mengintegrasikan data dari ERP baru dan sistem SAP yang ada secara standar, terkelola, dan terisolasi dari beban OLTP.

---

## Keputusan

Kami menetapkan adopsi standar **Cortex Framework V7 & Enterprise Data Product Architecture** untuk integrasi analitik dan koeksistensi dengan SAP:

1. **Prinsip Operasional Wajib**:
   - **Surface Assumptions**: Sebelum membuat model data, setiap asumsi bisnis mengenai field tabel, konversi mata uang, atau hierarki unit organisasi harus dipaparkan secara eksplisit ke pemangku kepentingan.
   - **Manage Confusion Actively**: Jika terdapat ketidaksesuaian antara skema SAP (mis. field BSEG vs ACDOCA) atau definisi akun lokal vs global, tim dilarang berasumsi diam-diam dan wajib mengonfirmasi versi target (**ECC vs S/4HANA vs Keduanya**).
   - **Namespace Kustom (`custom` / `sinergi`)**: Seluruh artefak data model, skrip transformasi, dan metadata analitik wajib menggunakan namespace kustom untuk mencegah tabrakan (*upgrade overwrite*) dengan pustaka Cortex upstream.

2. **Arsitektur 3-Lapisan Data Product**:
   - **Lapisan 1: Data Foundations (`foundations/`)**:
     - Mereplikasi tabel kamus data SAP (SAP DDIC: `DD02L`, `DD03L`, `DD08L`) serta tabel transaksi inti (`BKPF`, `BSEG`, `ACDOCA`, `MARC`, `VBAK`, `VBAP`, `EKKO`, `EKPO`) langsung ke BigQuery / Enterprise Lakehouse.
     - Mengalirkan event transaksional Kafka dari ERP baru (`erp.*.v1`) ke tabel raw data lakehouse secara asinkron.
   - **Lapisan 2: Annotations & Semantic Layer (`annotations/`)**:
     - Standardisasi tipe data, normalisasi field name (penamaan bisnis yang jelas), dan penyelarasan perbedaan struktur antara SAP ECC dan S/4HANA.
   - **Lapisan 3: Data Products (`products/`)**:
     - Model siap konsumsi yang dikompilasi menggunakan **Dataform (.js)** ke dalam skema bintang (*dimensional star schema*).

3. **Skema Bintang Terpadu (*Conformed Dimensions*)**:
   - `dim_organization`: Hierarki organisasi fleksibel berbasis closure table yang menyatukan kode unit ERP baru dan SAP Company Code / Business Area.
   - `dim_currency`: Dimensi mata uang dengan riwayat kurs harian untuk translasi multi-valuta otomatis.
   - `dim_fiscal_period`: Penyelaras periode fiskal multi-negara (mis. April–Maret untuk cabang Jepang/India vs Januari–Desember).
   - `dim_chart_of_accounts`: Pemetaan akun buku besar lokal anak perusahaan ke akun standar holding.
   - `fact_gl_transaction`, `fact_sales_order`, `fact_inventory_snapshot`: Tabel fakta berpartisi waktu (partitioned by date) yang dioptimasi untuk query agregasi instan.

4. **Isolasi Penuh Beban Kerja (OLTP vs OLAP)**:
   - Basis data operasional PostgreSQL fokus melayani transaksi atomik real-time (CRUD operasional).
   - Seluruh kebutuhan laporan analitik eksekutif, BI dashboard, konsolidasi holding, dan audit rekonsiliasi lintas sistem dilayani oleh Cortex Data Products di BigQuery / Lakehouse tanpa menyentuh pod PostgreSQL transaksional.

---

## Rasional

- Memberikan transparansi data konsolidasi tanpa memaksakan migrasi *big-bang* pada entitas yang masih menjalankan SAP.
- Menjamin performa basis data operasional (PostgreSQL) tetap stabil di bawah beban jutaan transaksi harian.
- Memberikan tata kelola skema dan dokumentasi data model yang ketat sesuai standar data engineering Cortex Framework.

---

## Konsekuensi

### Positif (+)
- Holding mendapatkan *Single Source of Truth* analitik lintas sistem (ERP baru + SAP legacy).
- Query agregasi data tahunan berjalan dalam hitungan detik di BigQuery tanpa membebani OLTP.
- Struktur namespace yang aman memudahkan upgrade sistem di masa depan.

### Negatif (−)
- Memerlukan infrastruktur pipeline replikasi data CDC / ETL ke BigQuery atau data warehouse.
- Terdapat jeda waktu *eventual consistency* (beberapa menit) antara transaksi OLTP dengan ketersediaan data di model analitik.

