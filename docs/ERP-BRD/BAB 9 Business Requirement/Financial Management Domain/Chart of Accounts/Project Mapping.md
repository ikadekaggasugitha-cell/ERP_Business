# 9.6.3.22 Project Mapping Requirements

---

# 9.6.3.22.1 Overview

Project Mapping mendefinisikan pemetaan antara Project pada modul Project Management dengan dimensi akuntansi pada Chart of Accounts, seperti Account dan Cost Center, untuk mendukung Project Costing dan Project Accounting. Project Mapping memastikan setiap transaksi yang timbul dari aktivitas project dapat ditelusuri, dialokasikan, dan dilaporkan secara akurat ke dimensi keuangan yang tepat, sehingga biaya, pendapatan, dan kinerja keuangan project dapat dianalisis secara konsisten lintas organisasi.

---

# 9.6.3.22.2 Business Capability

Project Mapping menyediakan kapabilitas berikut:

* Project as Accounting Dimension.
* Project-to-Account Mapping.
* Project-to-Cost Center Mapping.
* Project Work Breakdown Structure (WBS) to Account Mapping.
* Project Budget Line Mapping.
* Multi-Entity dan Multi-Cost Center Allocation.
* Project Revenue Recognition Mapping.
* Project Status Synchronization dengan Status Mapping.
* Effective Dating.
* Versioning.
* Approval Workflow.
* Audit Trail.

---

# 9.6.3.22.3 Business Requirement Catalogue

## BR-516 – Project Mapping Definition

| Attribute           | Description                                                                                                                                                                    |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-516                                                                                                                                                                          |
| Requirement Name    | Project Mapping Definition                                                                                                                                                     |
| Category            | Project Mapping                                                                                                                                                                |
| Priority            | Critical                                                                                                                                                                       |
| Business Objective  | BO-004                                                                                                                                                                          |
| Stakeholder Owner   | SH-003 (Chief Financial Officer), SH-014 (PMO)                                                                                                                                 |
| Description         | Sistem harus menyediakan Project Mapping sebagai objek konfigurasi yang memetakan setiap Project pada modul Project Management terhadap dimensi akuntansi (Account, Cost Center) pada Chart of Accounts. |
| Business Rationale  | Tanpa pemetaan yang eksplisit antara Project dan dimensi akuntansi, transaksi keuangan yang timbul dari aktivitas project tidak dapat ditelusuri, dialokasikan, maupun dilaporkan secara akurat, sehingga menyulitkan analisis biaya dan kinerja project. |
| Business Value      | Menjadi single source of truth bagi hubungan antara data project dan data akuntansi, sehingga Project Costing dan Project Accounting dapat dijalankan secara konsisten.        |
| Acceptance Criteria | Administrator dapat membuat, mengubah, dan menonaktifkan Project Mapping sesuai kewenangan; sistem menampilkan validasi apabila Project belum memiliki mapping yang valid.     |

---

## BR-517 – Project-to-Cost Center Mapping

| Attribute        | Description                                                                                                                          |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-517                                                                                                                                |
| Requirement Name  | Project-to-Cost Center Mapping                                                                                                        |
| Priority          | Critical                                                                                                                              |
| Description       | Sistem harus menyediakan mekanisme pemetaan antara Project dengan satu atau lebih Cost Center untuk keperluan alokasi biaya project. |
| Business Value    | Memastikan biaya project teralokasi ke Cost Center yang tepat sesuai struktur organisasi.                                            |

---

## BR-518 – Project Budget Line Mapping

| Attribute        | Description                                                                                                                              |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-518                                                                                                                                    |
| Requirement Name  | Project Budget Line Mapping                                                                                                              |
| Priority          | High                                                                                                                                      |
| Description       | Sistem harus memetakan setiap baris anggaran (budget line) pada Project ke Account dan Cost Center terkait untuk kontrol anggaran project. |
| Business Value    | Memungkinkan kontrol dan monitoring realisasi anggaran project terhadap dimensi akuntansi yang relevan.                                   |

---

## BR-519 – Project WBS to Account Mapping

| Attribute        | Description                                                                                                                                       |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-519                                                                                                                                             |
| Requirement Name  | Project WBS to Account Mapping                                                                                                                    |
| Priority          | High                                                                                                                                               |
| Description       | Sistem harus mendukung pemetaan Work Breakdown Structure (WBS) pada Project terhadap Account pada Chart of Accounts untuk kebutuhan Project Costing yang terperinci. |
| Acceptance Criteria | Setiap elemen WBS dapat dipetakan ke Account tanpa mengubah struktur project pada modul Project Management.                                     |

---

## BR-520 – Multi-Entity Project Allocation Mapping

| Attribute        | Description                                                                                                                                       |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-520                                                                                                                                             |
| Requirement Name  | Multi-Entity Project Allocation Mapping                                                                                                           |
| Priority          | High                                                                                                                                               |
| Description       | Sistem harus mendukung alokasi satu Project ke lebih dari satu Company/Business Unit berdasarkan persentase alokasi yang dikonfigurasi.           |
| Business Value    | Mendukung project lintas entitas dalam struktur Holding tanpa duplikasi data project.                                                             |

---

## BR-521 – Project Revenue Recognition Mapping

| Attribute        | Description                                                                                                                              |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-521                                                                                                                                    |
| Requirement Name  | Project Revenue Recognition Mapping                                                                                                      |
| Priority          | High                                                                                                                                      |
| Description       | Sistem harus memetakan Project terhadap Account pengakuan pendapatan (revenue recognition account) sesuai metode akuntansi project yang berlaku. |
| Business Value    | Memastikan pengakuan pendapatan project sesuai standar akuntansi dan kebijakan perusahaan.                                               |

---

## BR-522 – Project Status-Based Mapping Control

| Attribute        | Description                                                                                                                                |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-522                                                                                                                                     |
| Requirement Name  | Project Status-Based Mapping Control                                                                                                      |
| Priority          | Medium                                                                                                                                     |
| Description       | Sistem harus menyesuaikan ketersediaan Project Mapping berdasarkan status Project (Planning, Active, On Hold, Closed) pada modul Project Management. |
| Acceptance Criteria | Project Mapping pada Project berstatus Closed tidak dapat digunakan untuk transaksi baru.                                                |

---

## BR-523 – Project Mapping Validation

| Attribute        | Description                                                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-523                                                                                                                          |
| Requirement Name  | Project Mapping Validation                                                                                                     |
| Priority          | Critical                                                                                                                        |
| Description       | Sistem harus melakukan validasi otomatis terhadap konsistensi Project Mapping, termasuk keberadaan Account, Cost Center, dan Scope organisasi yang valid. |
| Business Value    | Mencegah kesalahan pencatatan transaksi akibat mapping yang tidak valid atau tidak konsisten.                                   |

---

## BR-524 – Project Mapping Change History

| Attribute        | Description                                                                                                            |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-524                                                                                                                  |
| Requirement Name  | Project Mapping Change History                                                                                         |
| Priority          | Medium                                                                                                                  |
| Description       | Sistem harus mencatat riwayat perubahan Project Mapping, termasuk perubahan alokasi, Account, dan Cost Center terkait. |
| Business Value    | Mendukung audit trail dan investigasi historis atas perubahan pemetaan project.                                        |

---

## BR-525 – Project Mapping Reporting Integration

| Attribute        | Description                                                                                                                             |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-525                                                                                                                                   |
| Requirement Name  | Project Mapping Reporting Integration                                                                                                   |
| Priority          | High                                                                                                                                     |
| Description       | Sistem harus menyediakan integrasi Project Mapping ke laporan Project Costing, Project Profitability, dan laporan keuangan konsolidasi. |
| Business Value    | Memberikan visibilitas kinerja keuangan project secara real-time bagi manajemen dan PMO.                                                |

---

# 9.6.3.22.4 Business Rules

| Business Rule ID | Description                                                                                                                                                     |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-PM-001      | Setiap Project berstatus Active harus memiliki minimal satu Project Mapping yang valid sebelum transaksi akuntansi dapat dicatat.                              |
| BRule-PM-002      | Project Mapping harus merujuk pada kombinasi Account dan Cost Center yang valid dan berstatus Active pada Chart of Accounts.                                   |
| BRule-PM-003      | Perubahan Project Mapping yang memengaruhi transaksi yang telah diposting harus melalui approval workflow dan tidak boleh mengubah data historis.              |
| BRule-PM-004      | Satu Project dapat memiliki lebih dari satu Project Mapping untuk alokasi biaya ke beberapa Cost Center atau Business Unit, dengan total persentase alokasi 100%. |
| BRule-PM-005      | Project Mapping berstatus Inactive tidak dapat digunakan untuk transaksi baru namun tetap dipertahankan untuk kebutuhan audit dan pelaporan historis.          |
| BRule-PM-006      | Sistem harus mencegah penghapusan Project Mapping yang telah digunakan pada transaksi yang telah diposting; mapping hanya dapat dinonaktifkan (soft delete).   |
| BRule-PM-007      | Project Mapping harus konsisten dengan Scope organisasi (Company/Business Unit) tempat Project tersebut terdaftar.                                             |

---

# 9.6.3.22.5 Data Attributes

| Attribute                  | Description                                              |
| --------------------------- | --------------------------------------------------------- |
| Project Mapping ID         | Identitas unik sistem.                                   |
| Project ID / Project Code  | Referensi ke Project pada modul Project Management.       |
| Account ID / Account Code  | Referensi ke Account pada Chart of Accounts.               |
| Cost Center ID             | Referensi ke Cost Center terkait alokasi biaya project.    |
| Allocation Percentage      | Persentase alokasi biaya/pendapatan project.               |
| Effective From / To        | Periode berlaku pemetaan.                                  |
| Version                    | Nomor versi.                                               |
| Approval Status            | Draft, Pending Approval, Approved, Published, Archived.   |
| Status                     | Active, Suspended, Inactive, Archived.                     |
| Created By / Created Date  | Informasi pembuatan data.                                  |
| Updated By / Updated Date  | Informasi perubahan terakhir.                              |

---

# 9.6.3.22.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder    | Future Process                       |
| --------------- | ------------------- | -------------- | -------------------------------------- |
| BR-516          | BO-004              | SH-003, SH-014 | Project Mapping Administration        |
| BR-517          | BO-004              | SH-014         | Project Cost Center Allocation        |
| BR-518          | BO-004              | SH-003         | Project Budget Control                |
| BR-519          | BO-004              | SH-014         | Project WBS Costing                   |
| BR-520          | BO-004              | SH-003         | Multi-Entity Project Allocation       |
| BR-521          | BO-004              | SH-003         | Project Revenue Recognition           |
| BR-522          | BO-004              | SH-014         | Project Lifecycle Mapping Control     |
| BR-523          | BO-004              | SH-003         | Project Mapping Validation Process    |
| BR-524          | BO-004              | SH-014         | Project Mapping Audit Trail           |
| BR-525          | BO-004              | SH-003, SH-014 | Project Financial Reporting           |

---

# 9.6.3.22.7 Summary

Project Mapping menyediakan fondasi pemetaan antara Project pada modul Project Management dengan dimensi akuntansi pada Chart of Accounts. Project Mapping menjadi single source of truth yang menghubungkan aktivitas project dengan pencatatan keuangan, sehingga Project Costing dan Project Accounting dapat dijalankan secara akurat, konsisten, dan dapat ditelusuri lintas organisasi.
