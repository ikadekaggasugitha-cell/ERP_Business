# 9.6.3.44 Manufacturing Account Integration Requirements

---

# 9.6.3.44.1 Overview

Dokumen ini mengatur integrasi **Chart of Accounts (COA) / Account** dengan modul **Manufacturing Management** yang telah dibahas secara menyeluruh dari sisi proses bisnis pada dokumen *Manufacturing Management Requirements* (9.9, BR-716 s/d BR-725). Dokumen tersebut mengatur Bill of Material, Routing, Work Order, Production Planning, MRP, Shop Floor Control, hingga perhitungan biaya produksi sebagai proses operasional manufaktur. Dokumen ini **tidak mengulang** requirement tersebut, melainkan membahas secara spesifik bagaimana struktur akun General Ledger (GL) **direferensikan, diserap, dan diposting secara otomatis** oleh transaksi produksi, dilihat murni dari sudut pandang Account/COA.

Fokus dokumen ini mencakup empat area akun yang menjadi titik kritis integrasi keuangan produksi, yaitu:

1. **Akun Work In Progress (WIP)** — akun aset sementara yang menampung akumulasi biaya produksi selama Work Order berjalan.
2. **Akun biaya bahan baku, tenaga kerja langsung, dan overhead pabrik yang diserap (absorbed) ke WIP** — bagaimana setiap komponen biaya produksi dipetakan ke akun penyerapan masing-masing sebelum menjadi bagian dari WIP.
3. **Akun Varians Produksi (Production/Manufacturing Variance)** — akun yang menampung selisih antara biaya standar dan biaya aktual pada material, tenaga kerja, dan overhead.
4. **Akun Barang Jadi (Finished Goods)** — akun persediaan yang menerima transfer nilai dari WIP pada saat Work Order diselesaikan.

Tanpa aturan akun yang eksplisit pada area ini, organisasi berisiko membiarkan saldo WIP menumpuk tanpa rekonsiliasi, mencampur varians produksi dengan akun WIP atau Harga Pokok Penjualan, atau mentransfer nilai ke Barang Jadi sebelum Work Order benar-benar selesai, yang seluruhnya mendistorsi akurasi Neraca dan Laporan Laba Rugi.

---

# 9.6.3.44.2 Business Capability

Manufacturing Account Integration menyediakan kapabilitas berikut:

* Work In Progress (WIP) Account Structure per Work Order/Plant/Product Line.
* Direct Material Cost Absorption Posting ke akun WIP saat konsumsi material dari Inventory.
* Direct Labor Cost Absorption Posting ke akun WIP berdasarkan tarif tenaga kerja standar/aktual.
* Manufacturing Overhead Absorption Posting ke akun WIP berdasarkan tarif overhead yang dikonfigurasi.
* Production/Manufacturing Variance Account untuk selisih material (usage & price).
* Production/Manufacturing Variance Account untuk selisih tenaga kerja (efficiency & rate).
* Production/Manufacturing Variance Account untuk selisih overhead (volume & efficiency).
* Finished Goods Account Mapping pada saat transfer keluaran produksi dari WIP.
* WIP Account Reconciliation & Aging berdasarkan status Work Order (Open/In-Process/Closed).
* Account Combination Validation untuk transaksi manufaktur berbasis scope Plant/Cost Center.
* Costing Method Configuration (Standard/Actual/Average) yang menentukan perilaku posting WIP dan Variance.
* Cross-Module Consistency Check antara akun WIP, akun Persediaan (Inventory), dan akun Harga Pokok Penjualan.

---

# 9.6.3.44.3 Business Requirement Catalogue

## BR-1016 – Work In Progress (WIP) Account Structure and Auto-Posting per Work Order

| Attribute           | Description                                                                                                                                                                                                                                    |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-1016                                                                                                                                                                                                                                        |
| Requirement Name    | Work In Progress (WIP) Account Structure and Auto-Posting per Work Order                                                                                                                                                                       |
| Category            | Manufacturing Account Integration                                                                                                                                                                                                              |
| Priority            | Critical                                                                                                                                                                                                                                        |
| Business Objective  | BO-003, BO-004                                                                                                                                                                                                                                  |
| Stakeholder Owner   | SH-003 (CFO)                                                                                                                                                                                                                                    |
| Description         | Sistem harus menyediakan **akun Work In Progress (WIP)** sebagai akun aset sementara yang menampung akumulasi seluruh biaya produksi (material, tenaga kerja, overhead) selama Work Order berstatus terbuka, dengan saldo yang dapat ditelusuri per Work Order, Plant, dan Product Line. |
| Business Rationale  | Tanpa akun WIP yang eksplisit dan terstruktur, biaya produksi yang belum selesai akan tercampur dengan akun Persediaan Barang Jadi atau akun Beban, sehingga Neraca tidak mencerminkan nilai produksi yang sesungguhnya sedang berjalan.        |
| Business Value      | Menjamin visibilitas akurat atas nilai produksi yang sedang berjalan (work-in-process) dan menjadi dasar valuasi persediaan serta perhitungan Harga Pokok Produksi yang andal.                                                                 |
| Acceptance Criteria | Setiap Work Order memiliki saldo WIP yang terakumulasi otomatis dari seluruh transaksi biaya terkait dan dapat direkonsiliasi per Work Order pada setiap saat.                                                                                 |

---

## BR-1017 – Direct Material Cost Absorption to WIP Account

| Attribute        | Description                                                                                                                                                                            |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1017                                                                                                                                                                              |
| Requirement Name | Direct Material Cost Absorption to WIP Account                                                                                                                                       |
| Priority         | Critical                                                                                                                                                                              |
| Description      | Sistem harus memposting konsumsi bahan baku dari Inventory ke Work Order sebagai debit akun WIP dan kredit akun Persediaan Bahan Baku, menggunakan metode valuasi persediaan yang berlaku (Standard/Actual/Average). |

---

## BR-1018 – Direct Labor Cost Absorption to WIP Account

| Attribute        | Description                                                                                                                                                                       |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1018                                                                                                                                                                          |
| Requirement Name | Direct Labor Cost Absorption to WIP Account                                                                                                                                     |
| Priority         | Critical                                                                                                                                                                        |
| Description      | Sistem harus menyerap biaya tenaga kerja langsung ke akun WIP berdasarkan waktu operasi yang dilaporkan pada Shop Floor Control dan tarif tenaga kerja per Work Center yang dikonfigurasi. |

---

## BR-1019 – Manufacturing Overhead Absorption to WIP Account

| Attribute        | Description                                                                                                                                                                     |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1019                                                                                                                                                                        |
| Requirement Name | Manufacturing Overhead Absorption to WIP Account                                                                                                                              |
| Priority         | Critical                                                                                                                                                                       |
| Description      | Sistem harus menyerap biaya overhead pabrik ke akun WIP berdasarkan tarif overhead standar (predetermined overhead rate) yang dikonfigurasi per Work Center atau Plant, terpisah dari akun Overhead aktual. |

---

## BR-1020 – Production Variance Account for Material Usage and Price Variance

| Attribute        | Description                                                                                                                                                                                 |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1020                                                                                                                                                                                    |
| Requirement Name | Production Variance Account for Material Usage and Price Variance                                                                                                                        |
| Priority         | Critical                                                                                                                                                                                    |
| Description      | Sistem harus memposting selisih antara konsumsi material aktual dan standar (usage variance), serta selisih harga material aktual dan standar (price variance), ke akun Varians Produksi yang terpisah dari akun WIP. |

---

## BR-1021 – Production Variance Account for Labor Efficiency and Rate Variance

| Attribute        | Description                                                                                                                                                                     |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1021                                                                                                                                                                        |
| Requirement Name | Production Variance Account for Labor Efficiency and Rate Variance                                                                                                            |
| Priority         | High                                                                                                                                                                            |
| Description      | Sistem harus memposting selisih efisiensi waktu tenaga kerja aktual terhadap standar (efficiency variance) dan selisih tarif tenaga kerja aktual terhadap standar (rate variance) ke akun Varians Produksi tenaga kerja. |

---

## BR-1022 – Production Variance Account for Overhead Volume and Efficiency Variance

| Attribute        | Description                                                                                                                                                                       |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1022                                                                                                                                                                          |
| Requirement Name | Production Variance Account for Overhead Volume and Efficiency Variance                                                                                                        |
| Priority         | High                                                                                                                                                                              |
| Description      | Sistem harus memposting selisih overhead akibat perbedaan volume produksi aktual terhadap kapasitas normal (volume variance) dan selisih efisiensi penyerapan overhead (efficiency variance) ke akun Varians Produksi overhead. |

---

## BR-1023 – Finished Goods Account Mapping upon Work Order Completion

| Attribute        | Description                                                                                                                                                                     |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1023                                                                                                                                                                        |
| Requirement Name | Finished Goods Account Mapping upon Work Order Completion                                                                                                                     |
| Priority         | Critical                                                                                                                                                                        |
| Description      | Sistem harus memposting transfer nilai dari akun WIP ke akun Barang Jadi (Finished Goods) hanya pada saat Work Order dinyatakan selesai (completed), sebesar biaya produksi yang telah terakumulasi pada Work Order tersebut. |

---

## BR-1024 – WIP Account Reconciliation and Aging by Work Order Status

| Attribute        | Description                                                                                                                                                                    |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1024                                                                                                                                                                       |
| Requirement Name | WIP Account Reconciliation and Aging by Work Order Status                                                                                                                    |
| Priority         | High                                                                                                                                                                          |
| Description      | Sistem harus menyediakan rekonsiliasi saldo akun WIP terhadap total biaya Work Order yang masih terbuka (Open/In-Process) dan menghasilkan laporan aging atas Work Order yang tidak kunjung ditutup melewati ambang waktu yang dikonfigurasi. |

---

## BR-1025 – Account Combination Validation for Manufacturing Transactions per Plant/Cost Center Scope

| Attribute        | Description                                                                                                                                                                         |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-1025                                                                                                                                                                            |
| Requirement Name | Account Combination Validation for Manufacturing Transactions per Plant/Cost Center Scope                                                                                          |
| Priority         | Critical                                                                                                                                                                            |
| Description      | Sistem harus memvalidasi kelengkapan kombinasi akun (Account, Plant, Cost Center, Work Center) sebelum transaksi konsumsi material, pelaporan tenaga kerja, atau penutupan Work Order dapat diposting ke General Ledger. |

---

# 9.6.3.44.4 Business Rules

| Business Rule ID | Description                                                                                                                                                          |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-MFGAI-001   | Setiap Work Order wajib memiliki akun WIP yang ditetapkan berdasarkan Plant/Product Line sebelum biaya produksi apapun dapat diserap ke Work Order tersebut.        |
| BRule-MFGAI-002   | Konsumsi bahan baku, tenaga kerja, dan overhead ke Work Order wajib diposting ke akun WIP menggunakan tarif standar yang berlaku, bukan tarif aktual, apabila metode costing yang dikonfigurasi adalah Standard Costing. |
| BRule-MFGAI-003   | Selisih antara biaya standar dan biaya aktual pada material, tenaga kerja, dan overhead wajib diposting ke akun Varians Produksi yang sesuai dan tidak boleh dicampur ke akun WIP. |
| BRule-MFGAI-004   | Akun Barang Jadi hanya dapat menerima transfer nilai dari akun WIP pada saat Work Order berstatus Completed; Work Order berstatus Open atau In-Process tidak diperbolehkan mentransfer nilai ke Barang Jadi. |
| BRule-MFGAI-005   | Saldo akun WIP wajib direkonsiliasi terhadap total biaya Work Order terbuka sebelum tutup buku periode; selisih yang tidak dapat dijelaskan wajib ditindaklanjuti sebelum periode ditutup. |
| BRule-MFGAI-006   | Work Order hanya dapat ditutup (closed) apabila seluruh varians produksi terkait telah direkonsiliasi dan saldo WIP Work Order tersebut telah nol atau telah dipindahkan seluruhnya ke Barang Jadi. |
| BRule-MFGAI-007   | Transaksi manufaktur tidak dapat diposting ke General Ledger apabila kombinasi akun (Account, Plant, Cost Center, Work Center) tidak lengkap atau tidak valid menurut Account Combination Rule yang berlaku. |

---

# 9.6.3.44.5 Data Attributes

| Attribute                                  | Description                                                                                     |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| WIP Account Reference                         | Referensi akun aset sementara untuk akumulasi biaya produksi per Work Order/Plant/Product Line.   |
| Material Absorption Account Rule              | Aturan penyerapan biaya bahan baku dari Inventory ke akun WIP.                                    |
| Labor Absorption Account Rule                 | Aturan penyerapan biaya tenaga kerja langsung ke akun WIP berdasarkan tarif Work Center.           |
| Overhead Absorption Rate Reference            | Tarif overhead standar (predetermined rate) yang digunakan untuk penyerapan biaya overhead ke WIP. |
| Material Variance Account Reference           | Referensi akun varians material (usage & price) yang terpisah dari akun WIP.                      |
| Labor Variance Account Reference              | Referensi akun varians tenaga kerja (efficiency & rate).                                          |
| Overhead Variance Account Reference           | Referensi akun varians overhead (volume & efficiency).                                            |
| Finished Goods Account Reference              | Referensi akun persediaan Barang Jadi penerima transfer nilai dari WIP.                            |
| WIP Reconciliation Reference                  | Referensi hasil rekonsiliasi saldo WIP terhadap total biaya Work Order terbuka.                    |
| WIP Aging Threshold                           | Ambang waktu (hari) Work Order terbuka sebelum dimunculkan pada laporan aging WIP.                 |
| Costing Method Configuration                  | Metode costing yang berlaku: Standard, Actual, atau Average, menentukan perilaku posting WIP/Variance. |
| Account Combination Validation Reference      | Referensi aturan validasi kombinasi akun (Account, Plant, Cost Center, Work Center) untuk transaksi manufaktur. |

---

# 9.6.3.44.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                                  |
| -------------- | ------------------- | ----------- | ------------------------------------------------ |
| BR-1016        | BO-003, BO-004       | SH-003      | WIP Account Governance                            |
| BR-1017        | BO-003, BO-009       | SH-036      | Material Cost Absorption                          |
| BR-1018        | BO-003, BO-009       | SH-036      | Labor Cost Absorption                             |
| BR-1019        | BO-003, BO-009       | SH-036      | Overhead Cost Absorption                          |
| BR-1020        | BO-004               | SH-003      | Material Variance Review                          |
| BR-1021        | BO-004               | SH-003      | Labor Variance Review                             |
| BR-1022        | BO-004               | SH-003      | Overhead Variance Review                          |
| BR-1023        | BO-003, BO-004       | SH-036      | Finished Goods Transfer                           |
| BR-1024        | BO-004               | SH-003      | WIP Reconciliation & Period-End Review            |
| BR-1025        | BO-003, BO-004       | SH-036      | Account Combination Governance                    |

---

# 9.6.3.44.7 Summary

Dokumen ini memperdalam integrasi Chart of Accounts dengan modul Manufacturing Management dari sisi Account, bukan dari sisi proses produksi itu sendiri. Dengan akun WIP yang terstruktur per Work Order, penyerapan biaya material, tenaga kerja, dan overhead yang eksplisit, serta pemisahan akun Varians Produksi dari akun WIP, organisasi memperoleh jaminan bahwa nilai produksi yang sedang berjalan dan telah selesai tercermin akurat pada Neraca, dan bahwa selisih biaya standar terhadap aktual dapat dianalisis secara terpisah tanpa mendistorsi nilai persediaan.

Kapabilitas Finished Goods Account Mapping, WIP Account Reconciliation, dan Account Combination Validation menjadikan Account sebagai kontrol preventif yang menjamin setiap transaksi manufaktur diposting ke akun yang tepat sejak awal, sekaligus mendukung kebutuhan **General Ledger**, **Inventory**, **Cost Accounting**, dan **Financial Statement Mapping** secara terintegrasi.
