# 9.6.3.45 Sales Account Integration Requirements

---

# 9.6.3.45.1 Overview

Sales Account Integration mendefinisikan bagaimana Chart of Accounts (Account) diintegrasikan, direferensikan, dan digunakan secara spesifik oleh modul Sales untuk mendukung pencatatan transaksi penjualan ke dalam General Ledger. Dokumen ini tidak membahas ulang requirement umum modul Sales (proses order-to-cash, manajemen pelanggan, atau proses penjualan itu sendiri, yang telah dibahas pada bagian 9.14 BR-766 s.d. BR-775), melainkan berfokus pada sudut pandang Account/COA: bagaimana Account pendapatan penjualan, Account diskon, Account retur, Account piutang, dan Account uang muka penjualan dikonfigurasi, dipetakan, dan diposting secara otomatis dari transaksi Sales.

Sales Account Integration memastikan setiap transaksi penjualan — mulai dari Sales Invoice, Sales Credit Note/Retur, hingga penerimaan uang muka atas kontrak penjualan jangka panjang — dapat dipetakan secara akurat ke kombinasi Account yang tepat berdasarkan kategori produk/jasa, channel penjualan, segmen pelanggan, dan organisasi (Company/Business Unit), sehingga laporan keuangan mencerminkan kinerja penjualan secara granular, konsisten, dan dapat ditelusuri (auditable) lintas organisasi.

---

# 9.6.3.45.2 Business Capability

Sales Account Integration menyediakan kapabilitas berikut:

* Sales Revenue Account Mapping per Kategori Produk/Jasa.
* Sales Revenue Account Mapping per Sales Channel.
* Sales Discount Account Mapping.
* Sales Return/Credit Note Account Mapping.
* Automated Sales Invoice-to-Revenue Account Posting.
* Automated Sales Invoice-to-Receivable Account Posting berdasarkan kombinasi Produk-Pelanggan-Organisasi.
* Unearned Revenue (Uang Muka Penjualan) Account Mapping untuk kontrak jangka panjang.
* Unearned-to-Earned Revenue Reclassification Mapping.
* Multi-Organization dan Multi-Currency Sales Account Derivation.
* Sales Account Mapping Validation dan Exception Handling.
* Sales Account Mapping Audit Trail.
* Sales Account Mapping Reporting Integration ke Financial Reporting dan Sales Profitability.

---

# 9.6.3.45.3 Business Requirement Catalogue

## BR-1026 – Sales Revenue Account Mapping by Product/Service Category

| Attribute           | Description                                                                                                                                                                                                                          |
| ------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-1026                                                                                                                                                                                                                               |
| Requirement Name    | Sales Revenue Account Mapping by Product/Service Category                                                                                                                                                                            |
| Category            | Sales Account Integration                                                                                                                                                                                                            |
| Priority            | Critical                                                                                                                                                                                                                              |
| Business Objective  | BO-004                                                                                                                                                                                                                                |
| Stakeholder Owner   | SH-003 (Chief Financial Officer), SH-034 (Sales Department)                                                                                                                                                                          |
| Description         | Sistem harus menyediakan Account pendapatan penjualan (Sales Revenue Account) yang dapat dipetakan secara spesifik terhadap setiap kategori produk, kategori jasa, dan kombinasi keduanya pada modul Sales.                         |
| Business Rationale  | Tanpa pemetaan Account pendapatan yang granular per kategori produk/jasa, kinerja penjualan tidak dapat dianalisis secara terpisah antar lini bisnis, sehingga menyulitkan evaluasi profitabilitas per kategori.                    |
| Business Value      | Menjadi single source of truth bagi klasifikasi pendapatan penjualan di General Ledger, mendukung analisis profitabilitas per kategori produk/jasa secara real-time.                                                                |
| Acceptance Criteria | Administrator dapat mengonfigurasi mapping Account pendapatan per kategori produk/jasa; sistem menolak posting transaksi Sales apabila kategori produk belum memiliki mapping Account yang valid.                                   |

---

## BR-1027 – Sales Revenue Account Mapping by Sales Channel

| Attribute        | Description                                                                                                                             |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1027                                                                                                                                  |
| Requirement Name  | Sales Revenue Account Mapping by Sales Channel                                                                                          |
| Priority          | High                                                                                                                                     |
| Description       | Sistem harus mendukung pemetaan Account pendapatan penjualan berdasarkan Sales Channel (Direct, Distributor, Online/E-Commerce, Retail). |

---

## BR-1028 – Sales Discount Account Mapping

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1028                                                                                                                           |
| Requirement Name  | Sales Discount Account Mapping                                                                                                    |
| Priority          | Critical                                                                                                                          |
| Description       | Sistem harus menyediakan Account diskon penjualan (Sales Discount Account) yang terpisah dari Account pendapatan untuk mencatat pengurangan nilai penjualan akibat diskon, baik trade discount maupun cash discount. |

---

## BR-1029 – Sales Return Account Mapping

| Attribute        | Description                                                                                                                          |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1029                                                                                                                              |
| Requirement Name  | Sales Return Account Mapping                                                                                                         |
| Priority          | Critical                                                                                                                             |
| Description       | Sistem harus menyediakan Account retur penjualan (Sales Return Account) untuk mencatat pengurangan pendapatan akibat retur/credit note, terpisah dari Account pendapatan kotor (gross revenue). |

---

## BR-1030 – Automated Sales Invoice-to-Revenue Account Posting

| Attribute        | Description                                                                                                                                        |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-1030                                                                                                                                            |
| Requirement Name  | Automated Sales Invoice-to-Revenue Account Posting                                                                                                 |
| Priority          | Critical                                                                                                                                            |
| Description       | Sistem harus melakukan derivasi dan posting otomatis Account pendapatan pada setiap baris Sales Invoice berdasarkan mapping kategori produk/jasa, Sales Channel, dan organisasi tanpa intervensi manual. |

---

## BR-1031 – Automated Sales Invoice-to-Receivable Account Posting by Product-Customer-Organization Combination

| Attribute        | Description                                                                                                                                                          |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1031                                                                                                                                                              |
| Requirement Name  | Automated Sales Invoice-to-Receivable Account Posting by Product-Customer-Organization Combination                                                                 |
| Priority          | Critical                                                                                                                                                              |
| Description       | Sistem harus melakukan derivasi Account piutang (Accounts Receivable Account) secara otomatis berdasarkan kombinasi Kategori Produk, Segmen/Grup Pelanggan, dan Organisasi (Company/Business Unit) pada setiap Sales Invoice. |

---

## BR-1032 – Unearned Revenue Account Mapping for Long-Term Sales Contracts

| Attribute        | Description                                                                                                                                              |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1032                                                                                                                                                  |
| Requirement Name  | Unearned Revenue Account Mapping for Long-Term Sales Contracts                                                                                          |
| Priority          | High                                                                                                                                                      |
| Description       | Sistem harus menyediakan Account uang muka penjualan (Unearned Revenue/Deferred Revenue Account) untuk mencatat penerimaan pembayaran di muka atas kontrak penjualan jangka panjang sebelum kewajiban penyerahan barang/jasa terpenuhi. |

---

## BR-1033 – Unearned-to-Earned Revenue Reclassification Mapping

| Attribute        | Description                                                                                                                                              |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1033                                                                                                                                                  |
| Requirement Name  | Unearned-to-Earned Revenue Reclassification Mapping                                                                                                     |
| Priority          | High                                                                                                                                                      |
| Description       | Sistem harus menyediakan mapping Account untuk reklasifikasi bertahap dari Unearned Revenue Account menjadi Sales Revenue Account sesuai jadwal pengakuan pendapatan kontrak jangka panjang. |

---

## BR-1034 – Sales Account Mapping Validation and Exception Handling

| Attribute        | Description                                                                                                                                    |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1034                                                                                                                                        |
| Requirement Name  | Sales Account Mapping Validation and Exception Handling                                                                                       |
| Priority          | High                                                                                                                                            |
| Description       | Sistem harus melakukan validasi otomatis terhadap kelengkapan dan validitas mapping Account penjualan, serta menampilkan exception ke Finance apabila kombinasi mapping tidak ditemukan. |

---

## BR-1035 – Sales Account Mapping Audit Trail and Change History

| Attribute        | Description                                                                                                                       |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1035                                                                                                                           |
| Requirement Name  | Sales Account Mapping Audit Trail and Change History                                                                             |
| Priority          | Medium                                                                                                                            |
| Description       | Sistem harus mencatat riwayat perubahan mapping Account penjualan, termasuk perubahan Account pendapatan, diskon, retur, dan Unearned Revenue. |

---

# 9.6.3.45.4 Business Rules

| Business Rule ID    | Description                                                                                                                                                                 |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| BRule-SALESAI-001    | Setiap kategori produk/jasa yang aktif pada modul Sales harus memiliki mapping Account pendapatan yang valid sebelum Sales Invoice dapat diposting.                       |
| BRule-SALESAI-002    | Account diskon dan Account retur penjualan harus merupakan Account tersendiri (contra-revenue) dan tidak boleh sama dengan Account pendapatan kotor.                      |
| BRule-SALESAI-003    | Derivasi Account piutang pada Sales Invoice harus mengikuti prioritas mapping: kombinasi Produk-Pelanggan-Organisasi lebih diutamakan daripada mapping default organisasi. |
| BRule-SALESAI-004    | Penerimaan uang muka atas kontrak penjualan jangka panjang wajib diposting ke Unearned Revenue Account dan tidak boleh langsung diakui sebagai Sales Revenue.              |
| BRule-SALESAI-005    | Reklasifikasi dari Unearned Revenue Account ke Sales Revenue Account hanya dapat dilakukan sesuai jadwal pengakuan pendapatan yang telah disetujui.                        |
| BRule-SALESAI-006    | Sistem harus mencegah posting Sales Invoice apabila kombinasi mapping Account pendapatan/piutang tidak ditemukan; transaksi harus ditahan pada status Exception.           |
| BRule-SALESAI-007    | Perubahan mapping Account penjualan yang berdampak pada transaksi yang telah diposting harus melalui approval workflow dan tidak boleh mengubah data historis.             |

---

# 9.6.3.45.5 Data Attributes

| Attribute                        | Description                                                                    |
| ---------------------------------- | --------------------------------------------------------------------------------- |
| Sales Account Mapping ID           | Identitas unik sistem untuk konfigurasi mapping Account penjualan.               |
| Product/Service Category           | Referensi kategori produk/jasa pada modul Sales.                                |
| Sales Channel                      | Direct, Distributor, Online/E-Commerce, Retail.                                 |
| Customer Segment/Group             | Referensi segmentasi/grup pelanggan pada modul Sales.                           |
| Organization/Company/Business Unit | Cakupan organisasi tempat mapping berlaku.                                      |
| Revenue Account ID                 | Referensi Account pendapatan penjualan pada Chart of Accounts.                  |
| Discount Account ID                | Referensi Account diskon penjualan.                                            |
| Return Account ID                  | Referensi Account retur penjualan.                                             |
| Receivable Account ID              | Referensi Account piutang usaha (Accounts Receivable).                         |
| Unearned Revenue Account ID        | Referensi Account uang muka penjualan/pendapatan tangguhan.                    |
| Contract Type                      | Referensi jenis kontrak penjualan (One-Time, Subscription, Multi-Year).         |
| Effective From / To                | Periode berlaku mapping.                                                       |
| Approval Status                    | Draft, Pending Approval, Approved, Published, Archived.                        |
| Status                             | Active, Suspended, Inactive, Archived.                                         |
| Created By / Created Date          | Informasi pembuatan data.                                                      |
| Updated By / Updated Date          | Informasi perubahan terakhir.                                                  |

---

# 9.6.3.45.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder     | Future Process                                  |
| --------------- | ------------------- | ---------------- | -------------------------------------------------- |
| BR-1026         | BO-004              | SH-003, SH-034   | Sales Revenue Classification by Category           |
| BR-1027         | BO-004              | SH-034           | Sales Channel Performance Analysis                 |
| BR-1028         | BO-004              | SH-003           | Sales Discount Accounting Control                  |
| BR-1029         | BO-004              | SH-003           | Sales Return/Credit Note Accounting Control         |
| BR-1030         | BO-003              | SH-003, SH-034   | Automated Sales Invoice Posting                    |
| BR-1031         | BO-003              | SH-003           | Automated Receivable Recognition                   |
| BR-1032         | BO-004              | SH-003           | Long-Term Contract Deferred Revenue Management     |
| BR-1033         | BO-008              | SH-003           | Revenue Recognition Schedule Processing            |
| BR-1034         | BO-004              | SH-003           | Sales Account Mapping Exception Management         |
| BR-1035         | BO-004              | SH-003, SH-034   | Sales Account Mapping Audit and Reporting          |

---

# 9.6.3.45.7 Summary

Sales Account Integration menjembatani transaksi operasional modul Sales dengan Chart of Accounts secara otomatis, konsisten, dan dapat ditelusuri. Dengan mapping Account pendapatan per kategori produk/jasa dan channel, pemisahan Account diskon dan retur, derivasi otomatis Account piutang berbasis kombinasi Produk-Pelanggan-Organisasi, serta pengelolaan Unearned Revenue untuk kontrak jangka panjang, organisasi memperoleh single source of truth atas kinerja keuangan penjualan yang mendukung pengambilan keputusan berbasis data secara real-time.
