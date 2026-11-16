# 9.6.3.21 Cost Center Mapping Requirements

---

# 9.6.3.21.1 Overview

Cost Center Mapping mendefinisikan pemetaan antara struktur organisasi operasional (Company, Business Unit, Branch, Department, Warehouse, Project Site) dengan Cost Center pada Chart of Accounts. Cost Center Mapping menjadi jembatan antara struktur organisasi yang bersifat operasional dengan struktur akuntansi yang bersifat finansial, sehingga setiap transaksi biaya dapat dialokasikan dan dilaporkan secara akurat sesuai unit organisasi yang bertanggung jawab.

---

# 9.6.3.21.2 Business Capability

Cost Center Mapping menyediakan kapabilitas berikut:

* Organization-to-Cost Center Mapping.
* Multi-Dimensional Mapping (Department, Branch, Warehouse, Project Site).
* Default Cost Center Assignment per Scope Organisasi.
* Transaction-Level Cost Center Override.
* Percentage-Based Cost Center Allocation.
* Mapping Validation terhadap Struktur Organisasi Aktif.
* Effective Dating dan Riwayat Perubahan Mapping.
* Approval Workflow untuk Perubahan Mapping.
* Cost Center Mapping Reporting dan Reconciliation.
* Multi Organization & Multi Entity Support.
* Audit Trail atas seluruh perubahan mapping.

---

# 9.6.3.21.3 Business Requirement Catalogue

## BR-506 – Cost Center Mapping Definition

| Attribute           | Description                                                                                                                                                        |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-506                                                                                                                                                             |
| Requirement Name    | Cost Center Mapping Definition                                                                                                                                    |
| Category            | Cost Center Mapping                                                                                                                                               |
| Priority            | Critical                                                                                                                                                          |
| Business Objective  | BO-004                                                                                                                                                            |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                  |
| Description         | Sistem harus menyediakan Cost Center Mapping sebagai objek bisnis untuk memetakan unit organisasi operasional terhadap Cost Center pada Chart of Accounts.        |
| Business Rationale  | Struktur organisasi operasional dan struktur akuntansi Cost Center dikelola secara terpisah, sehingga diperlukan mekanisme pemetaan tunggal yang konsisten.        |
| Business Value      | Menjadi single source of truth bagi relasi organisasi dan Cost Center yang digunakan seluruh modul transaksi dan pelaporan.                                       |
| Acceptance Criteria | Administrator dapat membuat, mengubah, dan mengelola Cost Center Mapping sesuai kewenangan pada setiap level organisasi.                                          |

---

## BR-507 – Organization Unit to Cost Center Mapping

| Attribute        | Description                                                                                                                                    |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-507                                                                                                                                          |
| Requirement Name | Organization Unit to Cost Center Mapping                                                                                                       |
| Priority         | Critical                                                                                                                                        |
| Description      | Sistem harus memungkinkan setiap unit organisasi (Company, Business Unit, Branch, Department, Warehouse, Project Site) dipetakan ke satu atau lebih Cost Center tanpa batasan level struktur organisasi. |
| Business Value   | Memastikan setiap unit organisasi memiliki representasi biaya yang jelas dalam laporan keuangan.                                               |

---

## BR-508 – Multi-Dimensional Cost Center Mapping

| Attribute        | Description                                                                                                                                              |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-508                                                                                                                                                   |
| Requirement Name | Multi-Dimensional Cost Center Mapping                                                                                                                    |
| Priority         | High                                                                                                                                                     |
| Description      | Sistem harus mendukung kombinasi dimensi organisasi (Department, Branch, Warehouse, Project Site) dalam satu mapping Cost Center untuk kebutuhan alokasi biaya yang bersifat multi dimensi. |
| Acceptance Criteria | Pengguna dapat mendefinisikan mapping berbasis kombinasi lebih dari satu dimensi organisasi ke satu Cost Center.                                     |

---

## BR-509 – Default Cost Center Assignment

| Attribute        | Description                                                                                                                             |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-509                                                                                                                                   |
| Requirement Name | Default Cost Center Assignment                                                                                                          |
| Priority         | Critical                                                                                                                                 |
| Description      | Sistem harus menyediakan Default Cost Center pada setiap scope organisasi sehingga transaksi yang tidak menyertakan Cost Center secara eksplisit tetap dapat diposting menggunakan Cost Center default. |
| Business Value   | Mengurangi risiko transaksi tanpa Cost Center yang dapat mengganggu keakuratan laporan biaya.                                           |

---

## BR-510 – Transaction-Level Cost Center Override

| Attribute        | Description                                                                                                                                       |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-510                                                                                                                                            |
| Requirement Name | Transaction-Level Cost Center Override                                                                                                            |
| Priority         | High                                                                                                                                              |
| Description      | Sistem harus memungkinkan pengguna melakukan override Cost Center pada level transaksi dari hasil mapping default, sesuai kewenangan dan aturan validasi yang berlaku. |
| Acceptance Criteria | Override hanya dapat dilakukan oleh pengguna dengan kewenangan yang sesuai dan tercatat pada audit trail.                                    |

---

## BR-511 – Percentage-Based Cost Center Allocation

| Attribute        | Description                                                                                                                                  |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-511                                                                                                                                        |
| Requirement Name | Percentage-Based Cost Center Allocation                                                                                                      |
| Priority         | High                                                                                                                                          |
| Description      | Sistem harus mendukung alokasi biaya dari satu unit organisasi ke beberapa Cost Center berdasarkan persentase alokasi yang dapat dikonfigurasi. |
| Business Value   | Mendukung distribusi biaya yang lebih akurat pada unit organisasi yang menanggung beban bersama (shared cost).                               |

---

## BR-512 – Cost Center Mapping Validation

| Attribute        | Description                                                                                                                                    |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-512                                                                                                                                         |
| Requirement Name | Cost Center Mapping Validation                                                                                                                 |
| Priority         | Critical                                                                                                                                       |
| Description      | Sistem harus melakukan validasi Cost Center Mapping terhadap struktur organisasi aktif dan status Cost Center pada Chart of Accounts sebelum mapping dapat digunakan pada transaksi. |
| Acceptance Criteria | Sistem menolak mapping yang mengacu pada organisasi non-aktif atau Cost Center yang berstatus Inactive/Archived.                          |

---

## BR-513 – Cost Center Mapping Effective Dating and History

| Attribute        | Description                                                                                                                        |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-513                                                                                                                             |
| Requirement Name | Cost Center Mapping Effective Dating and History                                                                                   |
| Priority         | High                                                                                                                              |
| Description      | Sistem harus mencatat periode berlaku (effective date) setiap Cost Center Mapping serta menyimpan riwayat perubahan mapping secara historis. |
| Business Value   | Memastikan ketertelusuran perubahan mapping organisasi terhadap Cost Center sepanjang waktu untuk kebutuhan audit dan rekonsiliasi.  |

---

## BR-514 – Cost Center Mapping Approval Workflow

| Attribute        | Description                                                                                                                    |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-514                                                                                                                          |
| Requirement Name | Cost Center Mapping Approval Workflow                                                                                          |
| Priority         | High                                                                                                                            |
| Description      | Sistem harus menyediakan configurable approval workflow untuk setiap pembuatan dan perubahan Cost Center Mapping sebelum mapping berstatus Approved dan dapat digunakan pada transaksi. |
| Acceptance Criteria | Mapping baru atau perubahan mapping wajib melalui approval sesuai konfigurasi workflow sebelum berlaku efektif.             |

---

## BR-515 – Cost Center Mapping Reporting and Reconciliation

| Attribute        | Description                                                                                                                       |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-515                                                                                                                             |
| Requirement Name | Cost Center Mapping Reporting and Reconciliation                                                                                   |
| Priority         | High                                                                                                                               |
| Description      | Sistem harus menyediakan laporan Cost Center Mapping yang menampilkan relasi organisasi dan Cost Center secara lengkap, termasuk mapping yang belum lengkap atau tidak konsisten, untuk kebutuhan rekonsiliasi berkala. |
| Business Value   | Mendukung transparansi alokasi biaya lintas unit organisasi dan mempercepat proses rekonsiliasi laporan keuangan.                 |

---

# 9.6.3.21.4 Business Rules

| Business Rule ID | Description                                                                                                                                |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-CCM-001    | Setiap unit organisasi hanya dapat memiliki satu Default Cost Center Mapping yang aktif dalam satu periode berlaku.                        |
| BRule-CCM-002    | Total persentase alokasi Cost Center pada satu unit organisasi harus berjumlah tepat 100% (seratus persen).                                |
| BRule-CCM-003    | Cost Center Mapping hanya dapat mengacu pada Cost Center dengan status Active pada Chart of Accounts.                                      |
| BRule-CCM-004    | Perubahan Cost Center Mapping tidak boleh mempengaruhi transaksi historis yang telah diposting sebelum tanggal efektif perubahan.           |
| BRule-CCM-005    | Override Cost Center pada level transaksi hanya diperbolehkan apabila kewenangan pengguna dan aturan validasi transaksi terpenuhi.         |
| BRule-CCM-006    | Cost Center Mapping yang belum berstatus Approved tidak dapat digunakan sebagai referensi pada transaksi.                                  |
| BRule-CCM-007    | Cost Center Mapping mengikuti scope organisasi sesuai prinsip Scope-Based Access Control yang berlaku pada struktur organisasi terkait.     |

---

# 9.6.3.21.5 Data Attributes

| Attribute                   | Description                                                       |
| ---------------------------- | ------------------------------------------------------------------ |
| Cost Center Mapping ID      | Identitas unik sistem.                                            |
| Organization Unit Reference | Referensi unit organisasi (Company, Business Unit, Branch, Department, Warehouse, Project Site). |
| Cost Center Reference       | Referensi Cost Center pada Chart of Accounts.                     |
| Mapping Type                | Default, Override, Allocation.                                    |
| Allocation Percentage       | Persentase alokasi biaya ke Cost Center terkait.                  |
| Effective From / To         | Periode berlaku mapping.                                          |
| Version                     | Nomor versi mapping.                                              |
| Approval Status             | Draft, Pending Approval, Approved, Published, Archived.           |
| Status                      | Active, Suspended, Inactive, Archived.                            |
| Created By / Created Date   | Informasi pembuatan data.                                         |
| Updated By / Updated Date   | Informasi perubahan terakhir.                                     |

---

# 9.6.3.21.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                          |
| --------------- | ------------------- | ------------ | ---------------------------------------- |
| BR-506         | BO-004             | SH-003      | Cost Center Mapping Administration       |
| BR-507         | BO-004             | SH-003      | Organization Unit to Cost Center Setup   |
| BR-508         | BO-004             | SH-003      | Multi-Dimensional Cost Allocation Setup  |
| BR-509         | BO-004             | SH-003      | Default Cost Center Assignment           |
| BR-510         | BO-004             | SH-003      | Transaction Posting Cost Center Override |
| BR-511         | BO-004             | SH-003      | Cost Allocation Processing               |
| BR-512         | BO-004             | SH-003      | Cost Center Mapping Validation           |
| BR-513         | BO-004             | SH-003      | Cost Center Mapping History Tracking     |
| BR-514         | BO-004             | SH-003      | Cost Center Mapping Approval             |
| BR-515         | BO-004             | SH-003      | Cost Center Mapping Reporting            |

---

# 9.6.3.21.7 Summary

Cost Center Mapping menyediakan pemetaan tunggal dan konsisten antara struktur organisasi operasional dengan Cost Center pada Chart of Accounts. Cost Center Mapping menjadi fondasi bagi keakuratan alokasi biaya dan pelaporan keuangan lintas unit organisasi pada Financial Management Domain.
