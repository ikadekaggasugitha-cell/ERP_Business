# 9.6.3.46 Project Accounting Account Integration Requirements

---

# 9.6.3.46.1 Overview

Project Accounting Account Integration mendefinisikan bagaimana Chart of Accounts (Account) diintegrasikan, direferensikan, dan digunakan secara spesifik oleh modul Project untuk mendukung pencatatan transaksi proyek ke dalam General Ledger. Dokumen ini tidak membahas ulang requirement umum modul Project (perencanaan proyek, WBS, resource management, atau proses eksekusi proyek itu sendiri, yang telah dibahas pada bagian 9.10 BR-726 s.d. BR-735), melainkan berfokus pada sudut pandang Account/COA: bagaimana Account Work In Progress (WIP), Account pengakuan pendapatan proyek, Account biaya proyek per Work Breakdown Structure (WBS), dan Account billing proyek dikonfigurasi, dipetakan, serta direkonsiliasi dengan Project Costing.

Project Accounting Account Integration memastikan setiap transaksi proyek — mulai dari akumulasi biaya per WBS, pengakuan pendapatan dengan metode persentase penyelesaian (percentage of completion), hingga proses billing (unbilled dan billed receivable) — dapat dipetakan secara akurat ke kombinasi Account yang tepat sesuai jenis kontrak, WBS, dan organisasi, sehingga posisi keuangan proyek dapat dipantau, direkonsiliasi, dan dilaporkan secara konsisten lintas organisasi.

---

# 9.6.3.46.2 Business Capability

Project Accounting Account Integration menyediakan kapabilitas berikut:

* Project Work In Progress (WIP) Account Mapping.
* Project Revenue Recognition Account Mapping (Percentage of Completion).
* Project Cost Account Mapping by Work Breakdown Structure (WBS).
* Unbilled Receivable Account Mapping.
* Billed Receivable Account Mapping.
* WIP-to-Revenue/Cost of Goods Sold Transfer Account Mapping.
* Project Account Mapping by Contract Type (Fixed Price, Time & Material, Cost Plus).
* Project Account Reconciliation dengan Project Costing.
* Multi-Company dan Multi-Currency Project Account Derivation.
* Project Account Mapping Validation dan Exception Handling.
* Project Account Mapping Audit Trail.
* Project Account Mapping Reporting Integration ke Financial Reporting dan Project Profitability.

---

# 9.6.3.46.3 Business Requirement Catalogue

## BR-1036 – Project Work In Progress (WIP) Account Mapping

| Attribute           | Description                                                                                                                                                                                                                    |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-1036                                                                                                                                                                                                                        |
| Requirement Name    | Project Work In Progress (WIP) Account Mapping                                                                                                                                                                                |
| Category            | Project Accounting Account Integration                                                                                                                                                                                        |
| Priority            | Critical                                                                                                                                                                                                                       |
| Business Objective  | BO-004                                                                                                                                                                                                                         |
| Stakeholder Owner   | SH-003 (Chief Financial Officer), SH-014 (PMO)                                                                                                                                                                                |
| Description         | Sistem harus menyediakan Account Work In Progress (WIP) yang dapat dipetakan terhadap setiap Project untuk mengakumulasi biaya proyek yang belum diakui sebagai pendapatan atau belum ditagihkan.                            |
| Business Rationale  | Tanpa Account WIP yang terdefinisi secara eksplisit, biaya proyek yang sedang berjalan tidak dapat dipisahkan dari beban periode berjalan, sehingga menyulitkan pengukuran posisi keuangan proyek yang belum selesai.        |
| Business Value      | Menjadi single source of truth atas nilai pekerjaan dalam proses (WIP) proyek, mendukung pelaporan neraca yang akurat dan pengambilan keputusan berbasis data real-time atas eksposur biaya proyek yang berjalan.            |
| Acceptance Criteria | Administrator dapat mengonfigurasi mapping Account WIP per Project/jenis kontrak; sistem menampilkan validasi apabila Project berstatus Active belum memiliki mapping Account WIP yang valid.                                |

---

## BR-1037 – Project Revenue Recognition Account Mapping (Percentage of Completion)

| Attribute        | Description                                                                                                                                                       |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1037                                                                                                                                                            |
| Requirement Name  | Project Revenue Recognition Account Mapping (Percentage of Completion)                                                                                           |
| Priority          | Critical                                                                                                                                                           |
| Description       | Sistem harus menyediakan mapping Account pengakuan pendapatan proyek berdasarkan metode persentase penyelesaian (percentage of completion) sesuai progres fisik atau progres biaya proyek. |

---

## BR-1038 – Project Cost Account Mapping by Work Breakdown Structure (WBS)

| Attribute        | Description                                                                                                                              |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1038                                                                                                                                  |
| Requirement Name  | Project Cost Account Mapping by Work Breakdown Structure (WBS)                                                                          |
| Priority          | Critical                                                                                                                                  |
| Description       | Sistem harus mendukung pemetaan Account biaya proyek pada setiap elemen Work Breakdown Structure (WBS) untuk kebutuhan Project Costing yang terperinci. |

---

## BR-1039 – Unbilled Receivable Account Mapping

| Attribute        | Description                                                                                                                              |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1039                                                                                                                                  |
| Requirement Name  | Unbilled Receivable Account Mapping                                                                                                      |
| Priority          | High                                                                                                                                       |
| Description       | Sistem harus menyediakan Account piutang belum ditagih (Unbilled Receivable Account) untuk mencatat pendapatan proyek yang telah diakui namun belum diterbitkan invoice ke pelanggan. |

---

## BR-1040 – Billed Receivable Account Mapping

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1040                                                                                                                            |
| Requirement Name  | Billed Receivable Account Mapping                                                                                                 |
| Priority          | High                                                                                                                               |
| Description       | Sistem harus menyediakan Account piutang tertagih (Billed Receivable Account) untuk mencatat reklasifikasi dari Unbilled Receivable pada saat invoice proyek diterbitkan. |

---

## BR-1041 – WIP-to-Revenue/Cost of Goods Sold Transfer Account Mapping

| Attribute        | Description                                                                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1041                                                                                                                                          |
| Requirement Name  | WIP-to-Revenue/Cost of Goods Sold Transfer Account Mapping                                                                                      |
| Priority          | High                                                                                                                                              |
| Description       | Sistem harus menyediakan mapping Account untuk transfer nilai dari Account WIP ke Account pendapatan proyek dan Account Cost of Goods Sold (COGS) sesuai progres pengakuan pendapatan. |

---

## BR-1042 – Project Account Reconciliation with Project Costing

| Attribute        | Description                                                                                                                                      |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1042                                                                                                                                           |
| Requirement Name  | Project Account Reconciliation with Project Costing                                                                                              |
| Priority          | Critical                                                                                                                                          |
| Description       | Sistem harus menyediakan mekanisme rekonsiliasi otomatis antara saldo Account WIP, Account biaya, dan Account pendapatan proyek pada General Ledger dengan data biaya dan progres pada Project Costing. |

---

## BR-1043 – Project Account Mapping by Contract Type

| Attribute        | Description                                                                                                                                    |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1043                                                                                                                                        |
| Requirement Name  | Project Account Mapping by Contract Type                                                                                                      |
| Priority          | Medium                                                                                                                                          |
| Description       | Sistem harus mendukung variasi mapping Account proyek berdasarkan jenis kontrak (Fixed Price, Time & Material, Cost Plus) sesuai kebijakan pengakuan pendapatan masing-masing jenis kontrak. |

---

## BR-1044 – Project Account Mapping Validation and Exception Handling

| Attribute        | Description                                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1044                                                                                                                                           |
| Requirement Name  | Project Account Mapping Validation and Exception Handling                                                                                        |
| Priority          | High                                                                                                                                               |
| Description       | Sistem harus melakukan validasi otomatis terhadap kelengkapan dan validitas mapping Account proyek, serta menampilkan exception ke Finance dan PMO apabila mapping tidak ditemukan atau tidak konsisten. |

---

## BR-1045 – Project Account Mapping Audit Trail and Change History

| Attribute        | Description                                                                                                                        |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-1045                                                                                                                            |
| Requirement Name  | Project Account Mapping Audit Trail and Change History                                                                            |
| Priority          | Medium                                                                                                                             |
| Description       | Sistem harus mencatat riwayat perubahan mapping Account proyek, termasuk perubahan Account WIP, Account pendapatan, dan Account billing. |

---

# 9.6.3.46.4 Business Rules

| Business Rule ID   | Description                                                                                                                                                                     |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| BRule-PRJAI-001      | Setiap Project berstatus Active harus memiliki mapping Account WIP, Account pendapatan, dan Account biaya per WBS yang valid sebelum transaksi proyek dapat diposting.        |
| BRule-PRJAI-002      | Pengakuan pendapatan proyek dengan metode percentage of completion harus mereklasifikasi nilai dari Account WIP ke Account pendapatan sesuai persentase progres yang divalidasi. |
| BRule-PRJAI-003      | Account Unbilled Receivable harus direklasifikasi ke Account Billed Receivable hanya pada saat invoice proyek diterbitkan dan disetujui.                                       |
| BRule-PRJAI-004      | Saldo Account WIP dan Account biaya proyek pada General Ledger harus direkonsiliasi dengan data Project Costing secara periodik; selisih di atas ambang batas wajib diinvestigasi. |
| BRule-PRJAI-005      | Mapping Account proyek harus konsisten dengan jenis kontrak (Fixed Price, Time & Material, Cost Plus) yang berlaku pada Project terkait.                                       |
| BRule-PRJAI-006      | Sistem harus mencegah posting transaksi biaya atau billing proyek apabila mapping Account WBS terkait tidak ditemukan atau berstatus Inactive.                                 |
| BRule-PRJAI-007      | Perubahan mapping Account proyek yang berdampak pada transaksi yang telah diposting harus melalui approval workflow dan tidak boleh mengubah data historis.                    |

---

# 9.6.3.46.5 Data Attributes

| Attribute                        | Description                                                                    |
| ---------------------------------- | --------------------------------------------------------------------------------- |
| Project Account Mapping ID         | Identitas unik sistem untuk konfigurasi mapping Account proyek.                 |
| Project ID / Project Code          | Referensi ke Project pada modul Project.                                       |
| WBS Element                        | Referensi elemen Work Breakdown Structure pada Project.                        |
| Contract Type                      | Fixed Price, Time & Material, Cost Plus.                                       |
| Organization/Company/Business Unit | Cakupan organisasi tempat mapping berlaku.                                     |
| WIP Account ID                     | Referensi Account Work In Progress pada Chart of Accounts.                     |
| Revenue Recognition Account ID     | Referensi Account pengakuan pendapatan proyek.                                 |
| Cost Account ID                    | Referensi Account biaya proyek per WBS.                                        |
| Unbilled Receivable Account ID     | Referensi Account piutang belum ditagih.                                       |
| Billed Receivable Account ID       | Referensi Account piutang tertagih.                                            |
| Percentage of Completion           | Persentase penyelesaian proyek yang digunakan untuk pengakuan pendapatan.       |
| Reconciliation Status              | Reconciled, Unreconciled, Variance Identified.                                 |
| Effective From / To                | Periode berlaku mapping.                                                       |
| Approval Status                    | Draft, Pending Approval, Approved, Published, Archived.                        |
| Status                             | Active, Suspended, Inactive, Archived.                                         |
| Created By / Created Date          | Informasi pembuatan data.                                                      |
| Updated By / Updated Date          | Informasi perubahan terakhir.                                                  |

---

# 9.6.3.46.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder     | Future Process                                    |
| --------------- | ------------------- | ---------------- | ------------------------------------------------------ |
| BR-1036         | BO-004              | SH-003, SH-014   | Project WIP Accounting                                 |
| BR-1037         | BO-004              | SH-003, SH-014   | Percentage of Completion Revenue Recognition            |
| BR-1038         | BO-003              | SH-014           | Project Cost Accounting by WBS                          |
| BR-1039         | BO-004              | SH-003           | Unbilled Receivable Management                          |
| BR-1040         | BO-004              | SH-003           | Project Billing and Invoicing                           |
| BR-1041         | BO-003              | SH-003, SH-014   | WIP-to-Revenue/COGS Transfer Processing                 |
| BR-1042         | BO-008              | SH-003, SH-014   | Project Costing Reconciliation                          |
| BR-1043         | BO-004              | SH-014           | Contract-Based Project Accounting Configuration         |
| BR-1044         | BO-004              | SH-003, SH-014   | Project Account Mapping Exception Management            |
| BR-1045         | BO-004              | SH-003, SH-014   | Project Account Mapping Audit and Reporting             |

---

# 9.6.3.46.7 Summary

Project Accounting Account Integration menjembatani transaksi operasional modul Project dengan Chart of Accounts secara terstruktur dan dapat direkonsiliasi. Dengan mapping Account WIP, Account pengakuan pendapatan berbasis percentage of completion, Account biaya per WBS, serta Account billing (unbilled dan billed receivable) yang terhubung dengan Project Costing, organisasi memperoleh single source of truth atas posisi keuangan proyek yang mendukung pelaporan akurat dan pengambilan keputusan berbasis data secara real-time bagi CFO dan PMO.
