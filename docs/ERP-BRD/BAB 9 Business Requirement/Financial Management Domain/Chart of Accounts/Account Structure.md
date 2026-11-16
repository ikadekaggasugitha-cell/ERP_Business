# 9.6.3.1 Account Structure Requirements

---

# 9.6.3.1.1 Overview

Account Structure merupakan kerangka dasar yang mendefinisikan komposisi kode akun dalam Chart of Accounts. Struktur ini menentukan segmen-segmen penyusun akun, urutan segmen, panjang setiap segmen, aturan validasi, serta hubungan antarsegmen sehingga setiap kombinasi akun dapat diinterpretasikan secara konsisten di seluruh modul ERP.

ERP harus memungkinkan organisasi mendefinisikan satu atau lebih Account Structure sesuai kebutuhan bisnis, regulasi, dan model organisasi. Setiap Account Structure harus dapat digunakan oleh Legal Entity, Ledger, Financial Book, maupun kelompok perusahaan tertentu tanpa memengaruhi struktur lain yang telah berjalan.

---

# 9.6.3.1.2 Business Capability

Account Structure menyediakan kapabilitas berikut:

* Multiple Account Structure.
* Segment-based Account Model.
* Flexible Segment Ordering.
* Variable Segment Length.
* Segment Data Type Validation.
* Mandatory dan Optional Segment.
* Reusable Structure Definition.
* Effective Dating.
* Versioning.
* Approval Workflow.
* Organization Assignment.
* Audit Trail.

---

# 9.6.3.1.3 Business Requirement Catalogue

## BR-301 – Account Structure Definition

| Attribute           | Description                                                                                                                           |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-301                                                                                                                                |
| Requirement Name    | Account Structure Definition                                                                                                          |
| Category            | Account Structure                                                                                                                     |
| Priority            | Critical                                                                                                                              |
| Business Objective  | BO-009                                                                                                                                |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                      |
| Description         | Sistem harus menyediakan definisi Account Structure sebagai objek bisnis yang mendeskripsikan komposisi kode akun berdasarkan segmen. |
| Business Value      | Menjamin konsistensi struktur akun di seluruh organisasi.                                                                             |
| Acceptance Criteria | Administrator dapat membuat, mengubah, dan mengelola Account Structure tanpa memengaruhi struktur lain.                               |

---

## BR-302 – Multiple Account Structure

| Attribute        | Description                                                                                                                                                      |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-302                                                                                                                                                           |
| Requirement Name | Multiple Account Structure                                                                                                                                       |
| Priority         | Critical                                                                                                                                                         |
| Description      | Sistem harus mendukung lebih dari satu Account Structure yang dapat digunakan oleh Holding, Sub Holding, Legal Entity, Ledger, atau Financial Book yang berbeda. |

---

## BR-303 – Segment Configuration

| Attribute        | Description                                                                                                                                                     |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-303                                                                                                                                                          |
| Requirement Name | Segment Configuration                                                                                                                                           |
| Priority         | Critical                                                                                                                                                        |
| Description      | Setiap Account Structure harus memungkinkan konfigurasi jumlah segmen, nama segmen, urutan segmen, panjang segmen, tipe data, serta status wajib atau opsional. |

---

## BR-304 – Structure Assignment

| Attribute        | Description                                                                                                                                                |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-304                                                                                                                                                     |
| Requirement Name | Account Structure Assignment                                                                                                                               |
| Priority         | Critical                                                                                                                                                   |
| Description      | Sistem harus memungkinkan penetapan Account Structure berdasarkan Holding, Sub Holding, Legal Entity, Financial Organization, Ledger, atau Financial Book. |

---

## BR-305 – Effective Dating

| Attribute        | Description                                                                                                      |
| ---------------- | ---------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-305                                                                                                           |
| Requirement Name | Account Structure Effective Dating                                                                               |
| Priority         | High                                                                                                             |
| Description      | Setiap Account Structure harus memiliki periode berlaku sehingga perubahan tidak memengaruhi transaksi historis. |

---

## BR-306 – Version Management

| Attribute        | Description                                                                              |
| ---------------- | ---------------------------------------------------------------------------------------- |
| Requirement ID   | BR-306                                                                                   |
| Requirement Name | Account Structure Versioning                                                             |
| Priority         | High                                                                                     |
| Description      | Sistem harus menyimpan histori perubahan Account Structure melalui mekanisme versioning. |

---

## BR-307 – Validation Rules

| Attribute        | Description                                                                                                                                |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-307                                                                                                                                     |
| Requirement Name | Account Structure Validation                                                                                                               |
| Priority         | Critical                                                                                                                                   |
| Description      | Sistem harus memvalidasi bahwa setiap kode akun mengikuti definisi struktur, panjang segmen, tipe data, dan aturan kombinasi yang berlaku. |

---

## BR-308 – Dependency Validation

| Attribute        | Description                                                                                                                     |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-308                                                                                                                          |
| Requirement Name | Dependency Validation                                                                                                           |
| Priority         | Critical                                                                                                                        |
| Description      | Sistem harus mencegah perubahan struktur yang akan menyebabkan akun aktif, jurnal, atau transaksi historis menjadi tidak valid. |

---

## BR-309 – Approval Workflow

| Attribute        | Description                                                                           |
| ---------------- | ------------------------------------------------------------------------------------- |
| Requirement ID   | BR-309                                                                                |
| Requirement Name | Account Structure Approval Workflow                                                   |
| Priority         | Critical                                                                              |
| Description      | Perubahan Account Structure harus melalui proses persetujuan sebelum berlaku efektif. |

---

## BR-310 – Account Structure Auditability

| Attribute        | Description                                                                                                                                                 |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-310                                                                                                                                                      |
| Requirement Name | Account Structure Auditability                                                                                                                              |
| Priority         | Critical                                                                                                                                                    |
| Description      | Seluruh aktivitas pembuatan, perubahan, persetujuan, aktivasi, dan penghentian penggunaan Account Structure harus dicatat secara lengkap dalam Audit Trail. |

---

# 9.6.3.1.4 Business Rules

| Business Rule ID | Description                                                                                                                                                                                                         |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-AS-001     | Setiap Account Structure harus memiliki identitas unik dan nama yang tidak boleh diduplikasi dalam ruang lingkup organisasi yang sama.                                                                              |
| BRule-AS-002     | Urutan segmen harus bersifat deterministik dan digunakan secara konsisten pada seluruh akun yang menggunakan struktur tersebut.                                                                                     |
| BRule-AS-003     | Perubahan terhadap definisi segmen tidak boleh mengubah kode akun yang telah digunakan pada transaksi historis.                                                                                                     |
| BRule-AS-004     | Satu Ledger atau Financial Book hanya boleh menggunakan satu Account Structure yang aktif pada satu periode efektif, kecuali terdapat konfigurasi parallel ledger yang secara eksplisit mendukung struktur berbeda. |
| BRule-AS-005     | Perubahan struktur harus berlaku mulai periode efektif berikutnya dan tidak boleh diterapkan secara retrospektif terhadap transaksi yang telah diposting.                                                           |
| BRule-AS-006     | Account Structure yang masih digunakan oleh akun aktif atau transaksi historis tidak boleh dihapus secara permanen.                                                                                                 |
| BRule-AS-007     | Seluruh perubahan Account Structure harus terdokumentasi dalam Audit Trail.                                                                                                                                         |

---

# 9.6.3.1.5 Data Attributes

| Attribute                 | Description                                                        |
| ------------------------- | ------------------------------------------------------------------ |
| Account Structure ID      | Identitas unik sistem.                                             |
| Structure Code            | Kode struktur akun.                                                |
| Structure Name            | Nama struktur akun.                                                |
| Description               | Deskripsi struktur akun.                                           |
| Number of Segments        | Jumlah segmen yang digunakan.                                      |
| Default Separator         | Karakter pemisah tampilan (misalnya "-", ".", atau tanpa pemisah). |
| Assignment Scope          | Holding, Sub Holding, Legal Entity, Ledger, atau Financial Book.   |
| Effective From            | Tanggal mulai berlaku.                                             |
| Effective To              | Tanggal akhir berlaku.                                             |
| Version                   | Nomor versi.                                                       |
| Approval Status           | Draft, Pending Approval, Approved, Published, Archived.            |
| Status                    | Active, Suspended, Inactive, Archived.                             |
| Remarks                   | Catatan bisnis.                                                    |
| Created By / Created Date | Informasi pembuatan data.                                          |
| Updated By / Updated Date | Informasi perubahan terakhir.                                      |

---

# 9.6.3.1.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                   |
| -------------- | ------------------ | ----------- | -------------------------------- |
| BR-301         | BO-009             | SH-003      | Account Structure Administration |
| BR-302         | BO-009             | SH-003      | Multi-Structure Configuration    |
| BR-303         | BO-011             | SH-003      | Segment Configuration            |
| BR-304         | BO-011             | SH-003      | Organizational Assignment        |
| BR-305         | BO-029             | SH-010      | Effective Date Governance        |
| BR-306         | BO-029             | SH-010      | Version Management               |
| BR-307         | BO-011             | SH-003      | Account Validation               |
| BR-308         | BO-029             | SH-010      | Dependency Validation            |
| BR-309         | BO-029             | SH-003      | Approval Workflow                |
| BR-310         | BO-029             | SH-010      | Audit & Compliance               |

---

# 9.6.3.1.7 Summary

Account Structure merupakan fondasi arsitektur Chart of Accounts yang mendefinisikan bagaimana kode akun dibentuk melalui kumpulan segmen yang terstruktur. Dengan dukungan multi-structure, konfigurasi segmen yang fleksibel, *effective dating*, *versioning*, *approval workflow*, dan *audit trail*, ERP dapat mendukung kebutuhan organisasi yang kompleks tanpa mengorbankan konsistensi dan integritas data keuangan.

Account Structure menjadi dasar bagi **Chart of Accounts Master**, **Account Master**, **Segment**, **Segment Value**, **Account Combination**, **General Ledger**, **Financial Reporting**, serta seluruh proses posting dan validasi transaksi di seluruh modul ERP.
