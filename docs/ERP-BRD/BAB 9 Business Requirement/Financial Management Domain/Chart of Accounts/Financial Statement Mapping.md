# 9.6.3.25 Financial Statement Mapping Requirements

---

# 9.6.3.25.1 Overview

Financial Statement Mapping mendefinisikan pemetaan setiap akun pada Chart of Accounts ke baris atau kelompok baris pada format laporan keuangan standar, yaitu Neraca (Balance Sheet), Laba Rugi (Income Statement), dan Arus Kas (Cash Flow Statement). Pemetaan ini menjadi jembatan antara struktur akun operasional dengan format pelaporan yang dipersyaratkan oleh standar akuntansi, regulator, maupun kebutuhan manajemen internal, sehingga penyusunan laporan keuangan dapat dilakukan secara otomatis, konsisten, dan akurat lintas entitas organisasi.

---

# 9.6.3.25.2 Business Capability

Financial Statement Mapping menyediakan kapabilitas berikut:

* Mapping Akun ke Baris Laporan Keuangan (Financial Statement Line Item).
* Multi Financial Statement Template (Neraca, Laba Rugi, Arus Kas).
* Multi Accounting Standard Mapping (PSAK, IFRS, Tax Basis, Management Reporting).
* Hierarchical Grouping dan Subtotal/Total Line Mapping.
* Cash Flow Mapping dengan Metode Langsung dan Tidak Langsung.
* Mapping Validation dan Completeness Check.
* Scope-Based Mapping Override per Organization Unit.
* Effective Dating dan Versioning Mapping.
* Approval Workflow untuk Perubahan Mapping.
* Automated Financial Statement Generation berbasis Mapping.
* Mapping Impact Analysis saat terjadi perubahan struktur akun.
* Audit Trail atas seluruh perubahan mapping.

---

# 9.6.3.25.3 Business Requirement Catalogue

## BR-546 – Financial Statement Mapping Definition

| Attribute           | Description                                                                                                                                          |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-546                                                                                                                                                 |
| Requirement Name    | Financial Statement Mapping Definition                                                                                                                |
| Category            | Financial Statement Mapping                                                                                                                           |
| Priority            | Critical                                                                                                                                               |
| Business Objective  | BO-022                                                                                                                                                 |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                       |
| Description         | Sistem harus menyediakan Financial Statement Mapping sebagai objek bisnis yang memetakan setiap akun pada Chart of Accounts ke baris atau kelompok baris pada format laporan keuangan standar. |
| Business Rationale  | Penyusunan laporan keuangan memerlukan pemetaan akun yang jelas dan terstruktur agar hasil laporan dapat diandalkan dan dapat ditelusuri kembali ke akun sumber. |
| Business Value      | Menyediakan fondasi pemetaan yang menjamin akurasi dan konsistensi penyusunan laporan keuangan.                                                       |
| Acceptance Criteria | Administrator dapat membuat, mengubah, dan mengelola Financial Statement Mapping sesuai kewenangan dan setiap akun dapat dipetakan ke baris laporan keuangan yang sesuai. |

---

## BR-547 – Multi Financial Statement Template Support

| Requirement ID | Requirement Name                          | Priority | Description                                                                                                                             |
| -------------- | ------------------------------------------ | -------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| BR-547         | Multi Financial Statement Template Support | Critical | Sistem harus mendukung mapping akun ke berbagai format laporan keuangan standar, meliputi Neraca, Laba Rugi, dan Arus Kas, secara terpisah maupun terintegrasi. |

---

## BR-548 – Account to Line Item Mapping

| Requirement ID | Requirement Name             | Priority | Description                                                                                                                          |
| -------------- | ------------------------------ | -------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| BR-548         | Account to Line Item Mapping | Critical | Sistem harus menyediakan mekanisme pemetaan satu atau beberapa akun ke satu baris laporan keuangan (Financial Statement Line Item) tertentu. |

---

## BR-549 – Multi Accounting Standard Mapping

| Requirement ID | Requirement Name                | Priority | Description                                                                                                                                             |
| -------------- | ---------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-549         | Multi Accounting Standard Mapping | High     | Sistem harus mendukung mapping akun yang berbeda untuk setiap standar akuntansi yang berlaku, seperti PSAK, IFRS, Tax Basis, dan Management Reporting, pada satu akun yang sama. |

---

## BR-550 – Mapping Hierarchy and Subtotal Structure

| Requirement ID | Requirement Name                        | Priority | Description                                                                                                                             |
| -------------- | ------------------------------------------ | -------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| BR-550         | Mapping Hierarchy and Subtotal Structure | High     | Sistem harus mendukung penyusunan hierarki baris laporan keuangan berjenjang, termasuk baris subtotal dan total, yang dihitung otomatis berdasarkan mapping akun. |

---

## BR-551 – Cash Flow Statement Mapping Method

| Requirement ID | Requirement Name                | Priority | Description                                                                                                                                         |
| -------------- | ---------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| BR-551         | Cash Flow Statement Mapping Method | High     | Sistem harus mendukung mapping Arus Kas dengan metode langsung (direct method) maupun tidak langsung (indirect method) sesuai kebijakan akuntansi organisasi. |

---

## BR-552 – Mapping Completeness Validation

| Requirement ID | Requirement Name              | Priority | Description                                                                                                                                       | Business Value                                                                                   |
| -------------- | -------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| BR-552         | Mapping Completeness Validation | Critical | Sistem harus melakukan validasi untuk memastikan seluruh akun aktif pada Chart of Accounts telah dipetakan ke baris laporan keuangan sebelum laporan disusun. | Mencegah akun yang belum dipetakan mengakibatkan laporan keuangan tidak lengkap atau tidak seimbang. |

---

## BR-553 – Effective Dating and Versioning of Mapping

| Requirement ID | Requirement Name                        | Priority | Description                                                                                                                          |
| -------------- | ------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| BR-553         | Effective Dating and Versioning of Mapping | High     | Sistem harus mendukung periode berlaku dan versi mapping agar perubahan format laporan keuangan tidak memengaruhi laporan periode sebelumnya. |

---

## BR-554 – Scope-Based Mapping Override

| Requirement ID | Requirement Name          | Priority | Description                                                                                                                                                  |
| -------------- | ---------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-554         | Scope-Based Mapping Override | Medium   | Sistem harus memungkinkan mapping berbeda pada level organisasi tertentu (Company, Business Unit) apabila terdapat kebutuhan pelaporan yang spesifik terhadap regulasi lokal. |

---

## BR-555 – Automated Financial Statement Generation

| Requirement ID | Requirement Name                     | Priority | Description                                                                                                                     | Acceptance Criteria                                                                                          |
| -------------- | ---------------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| BR-555         | Automated Financial Statement Generation | Critical | Sistem harus mampu menghasilkan Neraca, Laba Rugi, dan Arus Kas secara otomatis berdasarkan mapping akun dan saldo transaksi pada periode berjalan. | Laporan keuangan dapat dihasilkan secara otomatis dan saldo pada laporan sesuai dengan saldo akun sumber. |

---

# 9.6.3.25.4 Business Rules

| Business Rule ID | Description                                                                                                                     |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| BRule-FSM-001     | Setiap akun aktif pada Chart of Accounts harus dipetakan ke minimal satu baris laporan keuangan sebelum laporan dapat disusun.  |
| BRule-FSM-002     | Satu akun dapat dipetakan ke lebih dari satu baris laporan keuangan apabila diperlukan pemisahan pelaporan antar standar akuntansi. |
| BRule-FSM-003     | Perubahan mapping yang telah diterapkan pada periode berjalan harus melalui approval workflow sebelum dipublikasikan.          |
| BRule-FSM-004     | Baris subtotal dan total pada laporan keuangan harus dihitung otomatis berdasarkan agregasi baris mapping di bawahnya.          |
| BRule-FSM-005     | Mapping yang telah kedaluwarsa (expired) tidak dapat digunakan untuk penyusunan laporan pada periode aktif.                     |
| BRule-FSM-006     | Setiap perubahan mapping harus tercatat pada audit trail beserta alasan perubahan dan pihak yang melakukan perubahan.           |
| BRule-FSM-007     | Mapping Arus Kas harus konsisten dengan klasifikasi aktivitas Operasi, Investasi, dan Pendanaan sesuai standar akuntansi yang berlaku. |

---

# 9.6.3.25.5 Data Attributes

| Attribute                  | Description                                                        |
| ---------------------------- | --------------------------------------------------------------------- |
| Mapping ID                  | Identitas unik sistem untuk setiap mapping.                        |
| Account Reference            | Referensi akun pada Chart of Accounts yang dipetakan.               |
| Financial Statement Type    | Jenis laporan keuangan (Neraca, Laba Rugi, Arus Kas).              |
| Line Item Code / Name       | Kode dan nama baris pada format laporan keuangan.                  |
| Accounting Standard         | Standar akuntansi yang berlaku (PSAK, IFRS, Tax Basis, Management). |
| Hierarchy Level             | Tingkat hierarki baris laporan (detail, subtotal, total).          |
| Cash Flow Method            | Metode Arus Kas (Direct, Indirect), apabila relevan.                |
| Organization Scope           | Ruang lingkup organisasi berlakunya mapping.                        |
| Effective From / To         | Periode berlaku mapping.                                             |
| Version                     | Nomor versi mapping.                                                 |
| Approval Status             | Draft, Pending Approval, Approved, Published, Archived.            |
| Status                      | Active, Suspended, Inactive, Archived.                              |
| Created By / Created Date   | Informasi pembuatan data.                                            |
| Updated By / Updated Date   | Informasi perubahan terakhir.                                        |

---

# 9.6.3.25.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                             |
| -------------- | ------------------- | ----------- | --------------------------------------------- |
| BR-546         | BO-022              | SH-003      | Financial Statement Mapping Administration    |
| BR-547         | BO-022              | SH-003      | Financial Statement Template Configuration    |
| BR-548         | BO-022              | SH-028      | Account to Line Item Mapping Maintenance      |
| BR-549         | BO-022              | SH-003      | Multi Standard Financial Reporting            |
| BR-550         | BO-022              | SH-028      | Financial Statement Structuring               |
| BR-551         | BO-022              | SH-003      | Cash Flow Statement Preparation               |
| BR-552         | BO-022              | SH-010      | Mapping Completeness Audit                    |
| BR-553         | BO-022              | SH-028      | Mapping Version Control                       |
| BR-554         | BO-022              | SH-003      | Local Regulatory Reporting Adjustment         |
| BR-555         | BO-022              | SH-003      | Automated Financial Statement Generation      |

---

# 9.6.3.25.7 Summary

Financial Statement Mapping menyediakan pemetaan akun ke baris laporan keuangan standar yang menjadi fondasi bagi penyusunan Neraca, Laba Rugi, dan Arus Kas secara otomatis, konsisten, dan akurat. Financial Statement Mapping menjadi requirement lanjutan yang saling terhubung dengan requirement lain pada Financial Management Domain, khususnya Chart of Accounts dan Financial Reporting.
