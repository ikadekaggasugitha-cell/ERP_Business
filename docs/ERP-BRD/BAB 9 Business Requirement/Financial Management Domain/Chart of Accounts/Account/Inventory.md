# 9.6.3.42 Inventory Account Integration Requirements

---

# 9.6.3.42.1 Overview

Inventory Account Integration Requirements mendefinisikan bagaimana setiap akun pada Chart of Accounts diintegrasikan, direferensikan, dan dipakai secara spesifik oleh modul Inventory, dilihat dari sudut pandang Account/COA — bukan dari sudut pandang proses bisnis pergerakan stok yang telah dibahas secara menyeluruh pada Inventory Management Requirements (9.8, BR-706 s/d BR-715). Dokumen ini berfokus pada bagaimana akun persediaan, akun Harga Pokok Penjualan (HPP/COGS), akun selisih stock opname, akun barang dalam perjalanan (goods in transit), dan mapping metode valuasi persediaan (FIFO/Average) diposting, dipetakan, dan direkonsiliasi terhadap General Ledger.

Setiap transaksi pergerakan stok pada akhirnya harus tercermin secara akurat pada saldo akun terkait di General Ledger. Dokumen ini menjelaskan aturan mapping akun persediaan per kategori/kelompok barang, mekanisme penentuan akun HPP secara otomatis saat barang keluar, pemisahan akun untuk selisih surplus dan defisit hasil stock opname, penanganan akun untuk barang yang berstatus dalam perjalanan antar lokasi, serta bagaimana metode valuasi yang dipilih memengaruhi perhitungan nilai yang diposting ke akun persediaan.

Requirement pada dokumen ini tidak menduplikasi requirement umum Inventory Management (BR-706 s/d BR-715), melainkan menjembatani proses pergerakan stok tersebut dengan Chart of Accounts dari perspektif akun sebagai unit terkecil integrasi pencatatan keuangan persediaan.

---

# 9.6.3.42.2 Business Capability

Inventory Account Integration menyediakan kapabilitas berikut:

* Mapping Akun Persediaan per Kategori/Kelompok Barang (Item Category/Group).
* Penentuan Akun HPP (COGS) secara Otomatis pada saat Goods Issue.
* Pemisahan Akun Selisih Stock Opname (Adjustment Gain Account dan Adjustment Loss Account).
* Mapping Akun Barang dalam Perjalanan (Goods in Transit Account) untuk Transfer Antar Lokasi.
* Mapping Metode Valuasi Persediaan (FIFO/Average/Standard Cost) ke Perhitungan Nilai Akun Persediaan.
* Mapping Akun Persediaan per Scope Warehouse/Organisasi.
* Alokasi Komponen Biaya Tambahan (Landed Cost) ke Akun Persediaan Terkait.
* Posting Otomatis Akun Revaluasi Persediaan.
* Mapping Akun Provisi Persediaan Usang/Lambat Bergerak (Obsolete/Slow-Moving Inventory Provision).
* Rekonsiliasi Saldo Akun Persediaan pada General Ledger dengan Sub-Ledger Inventory.
* Validasi Kelengkapan Mapping Akun Persediaan sebelum Transaksi Stock Pertama Diposting.
* Audit Trail atas Perubahan Mapping Akun Persediaan.

---

# 9.6.3.42.3 Business Requirement Catalogue

## BR-996 – Inventory Account Mapping per Item Category/Group

| Attribute           | Description                                                                                                                                                                                                    |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-996                                                                                                                                                                                                          |
| Requirement Name    | Inventory Account Mapping per Item Category/Group                                                                                                                                                              |
| Category            | Inventory Account Integration                                                                                                                                                                                  |
| Priority            | Critical                                                                                                                                                                                                        |
| Business Objective  | BO-004                                                                                                                                                                                                          |
| Stakeholder Owner   | SH-032 (Inventory Department)                                                                                                                                                                                  |
| Description         | Sistem harus menyediakan mekanisme mapping akun persediaan pada Chart of Accounts untuk setiap kategori atau kelompok barang (Item Category/Group), sehingga setiap transaksi stock in atas suatu kategori barang secara otomatis diposting ke akun persediaan yang sesuai. |
| Business Rationale  | Tanpa mapping akun per kategori barang, pencatatan nilai persediaan pada General Ledger tidak dapat memisahkan jenis persediaan (bahan baku, barang jadi, barang dagangan, dsb.) sehingga analisis keuangan menjadi tidak akurat. |
| Business Value      | Menjamin nilai persediaan pada General Ledger tersegmentasi dan akurat sesuai jenis/kategori barang, mendukung analisis dan pelaporan keuangan yang lebih rinci.                                              |
| Acceptance Criteria | Setiap kategori/kelompok barang memiliki akun persediaan yang dipetakan pada Chart of Accounts, dan seluruh transaksi stock in atas kategori tersebut secara otomatis diposting ke akun yang telah dipetakan. |

---

## BR-997 – Automatic COGS Account Posting on Goods Issue

| Requirement ID | Requirement Name                        | Priority | Description                                                                                                                                              |
| -------------- | ------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| BR-997         | Automatic COGS Account Posting on Goods Issue | Critical | Sistem harus secara otomatis menentukan dan memposting akun Harga Pokok Penjualan (HPP/COGS) pada saat barang keluar (goods issue) untuk penjualan, produksi, maupun konsumsi internal, berdasarkan mapping kategori barang dan metode valuasi yang berlaku. |

---

## BR-998 – Stock Opname Adjustment Gain/Loss Account Mapping

| Requirement ID | Requirement Name                           | Priority | Description                                                                                                                                          |
| -------------- | --------------------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-998         | Stock Opname Adjustment Gain/Loss Account Mapping | Critical | Sistem harus menyediakan mapping akun terpisah untuk selisih surplus (adjustment gain) dan defisit (adjustment loss) hasil stock opname, agar setiap selisih diposting ke akun yang sesuai dengan sifatnya. |

---

## BR-999 – Goods-in-Transit Account Mapping for Inter-Location Transfer

| Requirement ID | Requirement Name                                     | Priority | Description                                                                                                                                       |
| -------------- | --------------------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-999         | Goods-in-Transit Account Mapping for Inter-Location Transfer | High     | Sistem harus memetakan nilai barang yang berstatus dalam perjalanan (goods in transit) antar Warehouse/organisasi ke akun Goods in Transit yang terpisah dari akun persediaan reguler, hingga barang diterima di lokasi tujuan. |

---

## BR-1000 – Valuation Method to Inventory Account Value Calculation Mapping

| Requirement ID | Requirement Name                                        | Priority | Description                                                                                                                                    |
| -------------- | ------------------------------------------------------------ | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-1000        | Valuation Method to Inventory Account Value Calculation Mapping | Critical | Sistem harus memetakan metode valuasi persediaan (FIFO, Average, Standard Cost) yang berlaku per item ke mekanisme perhitungan nilai yang diposting ke akun persediaan, agar saldo akun konsisten dengan metode valuasi yang dikonfigurasi. |

---

## BR-1001 – Inventory Account Mapping per Warehouse/Organization Scope

| Requirement ID | Requirement Name                                     | Priority | Description                                                                                                                                 |
| -------------- | --------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| BR-1001        | Inventory Account Mapping per Warehouse/Organization Scope | High     | Sistem harus mendukung mapping akun persediaan yang berbeda per Warehouse atau unit organisasi, apabila diperlukan pemisahan pencatatan nilai persediaan antar lokasi/entitas. |

---

## BR-1002 – Landed Cost Component Account Allocation to Inventory Value

| Requirement ID | Requirement Name                                       | Priority | Description                                                                                                                                    |
| -------------- | ------------------------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-1002        | Landed Cost Component Account Allocation to Inventory Value | Medium   | Sistem harus mengalokasikan komponen biaya tambahan (landed cost seperti biaya pengiriman, bea masuk, asuransi) ke akun persediaan barang terkait, bukan ke akun biaya terpisah, sesuai kebijakan kapitalisasi persediaan. |

---

## BR-1003 – Inventory Revaluation Account Posting

| Requirement ID | Requirement Name                     | Priority | Description                                                                                                                                |
| -------------- | ---------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| BR-1003        | Inventory Revaluation Account Posting   | Medium   | Sistem harus memposting selisih revaluasi persediaan (akibat perubahan standard cost atau penyesuaian nilai wajar) ke akun revaluasi yang terpisah dari akun persediaan reguler. |

---

## BR-1004 – Obsolete/Slow-Moving Inventory Provision Account Mapping

| Requirement ID | Requirement Name                                       | Priority | Description                                                                                                                              |
| -------------- | ------------------------------------------------------------ | -------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| BR-1004        | Obsolete/Slow-Moving Inventory Provision Account Mapping | Medium   | Sistem harus menyediakan mapping akun provisi (penyisihan) untuk persediaan usang atau lambat bergerak berdasarkan hasil inventory aging, agar pencadangan nilai persediaan tercermin pada akun terpisah. |

---

## BR-1005 – Inventory-to-GL Reconciliation Account Validation

| Requirement ID | Requirement Name                          | Priority | Description                                                                                                                              |
| -------------- | ---------------------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| BR-1005        | Inventory-to-GL Reconciliation Account Validation | Critical | Sistem harus menyediakan validasi rekonsiliasi antara saldo akun persediaan pada General Ledger dengan saldo nilai persediaan pada sub-ledger Inventory, dan menandai selisih yang melampaui ambang batas toleransi sebelum periode ditutup. |

---

# 9.6.3.42.4 Business Rules

| Business Rule ID | Description                                                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-INVAI-001    | Setiap kategori/kelompok barang wajib dipetakan ke akun persediaan tertentu pada Chart of Accounts sebelum transaksi stock in pertama kali dapat diposting.       |
| BRule-INVAI-002    | Akun HPP (COGS) yang digunakan pada saat goods issue ditentukan otomatis berdasarkan mapping kategori barang dan metode valuasi yang berlaku, tanpa memerlukan input manual pengguna. |
| BRule-INVAI-003    | Selisih hasil stock opname yang bersifat surplus (gain) dan defisit (loss) wajib diposting ke akun yang berbeda sesuai konfigurasi, guna memisahkan analisis penyebab selisih. |
| BRule-INVAI-004    | Barang yang berstatus dalam perjalanan (goods in transit) antar lokasi/organisasi wajib dicatat pada akun Goods in Transit terpisah dari akun persediaan reguler hingga barang diterima di lokasi tujuan. |
| BRule-INVAI-005    | Perubahan metode valuasi persediaan (FIFO/Average) untuk suatu item tidak boleh diterapkan secara retroaktif terhadap nilai akun persediaan yang telah diposting pada periode yang telah ditutup. |
| BRule-INVAI-006    | Komponen biaya tambahan (landed cost) yang dialokasikan ke nilai persediaan wajib diposting ke akun persediaan yang sama dengan barang terkait, bukan ke akun biaya terpisah. |
| BRule-INVAI-007    | Saldo akun persediaan pada General Ledger wajib direkonsiliasi secara berkala dengan saldo nilai persediaan pada sub-ledger Inventory; selisih di atas ambang batas wajib diinvestigasi sebelum periode ditutup. |

---

# 9.6.3.42.5 Data Attributes

| Attribute                          | Description                                                                                     |
| ------------------------------------ | --------------------------------------------------------------------------------------------------- |
| Item Category/Group Reference        | Referensi kategori atau kelompok barang yang dipetakan ke akun persediaan.                        |
| Inventory Account Reference          | Referensi akun persediaan pada Chart of Accounts untuk kategori barang terkait.                   |
| COGS Account Reference               | Referensi akun HPP (COGS) yang digunakan pada saat goods issue.                                    |
| Valuation Method                     | Metode valuasi persediaan yang berlaku (FIFO, Average, Standard Cost).                             |
| Adjustment Gain Account Reference    | Referensi akun untuk selisih surplus hasil stock opname.                                            |
| Adjustment Loss Account Reference    | Referensi akun untuk selisih defisit hasil stock opname.                                            |
| Goods-in-Transit Account Reference   | Referensi akun untuk barang berstatus dalam perjalanan antar lokasi/organisasi.                    |
| Landed Cost Allocation Mapping       | Mapping komponen biaya tambahan ke akun persediaan terkait.                                        |
| Revaluation Account Reference        | Referensi akun untuk selisih revaluasi persediaan.                                                  |
| Provision Account Reference          | Referensi akun provisi untuk persediaan usang/lambat bergerak.                                     |
| Organization/Warehouse Scope         | Ruang lingkup organisasi/warehouse berlakunya mapping akun persediaan.                             |
| Reconciliation Status                | Status rekonsiliasi saldo akun persediaan dengan sub-ledger Inventory (Balanced, Variance, Under Review). |
| Reconciliation Tolerance Threshold   | Ambang batas toleransi selisih yang diizinkan sebelum ditandai sebagai variance signifikan.        |
| Last Reconciliation Date             | Tanggal terakhir proses rekonsiliasi dilakukan.                                                     |
| Effective From / To                  | Periode berlaku mapping akun persediaan.                                                            |
| Created By / Created Date            | Informasi pembuatan data mapping akun persediaan.                                                   |
| Updated By / Updated Date            | Informasi perubahan terakhir data mapping akun persediaan.                                          |

---

# 9.6.3.42.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                               |
| -------------- | ------------------- | ----------- | ---------------------------------------------- |
| BR-996         | BO-004              | SH-032      | Inventory Account Mapping Configuration          |
| BR-997         | BO-009              | SH-032      | Automatic COGS Posting Engine                    |
| BR-998         | BO-022              | SH-032      | Stock Opname Adjustment Posting                  |
| BR-999         | BO-004              | SH-032      | Goods-in-Transit Account Tracking                |
| BR-1000        | BO-022              | SH-032      | Valuation Method to GL Posting Mapping           |
| BR-1001        | BO-004              | SH-032      | Warehouse-Scoped Inventory Account Configuration |
| BR-1002        | BO-022              | SH-032      | Landed Cost Capitalization Posting               |
| BR-1003        | BO-022              | SH-032      | Inventory Revaluation Posting                    |
| BR-1004        | BO-022              | SH-032      | Inventory Provision Posting                      |
| BR-1005        | BO-022              | SH-003, SH-032 | Inventory-to-GL Reconciliation Review        |

---

# 9.6.3.42.7 Summary

Inventory Account Integration Requirements menetapkan bagaimana Account/Chart of Accounts berperan aktif dalam pencatatan keuangan atas transaksi persediaan, mencakup mapping akun per kategori barang, penentuan otomatis akun HPP saat barang keluar, pemisahan akun selisih stock opname, akun barang dalam perjalanan, mapping metode valuasi ke perhitungan nilai akun persediaan, alokasi landed cost, revaluasi, provisi persediaan usang, hingga rekonsiliasi saldo akun persediaan dengan sub-ledger Inventory. Requirement pada dokumen ini melengkapi, bukan menduplikasi, requirement umum Inventory Management (9.8), dengan fokus khusus pada perspektif integrasi Account.
