# 9.6.3.23 Intercompany Mapping Requirements

---

# 9.6.3.23.1 Overview

Intercompany Mapping mendefinisikan pemetaan akun dan aturan eliminasi untuk transaksi antar perusahaan dalam satu grup usaha (intercompany transaction), mencakup pasangan entitas (trading partner), pemetaan akun Due To/Due From, serta konfigurasi elimination rule. Intercompany Mapping menjadi dasar bagi proses Financial Consolidation agar saldo dan transaksi resiprokal antar entitas dalam struktur Holding, Sub Holding, Company, hingga Business Unit dapat diidentifikasi, direkonsiliasi, dan dieliminasi secara akurat, konsisten dengan prinsip Organization Tree First dan Scope-Based Everything.

---

# 9.6.3.23.2 Business Capability

Intercompany Mapping menyediakan kapabilitas berikut:

* Intercompany Account Mapping Definition.
* Intercompany Trading Partner Configuration.
* Due To / Due From Account Pairing.
* Elimination Rule Configuration.
* Intercompany Transaction Matching & Reconciliation.
* Intercompany Balance Confirmation.
* Multi-Currency Intercompany Support.
* Approval Workflow.
* Effective Dating.
* Versioning.
* Audit Trail.
* Consolidation Reporting Integration.

---

# 9.6.3.23.3 Business Requirement Catalogue

## BR-526 – Intercompany Account Mapping Definition

| Attribute           | Description                                                                                                                                                                                             |
| -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-526                                                                                                                                                                                                   |
| Requirement Name    | Intercompany Account Mapping Definition                                                                                                                                                                 |
| Category            | Intercompany Mapping                                                                                                                                                                                     |
| Priority            | Critical                                                                                                                                                                                                 |
| Business Objective  | BO-020                                                                                                                                                                                                   |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                                                        |
| Description         | Sistem harus menyediakan Intercompany Mapping sebagai objek bisnis untuk mendefinisikan pemetaan akun antar entitas dalam satu grup usaha, sebagai dasar proses eliminasi pada Financial Consolidation. |
| Business Rationale  | Transaksi antar entitas dalam satu grup usaha memerlukan pemetaan akun yang konsisten agar saldo resiprokal dapat diidentifikasi dan dieliminasi secara akurat pada saat konsolidasi laporan keuangan.  |
| Business Value      | Menyediakan fondasi pemetaan akun intercompany yang konsisten dan dapat digunakan berulang pada seluruh siklus konsolidasi keuangan grup.                                                              |
| Acceptance Criteria | Administrator dapat membuat, mengubah, dan mengelola Intercompany Mapping antar pasangan entitas sesuai kewenangan yang dimiliki.                                                                       |

---

## BR-527 – Intercompany Trading Partner Configuration

| Attribute           | Description                                                                                                                                                        |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-527                                                                                                                                                             |
| Requirement Name    | Intercompany Trading Partner Configuration                                                                                                                        |
| Priority            | High                                                                                                                                                               |
| Description         | Sistem harus memungkinkan konfigurasi Trading Partner (entitas pasangan intercompany) untuk setiap entitas dalam struktur organisasi, termasuk penetapan pasangan entitas yang diperbolehkan bertransaksi intercompany. |
| Acceptance Criteria | Setiap entitas dapat memiliki daftar trading partner yang terdefinisi dan dapat diverifikasi pada saat transaksi intercompany dibuat.                              |

---

## BR-528 – Due To / Due From Account Pairing

| Attribute      | Description                                                                                                                                                                    |
| --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-528                                                                                                                                                                         |
| Requirement Name | Due To / Due From Account Pairing                                                                                                                                            |
| Priority       | Critical                                                                                                                                                                       |
| Description    | Sistem harus mendukung pemetaan akun Due To dan Due From secara berpasangan antar entitas agar setiap transaksi intercompany menghasilkan jurnal timbal balik yang seimbang antar entitas terkait. |
| Business Value | Memastikan saldo intercompany antar entitas selalu dapat direkonsiliasi secara konsisten.                                                                                     |

---

## BR-529 – Elimination Rule Configuration

| Attribute           | Description                                                                                                                                                              |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-529                                                                                                                                                                   |
| Requirement Name    | Elimination Rule Configuration                                                                                                                                           |
| Priority            | Critical                                                                                                                                                                 |
| Description         | Sistem harus menyediakan konfigurasi aturan eliminasi (elimination rule) yang menentukan akun, persentase kepemilikan, dan metode eliminasi yang diterapkan pada saldo dan transaksi intercompany saat proses konsolidasi. |
| Acceptance Criteria | Elimination rule dapat dikonfigurasi per pasangan entitas dan diterapkan secara otomatis pada proses konsolidasi.                                                        |

---

## BR-530 – Intercompany Transaction Matching & Reconciliation

| Attribute      | Description                                                                                                                                                     |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-530                                                                                                                                                          |
| Requirement Name | Intercompany Transaction Matching & Reconciliation                                                                                                           |
| Priority       | High                                                                                                                                                             |
| Description    | Sistem harus menyediakan mekanisme pencocokan (matching) transaksi intercompany antar entitas guna mengidentifikasi selisih saldo sebelum proses eliminasi dijalankan. |
| Business Value | Mengurangi risiko selisih hasil konsolidasi akibat transaksi intercompany yang tidak sepadan antar entitas.                                                     |

---

## BR-531 – Intercompany Balance Confirmation

| Attribute           | Description                                                                                                                            |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-531                                                                                                                                 |
| Requirement Name    | Intercompany Balance Confirmation                                                                                                     |
| Priority            | Medium                                                                                                                                 |
| Description         | Sistem harus mendukung proses konfirmasi saldo intercompany antar entitas pada periode tertentu sebagai bagian dari kontrol sebelum proses konsolidasi dijalankan. |
| Acceptance Criteria | Saldo intercompany dapat dikonfirmasi oleh masing-masing entitas terkait dan status konfirmasi tercatat dalam sistem.                 |

---

## BR-532 – Multi-Currency Intercompany Mapping

| Attribute      | Description                                                                                                                                       |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-532                                                                                                                                          |
| Requirement Name | Multi-Currency Intercompany Mapping                                                                                                           |
| Priority       | High                                                                                                                                             |
| Description    | Sistem harus mendukung pemetaan intercompany antar entitas yang menggunakan mata uang fungsional berbeda, termasuk penerapan kurs konsolidasi yang telah ditetapkan. |
| Business Value | Mendukung konsolidasi keuangan grup usaha yang beroperasi lintas mata uang.                                                                     |

---

## BR-533 – Intercompany Mapping Approval Workflow

| Attribute           | Description                                                                                                                                          |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-533                                                                                                                                               |
| Requirement Name    | Intercompany Mapping Approval Workflow                                                                                                              |
| Priority            | High                                                                                                                                                 |
| Description         | Sistem harus menyediakan approval workflow yang configurable untuk setiap perubahan pada Intercompany Mapping sebelum berstatus Published dan dapat digunakan pada proses konsolidasi. |
| Acceptance Criteria | Perubahan Intercompany Mapping tidak dapat diterapkan tanpa melalui persetujuan sesuai workflow yang ditetapkan.                                     |

---

## BR-534 – Intercompany Mapping Versioning & Effective Dating

| Attribute      | Description                                                                                                                                                        |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-534                                                                                                                                                            |
| Requirement Name | Intercompany Mapping Versioning & Effective Dating                                                                                                             |
| Priority       | Medium                                                                                                                                                            |
| Description    | Sistem harus mendukung version dan effective dating pada Intercompany Mapping sehingga perubahan pemetaan dapat diberlakukan pada periode tertentu tanpa mempengaruhi data historis. |
| Business Value | Menjaga konsistensi hasil konsolidasi periode sebelumnya meskipun terjadi perubahan mapping di kemudian hari.                                                     |

---

## BR-535 – Intercompany Mapping Audit Trail & Reporting Integration

| Attribute           | Description                                                                                                                              |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-535                                                                                                                                   |
| Requirement Name    | Intercompany Mapping Audit Trail & Reporting Integration                                                                               |
| Priority            | Medium                                                                                                                                   |
| Description         | Sistem harus mencatat seluruh perubahan pada Intercompany Mapping dalam audit trail serta menyediakan integrasi data mapping ke laporan dan dashboard konsolidasi. |
| Acceptance Criteria | Seluruh riwayat perubahan Intercompany Mapping dapat ditelusuri dan tersedia sebagai referensi pada laporan konsolidasi.                 |

---

# 9.6.3.23.4 Business Rules

| Business Rule ID | Description                                                                                                                                       |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-ICM-001    | Setiap pasangan entitas intercompany harus memiliki mapping akun Due To dan Due From yang saling berpasangan dan seimbang.                       |
| BRule-ICM-002    | Elimination rule hanya dapat diterapkan pada akun yang telah terdaftar dalam Intercompany Mapping yang berstatus Published.                      |
| BRule-ICM-003    | Saldo intercompany antar entitas harus melalui proses matching dan konfirmasi sebelum proses eliminasi konsolidasi dijalankan.                    |
| BRule-ICM-004    | Perubahan pada Intercompany Mapping yang telah berstatus Published harus dibuat sebagai versi baru dan melalui approval workflow ulang.           |
| BRule-ICM-005    | Transaksi intercompany antar entitas dengan mata uang fungsional berbeda harus dikonversi menggunakan kurs konsolidasi yang ditetapkan pada periode terkait. |
| BRule-ICM-006    | Intercompany Mapping hanya berlaku untuk pasangan entitas yang berada dalam struktur organisasi grup yang sama (within group scope).             |
| BRule-ICM-007    | Setiap perubahan data dan status approval pada Intercompany Mapping harus tercatat dalam audit trail beserta informasi pengguna dan waktu perubahan. |

---

# 9.6.3.23.5 Data Attributes

| Attribute                  | Description                                             |
| --------------------------- | -------------------------------------------------------- |
| Intercompany Mapping ID    | Identitas unik sistem.                                  |
| Source Entity              | Entitas asal transaksi intercompany.                    |
| Counterparty Entity        | Entitas pasangan (trading partner) transaksi intercompany. |
| Source Account             | Akun pada entitas asal (Due From).                      |
| Counterparty Account       | Akun pada entitas pasangan (Due To).                    |
| Elimination Account        | Akun yang digunakan pada saat proses eliminasi konsolidasi. |
| Currency                   | Mata uang transaksi intercompany.                       |
| Effective From / To        | Periode berlaku mapping.                                |
| Version                    | Nomor versi.                                            |
| Approval Status            | Draft, Pending Approval, Approved, Published, Archived. |
| Status                     | Active, Suspended, Inactive, Archived.                  |
| Created By / Created Date  | Informasi pembuatan data.                               |
| Updated By / Updated Date  | Informasi perubahan terakhir.                           |

---

# 9.6.3.23.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                              |
| -------------- | ------------------- | ----------- | -------------------------------------------- |
| BR-526         | BO-020              | SH-003      | Intercompany Mapping Administration          |
| BR-527         | BO-020              | SH-003      | Trading Partner Configuration                |
| BR-528         | BO-020              | SH-003      | Due To / Due From Account Configuration      |
| BR-529         | BO-020              | SH-003      | Elimination Rule Configuration               |
| BR-530         | BO-020              | SH-003      | Intercompany Transaction Matching            |
| BR-531         | BO-020              | SH-003      | Intercompany Balance Confirmation            |
| BR-532         | BO-020              | SH-003      | Multi-Currency Intercompany Processing       |
| BR-533         | BO-020              | SH-003      | Intercompany Mapping Approval                |
| BR-534         | BO-020              | SH-003      | Intercompany Mapping Version Control         |
| BR-535         | BO-020              | SH-003      | Intercompany Mapping Audit & Reporting       |

---

# 9.6.3.23.7 Summary

Intercompany Mapping menyediakan pemetaan akun dan aturan eliminasi bagi transaksi antar perusahaan dalam satu grup usaha, mencakup konfigurasi trading partner, pasangan akun Due To/Due From, elimination rule, matching, konfirmasi saldo, hingga dukungan multi-currency. Intercompany Mapping menjadi fondasi bagi requirement lain yang terkait pada Financial Consolidation dalam Financial Management Domain.
