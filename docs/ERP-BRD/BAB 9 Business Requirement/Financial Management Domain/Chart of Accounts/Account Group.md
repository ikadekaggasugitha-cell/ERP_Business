# 9.6.3.6 Account Group Requirements

---

# 9.6.3.6.1 Overview

Account Group merupakan objek bisnis yang digunakan untuk mengelompokkan Account Master berdasarkan karakteristik administrasi, tata kelola, dan konfigurasi operasional yang serupa. Account Group berfungsi sebagai template konfigurasi yang menentukan aturan pembuatan akun, validasi atribut, penomoran, persyaratan data, kontrol proses, dan pengaturan default sehingga administrasi Chart of Accounts dapat dilakukan secara konsisten.

ERP harus memungkinkan organisasi mendefinisikan Account Group sesuai kebutuhan bisnis, standar akuntansi, dan kebijakan internal tanpa memengaruhi klasifikasi bisnis yang ditentukan oleh Account Category, Account Class, maupun perilaku sistem yang ditentukan oleh Account Type.

---

# 9.6.3.6.2 Business Capability

Account Group menyediakan kapabilitas berikut:

* Standard Account Groups.
* Custom Account Groups.
* Number Range Assignment.
* Default Attribute Assignment.
* Mandatory Field Configuration.
* Optional Field Configuration.
* Validation Rule Assignment.
* Approval Workflow Assignment.
* Organizational Assignment.
* Effective Dating.
* Versioning.
* Lifecycle Management.
* Audit Trail.

---

# 9.6.3.6.3 Business Requirement Catalogue

## BR-353 – Account Group Definition

| Attribute           | Description                                                                                                                                     |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-353                                                                                                                                          |
| Requirement Name    | Account Group Definition                                                                                                                        |
| Category            | Account Group                                                                                                                                   |
| Priority            | Critical                                                                                                                                        |
| Business Objective  | BO-009                                                                                                                                          |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                |
| Description         | Sistem harus menyediakan Account Group sebagai objek bisnis yang mengelompokkan Account Master berdasarkan aturan administrasi dan tata kelola. |
| Business Value      | Menjamin konsistensi konfigurasi dan pengelolaan Account Master.                                                                                |
| Acceptance Criteria | Administrator dapat membuat, mengubah, dan mengelola Account Group tanpa memengaruhi transaksi yang telah diposting.                            |

---

## BR-354 – Standard Account Groups

| Attribute        | Description                                                                                                                                                                                                                                  |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-354                                                                                                                                                                                                                                       |
| Requirement Name | Standard Account Groups                                                                                                                                                                                                                      |
| Priority         | Critical                                                                                                                                                                                                                                     |
| Description      | Sistem harus menyediakan Account Group standar seperti Cash Accounts, Bank Accounts, Accounts Receivable, Accounts Payable, Inventory Accounts, Fixed Asset Accounts, Revenue Accounts, Expense Accounts, Equity Accounts, dan Tax Accounts. |

---

## BR-355 – Custom Account Groups

| Attribute        | Description                                                                                                                                                          |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-355                                                                                                                                                               |
| Requirement Name | Custom Account Groups                                                                                                                                                |
| Priority         | High                                                                                                                                                                 |
| Description      | Sistem harus memungkinkan organisasi membuat Account Group tambahan sesuai kebutuhan industri, regulasi, atau kebijakan internal tanpa mengubah konfigurasi standar. |

---

## BR-356 – Number Range Assignment

| Attribute        | Description                                                                                                                                                   |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-356                                                                                                                                                        |
| Requirement Name | Number Range Assignment                                                                                                                                       |
| Priority         | Critical                                                                                                                                                      |
| Description      | Setiap Account Group harus dapat memiliki rentang nomor akun (number range) atau pola penomoran yang digunakan sebagai dasar pembentukan Account Master baru. |

---

## BR-357 – Default Configuration Assignment

| Attribute        | Description                                                                                                                                                                                                 |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-357                                                                                                                                                                                                      |
| Requirement Name | Default Configuration Assignment                                                                                                                                                                            |
| Priority         | Critical                                                                                                                                                                                                    |
| Description      | Sistem harus memungkinkan Account Group menentukan nilai default seperti mata uang, Account Type, Account Class, Financial Statement Mapping, Posting Rule, dan atribut lainnya saat Account Master dibuat. |

---

## BR-358 – Field Configuration

| Attribute        | Description                                                                                                                                                                          |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-358                                                                                                                                                                               |
| Requirement Name | Field Configuration                                                                                                                                                                  |
| Priority         | Critical                                                                                                                                                                             |
| Description      | Account Group harus dapat menentukan atribut mana yang bersifat Mandatory, Optional, Read Only, Hidden, atau System Generated pada proses pembuatan dan pemeliharaan Account Master. |

---

## BR-359 – Validation Rule Assignment

| Attribute        | Description                                                                                                                                               |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-359                                                                                                                                                    |
| Requirement Name | Validation Rule Assignment                                                                                                                                |
| Priority         | High                                                                                                                                                      |
| Description      | Account Group harus dapat dikaitkan dengan Validation Rule untuk memastikan bahwa setiap Account Master memenuhi kebijakan organisasi sebelum diaktifkan. |

---

## BR-360 – Organizational Assignment

| Attribute        | Description                                                                                                                                                                 |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-360                                                                                                                                                                      |
| Requirement Name | Organizational Assignment                                                                                                                                                   |
| Priority         | High                                                                                                                                                                        |
| Description      | Account Group harus dapat diterapkan pada satu atau lebih Chart of Accounts, Holding, Sub Holding, Legal Entity, Ledger, atau Financial Book sesuai konfigurasi organisasi. |

---

## BR-361 – Effective Dating

| Attribute        | Description                                                                                                                                    |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-361                                                                                                                                         |
| Requirement Name | Account Group Effective Dating                                                                                                                 |
| Priority         | High                                                                                                                                           |
| Description      | Setiap Account Group harus memiliki periode berlaku sehingga perubahan konfigurasi tidak memengaruhi Account Master maupun transaksi historis. |

---

## BR-362 – Version Management

| Attribute        | Description                                                                                                                                      |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-362                                                                                                                                           |
| Requirement Name | Account Group Version Management                                                                                                                 |
| Priority         | High                                                                                                                                             |
| Description      | Sistem harus menyimpan histori perubahan Account Group melalui mekanisme versioning sehingga perubahan konfigurasi dapat ditelusuri dan diaudit. |

---

## BR-363 – Approval Workflow

| Attribute        | Description                                                                                                                                                    |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-363                                                                                                                                                         |
| Requirement Name | Account Group Approval Workflow                                                                                                                                |
| Priority         | Critical                                                                                                                                                       |
| Description      | Pembuatan, perubahan, aktivasi, maupun penghentian penggunaan Account Group harus melalui proses persetujuan sesuai matriks otorisasi sebelum berlaku efektif. |

---

## BR-364 – Dependency Validation

| Attribute        | Description                                                                                                                                         |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-364                                                                                                                                              |
| Requirement Name | Dependency Validation                                                                                                                               |
| Priority         | Critical                                                                                                                                            |
| Description      | Sistem harus mencegah perubahan atau penghapusan Account Group yang masih direferensikan oleh Account Master aktif atau konfigurasi sistem lainnya. |

---

## BR-365 – Account Group Auditability

| Attribute        | Description                                                                                                                                                 |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-365                                                                                                                                                      |
| Requirement Name | Account Group Auditability                                                                                                                                  |
| Priority         | Critical                                                                                                                                                    |
| Description      | Seluruh aktivitas pembuatan, perubahan, persetujuan, aktivasi, penonaktifan, serta penggunaan Account Group harus dicatat secara lengkap dalam Audit Trail. |

---

# 9.6.3.6.4 Business Rules

| Business Rule ID | Description                                                                                                                                                                                                    |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-AG-001     | Setiap Account Group harus memiliki kode dan nama yang unik dalam ruang lingkup Chart of Accounts yang sama.                                                                                                   |
| BRule-AG-002     | Setiap Account Master wajib dikaitkan dengan tepat satu Account Group pada saat pembentukan akun.                                                                                                              |
| BRule-AG-003     | Account Group dapat menetapkan nilai default untuk Account Category, Account Class, Account Type, Number Range, dan atribut lain, namun administrator dapat mengubahnya jika kebijakan organisasi mengizinkan. |
| BRule-AG-004     | Perubahan konfigurasi Account Group hanya berlaku bagi Account Master yang dibuat setelah tanggal efektif, kecuali organisasi secara eksplisit melakukan proses migrasi massal yang terdokumentasi.            |
| BRule-AG-005     | Account Group yang masih digunakan oleh Account Master aktif tidak boleh dihapus secara permanen dan hanya dapat diubah statusnya menjadi Inactive atau Archived.                                              |
| BRule-AG-006     | Perubahan terhadap aturan Mandatory Field atau Validation Rule tidak boleh menyebabkan Account Master yang telah aktif menjadi tidak valid secara retrospektif.                                                |
| BRule-AG-007     | Seluruh perubahan Account Group harus tercatat dalam Audit Trail beserta identitas pengguna, waktu perubahan, dan alasan bisnis.                                                                               |

---

# 9.6.3.6.5 Data Attributes

| Attribute                 | Description                                                    |
| ------------------------- | -------------------------------------------------------------- |
| Account Group ID          | Identitas unik sistem.                                         |
| Group Code                | Kode Account Group.                                            |
| Group Name                | Nama Account Group.                                            |
| Description               | Deskripsi bisnis.                                              |
| Number Range              | Rentang atau pola penomoran akun.                              |
| Default Account Category  | Nilai default kategori akun (opsional).                        |
| Default Account Class     | Nilai default kelas akun (opsional).                           |
| Default Account Type      | Nilai default tipe akun (opsional).                            |
| Default Currency          | Mata uang default (opsional).                                  |
| Validation Rule           | Referensi ke aturan validasi.                                  |
| Approval Workflow         | Referensi ke workflow persetujuan.                             |
| Chart of Accounts ID      | Referensi ke Chart of Accounts Master.                         |
| Effective From            | Tanggal mulai berlaku.                                         |
| Effective To              | Tanggal akhir berlaku.                                         |
| Version                   | Nomor versi.                                                   |
| Approval Status           | Draft, Pending Approval, Approved, Published, Archived.        |
| Status                    | Draft, Review, Approved, Active, Suspended, Retired, Archived. |
| Remarks                   | Catatan bisnis.                                                |
| Created By / Created Date | Informasi pembuatan data.                                      |
| Updated By / Updated Date | Informasi perubahan terakhir.                                  |

---

# 9.6.3.6.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                    |
| -------------- | ------------------ | ----------- | --------------------------------- |
| BR-353         | BO-009             | SH-003      | Account Group Administration      |
| BR-354         | BO-011             | SH-003      | Standard Account Group Management |
| BR-355         | BO-011             | SH-003      | Custom Account Group Management   |
| BR-356         | BO-011             | SH-003      | Number Range Configuration        |
| BR-357         | BO-011             | SH-003      | Default Configuration Management  |
| BR-358         | BO-011             | SH-003      | Field Configuration Management    |
| BR-359         | BO-029             | SH-010      | Validation Rule Management        |
| BR-360         | BO-011             | SH-003      | Organizational Assignment         |
| BR-361         | BO-029             | SH-010      | Effective Date Governance         |
| BR-362         | BO-029             | SH-010      | Version Management                |
| BR-363         | BO-029             | SH-003      | Approval Workflow                 |
| BR-364         | BO-029             | SH-010      | Dependency Validation             |
| BR-365         | BO-029             | SH-010      | Audit & Compliance                |

---

# 9.6.3.6.7 Summary

Account Group merupakan mekanisme tata kelola yang menyederhanakan administrasi Chart of Accounts melalui penggunaan template konfigurasi yang konsisten. Dengan mengelompokkan Account Master berdasarkan aturan administrasi yang sama, organisasi dapat mempercepat pembuatan akun baru, memastikan kepatuhan terhadap kebijakan internal, serta mengurangi risiko kesalahan konfigurasi.

Kapabilitas seperti **number range management**, **default value assignment**, **field configuration**, **validation rules**, **effective dating**, **versioning**, **approval workflow**, dan **audit trail** menjadikan Account Group sebagai fondasi penting dalam pengelolaan siklus hidup Account Master pada ERP enterprise.
