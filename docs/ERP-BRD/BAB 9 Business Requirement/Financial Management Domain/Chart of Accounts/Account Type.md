# 9.6.3.4 Account Type Requirements

---

# 9.6.3.4.1 Overview

Account Type merupakan klasifikasi perilaku (*behavioral classification*) yang menentukan bagaimana suatu akun digunakan, divalidasi, dan diproses oleh sistem ERP. Berbeda dengan **Account Category** yang mengelompokkan akun berdasarkan sifat ekonominya, Account Type mengatur fungsi operasional akun dalam proses transaksi, posting, konsolidasi, alokasi, pelaporan, dan pengendalian internal.

ERP harus menyediakan Account Type yang dapat dikonfigurasi, diperluas, dan diterapkan secara konsisten pada seluruh Account Master sehingga perilaku setiap akun dapat dikendalikan melalui aturan sistem, bukan melalui logika aplikasi yang tersebar.

---

# 9.6.3.4.2 Business Capability

Account Type menyediakan kapabilitas berikut:

* Standard Account Types.
* Custom Account Types.
* Posting Behavior Configuration.
* Transaction Validation Rules.
* Journal Processing Rules.
* Financial Reporting Behavior.
* Organization Assignment.
* Effective Dating.
* Versioning.
* Approval Workflow.
* Lifecycle Management.
* Audit Trail.

---

# 9.6.3.4.3 Business Requirement Catalogue

## BR-331 – Account Type Definition

| Attribute           | Description                                                                                                         |
| ------------------- | ------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-331                                                                                                              |
| Requirement Name    | Account Type Definition                                                                                             |
| Category            | Account Type                                                                                                        |
| Priority            | Critical                                                                                                            |
| Business Objective  | BO-009                                                                                                              |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                    |
| Description         | Sistem harus menyediakan Account Type sebagai objek bisnis yang mendefinisikan perilaku operasional akun dalam ERP. |
| Business Value      | Menjamin konsistensi perilaku akun di seluruh proses keuangan.                                                      |
| Acceptance Criteria | Administrator dapat membuat dan mengelola Account Type sesuai kewenangan.                                           |

---

## BR-332 – Standard Account Types

| Attribute        | Description                                                                                                                                                     |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-332                                                                                                                                                          |
| Requirement Name | Standard Account Types                                                                                                                                          |
| Priority         | Critical                                                                                                                                                        |
| Description      | Sistem harus menyediakan Account Type standar seperti Posting, Header, Control, Summary, Statistical, Memo, Allocation, Clearing, Suspense, dan System Account. |

---

## BR-333 – Custom Account Types

| Attribute        | Description                                                                                                                                   |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-333                                                                                                                                        |
| Requirement Name | Custom Account Types                                                                                                                          |
| Priority         | High                                                                                                                                          |
| Description      | Sistem harus memungkinkan organisasi membuat Account Type tambahan untuk memenuhi kebutuhan bisnis atau regulasi tanpa mengubah tipe standar. |

---

## BR-334 – Processing Behavior Configuration

| Attribute        | Description                                                                                                                                                                              |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-334                                                                                                                                                                                   |
| Requirement Name | Processing Behavior Configuration                                                                                                                                                        |
| Priority         | Critical                                                                                                                                                                                 |
| Description      | Setiap Account Type harus memiliki konfigurasi perilaku yang menentukan apakah akun dapat diposting, menerima jurnal manual, digunakan dalam alokasi, atau hanya sebagai akun ringkasan. |

---

## BR-335 – Organizational Assignment

| Attribute        | Description                                                                                                                                                                                |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-335                                                                                                                                                                                     |
| Requirement Name | Organizational Assignment                                                                                                                                                                  |
| Priority         | High                                                                                                                                                                                       |
| Description      | Account Type harus dapat diterapkan pada satu atau lebih Chart of Accounts dan digunakan oleh Holding, Sub Holding, Legal Entity, Ledger, atau Financial Book sesuai kebutuhan organisasi. |

---

## BR-336 – Effective Dating

| Attribute        | Description                                                                                                 |
| ---------------- | ----------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-336                                                                                                      |
| Requirement Name | Account Type Effective Dating                                                                               |
| Priority         | High                                                                                                        |
| Description      | Setiap Account Type harus memiliki periode berlaku sehingga perubahan tidak memengaruhi transaksi historis. |

---

## BR-337 – Version Management

| Attribute        | Description                                                                         |
| ---------------- | ----------------------------------------------------------------------------------- |
| Requirement ID   | BR-337                                                                              |
| Requirement Name | Account Type Version Management                                                     |
| Priority         | High                                                                                |
| Description      | Sistem harus menyimpan histori perubahan Account Type melalui mekanisme versioning. |

---

## BR-338 – Dependency Validation

| Attribute        | Description                                                                                                                                                                      |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-338                                                                                                                                                                           |
| Requirement Name | Dependency Validation                                                                                                                                                            |
| Priority         | Critical                                                                                                                                                                         |
| Description      | Sistem harus mencegah perubahan atau penghapusan Account Type yang masih direferensikan oleh Account Master, Posting Rule, Allocation Rule, atau transaksi yang telah diposting. |

---

## BR-339 – Approval Workflow

| Attribute        | Description                                                                                                                 |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-339                                                                                                                      |
| Requirement Name | Account Type Approval Workflow                                                                                              |
| Priority         | Critical                                                                                                                    |
| Description      | Penambahan, perubahan, maupun penghentian penggunaan Account Type harus melalui proses persetujuan sebelum berlaku efektif. |

---

## BR-340 – Account Type Auditability

| Attribute        | Description                                                                                                                                              |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-340                                                                                                                                                   |
| Requirement Name | Account Type Auditability                                                                                                                                |
| Priority         | Critical                                                                                                                                                 |
| Description      | Seluruh aktivitas pembuatan, perubahan, persetujuan, aktivasi, penonaktifan, dan penggunaan Account Type harus dicatat secara lengkap dalam Audit Trail. |

---

# 9.6.3.4.4 Business Rules

| Business Rule ID | Description                                                                                                                                        |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-AT-001     | Setiap Account Type harus memiliki kode dan nama yang unik dalam ruang lingkup Chart of Accounts yang sama.                                        |
| BRule-AT-002     | Setiap Account Master wajib memiliki tepat satu Account Type aktif.                                                                                |
| BRule-AT-003     | Hanya Account Type dengan perilaku **Posting Enabled** yang dapat menerima jurnal secara langsung.                                                 |
| BRule-AT-004     | Account Type **Header** dan **Summary** tidak boleh menerima transaksi atau jurnal secara langsung.                                                |
| BRule-AT-005     | Account Type **Control** hanya dapat digunakan oleh modul yang ditentukan sesuai konfigurasi (misalnya Accounts Receivable atau Accounts Payable). |
| BRule-AT-006     | Perubahan Account Type tidak boleh mengubah perilaku transaksi historis yang telah diposting.                                                      |
| BRule-AT-007     | Account Type yang masih digunakan oleh Account Master aktif tidak boleh dihapus secara permanen.                                                   |
| BRule-AT-008     | Seluruh perubahan terhadap Account Type harus terdokumentasi dalam Audit Trail.                                                                    |

---

# 9.6.3.4.5 Data Attributes

| Attribute                    | Description                                                 |
| ---------------------------- | ----------------------------------------------------------- |
| Account Type ID              | Identitas unik sistem.                                      |
| Type Code                    | Kode Account Type.                                          |
| Type Name                    | Nama Account Type.                                          |
| Description                  | Deskripsi bisnis.                                           |
| Posting Allowed              | Menentukan apakah akun dapat menerima posting langsung.     |
| Manual Journal Allowed       | Menentukan apakah jurnal manual diperbolehkan.              |
| System Managed               | Menentukan apakah akun hanya dapat dikelola sistem.         |
| Allocation Enabled           | Menentukan apakah akun dapat digunakan pada proses alokasi. |
| Financial Reporting Behavior | Perilaku akun dalam pelaporan.                              |
| Chart of Accounts ID         | Referensi ke Chart of Accounts Master.                      |
| Effective From               | Tanggal mulai berlaku.                                      |
| Effective To                 | Tanggal akhir berlaku.                                      |
| Version                      | Nomor versi.                                                |
| Approval Status              | Draft, Pending Approval, Approved, Published, Archived.     |
| Status                       | Active, Suspended, Inactive, Archived.                      |
| Remarks                      | Catatan bisnis.                                             |
| Created By / Created Date    | Informasi pembuatan data.                                   |
| Updated By / Updated Date    | Informasi perubahan terakhir.                               |

---

# 9.6.3.4.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                   |
| -------------- | ------------------ | ----------- | -------------------------------- |
| BR-331         | BO-009             | SH-003      | Account Type Administration      |
| BR-332         | BO-011             | SH-003      | Standard Account Type Management |
| BR-333         | BO-011             | SH-003      | Custom Account Type Management   |
| BR-334         | BO-011             | SH-003      | Posting Behavior Configuration   |
| BR-335         | BO-011             | SH-003      | Organizational Assignment        |
| BR-336         | BO-029             | SH-010      | Effective Date Governance        |
| BR-337         | BO-029             | SH-010      | Version Management               |
| BR-338         | BO-029             | SH-010      | Dependency Validation            |
| BR-339         | BO-029             | SH-003      | Approval Workflow                |
| BR-340         | BO-029             | SH-010      | Audit & Compliance               |

---

# 9.6.3.4.7 Summary

Account Type mendefinisikan perilaku operasional setiap akun dalam sistem ERP, seperti kemampuan menerima posting, menjadi akun kontrol, mendukung proses alokasi, atau hanya berfungsi sebagai akun ringkasan. Dengan pemisahan yang jelas antara **Account Category**, **Account Class**, dan **Account Type**, organisasi memperoleh model Chart of Accounts yang lebih modular, mudah dipelihara, dan mampu mendukung otomasi proses akuntansi lintas modul.

Dukungan terhadap *effective dating*, *versioning*, *approval workflow*, *dependency validation*, dan *audit trail* memastikan bahwa perubahan pada Account Type tetap terkendali, terdokumentasi, dan tidak memengaruhi integritas transaksi historis.
