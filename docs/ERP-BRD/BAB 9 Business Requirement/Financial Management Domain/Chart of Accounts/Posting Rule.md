# 9.6.3.16 Posting Rule Requirements

---

# 9.6.3.16.1 Overview

Posting Rule merupakan objek konfigurasi bisnis yang mendefinisikan bagaimana suatu jenis transaksi bisnis (invoice, payment, goods receipt, journal manual, dan sejenisnya) diterjemahkan secara otomatis menjadi entri jurnal debit-kredit ke General Ledger. Posting Rule menjembatani proses transaksional pada modul operasional dengan pencatatan akuntansi, sehingga memastikan setiap transaksi tercatat secara konsisten, akurat, dan sesuai dengan struktur Chart of Accounts serta lingkup organisasi yang berlaku, tanpa memerlukan perubahan source code ketika terjadi perubahan kebijakan akuntansi maupun perluasan struktur organisasi.

---

# 9.6.3.16.2 Business Capability

Posting Rule menyediakan kapabilitas berikut:

* Transaction Type to Posting Rule Mapping.
* Automated Debit-Credit Journal Generation.
* Multi-Dimensional Account Determination berbasis Organization Tree (Holding, Company, Business Unit, Branch, Department, Warehouse, Project Site).
* Configurable Posting Templates.
* Conditional Posting Logic berbasis parameter transaksi.
* Multi-Currency Posting Support.
* Posting Rule Versioning & Effective Dating.
* Approval Workflow untuk perubahan Posting Rule.
* Posting Simulation & Preview sebelum posting aktual.
* Suspense Account & Exception Handling.
* Multi-Organization Posting Rule Scope.
* Audit Trail atas seluruh perubahan Posting Rule.

---

# 9.6.3.16.3 Business Requirement Catalogue

## BR-456 – Posting Rule Definition

| Attribute           | Description                                                                                                                                                    |
| -------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-456                                                                                                                                                        |
| Requirement Name    | Posting Rule Definition                                                                                                                                       |
| Category            | Posting Rule                                                                                                                                                  |
| Priority            | Critical                                                                                                                                                      |
| Business Objective  | BO-009, BO-022                                                                                                                                                |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                              |
| Description         | Sistem harus menyediakan Posting Rule sebagai objek konfigurasi bisnis yang mendefinisikan logika penerjemahan suatu jenis transaksi menjadi entri jurnal debit-kredit ke General Ledger secara otomatis. |
| Business Rationale  | Otomasi penjurnalan diperlukan agar setiap transaksi bisnis tercatat secara konsisten ke buku besar tanpa intervensi manual yang rentan kesalahan.             |
| Business Value      | Mengurangi risiko kesalahan pencatatan manual dan mempercepat proses tutup buku (closing) lintas departemen.                                                  |
| Acceptance Criteria | Administrator dapat membuat, mengubah, menonaktifkan, dan mengelola Posting Rule sesuai kewenangan, serta sistem menghasilkan jurnal balanced secara otomatis. |

---

## BR-457 – Transaction Type to Posting Rule Mapping

| Attribute        | Description                                                                                                                                       |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-457                                                                                                                                            |
| Requirement Name | Transaction Type to Posting Rule Mapping                                                                                                          |
| Priority         | Critical                                                                                                                                          |
| Description      | Sistem harus memungkinkan setiap Transaction Type (invoice, payment, goods receipt, dsb.) dipetakan ke satu atau lebih Posting Rule berdasarkan lingkup organisasi. |
| Business Value   | Menjamin setiap jenis transaksi memiliki jalur penjurnalan yang jelas dan dapat ditelusuri.                                                        |

---

## BR-458 – Debit-Credit Account Determination Logic

| Attribute        | Description                                                                                                                                                    |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-458                                                                                                                                                          |
| Requirement Name | Debit-Credit Account Determination Logic                                                                                                                       |
| Priority         | Critical                                                                                                                                                        |
| Description      | Sistem harus menyediakan logika penentuan akun debit dan kredit secara konfigurasi, berdasarkan kombinasi Transaction Type, Account Category, dan atribut transaksi lainnya. |
| Acceptance Criteria | Administrator dapat mengonfigurasi aturan penentuan akun tanpa memerlukan perubahan source code.                                                            |

---

## BR-459 – Multi-Dimensional Posting berbasis Organization Tree

| Attribute        | Description                                                                                                                                                                  |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-459                                                                                                                                                                       |
| Requirement Name | Multi-Dimensional Posting berbasis Organization Tree                                                                                                                        |
| Priority         | High                                                                                                                                                                         |
| Description      | Sistem harus mendukung posting jurnal yang mempertimbangkan dimensi organisasi (Company, Business Unit, Branch, Department, Warehouse, Project Site) sebagai bagian dari lingkup Posting Rule. |
| Business Value   | Mendukung pelaporan keuangan yang granular per unit organisasi tanpa duplikasi konfigurasi.                                                                                  |

---

## BR-460 – Conditional Posting Rule

| Attribute        | Description                                                                                                                                              |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-460                                                                                                                                                    |
| Requirement Name | Conditional Posting Rule                                                                                                                                 |
| Priority         | High                                                                                                                                                      |
| Description      | Sistem harus mendukung logika kondisional pada Posting Rule (misalnya berdasarkan nilai nominal, jenis mata uang, kategori vendor/pelanggan) untuk menentukan variasi jurnal yang dihasilkan. |
| Acceptance Criteria | Sistem dapat menerapkan Posting Rule yang berbeda pada transaksi sejenis berdasarkan kondisi yang dikonfigurasi.                                        |

---

## BR-461 – Multi-Currency Posting Rule

| Attribute        | Description                                                                                                                                        |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-461                                                                                                                                              |
| Requirement Name | Multi-Currency Posting Rule                                                                                                                        |
| Priority         | High                                                                                                                                                |
| Description      | Sistem harus mendukung posting jurnal untuk transaksi lintas mata uang, termasuk konversi ke mata uang fungsional dan pencatatan selisih kurs (realized/unrealized gain-loss). |
| Business Value   | Menjamin akurasi laporan keuangan konsolidasi pada organisasi yang beroperasi dengan multi mata uang.                                              |

---

## BR-462 – Posting Rule Versioning & Effective Dating

| Attribute        | Description                                                                                                                        |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-462                                                                                                                             |
| Requirement Name | Posting Rule Versioning & Effective Dating                                                                                        |
| Priority         | High                                                                                                                               |
| Description      | Sistem harus mendukung pengelolaan versi Posting Rule dengan periode efektif (effective date) agar perubahan kebijakan akuntansi dapat diterapkan tanpa mengganggu transaksi historis. |

---

## BR-463 – Posting Rule Approval Workflow

| Attribute        | Description                                                                                                                                    |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-463                                                                                                                                          |
| Requirement Name | Posting Rule Approval Workflow                                                                                                                 |
| Priority         | Critical                                                                                                                                        |
| Description      | Sistem harus menyediakan alur persetujuan (approval workflow) yang configurable sebelum suatu Posting Rule baru atau perubahannya berstatus Published dan berlaku efektif. |
| Business Value   | Menjamin tata kelola dan pengendalian internal atas perubahan logika penjurnalan yang berdampak pada laporan keuangan.                          |

---

## BR-464 – Posting Rule Simulation & Preview

| Attribute        | Description                                                                                                                     |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-464                                                                                                                            |
| Requirement Name | Posting Rule Simulation & Preview                                                                                                |
| Priority         | Medium                                                                                                                            |
| Description      | Sistem harus menyediakan fitur simulasi yang menampilkan pratinjau entri jurnal debit-kredit yang akan dihasilkan sebelum Posting Rule diterapkan pada transaksi aktual. |
| Acceptance Criteria | Pengguna dapat menjalankan simulasi posting dan meninjau hasil jurnal tanpa memengaruhi General Ledger.                        |

---

## BR-465 – Exception & Suspense Account Handling

| Attribute        | Description                                                                                                                                  |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-465                                                                                                                                        |
| Requirement Name | Exception & Suspense Account Handling                                                                                                        |
| Priority         | High                                                                                                                                          |
| Description      | Sistem harus mengarahkan transaksi yang tidak memenuhi kondisi Posting Rule manapun ke Suspense Account serta memicu notifikasi eksepsi kepada pihak berwenang. |
| Business Value   | Mencegah transaksi tidak terposting atau hilang, sekaligus menjaga integritas dan kelengkapan laporan keuangan.                               |

---

# 9.6.3.16.4 Business Rules

| Business Rule ID | Description                                                                                                                                             |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-PR-001      | Setiap Posting Rule harus menghasilkan jurnal yang balanced (total debit sama dengan total kredit) sebelum dapat diposting ke General Ledger.           |
| BRule-PR-002      | Setiap Transaction Type hanya dapat memiliki satu Posting Rule aktif dalam lingkup organisasi yang sama pada satu periode efektif tertentu.             |
| BRule-PR-003      | Posting Rule yang berstatus Published tidak dapat dihapus; perubahan hanya dapat dilakukan melalui penerbitan versi baru dengan periode efektif berikutnya. |
| BRule-PR-004      | Perubahan Posting Rule yang berdampak pada akun General Ledger wajib melalui approval workflow sebelum berstatus Published dan berlaku efektif.         |
| BRule-PR-005      | Sistem harus mencatat audit trail atas seluruh perubahan Posting Rule, termasuk riwayat versi sebelumnya dan pihak yang melakukan perubahan.            |
| BRule-PR-006      | Apabila tidak ada Posting Rule yang memenuhi kondisi transaksi, sistem harus mengarahkan transaksi ke Suspense Account dan memicu notifikasi eksepsi.   |
| BRule-PR-007      | Posting Rule harus divalidasi terhadap Chart of Accounts dan Account Category yang berlaku pada lingkup organisasi terkait sebelum dapat dipublikasikan. |

---

# 9.6.3.16.5 Data Attributes

| Attribute                     | Description                                                    |
| ------------------------------ | --------------------------------------------------------------- |
| Posting Rule ID               | Identitas unik sistem.                                          |
| Posting Rule Code             | Kode Posting Rule.                                              |
| Posting Rule Name             | Nama Posting Rule.                                               |
| Transaction Type              | Jenis transaksi bisnis yang dipetakan (invoice, payment, dsb.). |
| Organization Scope            | Lingkup organisasi tempat Posting Rule berlaku.                 |
| Debit Account Determination   | Aturan penentuan akun debit.                                    |
| Credit Account Determination  | Aturan penentuan akun kredit.                                   |
| Posting Condition              | Kondisi yang menentukan variasi jurnal yang dihasilkan.        |
| Currency Handling              | Ketentuan konversi dan pencatatan selisih kurs.                |
| Effective From / To           | Periode berlaku.                                                |
| Version                       | Nomor versi.                                                     |
| Approval Status               | Draft, Pending Approval, Approved, Published, Archived.        |
| Status                        | Active, Suspended, Inactive, Archived.                          |
| Created By / Created Date     | Informasi pembuatan data.                                        |
| Updated By / Updated Date     | Informasi perubahan terakhir.                                    |

---

# 9.6.3.16.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                          |
| --------------- | ------------------- | ------------ | ---------------------------------------- |
| BR-456          | BO-009, BO-022      | SH-003       | Posting Rule Administration              |
| BR-457          | BO-009              | SH-003       | Transaction Type Mapping Configuration   |
| BR-458          | BO-022              | SH-003       | Account Determination Configuration      |
| BR-459          | BO-009              | SH-003       | Multi-Dimensional Journal Posting        |
| BR-460          | BO-009              | SH-003       | Conditional Posting Configuration        |
| BR-461          | BO-022              | SH-003       | Multi-Currency Journal Posting           |
| BR-462          | BO-022              | SH-003       | Posting Rule Version Management          |
| BR-463          | BO-022              | SH-003       | Posting Rule Approval Administration     |
| BR-464          | BO-009              | SH-003       | Posting Simulation & Preview             |
| BR-465          | BO-022              | SH-003       | Exception & Suspense Account Management  |

---

# 9.6.3.16.7 Summary

Posting Rule menyediakan mekanisme otomasi penjurnalan yang menjembatani transaksi operasional dengan pencatatan akuntansi pada General Ledger secara konsisten, akurat, dan configurable. Posting Rule menjadi fondasi bagi keandalan proses financial closing dan akurasi laporan keuangan pada Financial Management Domain.
