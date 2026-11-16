# 9.6.3.5 Account Class Requirements

---

# 9.6.3.5.1 Overview

Account Class merupakan klasifikasi operasional yang mengelompokkan Account Master ke dalam kelompok keuangan yang lebih spesifik berdasarkan karakteristik bisnis, tujuan pelaporan, dan perlakuan akuntansi. Account Class berada di bawah **Account Category** dan menjadi dasar untuk pengelompokan akun pada laporan keuangan, analisis manajemen, budgeting, cost accounting, serta financial consolidation.

ERP harus memungkinkan organisasi mendefinisikan Account Class standar maupun kustom, sehingga struktur Chart of Accounts dapat menyesuaikan kebutuhan industri, standar akuntansi, dan kebijakan internal tanpa mengubah Account Category ataupun perilaku sistem yang didefinisikan oleh Account Type.

---

# 9.6.3.5.2 Business Capability

Account Class menyediakan kapabilitas berikut:

* Standard Account Classes.
* Custom Account Classes.
* Financial Statement Classification.
* Management Reporting Classification.
* Budget Classification.
* Cost Accounting Classification.
* Hierarchical Classification.
* Organization Assignment.
* Effective Dating.
* Versioning.
* Approval Workflow.
* Lifecycle Management.
* Audit Trail.

---

# 9.6.3.5.3 Business Requirement Catalogue

## BR-341 – Account Class Definition

| Attribute           | Description                                                                                                                                                 |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-341                                                                                                                                                      |
| Requirement Name    | Account Class Definition                                                                                                                                    |
| Category            | Account Class                                                                                                                                               |
| Priority            | Critical                                                                                                                                                    |
| Business Objective  | BO-009                                                                                                                                                      |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                            |
| Description         | Sistem harus menyediakan Account Class sebagai objek bisnis yang mengelompokkan Account Master berdasarkan klasifikasi operasional dan kebutuhan pelaporan. |
| Business Value      | Menyediakan lapisan klasifikasi yang fleksibel antara Account Category dan Account Type.                                                                    |
| Acceptance Criteria | Administrator dapat membuat, mengubah, dan mengelola Account Class sesuai hak akses yang diberikan.                                                         |

---

## BR-342 – Standard Account Classes

| Attribute        | Description                                                                                                                                                                                                                                                                        |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-342                                                                                                                                                                                                                                                                             |
| Requirement Name | Standard Account Classes                                                                                                                                                                                                                                                           |
| Priority         | Critical                                                                                                                                                                                                                                                                           |
| Description      | Sistem harus menyediakan Account Class standar seperti Current Asset, Non-Current Asset, Current Liability, Non-Current Liability, Equity, Operating Revenue, Non-Operating Revenue, Cost of Sales, Operating Expense, Administrative Expense, Financial Expense, dan Tax Expense. |

---

## BR-343 – Custom Account Classes

| Attribute        | Description                                                                                                                                                                 |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-343                                                                                                                                                                      |
| Requirement Name | Custom Account Classes                                                                                                                                                      |
| Priority         | High                                                                                                                                                                        |
| Description      | Sistem harus memungkinkan organisasi mendefinisikan Account Class tambahan sesuai kebutuhan industri, regulasi, atau kebijakan internal tanpa mengubah klasifikasi standar. |

---

## BR-344 – Account Category Association

| Attribute        | Description                                                                                                  |
| ---------------- | ------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-344                                                                                                       |
| Requirement Name | Account Category Association                                                                                 |
| Priority         | Critical                                                                                                     |
| Description      | Setiap Account Class harus dikaitkan dengan tepat satu Account Category sebagai induk klasifikasi bisnisnya. |

---

## BR-345 – Financial Statement Mapping

| Attribute        | Description                                                                                                                                                                                                                 |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-345                                                                                                                                                                                                                      |
| Requirement Name | Financial Statement Mapping                                                                                                                                                                                                 |
| Priority         | Critical                                                                                                                                                                                                                    |
| Description      | Account Class harus dapat dipetakan ke struktur laporan keuangan, termasuk Statement of Financial Position, Statement of Profit or Loss, Statement of Cash Flows, Statement of Changes in Equity, maupun laporan manajemen. |

---

## BR-346 – Organizational Assignment

| Attribute        | Description                                                                                                                                          |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-346                                                                                                                                               |
| Requirement Name | Organizational Assignment                                                                                                                            |
| Priority         | High                                                                                                                                                 |
| Description      | Account Class harus dapat digunakan pada satu atau lebih Chart of Accounts, Legal Entity, Ledger, atau Financial Book sesuai konfigurasi organisasi. |

---

## BR-347 – Effective Dating

| Attribute        | Description                                                                                                                                                       |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-347                                                                                                                                                            |
| Requirement Name | Account Class Effective Dating                                                                                                                                    |
| Priority         | High                                                                                                                                                              |
| Description      | Setiap Account Class harus memiliki periode berlaku sehingga perubahan klasifikasi tidak memengaruhi transaksi historis maupun laporan yang telah dipublikasikan. |

---

## BR-348 – Version Management

| Attribute        | Description                                                                                                                               |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-348                                                                                                                                    |
| Requirement Name | Account Class Version Management                                                                                                          |
| Priority         | High                                                                                                                                      |
| Description      | Sistem harus menyimpan histori perubahan Account Class melalui mekanisme versioning sehingga perubahan dapat ditelusuri dan dibandingkan. |

---

## BR-349 – Dependency Validation

| Attribute        | Description                                                                                                                                                                                                   |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-349                                                                                                                                                                                                        |
| Requirement Name | Dependency Validation                                                                                                                                                                                         |
| Priority         | Critical                                                                                                                                                                                                      |
| Description      | Sistem harus mencegah perubahan atau penghapusan Account Class yang masih digunakan oleh Account Master, Financial Statement Mapping, Budget Structure, Cost Accounting, atau transaksi yang telah diposting. |

---

## BR-350 – Approval Workflow

| Attribute        | Description                                                                                                                                           |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-350                                                                                                                                                |
| Requirement Name | Account Class Approval Workflow                                                                                                                       |
| Priority         | Critical                                                                                                                                              |
| Description      | Penambahan, perubahan, maupun penghentian penggunaan Account Class harus melalui proses persetujuan sesuai matriks otorisasi sebelum berlaku efektif. |

---

## BR-351 – Lifecycle Management

| Attribute        | Description                                                                                                                        |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-351                                                                                                                             |
| Requirement Name | Account Class Lifecycle Management                                                                                                 |
| Priority         | High                                                                                                                               |
| Description      | Sistem harus mendukung siklus hidup Account Class dengan status Draft, Review, Approved, Active, Suspended, Retired, dan Archived. |

---

## BR-352 – Account Class Auditability

| Attribute        | Description                                                                                                                                                 |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-352                                                                                                                                                      |
| Requirement Name | Account Class Auditability                                                                                                                                  |
| Priority         | Critical                                                                                                                                                    |
| Description      | Seluruh aktivitas pembuatan, perubahan, persetujuan, aktivasi, penonaktifan, serta penggunaan Account Class harus dicatat secara lengkap dalam Audit Trail. |

---

# 9.6.3.5.4 Business Rules

| Business Rule ID | Description                                                                                                                 |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------- |
| BRule-ACL-001    | Setiap Account Class harus memiliki kode dan nama yang unik dalam ruang lingkup Chart of Accounts yang sama.                |
| BRule-ACL-002    | Setiap Account Class wajib memiliki tepat satu Account Category sebagai induk klasifikasi.                                  |
| BRule-ACL-003    | Setiap Account Master wajib memiliki tepat satu Account Class aktif.                                                        |
| BRule-ACL-004    | Perubahan Account Class tidak boleh mengubah klasifikasi transaksi historis maupun laporan keuangan yang telah diterbitkan. |
| BRule-ACL-005    | Account Class yang masih digunakan oleh Account Master aktif tidak boleh dihapus secara permanen.                           |
| BRule-ACL-006    | Perubahan hanya berlaku mulai periode efektif yang telah disetujui.                                                         |
| BRule-ACL-007    | Seluruh perubahan terhadap Account Class harus terdokumentasi dalam Audit Trail.                                            |

---

# 9.6.3.5.5 Data Attributes

| Attribute                      | Description                                                    |
| ------------------------------ | -------------------------------------------------------------- |
| Account Class ID               | Identitas unik sistem.                                         |
| Class Code                     | Kode Account Class.                                            |
| Class Name                     | Nama Account Class.                                            |
| Description                    | Deskripsi bisnis.                                              |
| Account Category ID            | Referensi ke Account Category.                                 |
| Financial Statement Section    | Kelompok laporan keuangan utama.                               |
| Budget Classification          | Klasifikasi anggaran (opsional).                               |
| Cost Accounting Classification | Klasifikasi biaya (opsional).                                  |
| Reporting Sequence             | Urutan penyajian dalam laporan.                                |
| Chart of Accounts ID           | Referensi ke Chart of Accounts Master.                         |
| Effective From                 | Tanggal mulai berlaku.                                         |
| Effective To                   | Tanggal akhir berlaku.                                         |
| Version                        | Nomor versi.                                                   |
| Approval Status                | Draft, Pending Approval, Approved, Published, Archived.        |
| Status                         | Draft, Review, Approved, Active, Suspended, Retired, Archived. |
| Remarks                        | Catatan bisnis.                                                |
| Created By / Created Date      | Informasi pembuatan data.                                      |
| Updated By / Updated Date      | Informasi perubahan terakhir.                                  |

---

# 9.6.3.5.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                     |
| -------------- | ------------------ | ----------- | ---------------------------------- |
| BR-341         | BO-009             | SH-003      | Account Class Administration       |
| BR-342         | BO-011             | SH-003      | Standard Classification Management |
| BR-343         | BO-011             | SH-003      | Custom Classification Management   |
| BR-344         | BO-011             | SH-003      | Account Category Association       |
| BR-345         | BO-011             | SH-003      | Financial Statement Mapping        |
| BR-346         | BO-011             | SH-003      | Organizational Assignment          |
| BR-347         | BO-029             | SH-010      | Effective Date Governance          |
| BR-348         | BO-029             | SH-010      | Version Management                 |
| BR-349         | BO-029             | SH-010      | Dependency Validation              |
| BR-350         | BO-029             | SH-003      | Approval Workflow                  |
| BR-351         | BO-029             | SH-003      | Lifecycle Management               |
| BR-352         | BO-029             | SH-010      | Audit & Compliance                 |

---

# 9.6.3.5.7 Summary

Account Class menyediakan lapisan klasifikasi operasional yang menghubungkan **Account Category** dengan **Account Type**, sehingga Chart of Accounts memiliki struktur yang lebih kaya untuk kebutuhan pelaporan keuangan, budgeting, cost accounting, dan analisis manajemen. Dengan dukungan klasifikasi standar maupun kustom, *effective dating*, *versioning*, *approval workflow*, dan *audit trail*, ERP dapat mengakomodasi perubahan organisasi dan regulasi tanpa mengganggu integritas transaksi historis.

Account Class menjadi fondasi bagi **Account Master**, **Financial Statement Mapping**, **Budget Structure**, **Cost Accounting**, **General Ledger**, **Financial Consolidation**, **Management Reporting**, dan **Enterprise Performance Management (EPM)**.
