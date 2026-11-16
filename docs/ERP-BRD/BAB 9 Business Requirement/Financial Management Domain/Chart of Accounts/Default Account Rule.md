# 9.6.3.15 Default Account Rule Requirements

---

# 9.6.3.15.1 Overview

Default Account Rule merupakan mekanisme konfigurasi yang mendefinisikan akun bawaan (default account) yang digunakan oleh sistem ketika suatu transaksi tidak secara eksplisit menentukan akun tujuan, misalnya akun untuk menampung selisih pembulatan, transaksi yang belum teridentifikasi (suspense), selisih kurs, biaya bank, maupun pajak. Default Account Rule memastikan setiap transaksi tetap dapat diposting ke Chart of Accounts secara konsisten tanpa memerlukan intervensi manual, sekaligus menjaga akurasi dan keterlacakan laporan keuangan pada seluruh unit organisasi.

---

# 9.6.3.15.2 Business Capability

Default Account Rule menyediakan kapabilitas berikut:

* Default Account Rule Definition.
* Rounding Difference Account Configuration.
* Suspense Account Configuration.
* Scope-Based Default Account Rule Assignment.
* Transaction Type to Default Account Mapping.
* Multi-Currency Realized/Unrealized Gain-Loss Default Account.
* Bank Charges and Fee Default Account.
* Tax Default Account Mapping.
* Rule Inheritance melalui Organization Tree.
* Effective Dating.
* Versioning.
* Approval Workflow.
* Audit Trail.

---

# 9.6.3.15.3 Business Requirement Catalogue

## BR-446 – Default Account Rule Definition

| Attribute           | Description                                                                                                                                                                       |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-446                                                                                                                                                                             |
| Requirement Name    | Default Account Rule Definition                                                                                                                                                    |
| Category            | Default Account Rule                                                                                                                                                               |
| Priority            | Critical                                                                                                                                                                            |
| Business Objective  | BO-009, BO-014                                                                                                                                                                      |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                                   |
| Description         | Sistem harus menyediakan Default Account Rule sebagai objek bisnis untuk mendefinisikan akun bawaan yang digunakan secara otomatis ketika transaksi tidak menentukan akun secara eksplisit. |
| Business Rationale  | Transaksi lintas modul (Sales, Purchasing, Inventory, Manufacturing, Asset) sering kali tidak menentukan akun secara manual sehingga dibutuhkan aturan default agar posting tetap konsisten. |
| Business Value      | Mengurangi kegagalan posting transaksi dan memastikan otomatisasi jurnal lintas departemen berjalan tanpa hambatan.                                                                |
| Acceptance Criteria | Administrator dapat membuat, mengubah, menonaktifkan, dan mengelola Default Account Rule sesuai kewenangan dan ruang lingkup organisasi.                                           |

---

## BR-447 – Rounding Difference Account Configuration

| Attribute         | Description                                                                                                                                          |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-447                                                                                                                                                |
| Requirement Name  | Rounding Difference Account Configuration                                                                                                            |
| Priority          | High                                                                                                                                                  |
| Description       | Sistem harus menyediakan konfigurasi akun default untuk menampung selisih pembulatan (rounding difference) yang timbul akibat perhitungan transaksi. |
| Business Value    | Menjaga keseimbangan jurnal secara otomatis tanpa memerlukan penyesuaian manual berulang.                                                            |

---

## BR-448 – Suspense Account Configuration

| Attribute         | Description                                                                                                                                                     |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-448                                                                                                                                                            |
| Requirement Name  | Suspense Account Configuration                                                                                                                                    |
| Priority          | Critical                                                                                                                                                          |
| Description       | Sistem harus menyediakan konfigurasi akun suspense yang digunakan sementara untuk menampung transaksi yang belum dapat diklasifikasikan ke akun definitif.       |
| Acceptance Criteria | Sistem harus dapat menampilkan daftar transaksi yang tercatat pada Suspense Account untuk ditindaklanjuti oleh Accounting Department.                          |

---

## BR-449 – Scope-Based Default Account Rule Assignment

| Attribute         | Description                                                                                                                                                             |
| ------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-449                                                                                                                                                                   |
| Requirement Name  | Scope-Based Default Account Rule Assignment                                                                                                                             |
| Priority          | Critical                                                                                                                                                                 |
| Description       | Sistem harus mendukung penetapan Default Account Rule berdasarkan ruang lingkup organisasi (Holding, Sub Holding, Company, Business Unit, Branch, Department, Warehouse, Project Site) dengan mekanisme inheritance dan override. |
| Business Value    | Memungkinkan setiap entitas organisasi memiliki aturan akun default yang sesuai kebutuhan lokal tanpa mengubah source code.                                             |

---

## BR-450 – Transaction Type to Default Account Mapping

| Attribute         | Description                                                                                                                                                 |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-450                                                                                                                                                       |
| Requirement Name  | Transaction Type to Default Account Mapping                                                                                                                 |
| Priority          | High                                                                                                                                                         |
| Description       | Sistem harus mendukung pemetaan Default Account Rule berdasarkan jenis transaksi (misalnya Sales Invoice, Purchase Invoice, Inventory Adjustment, Payroll). |
| Acceptance Criteria | Sistem harus dapat menerapkan akun default yang berbeda untuk setiap jenis transaksi sesuai konfigurasi yang ditetapkan.                                  |

---

## BR-451 – Currency Realized/Unrealized Gain-Loss Default Account

| Attribute         | Description                                                                                                                                                        |
| ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-451                                                                                                                                                              |
| Requirement Name  | Currency Realized/Unrealized Gain-Loss Default Account                                                                                                             |
| Priority          | High                                                                                                                                                                |
| Description       | Sistem harus menyediakan konfigurasi akun default untuk mencatat selisih kurs realized dan unrealized pada transaksi multi currency secara otomatis.               |
| Business Value    | Mendukung akurasi konsolidasi laporan keuangan lintas mata uang dan lintas perusahaan.                                                                             |

---

## BR-452 – Bank Charges and Fee Default Account

| Attribute         | Description                                                                                                                    |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-452                                                                                                                            |
| Requirement Name  | Bank Charges and Fee Default Account                                                                                             |
| Priority          | Medium                                                                                                                            |
| Description       | Sistem harus menyediakan konfigurasi akun default untuk mencatat biaya administrasi bank dan biaya transaksi lain secara otomatis pada proses rekonsiliasi bank. |

---

## BR-453 – Tax Default Account Mapping

| Attribute         | Description                                                                                                                              |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-453                                                                                                                                      |
| Requirement Name  | Tax Default Account Mapping                                                                                                                 |
| Priority          | High                                                                                                                                        |
| Description       | Sistem harus menyediakan pemetaan akun default untuk setiap jenis pajak (misalnya PPN Masukan, PPN Keluaran, PPh) berdasarkan ruang lingkup organisasi dan yurisdiksi. |
| Acceptance Criteria | Sistem harus dapat menerapkan akun default pajak sesuai jenis pajak dan wilayah operasional entitas terkait.                              |

---

## BR-454 – Approval Workflow for Default Account Rule Changes

| Attribute         | Description                                                                                                                       |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-454                                                                                                                            |
| Requirement Name  | Approval Workflow for Default Account Rule Changes                                                                               |
| Priority          | High                                                                                                                              |
| Description       | Sistem harus mendukung alur persetujuan (approval workflow) yang configurable sebelum perubahan Default Account Rule berlaku efektif. |
| Business Value    | Menjaga tata kelola dan mencegah perubahan akun default yang tidak sah atau tidak sesuai kebijakan akuntansi.                     |

---

## BR-455 – Default Account Rule Audit Trail and Versioning

| Attribute         | Description                                                                                                                                     |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-455                                                                                                                                           |
| Requirement Name  | Default Account Rule Audit Trail and Versioning                                                                                                  |
| Priority          | Medium                                                                                                                                            |
| Description       | Sistem harus mencatat seluruh riwayat perubahan Default Account Rule, termasuk versi sebelumnya, pihak yang mengubah, dan waktu perubahan.       |
| Acceptance Criteria | Sistem harus dapat menampilkan histori versi Default Account Rule secara lengkap untuk kebutuhan audit.                                        |

---

# 9.6.3.15.4 Business Rules

| Business Rule ID | Description                                                                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| BRule-DAR-001     | Setiap organization scope wajib memiliki minimal satu Default Account Rule aktif untuk kategori Suspense Account.                                |
| BRule-DAR-002     | Default Account Rule tidak dapat dihapus apabila masih digunakan secara aktif oleh transaksi atau scope tertentu.                                |
| BRule-DAR-003     | Perubahan Default Account Rule wajib melalui approval workflow sebelum berlaku efektif pada scope terkait.                                       |
| BRule-DAR-004     | Default Account Rule mengikuti mekanisme inheritance pada Organization Tree; child scope mewarisi rule dari parent kecuali terdapat override eksplisit. |
| BRule-DAR-005     | Akun yang ditetapkan sebagai Default Account harus berstatus Active dan Postable pada Chart of Accounts.                                         |
| BRule-DAR-006     | Sistem harus mencegah konflik apabila lebih dari satu Default Account Rule berlaku pada scope, jenis transaksi, dan periode efektif yang sama.   |
| BRule-DAR-007     | Setiap perubahan Default Account Rule harus tercatat pada audit trail beserta nomor versi dan status persetujuan.                                |

---

# 9.6.3.15.5 Data Attributes

| Attribute                  | Description                                                                                |
| ---------------------------- | --------------------------------------------------------------------------------------------- |
| Default Account Rule ID     | Identitas unik sistem.                                                                       |
| Rule Code / Name            | Kode dan nama Default Account Rule.                                                          |
| Organization Scope          | Ruang lingkup organisasi tempat rule berlaku (Holding, Company, Business Unit, Branch, dsb.). |
| Rule Type                   | Rounding, Suspense, Bank Charges, Tax, Realized/Unrealized Gain-Loss, dan tipe lainnya.       |
| Transaction Type            | Jenis transaksi pemicu penerapan Default Account Rule.                                        |
| Default Account             | Referensi akun pada Chart of Accounts yang digunakan sebagai akun bawaan.                     |
| Currency                    | Mata uang terkait rule, jika berlaku.                                                         |
| Effective From / To         | Periode berlaku Default Account Rule.                                                         |
| Version                     | Nomor versi Default Account Rule.                                                             |
| Approval Status             | Draft, Pending Approval, Approved, Published, Archived.                                       |
| Status                      | Active, Suspended, Inactive, Archived.                                                        |
| Created By / Created Date   | Informasi pembuatan data.                                                                     |
| Updated By / Updated Date   | Informasi perubahan terakhir.                                                                 |

---

# 9.6.3.15.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                          |
| --------------- | -------------------- | ------------- | ------------------------------------------ |
| BR-446         | BO-009, BO-014       | SH-003        | Default Account Rule Administration       |
| BR-447         | BO-009               | SH-003        | Rounding Difference Handling               |
| BR-448         | BO-009               | SH-028        | Suspense Account Monitoring                |
| BR-449         | BO-009, BO-015       | SH-003        | Scope-Based Rule Assignment                |
| BR-450         | BO-009               | SH-028        | Transaction Posting Automation             |
| BR-451         | BO-009, BO-021       | SH-003        | Multi-Currency Revaluation                 |
| BR-452         | BO-009               | SH-027        | Bank Reconciliation                        |
| BR-453         | BO-009               | SH-028        | Tax Posting Automation                     |
| BR-454         | BO-009               | SH-003        | Default Account Rule Approval              |
| BR-455         | BO-009, BO-028       | SH-010        | Default Account Rule Audit                 |

---

# 9.6.3.15.7 Summary

Default Account Rule menyediakan mekanisme konfigurasi akun bawaan yang mendukung otomatisasi posting transaksi lintas departemen ketika akun tidak ditentukan secara eksplisit. Default Account Rule menjadi fondasi bagi konsistensi dan akurasi jurnal, sekaligus melengkapi requirement lain yang terkait pada Financial Management Domain.
