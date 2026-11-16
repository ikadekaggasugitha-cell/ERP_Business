# 9.6.3.43 Procurement Account Integration Requirements

---

# 9.6.3.43.1 Overview

Dokumen ini mengatur integrasi **Chart of Accounts (COA) / Account** dengan modul **Procurement Management** yang telah dibahas secara menyeluruh dari sisi proses bisnis pada dokumen *Procurement Management Requirements* (9.7, BR-696 s/d BR-705). Dokumen tersebut mengatur alur Purchase Requisition, RFQ, evaluasi vendor, Purchase Order (PO), hingga Goods Receipt sebagai proses operasional pengadaan. Dokumen ini **tidak mengulang** requirement tersebut, melainkan membahas secara spesifik bagaimana struktur akun General Ledger (GL) **direferensikan, dipetakan, dan diposting secara otomatis** oleh transaksi procurement, dilihat murni dari sudut pandang Account/COA.

Fokus dokumen ini mencakup empat area akun yang menjadi titik kritis integrasi keuangan pengadaan, yaitu:

1. **Akun Purchase Accrual** — akun kewajiban sementara atas barang/jasa yang telah diterima namun belum ditagih oleh vendor (accrued liability).
2. **Akun GR/IR (Goods Receipt/Invoice Receipt) Clearing** — akun kliring yang menjembatani selisih waktu antara pencatatan penerimaan barang dan penerimaan invoice vendor, dilihat dari sisi procurement sebagai pemicu (trigger) posting, bukan dari sisi rekonsiliasi Accounts Payable.
3. **Akun Uang Muka Pembelian (Purchase Advance)** — akun aset yang menampung pembayaran di muka kepada vendor sebelum barang/jasa diterima.
4. **Mapping otomatis kategori pembelian ke akun beban, aset, atau persediaan** — penentuan akun tujuan posting berdasarkan jenis Purchase Order (Inventory Item, Expense/Service, Capital/Asset Item).

Tanpa aturan akun yang eksplisit pada area ini, organisasi berisiko mencatat kewajiban pengadaan pada akun yang keliru, kehilangan visibilitas atas saldo GR/IR yang menumpuk (aged), salah menempatkan uang muka pembelian sebagai beban, atau salah memetakan pembelian barang modal sebagai beban operasional.

---

# 9.6.3.43.2 Business Capability

Procurement Account Integration menyediakan kapabilitas berikut:

* Purchase Accrual Account Auto-Posting pada saat Goods Receipt/Service Receipt dicatat.
* Accrual Reversal Rule terkonfigurasi (reversal otomatis pada periode berikutnya atau menunggu invoice vendor).
* GR/IR Clearing Account Structure per kategori pembelian (barang vs jasa).
* GR/IR Aging Monitoring & Exception Alerting untuk saldo yang belum ter-clear dalam ambang waktu tertentu.
* Purchase Advance Account Mapping sebagai akun aset, terpisah dari akun beban/persediaan.
* Purchase Advance Offset Otomatis terhadap invoice vendor final.
* PO Category-to-Account Auto-Mapping (Inventory, Expense, Asset) berbasis Item Category.
* Expense Account Derivation Rule untuk PO non-inventori (jasa/beban operasional).
* Fixed Asset Clearing Account Mapping untuk PO belanja modal (capital expenditure).
* Purchase Price Variance (PPV) Account Posting untuk item berbasis standard costing pada saat Goods Receipt.
* Multi-Currency Purchase Accrual Revaluation Account Handling.
* Account Combination Completeness Validation sebelum PO/Goods Receipt dapat diposting ke GL.

---

# 9.6.3.43.3 Business Requirement Catalogue

## BR-1006 – Purchase Accrual Account Auto-Posting for Goods/Services Received Not Yet Invoiced

| Attribute           | Description                                                                                                                                                                                                                                                    |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-1006                                                                                                                                                                                                                                                        |
| Requirement Name    | Purchase Accrual Account Auto-Posting for Goods/Services Received Not Yet Invoiced                                                                                                                                                                            |
| Category            | Procurement Account Integration                                                                                                                                                                                                                                |
| Priority            | Critical                                                                                                                                                                                                                                                       |
| Business Objective  | BO-003, BO-004                                                                                                                                                                                                                                                 |
| Stakeholder Owner   | SH-003 (CFO)                                                                                                                                                                                                                                                   |
| Description         | Sistem harus secara otomatis memposting jurnal ke **akun Purchase Accrual** (kewajiban sementara) pada saat barang atau jasa diterima namun invoice vendor belum diterima, sehingga kewajiban tersebut tercermin di Neraca sejak titik penerimaan, bukan menunggu invoice. |
| Business Rationale  | Tanpa akun accrual yang eksplisit, kewajiban atas barang/jasa yang sudah diterima namun belum ditagih tidak tercermin di laporan keuangan periode berjalan, sehingga menyebabkan understatement liabilitas dan beban pada periode terkait.                    |
| Business Value      | Menjamin kelengkapan pengakuan kewajiban (completeness of liabilities) sesuai basis akrual dan mendukung penyajian laporan keuangan yang akurat pada setiap periode tutup buku.                                                                              |
| Acceptance Criteria | Setiap Goods Receipt/Service Receipt yang belum memiliki invoice vendor terkait menghasilkan jurnal debit akun beban/persediaan dan kredit akun Purchase Accrual secara otomatis, tanpa entri manual.                                                          |

---

## BR-1007 – GR/IR Clearing Account Structure and Matching Logic

| Attribute        | Description                                                                                                                                                                                                                    |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1007                                                                                                                                                                                                                       |
| Requirement Name | GR/IR Clearing Account Structure and Matching Logic                                                                                                                                                                          |
| Priority         | Critical                                                                                                                                                                                                                      |
| Description      | Sistem harus menyediakan akun kliring **Goods Receipt/Invoice Receipt (GR/IR)** yang menampung selisih sementara antara nilai Goods Receipt dan nilai Invoice vendor, serta melakukan matching otomatis saat invoice diterima untuk menutup (clear) saldo akun tersebut. |

---

## BR-1008 – GR/IR Clearing Aging Monitoring and Exception Reporting

| Attribute        | Description                                                                                                                                                                    |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1008                                                                                                                                                                       |
| Requirement Name | GR/IR Clearing Aging Monitoring and Exception Reporting                                                                                                                     |
| Priority         | High                                                                                                                                                                          |
| Description      | Sistem harus memantau umur (aging) saldo akun GR/IR per Purchase Order dan menghasilkan laporan eksepsi untuk saldo yang belum ter-clear melewati ambang waktu yang dikonfigurasi, agar dapat ditindaklanjuti sebelum tutup buku. |

---

## BR-1009 – Purchase Advance Account Mapping as Asset

| Attribute        | Description                                                                                                                                                            |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1009                                                                                                                                                               |
| Requirement Name | Purchase Advance Account Mapping as Asset                                                                                                                            |
| Priority         | Critical                                                                                                                                                              |
| Description      | Sistem harus memetakan setiap pembayaran uang muka pembelian (purchase advance) ke akun aset khusus (Uang Muka Pembelian), terpisah dari akun beban maupun akun Accounts Payable reguler. |

---

## BR-1010 – Automatic Purchase Advance Offset Against Vendor Final Invoice

| Attribute        | Description                                                                                                                                                                       |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1010                                                                                                                                                                          |
| Requirement Name | Automatic Purchase Advance Offset Against Vendor Final Invoice                                                                                                                  |
| Priority         | Critical                                                                                                                                                                        |
| Description      | Sistem harus secara otomatis meng-offset saldo akun Uang Muka Pembelian terhadap invoice vendor final terkait Purchase Order yang sama, sehingga sisa kewajiban yang dibayarkan ke vendor sudah dikurangi uang muka. |

---

## BR-1011 – PO Category-to-Account Auto-Mapping (Inventory/Expense/Asset)

| Attribute        | Description                                                                                                                                                                                        |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1011                                                                                                                                                                                          |
| Requirement Name | PO Category-to-Account Auto-Mapping (Inventory/Expense/Asset)                                                                                                                                   |
| Priority         | Critical                                                                                                                                                                                         |
| Description      | Sistem harus menentukan akun tujuan posting secara otomatis berdasarkan kategori item pada baris Purchase Order: item Inventory diposting ke akun Persediaan, item Jasa/Beban diposting ke akun Beban sesuai kategori, dan item Modal diposting ke akun Aset/Clearing Aset. |

---

## BR-1012 – Expense Account Derivation for Non-Inventory/Service Purchase Order

| Attribute        | Description                                                                                                                                                                 |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1012                                                                                                                                                                   |
| Requirement Name | Expense Account Derivation for Non-Inventory/Service Purchase Order                                                                                                       |
| Priority         | High                                                                                                                                                                      |
| Description      | Sistem harus menurunkan akun beban secara otomatis untuk PO non-inventori (jasa, sewa, konsultasi, dsb.) berdasarkan kategori beban yang dikonfigurasi pada Item/Vendor Category, tanpa memerlukan pemilihan akun manual oleh pengguna. |

---

## BR-1013 – Capital Purchase Order to Fixed Asset Clearing Account Mapping

| Attribute        | Description                                                                                                                                                                                       |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-1013                                                                                                                                                                                          |
| Requirement Name | Capital Purchase Order to Fixed Asset Clearing Account Mapping                                                                                                                                   |
| Priority         | Critical                                                                                                                                                                                          |
| Description      | Sistem harus memposting PO belanja modal ke akun **Fixed Asset Clearing** (Aset Dalam Penyelesaian sementara) dan bukan langsung ke akun Aset Tetap final, sampai proses kapitalisasi aset diselesaikan oleh modul Fixed Assets. |

---

## BR-1014 – Purchase Price Variance Account Posting at Goods Receipt

| Attribute        | Description                                                                                                                                                                                     |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1014                                                                                                                                                                                        |
| Requirement Name | Purchase Price Variance Account Posting at Goods Receipt                                                                                                                                       |
| Priority         | High                                                                                                                                                                                            |
| Description      | Untuk item yang menggunakan metode standard costing, sistem harus memposting selisih antara harga PO aktual dan harga standar ke **akun Purchase Price Variance**, bukan ke akun Persediaan, pada saat Goods Receipt. |

---

## BR-1015 – Account Combination Completeness Validation Prior to PO/GR Posting

| Attribute        | Description                                                                                                                                                                            |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1015                                                                                                                                                                               |
| Requirement Name | Account Combination Completeness Validation Prior to PO/GR Posting                                                                                                                    |
| Priority         | Critical                                                                                                                                                                              |
| Description      | Sistem harus memvalidasi kelengkapan kombinasi akun (Account, Cost Center, Project, Segment lain yang wajib) sebelum Purchase Order atau Goods Receipt dapat diposting ke General Ledger, sehingga tidak ada transaksi procurement yang tersimpan tanpa kombinasi akun yang valid. |

---

# 9.6.3.43.4 Business Rules

| Business Rule ID | Description                                                                                                                                                          |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-PROCAI-001  | Setiap baris Purchase Order wajib memiliki kategori item (Inventory/Expense/Asset) yang menentukan akun tujuan posting sebelum PO dapat diterbitkan.               |
| BRule-PROCAI-002  | Saldo akun GR/IR harus nol pada saat Purchase Order telah sepenuhnya diterima dan seluruh invoice vendor terkait telah dicocokkan (matched); selisih residual hanya diperbolehkan akibat perbedaan waktu (timing) yang dikonfigurasi toleransinya. |
| BRule-PROCAI-003  | Jurnal Purchase Accrual harus dibalik (reversal) secara otomatis pada awal periode berikutnya apabila invoice vendor belum diterima, kecuali kebijakan akrual organisasi dikonfigurasi untuk tetap terbuka hingga invoice diterima. |
| BRule-PROCAI-004  | Akun Uang Muka Pembelian tidak boleh digunakan sebagai akun beban biasa dan hanya dapat berkurang melalui transaksi offset terhadap invoice vendor final atau pengembalian uang muka. |
| BRule-PROCAI-005  | Saldo akun GR/IR yang melewati ambang waktu aging yang dikonfigurasi wajib muncul pada laporan eksepsi dan ditinjau oleh Procurement Department serta Finance sebelum tutup buku periode. |
| BRule-PROCAI-006  | Purchase Price Variance yang timbul pada Goods Receipt untuk item standard costing wajib diposting ke akun variance yang ditetapkan dan tidak boleh mengubah nilai akun Persediaan. |
| BRule-PROCAI-007  | Purchase Order dan Goods Receipt tidak dapat diposting ke General Ledger apabila kombinasi akun yang dihasilkan tidak lengkap atau tidak valid menurut Account Combination Rule yang berlaku. |

---

# 9.6.3.43.5 Data Attributes

| Attribute                                    | Description                                                                                     |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| Purchase Accrual Account Reference             | Referensi akun kewajiban sementara atas barang/jasa diterima belum ditagih.                       |
| Accrual Reversal Method                        | Metode pembalikan jurnal accrual: Auto-Reverse Periode Berikutnya atau Held-Until-Invoiced.        |
| GR/IR Clearing Account Reference               | Referensi akun kliring antara Goods Receipt dan Invoice Receipt.                                   |
| GR/IR Aging Threshold                          | Ambang waktu (hari) sebelum saldo GR/IR yang belum ter-clear dimunculkan pada laporan eksepsi.     |
| Purchase Advance Account Reference             | Referensi akun aset untuk uang muka pembelian.                                                    |
| Purchase Advance Offset Rule                   | Aturan offset otomatis uang muka terhadap invoice vendor final per Purchase Order.                 |
| PO Item Category Account Mapping               | Pemetaan kategori item PO (Inventory/Expense/Asset) ke akun tujuan posting.                        |
| Expense Account Derivation Rule                | Aturan penurunan akun beban untuk PO non-inventori berdasarkan kategori beban.                     |
| Fixed Asset Clearing Account Reference         | Referensi akun kliring aset untuk PO belanja modal sebelum kapitalisasi final.                     |
| Purchase Price Variance Account Reference      | Referensi akun varians harga beli untuk item standard costing.                                    |
| Multi-Currency Revaluation Account Reference   | Referensi akun untuk penyesuaian selisih kurs atas saldo accrual/GR-IR dalam valuta asing.         |
| Account Combination Validation Reference       | Referensi aturan validasi kombinasi akun yang berlaku untuk transaksi procurement.                 |

---

# 9.6.3.43.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                                  |
| -------------- | ------------------- | ----------- | ------------------------------------------------ |
| BR-1006        | BO-003, BO-004       | SH-003      | Purchase Accrual Posting                          |
| BR-1007        | BO-003, BO-009       | SH-003      | GR/IR Clearing Management                         |
| BR-1008        | BO-004               | SH-030      | GR/IR Aging Review                                |
| BR-1009        | BO-004               | SH-003      | Purchase Advance Governance                       |
| BR-1010        | BO-003, BO-009       | SH-003      | Vendor Invoice Settlement                         |
| BR-1011        | BO-003, BO-009       | SH-030      | Purchase Category Account Mapping                 |
| BR-1012        | BO-009               | SH-030      | Service/Expense Purchase Posting                  |
| BR-1013        | BO-004               | SH-003      | Capital Expenditure Account Governance             |
| BR-1014        | BO-004               | SH-003      | Purchase Price Variance Review                    |
| BR-1015        | BO-003, BO-004       | SH-003      | Account Combination Governance                    |

---

# 9.6.3.43.7 Summary

Dokumen ini memperdalam integrasi Chart of Accounts dengan modul Procurement Management dari sisi Account, bukan dari sisi proses pengadaan itu sendiri. Dengan akun Purchase Accrual, GR/IR Clearing, dan Uang Muka Pembelian yang terdefinisi secara eksplisit, organisasi memperoleh jaminan bahwa kewajiban atas barang/jasa yang diterima, selisih waktu penerimaan-invoice, dan pembayaran di muka kepada vendor tercermin akurat pada Neraca sejak titik transaksi terjadi, bukan hanya pada saat invoice diproses oleh Accounts Payable.

Kapabilitas PO Category-to-Account Auto-Mapping, Fixed Asset Clearing Account, Purchase Price Variance Account, dan Account Combination Completeness Validation menjadikan Account sebagai kontrol preventif yang menjamin setiap transaksi procurement diposting ke akun yang tepat sejak awal, sekaligus mendukung kebutuhan **General Ledger**, **Accounts Payable**, **Fixed Assets**, **Inventory**, dan **Financial Statement Mapping** secara terintegrasi.
