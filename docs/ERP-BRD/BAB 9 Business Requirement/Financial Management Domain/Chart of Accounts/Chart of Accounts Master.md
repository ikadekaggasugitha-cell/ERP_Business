# 9.6.3.2 Chart of Accounts Master Requirements

---

# 9.6.3.2.1 Overview

Chart of Accounts Master (CoA Master) merupakan objek bisnis utama yang mendefinisikan kumpulan akun keuangan beserta struktur, aturan, dan ruang lingkup penggunaannya dalam sistem ERP. CoA Master berfungsi sebagai kerangka acuan bagi seluruh proses pencatatan akuntansi, pelaporan keuangan, konsolidasi, dan integrasi lintas modul.

ERP harus mendukung lebih dari satu Chart of Accounts untuk memenuhi kebutuhan organisasi yang memiliki beberapa standar akuntansi, beberapa Ledger, beberapa Financial Book, atau beberapa wilayah operasional. Setiap CoA Master harus dapat dikelola secara independen dengan mekanisme *effective dating*, *versioning*, *approval workflow*, dan *audit trail*.

---

# 9.6.3.2.2 Business Capability

Chart of Accounts Master menyediakan kapabilitas berikut:

* Multiple Chart of Accounts.
* Corporate Chart of Accounts.
* Local Chart of Accounts.
* Statutory Chart of Accounts.
* Management Chart of Accounts.
* Tax Chart of Accounts.
* Assignment berdasarkan Holding, Sub Holding, Legal Entity, Ledger, dan Financial Book.
* Effective Dating.
* Versioning.
* Approval Workflow.
* Status Lifecycle Management.
* Audit Trail.

---

# 9.6.3.2.3 Business Requirement Catalogue

## BR-311 – Chart of Accounts Master Definition

| Attribute           | Description                                                                                                                                                            |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-311                                                                                                                                                                 |
| Requirement Name    | Chart of Accounts Master Definition                                                                                                                                    |
| Category            | Chart of Accounts Master                                                                                                                                               |
| Priority            | Critical                                                                                                                                                               |
| Business Objective  | BO-009                                                                                                                                                                 |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                       |
| Description         | Sistem harus menyediakan Chart of Accounts Master sebagai objek bisnis utama yang menjadi induk bagi seluruh Account Master, Segment, Hierarchy, dan aturan akuntansi. |
| Business Value      | Menyediakan fondasi yang konsisten bagi pencatatan dan pelaporan keuangan.                                                                                             |
| Acceptance Criteria | Administrator dapat membuat dan mengelola lebih dari satu Chart of Accounts Master sesuai kebutuhan organisasi.                                                        |

---

## BR-312 – Multiple Chart of Accounts

| Attribute        | Description                                                                                                                                                             |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-312                                                                                                                                                                  |
| Requirement Name | Multiple Chart of Accounts                                                                                                                                              |
| Priority         | Critical                                                                                                                                                                |
| Description      | Sistem harus mendukung beberapa Chart of Accounts yang dapat digunakan secara bersamaan untuk kebutuhan Corporate, Local, Tax, Statutory, maupun Management Accounting. |

---

## BR-313 – Organizational Assignment

| Attribute        | Description                                                                                                                                                                        |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-313                                                                                                                                                                             |
| Requirement Name | Organizational Assignment                                                                                                                                                          |
| Priority         | Critical                                                                                                                                                                           |
| Description      | Chart of Accounts Master harus dapat ditetapkan pada Holding, Sub Holding, Legal Entity, Financial Organization, Ledger, Financial Book, atau kombinasi dari objek-objek tersebut. |

---

## BR-314 – Account Structure Association

| Attribute        | Description                                                                                                                    |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-314                                                                                                                         |
| Requirement Name | Account Structure Association                                                                                                  |
| Priority         | Critical                                                                                                                       |
| Description      | Setiap Chart of Accounts Master harus memiliki tepat satu Account Structure aktif sebagai dasar pembentukan seluruh kode akun. |

---

## BR-315 – Effective Dating

| Attribute        | Description                                                                                                                  |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-315                                                                                                                       |
| Requirement Name | Chart of Accounts Effective Dating                                                                                           |
| Priority         | High                                                                                                                         |
| Description      | Chart of Accounts Master harus memiliki periode berlaku sehingga perubahan konfigurasi tidak memengaruhi transaksi historis. |

---

## BR-316 – Version Management

| Attribute        | Description                                                                                                                                                 |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-316                                                                                                                                                      |
| Requirement Name | Chart of Accounts Version Management                                                                                                                        |
| Priority         | High                                                                                                                                                        |
| Description      | Sistem harus menyimpan histori perubahan Chart of Accounts Master melalui mekanisme versioning sehingga setiap perubahan dapat ditelusuri dan dibandingkan. |

---

## BR-317 – Dependency Validation

| Attribute        | Description                                                                                                                                                                        |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-317                                                                                                                                                                             |
| Requirement Name | Dependency Validation                                                                                                                                                              |
| Priority         | Critical                                                                                                                                                                           |
| Description      | Sistem harus mencegah perubahan atau penghapusan Chart of Accounts Master yang masih digunakan oleh Account Master, Ledger, Financial Book, maupun transaksi yang telah diposting. |

---

## BR-318 – Approval Workflow

| Attribute        | Description                                                                                                                            |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-318                                                                                                                                 |
| Requirement Name | Chart of Accounts Approval Workflow                                                                                                    |
| Priority         | Critical                                                                                                                               |
| Description      | Pembuatan maupun perubahan Chart of Accounts Master harus melalui proses persetujuan sesuai matriks otorisasi sebelum berlaku efektif. |

---

## BR-319 – Lifecycle Management

| Attribute        | Description                                                                                                                                   |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-319                                                                                                                                        |
| Requirement Name | Chart of Accounts Lifecycle Management                                                                                                        |
| Priority         | High                                                                                                                                          |
| Description      | Sistem harus mendukung siklus hidup Chart of Accounts Master dengan status Draft, Review, Approved, Active, Suspended, Retired, dan Archived. |

---

## BR-320 – Chart of Accounts Auditability

| Attribute        | Description                                                                                                                                                                       |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-320                                                                                                                                                                            |
| Requirement Name | Chart of Accounts Auditability                                                                                                                                                    |
| Priority         | Critical                                                                                                                                                                          |
| Description      | Seluruh aktivitas pembuatan, perubahan, persetujuan, aktivasi, penonaktifan, dan penghentian penggunaan Chart of Accounts Master harus tercatat secara lengkap dalam Audit Trail. |

---

# 9.6.3.2.4 Business Rules

| Business Rule ID | Description                                                                                                                                                                                   |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-COA-001    | Setiap Chart of Accounts Master harus memiliki kode dan nama yang unik dalam ruang lingkup organisasi yang sama.                                                                              |
| BRule-COA-002    | Setiap Chart of Accounts Master wajib memiliki satu Account Structure aktif sebelum dapat digunakan.                                                                                          |
| BRule-COA-003    | Satu Ledger hanya dapat menggunakan satu Chart of Accounts Master yang aktif pada satu periode efektif, kecuali terdapat konfigurasi parallel ledger yang mengizinkan penggunaan CoA berbeda. |
| BRule-COA-004    | Perubahan Chart of Accounts Master tidak boleh mengubah Account Master maupun transaksi historis yang telah diposting.                                                                        |
| BRule-COA-005    | Chart of Accounts Master yang masih memiliki Account Master aktif atau digunakan oleh Ledger tidak boleh dihapus secara permanen.                                                             |
| BRule-COA-006    | Perubahan konfigurasi hanya berlaku mulai periode efektif yang telah disetujui.                                                                                                               |
| BRule-COA-007    | Seluruh perubahan terhadap Chart of Accounts Master harus terdokumentasi dalam Audit Trail.                                                                                                   |

---

# 9.6.3.2.5 Data Attributes

| Attribute                 | Description                                                                 |
| ------------------------- | --------------------------------------------------------------------------- |
| Chart of Accounts ID      | Identitas unik sistem.                                                      |
| CoA Code                  | Kode Chart of Accounts.                                                     |
| CoA Name                  | Nama Chart of Accounts.                                                     |
| Description               | Deskripsi bisnis.                                                           |
| CoA Type                  | Corporate, Local, Statutory, Management, Tax, Consolidation, Custom.        |
| Account Structure ID      | Referensi ke Account Structure yang digunakan.                              |
| Assignment Scope          | Holding, Sub Holding, Legal Entity, Ledger, Financial Book, atau kombinasi. |
| Accounting Standard       | IFRS, PSAK, US GAAP, Local GAAP, atau standar lainnya.                      |
| Default Currency          | Mata uang pelaporan utama yang direkomendasikan untuk CoA ini (opsional).   |
| Effective From            | Tanggal mulai berlaku.                                                      |
| Effective To              | Tanggal akhir berlaku.                                                      |
| Version                   | Nomor versi.                                                                |
| Approval Status           | Draft, Pending Approval, Approved, Published, Archived.                     |
| Status                    | Draft, Review, Approved, Active, Suspended, Retired, Archived.              |
| Remarks                   | Catatan bisnis.                                                             |
| Created By / Created Date | Informasi pembuatan data.                                                   |
| Updated By / Updated Date | Informasi perubahan terakhir.                                               |

---

# 9.6.3.2.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                     |
| -------------- | ------------------ | ----------- | ---------------------------------- |
| BR-311         | BO-009             | SH-003      | Chart of Accounts Administration   |
| BR-312         | BO-011             | SH-003      | Multi-Chart of Accounts Management |
| BR-313         | BO-011             | SH-003      | Organizational Assignment          |
| BR-314         | BO-011             | SH-003      | Account Structure Binding          |
| BR-315         | BO-029             | SH-010      | Effective Date Governance          |
| BR-316         | BO-029             | SH-010      | Version Management                 |
| BR-317         | BO-029             | SH-010      | Dependency Validation              |
| BR-318         | BO-029             | SH-003      | Approval Workflow                  |
| BR-319         | BO-029             | SH-003      | Lifecycle Management               |
| BR-320         | BO-029             | SH-010      | Audit & Compliance                 |

---

# 9.6.3.2.7 Summary

Chart of Accounts Master merupakan fondasi utama dari seluruh model akuntansi dalam ERP. Objek ini mengelola kumpulan akun beserta struktur, ruang lingkup, standar akuntansi, dan siklus hidupnya sehingga organisasi dapat mengoperasikan beberapa Chart of Accounts secara bersamaan untuk kebutuhan pelaporan korporat, lokal, perpajakan, maupun manajemen.

Dengan dukungan **multi-CoA**, **assignment lintas organisasi**, **effective dating**, **versioning**, **approval workflow**, dan **audit trail**, ERP mampu mengakomodasi perusahaan berskala global yang menerapkan berbagai standar akuntansi dan model pelaporan tanpa mengorbankan konsistensi maupun integritas data historis.

Chart of Accounts Master menjadi referensi utama bagi **Account Master**, **General Ledger**, **Accounts Payable**, **Accounts Receivable**, **Fixed Assets**, **Cash Management**, **Treasury**, **Project Accounting**, **Budgeting**, **Financial Consolidation**, dan **Financial Reporting**.
