# 9.6.3.3 Account Category Requirements

---

# 9.6.3.3.1 Overview

Account Category merupakan klasifikasi bisnis tingkat tertinggi dalam Chart of Accounts yang mengelompokkan akun berdasarkan sifat ekonominya. Account Category menjadi dasar penyusunan laporan keuangan, penentuan perilaku akuntansi, validasi proses posting, serta pengelompokan akun untuk kebutuhan pelaporan, konsolidasi, dan analisis.

ERP harus menyediakan mekanisme pengelolaan Account Category yang fleksibel, mendukung berbagai standar akuntansi (misalnya IFRS, PSAK, US GAAP), serta memungkinkan organisasi mendefinisikan kategori tambahan sesuai kebutuhan bisnis tanpa mengubah struktur dasar Chart of Accounts.

---

# 9.6.3.3.2 Business Capability

Account Category menyediakan kapabilitas berikut:

* Standard Account Categories.
* Custom Account Categories.
* Multi Accounting Standard Support.
* Financial Statement Classification.
* Category Hierarchy.
* Organizational Assignment.
* Effective Dating.
* Versioning.
* Approval Workflow.
* Lifecycle Management.
* Audit Trail.

---

# 9.6.3.3.3 Business Requirement Catalogue

## BR-321 – Account Category Definition

| Attribute           | Description                                                                                                                |
| ------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-321                                                                                                                     |
| Requirement Name    | Account Category Definition                                                                                                |
| Category            | Account Category                                                                                                           |
| Priority            | Critical                                                                                                                   |
| Business Objective  | BO-009                                                                                                                     |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                           |
| Description         | Sistem harus menyediakan Account Category sebagai objek bisnis untuk mengklasifikasikan akun berdasarkan sifat ekonominya. |
| Business Value      | Menyediakan klasifikasi standar bagi seluruh akun keuangan.                                                                |
| Acceptance Criteria | Administrator dapat membuat, mengubah, dan mengelola Account Category sesuai kewenangan.                                   |

---

## BR-322 – Standard Account Categories

| Attribute        | Description                                                                                                                                                                 |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-322                                                                                                                                                                      |
| Requirement Name | Standard Account Categories                                                                                                                                                 |
| Priority         | Critical                                                                                                                                                                    |
| Description      | Sistem harus menyediakan kategori standar seperti Asset, Liability, Equity, Revenue, Expense, serta kategori lain yang dibutuhkan sesuai standar akuntansi yang diterapkan. |

---

## BR-323 – Custom Account Categories

| Attribute        | Description                                                                                                                                                       |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-323                                                                                                                                                            |
| Requirement Name | Custom Account Categories                                                                                                                                         |
| Priority         | High                                                                                                                                                              |
| Description      | Sistem harus memungkinkan organisasi mendefinisikan Account Category tambahan sesuai kebutuhan bisnis dengan tetap menjaga integritas struktur Chart of Accounts. |

---

## BR-324 – Financial Statement Mapping

| Attribute        | Description                                                                                                                                                                                                   |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-324                                                                                                                                                                                                        |
| Requirement Name | Financial Statement Mapping                                                                                                                                                                                   |
| Priority         | Critical                                                                                                                                                                                                      |
| Description      | Setiap Account Category harus dapat dipetakan ke kelompok laporan keuangan seperti Statement of Financial Position, Statement of Profit or Loss, Statement of Cash Flows, dan Statement of Changes in Equity. |

---

## BR-325 – Organizational Assignment

| Attribute        | Description                                                                                                                                                         |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-325                                                                                                                                                              |
| Requirement Name | Organizational Assignment                                                                                                                                           |
| Priority         | High                                                                                                                                                                |
| Description      | Account Category harus dapat diterapkan pada satu atau lebih Chart of Accounts yang digunakan oleh Holding, Sub Holding, Legal Entity, Ledger, atau Financial Book. |

---

## BR-326 – Effective Dating

| Attribute        | Description                                                                                                                 |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-326                                                                                                                      |
| Requirement Name | Account Category Effective Dating                                                                                           |
| Priority         | High                                                                                                                        |
| Description      | Setiap Account Category harus memiliki periode berlaku sehingga perubahan klasifikasi tidak memengaruhi transaksi historis. |

---

## BR-327 – Version Management

| Attribute        | Description                                                                             |
| ---------------- | --------------------------------------------------------------------------------------- |
| Requirement ID   | BR-327                                                                                  |
| Requirement Name | Account Category Version Management                                                     |
| Priority         | High                                                                                    |
| Description      | Sistem harus menyimpan histori perubahan Account Category melalui mekanisme versioning. |

---

## BR-328 – Dependency Validation

| Attribute        | Description                                                                                                                                                                                      |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-328                                                                                                                                                                                           |
| Requirement Name | Dependency Validation                                                                                                                                                                            |
| Priority         | Critical                                                                                                                                                                                         |
| Description      | Sistem harus mencegah perubahan atau penghapusan Account Category yang masih direferensikan oleh Account Master, Financial Statement Mapping, Posting Rule, atau transaksi yang telah diposting. |

---

## BR-329 – Approval Workflow

| Attribute        | Description                                                                                                                     |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-329                                                                                                                          |
| Requirement Name | Account Category Approval Workflow                                                                                              |
| Priority         | Critical                                                                                                                        |
| Description      | Penambahan, perubahan, maupun penghentian penggunaan Account Category harus melalui proses persetujuan sebelum berlaku efektif. |

---

## BR-330 – Account Category Auditability

| Attribute        | Description                                                                                                                                                  |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-330                                                                                                                                                       |
| Requirement Name | Account Category Auditability                                                                                                                                |
| Priority         | Critical                                                                                                                                                     |
| Description      | Seluruh aktivitas pembuatan, perubahan, persetujuan, aktivasi, penonaktifan, dan penggunaan Account Category harus dicatat secara lengkap dalam Audit Trail. |

---

# 9.6.3.3.4 Business Rules

| Business Rule ID | Description                                                                                                                |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------- |
| BRule-AC-001     | Setiap Account Category harus memiliki kode dan nama yang unik dalam ruang lingkup Chart of Accounts yang sama.            |
| BRule-AC-002     | Setiap Account Master wajib memiliki tepat satu Account Category.                                                          |
| BRule-AC-003     | Perubahan Account Category tidak boleh mengubah klasifikasi transaksi historis yang telah diposting.                       |
| BRule-AC-004     | Account Category yang telah digunakan oleh Account Master aktif tidak boleh dihapus secara permanen.                       |
| BRule-AC-005     | Account Category harus dapat dipetakan ke satu atau lebih struktur laporan keuangan sesuai standar akuntansi yang berlaku. |
| BRule-AC-006     | Perubahan kategori hanya berlaku mulai periode efektif yang telah disetujui.                                               |
| BRule-AC-007     | Seluruh perubahan Account Category harus tercatat dalam Audit Trail.                                                       |

---

# 9.6.3.3.5 Data Attributes

| Attribute                 | Description                                             |
| ------------------------- | ------------------------------------------------------- |
| Account Category ID       | Identitas unik sistem.                                  |
| Category Code             | Kode kategori akun.                                     |
| Category Name             | Nama kategori akun.                                     |
| Description               | Deskripsi bisnis.                                       |
| Parent Category           | Referensi kategori induk (opsional).                    |
| Financial Statement Group | Kelompok laporan keuangan yang terkait.                 |
| Accounting Standard       | IFRS, PSAK, US GAAP, Local GAAP, atau standar lainnya.  |
| Chart of Accounts ID      | Referensi ke Chart of Accounts Master.                  |
| Effective From            | Tanggal mulai berlaku.                                  |
| Effective To              | Tanggal akhir berlaku.                                  |
| Version                   | Nomor versi.                                            |
| Approval Status           | Draft, Pending Approval, Approved, Published, Archived. |
| Status                    | Active, Suspended, Inactive, Archived.                  |
| Remarks                   | Catatan bisnis.                                         |
| Created By / Created Date | Informasi pembuatan data.                               |
| Updated By / Updated Date | Informasi perubahan terakhir.                           |

---

# 9.6.3.3.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                     |
| -------------- | ------------------ | ----------- | ---------------------------------- |
| BR-321         | BO-009             | SH-003      | Account Category Administration    |
| BR-322         | BO-011             | SH-003      | Standard Classification Management |
| BR-323         | BO-011             | SH-003      | Custom Classification Management   |
| BR-324         | BO-011             | SH-003      | Financial Statement Mapping        |
| BR-325         | BO-011             | SH-003      | Organizational Assignment          |
| BR-326         | BO-029             | SH-010      | Effective Date Governance          |
| BR-327         | BO-029             | SH-010      | Version Management                 |
| BR-328         | BO-029             | SH-010      | Dependency Validation              |
| BR-329         | BO-029             | SH-003      | Approval Workflow                  |
| BR-330         | BO-029             | SH-010      | Audit & Compliance                 |

---

# 9.6.3.3.7 Summary

Account Category menyediakan klasifikasi bisnis tingkat tertinggi dalam Chart of Accounts yang menjadi dasar penyusunan laporan keuangan, validasi proses akuntansi, dan pengelompokan akun untuk analisis bisnis. Dengan dukungan kategori standar maupun kustom, *effective dating*, *versioning*, *approval workflow*, dan *audit trail*, ERP mampu mendukung berbagai standar akuntansi serta kebutuhan organisasi yang kompleks tanpa mengorbankan konsistensi data.

Account Category menjadi fondasi bagi **Account Type**, **Account Class**, **Account Master**, **Financial Statement Mapping**, **General Ledger**, **Financial Consolidation**, **Budgeting**, **Financial Reporting**, dan **Enterprise Performance Management (EPM)**.
