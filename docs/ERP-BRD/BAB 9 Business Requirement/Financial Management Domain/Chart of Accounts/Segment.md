# 9.6.3.7 Segment Requirements

---

# 9.6.3.7.1 Overview

Segment merupakan komponen struktur Chart of Accounts yang merepresentasikan satu dimensi pengkodean akun, seperti Company, Cost Center, Account, Product, atau Project. Setiap Segment terdiri atas sekumpulan Segment Value yang dikonfigurasi, divalidasi, dan dikelola secara independen. Kombinasi beberapa Segment membentuk Account Combination yang digunakan sebagai unit pencatatan transaksi keuangan. Melalui pendekatan Configuration over Customization, struktur Segment dapat ditambah, diubah, atau disesuaikan tanpa perubahan source code, sehingga mendukung fleksibilitas Chart of Accounts bagi organisasi dengan struktur Holding, Sub Holding, Company, Business Unit, Branch, Department, Warehouse, maupun Project Site.

---

# 9.6.3.7.2 Business Capability

Segment menyediakan kapabilitas berikut:

* Segment Definition and Configuration.
* Multiple Segment Types (Company, Cost Center, Account, Product, Project, Location, Intercompany, dan lainnya).
* Segment Value Master Management.
* Segment Value Hierarchy and Rollup.
* Segment Length and Format Configuration.
* Segment Value Validation Rules.
* Cross-Validation Rules Between Segments.
* Segment Value Effective Dating.
* Segment Value Status Control.
* Segment Security by Organization Scope.
* Segment Mapping and Crosswalk for Reporting.
* Approval Workflow and Audit Trail.

---

# 9.6.3.7.3 Business Requirement Catalogue

## BR-366 – Segment Definition and Configuration

| Attribute           | Description                                                                                                                                                                                    |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID      | BR-366                                                                                                                                                                                          |
| Requirement Name    | Segment Definition and Configuration                                                                                                                                                            |
| Category            | Segment                                                                                                                                                                                         |
| Priority            | Critical                                                                                                                                                                                        |
| Business Objective  | BO-004                                                                                                                                                                                          |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                                                |
| Description         | Sistem harus menyediakan Segment sebagai objek konfigurasi untuk merepresentasikan satu dimensi pengkodean akun (Company, Cost Center, Account, Product, Project, dan sejenisnya) dalam Chart of Accounts, tanpa memerlukan perubahan source code. |
| Business Rationale  | Struktur akun yang bersifat multi-dimensi memerlukan mekanisme konfigurasi terpusat agar setiap entitas organisasi dapat mendefinisikan dimensi pengkodean sesuai kebutuhan bisnisnya.          |
| Business Value      | Menjadikan Segment sebagai satu sumber kebenaran (single source of truth) atas struktur dimensi akun bagi seluruh modul yang menggunakan Chart of Accounts.                                    |
| Acceptance Criteria | Administrator dapat membuat, mengubah, dan mengelola definisi Segment beserta atributnya sesuai kewenangan yang dimiliki.                                                                       |

---

## BR-367 – Segment Value Master Management

| Attribute        | Description                                                                                                                    |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-367                                                                                                                         |
| Requirement Name  | Segment Value Master Management                                                                                                |
| Priority          | Critical                                                                                                                       |
| Description       | Sistem harus menyediakan pengelolaan Segment Value sebagai daftar induk nilai yang sah bagi setiap Segment dalam Chart of Accounts. |
| Business Value    | Menjamin konsistensi nilai Segment yang digunakan dalam Account Combination di seluruh organisasi.                             |

---

## BR-368 – Segment Value Hierarchy and Rollup

| Attribute           | Description                                                                                                                       |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-368                                                                                                                            |
| Requirement Name    | Segment Value Hierarchy and Rollup                                                                                               |
| Priority            | High                                                                                                                              |
| Description         | Sistem harus mendukung penyusunan hierarki Segment Value dengan hubungan induk-anak (parent-child) untuk kebutuhan rollup pelaporan. |
| Acceptance Criteria | Segment Value dapat disusun bertingkat dan hasil rollup konsisten hingga level tertinggi hierarki.                               |

---

## BR-369 – Segment Length and Format Configuration

| Attribute        | Description                                                                                                                   |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-369                                                                                                                        |
| Requirement Name  | Segment Length and Format Configuration                                                                                      |
| Priority          | Critical                                                                                                                     |
| Description       | Sistem harus memungkinkan konfigurasi panjang karakter dan format data (numerik, alfanumerik) untuk setiap Segment secara independen. |

---

## BR-370 – Cross-Validation Rules Between Segments

| Attribute           | Description                                                                                                                                    |
| ------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-370                                                                                                                                        |
| Requirement Name    | Cross-Validation Rules Between Segments                                                                                                      |
| Priority            | Critical                                                                                                                                     |
| Description         | Sistem harus menyediakan aturan validasi silang (cross-validation rule) antar Segment untuk memastikan kombinasi nilai Segment yang dibentuk merupakan Account Combination yang sah. |
| Acceptance Criteria | Kombinasi Segment Value yang tidak memenuhi cross-validation rule ditolak sistem sebelum digunakan pada transaksi.                            |

---

## BR-371 – Segment Value Effective Dating

| Attribute        | Description                                                                                                    |
| ---------------- | ---------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-371                                                                                                          |
| Requirement Name  | Segment Value Effective Dating                                                                                 |
| Priority          | High                                                                                                            |
| Description       | Sistem harus mendukung periode berlaku (effective from/to) bagi setiap Segment Value agar sejalan dengan siklus bisnis organisasi. |

---

## BR-372 – Segment Value Status Control

| Attribute        | Description                                                                                                             |
| ---------------- | --------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-372                                                                                                                 |
| Requirement Name  | Segment Value Status Control                                                                                          |
| Priority          | High                                                                                                                  |
| Description       | Sistem harus menyediakan status pengendalian (Active, Suspended, Inactive, Archived) bagi setiap Segment Value untuk mengatur kelayakan penggunaannya dalam transaksi baru. |

---

## BR-373 – Segment Security by Organization Scope

| Attribute           | Description                                                                                                                             |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID      | BR-373                                                                                                                                  |
| Requirement Name    | Segment Security by Organization Scope                                                                                                 |
| Priority            | Critical                                                                                                                               |
| Description         | Sistem harus membatasi akses pembuatan, perubahan, dan penggunaan Segment Value berdasarkan ruang lingkup (scope) organisasi pengguna. |
| Acceptance Criteria | Pengguna hanya dapat mengakses Segment Value sesuai scope organisasi yang menjadi kewenangannya.                                       |

---

## BR-374 – Segment Mapping and Crosswalk for Reporting

| Attribute        | Description                                                                                                                     |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-374                                                                                                                        |
| Requirement Name  | Segment Mapping and Crosswalk for Reporting                                                                                   |
| Priority          | Medium                                                                                                                       |
| Description       | Sistem harus menyediakan pemetaan (mapping/crosswalk) antar Segment Value untuk kebutuhan konsolidasi pelaporan lintas entitas atau lintas standar akuntansi. |

---

## BR-375 – Segment Approval Workflow and Audit Trail

| Attribute           | Description                                                                                                             |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-375                                                                                                                  |
| Requirement Name    | Segment Approval Workflow and Audit Trail                                                                             |
| Priority            | High                                                                                                                   |
| Description         | Sistem harus menyediakan alur persetujuan (approval workflow) yang configurable serta pencatatan jejak audit (audit trail) atas setiap perubahan definisi Segment dan Segment Value. |
| Business Value      | Menjamin tata kelola, transparansi, dan integritas data pada struktur Segment Chart of Accounts.                        |

---

# 9.6.3.7.4 Business Rules

| Business Rule ID | Description                                                                                                                                       |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-SEG-001    | Setiap Segment harus memiliki kode dan nama yang unik dalam ruang lingkup Chart of Accounts yang sama.                                             |
| BRule-SEG-002    | Setiap Segment Value harus memiliki kode yang unik dalam ruang lingkup Segment yang sama.                                                          |
| BRule-SEG-003    | Kombinasi nilai Segment yang membentuk Account Combination wajib melalui Cross-Validation Rule sebelum dapat digunakan dalam transaksi.            |
| BRule-SEG-004    | Perubahan panjang atau format Segment yang telah memiliki transaksi tidak diperbolehkan, kecuali melalui mekanisme versioning baru.                |
| BRule-SEG-005    | Segment Value dengan status Inactive atau Archived tidak dapat digunakan pada Account Combination baru.                                            |
| BRule-SEG-006    | Segment Value yang memiliki hierarki wajib menghasilkan rollup yang konsisten hingga level tertinggi.                                              |
| BRule-SEG-007    | Akses terhadap Segment Value dibatasi berdasarkan scope organisasi (Organization Tree) yang berlaku bagi pengguna.                                 |

---

# 9.6.3.7.5 Data Attributes

| Attribute                    | Description                                              |
| ------------------------------ | --------------------------------------------------------- |
| Segment ID                    | Identitas unik sistem.                                    |
| Segment Code                  | Kode Segment.                                              |
| Segment Name                  | Nama Segment.                                              |
| Segment Type                  | Company, Cost Center, Account, Product, Project, dan lainnya. |
| Data Type / Format            | Numerik atau alfanumerik.                                 |
| Segment Length                | Panjang karakter yang diizinkan.                          |
| Segment Value ID              | Identitas unik nilai Segment.                             |
| Segment Value Code            | Kode nilai Segment.                                       |
| Parent Segment Value          | Nilai induk dalam hierarki Segment Value.                 |
| Hierarchy Level                | Tingkat kedalaman hierarki Segment Value.                  |
| Effective From / To           | Periode berlaku Segment Value.                             |
| Cross-Validation Rule ID      | Identitas aturan validasi silang antar Segment.            |
| Approval Status               | Draft, Pending Approval, Approved, Published, Archived.   |
| Status                        | Active, Suspended, Inactive, Archived.                     |
| Created By / Created Date     | Informasi pembuatan data.                                  |
| Updated By / Updated Date     | Informasi perubahan terakhir.                              |

---

# 9.6.3.7.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                            |
| -------------- | ------------------- | ----------- | ------------------------------------------ |
| BR-366         | BO-004              | SH-003      | Segment Administration                    |
| BR-367         | BO-004              | SH-003      | Segment Value Administration               |
| BR-368         | BO-004              | SH-003      | Segment Value Hierarchy Maintenance        |
| BR-369         | BO-004              | SH-003      | Segment Configuration Setup                |
| BR-370         | BO-004              | SH-003      | Account Combination Validation             |
| BR-371         | BO-004              | SH-003      | Segment Value Lifecycle Management          |
| BR-372         | BO-004              | SH-003      | Segment Value Status Maintenance            |
| BR-373         | BO-004              | SH-003      | Segment Access Control Administration       |
| BR-374         | BO-004              | SH-003      | Financial Consolidation Reporting          |
| BR-375         | BO-004              | SH-003      | Segment Change Approval and Audit          |

---

# 9.6.3.7.7 Summary

Segment merupakan komponen dasar dalam struktur Chart of Accounts yang merepresentasikan dimensi pengkodean akun secara independen dan configurable. Melalui pengelolaan Segment Value, hierarki, validasi silang, keamanan berbasis scope, serta approval workflow dan audit trail, Segment menjadi fondasi utama pembentukan Account Combination yang konsisten dan dapat dipercaya sebagai single source of truth bagi seluruh proses pencatatan dan pelaporan keuangan pada Financial Management Domain.
