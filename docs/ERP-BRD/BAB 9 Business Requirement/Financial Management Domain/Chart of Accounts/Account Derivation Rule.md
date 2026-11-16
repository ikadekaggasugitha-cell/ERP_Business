# 9.6.3.14 Account Derivation Rule Requirements

---

# 9.6.3.14.1 Overview

Account Derivation Rule merupakan mekanisme konfigurasi yang menentukan akun tujuan (Debit dan/atau Kredit) secara otomatis berdasarkan kondisi transaksi tertentu, seperti jenis transaksi, tipe pelanggan, kategori item, atau kombinasi dimensi bisnis lainnya. Kapabilitas ini mengurangi ketergantungan pada input akun secara manual, menekan potensi kesalahan pencatatan, serta memastikan konsistensi posting jurnal di seluruh entitas organisasi tanpa memerlukan perubahan source code.

---

# 9.6.3.14.2 Business Capability

Account Derivation Rule menyediakan kapabilitas berikut:

* Rule-Based Account Derivation Engine.
* Multi-Condition Criteria Matching (jenis transaksi, tipe pelanggan, kategori item, dan dimensi lainnya).
* Priority-Based Rule Resolution.
* Default/Fallback Account Assignment.
* Scope-Based Rule Configuration mengikuti Organization Tree.
* Multi-Dimensional Account Derivation (Debit dan Kredit secara simultan).
* Rule Simulation & Testing sebelum dipublikasikan.
* Effective Dating.
* Versioning.
* Approval Workflow.
* Audit Trail.
* Integrasi dengan modul transaksi (Sales, Purchasing, Inventory, dan modul lain).

---

# 9.6.3.14.3 Business Requirement Catalogue

## BR-436 – Account Derivation Rule Definition

| Attribute           | Description                                                                                                                                                    |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-436                                                                                                                                                          |
| Requirement Name    | Account Derivation Rule Definition                                                                                                                              |
| Category            | Account Derivation Rule                                                                                                                                         |
| Priority            | Critical                                                                                                                                                        |
| Business Objective  | BO-009                                                                                                                                                          |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                               |
| Description         | Sistem harus menyediakan Account Derivation Rule sebagai objek bisnis untuk mendefinisikan aturan penentuan akun tujuan secara otomatis berdasarkan kondisi transaksi tertentu. |
| Business Rationale  | Penentuan akun secara manual pada setiap transaksi rawan kesalahan dan tidak efisien ketika volume transaksi tinggi serta struktur organisasi kompleks.        |
| Business Value      | Mengurangi input manual dan potensi kesalahan posting akun pada seluruh entitas organisasi.                                                                    |
| Acceptance Criteria | Administrator dapat membuat, mengubah, dan mengelola Account Derivation Rule sesuai kewenangan.                                                                |

---

## BR-437 – Multi-Condition Criteria Configuration

| Attribute       | Description                                                                                                                                       |
| --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-437                                                                                                                                             |
| Requirement Name| Multi-Condition Criteria Configuration                                                                                                            |
| Priority        | Critical                                                                                                                                           |
| Description     | Sistem harus mendukung konfigurasi kriteria kondisi majemuk (transaction type, customer type, item category, project, business unit, dan dimensi lain) dalam satu rule. |
| Business Value  | Memungkinkan derivasi akun yang presisi sesuai konteks bisnis yang bervariasi tanpa perubahan aplikasi.                                           |

---

## BR-438 – Rule Priority & Sequencing

| Attribute       | Description                                                                                                                                   |
| --------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-438                                                                                                                                         |
| Requirement Name| Rule Priority & Sequencing                                                                                                                    |
| Priority        | High                                                                                                                                           |
| Description     | Sistem harus mendukung penetapan prioritas dan urutan eksekusi antar-rule ketika terdapat lebih dari satu rule yang berpotensi cocok pada satu transaksi. |
| Acceptance Criteria | Sistem menerapkan rule dengan prioritas tertinggi dan menghentikan evaluasi rule berikutnya sesuai konfigurasi.                           |

---

## BR-439 – Default/Fallback Account Handling

| Attribute       | Description                                                                                                                             |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID  | BR-439                                                                                                                                   |
| Requirement Name| Default/Fallback Account Handling                                                                                                       |
| Priority        | Critical                                                                                                                                 |
| Description     | Sistem harus menyediakan default account yang digunakan apabila tidak ada satu pun Account Derivation Rule yang cocok dengan kondisi transaksi. |
| Business Value  | Mencegah transaksi gagal diposting akibat tidak ditemukannya akun tujuan yang sesuai.                                                   |

---

## BR-440 – Scope-Based Rule Applicability

| Attribute       | Description                                                                                                                                     |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-440                                                                                                                                           |
| Requirement Name| Scope-Based Rule Applicability                                                                                                                  |
| Priority        | Critical                                                                                                                                         |
| Description     | Sistem harus mendukung penerapan Account Derivation Rule berdasarkan cakupan Organization Tree (Holding, Sub Holding, Company, Business Unit, Branch, dan level lain). |
| Acceptance Criteria | Rule dapat dikonfigurasi berlaku pada level organisasi tertentu beserta seluruh unit turunannya sesuai kebutuhan bisnis.                    |

---

## BR-441 – Multi-Dimensional Account Derivation

| Attribute       | Description                                                                                                                       |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-441                                                                                                                            |
| Requirement Name| Multi-Dimensional Account Derivation                                                                                              |
| Priority        | High                                                                                                                               |
| Description     | Sistem harus mendukung derivasi akun Debit dan Kredit secara simultan dalam satu rule untuk menghasilkan pasangan jurnal yang lengkap. |
| Business Value  | Memastikan konsistensi double-entry pada seluruh transaksi yang menggunakan derivasi akun otomatis.                               |

---

## BR-442 – Rule Simulation & Testing

| Attribute       | Description                                                                                                                     |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-442                                                                                                                         |
| Requirement Name| Rule Simulation & Testing                                                                                                      |
| Priority        | High                                                                                                                            |
| Description     | Sistem harus menyediakan fitur simulasi untuk menguji hasil derivasi akun terhadap contoh data transaksi sebelum rule dipublikasikan. |
| Acceptance Criteria | Pengguna dapat menjalankan simulasi dan melihat hasil akun tujuan sebelum rule berstatus aktif.                            |

---

## BR-443 – Effective Dating & Versioning

| Attribute       | Description                                                                                                       |
| --------------- | ------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-443                                                                                                             |
| Requirement Name| Effective Dating & Versioning                                                                                     |
| Priority        | High                                                                                                                |
| Description     | Sistem harus mendukung periode berlaku dan riwayat versi setiap Account Derivation Rule agar perubahan aturan tetap dapat ditelusuri. |
| Business Value  | Menjaga keakuratan historis derivasi akun sesuai periode transaksi yang bersangkutan.                             |

---

## BR-444 – Approval Workflow for Rule Changes

| Attribute       | Description                                                                                                          |
| --------------- | ------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-444                                                                                                                |
| Requirement Name| Approval Workflow for Rule Changes                                                                                    |
| Priority        | High                                                                                                                  |
| Description     | Sistem harus menerapkan approval workflow yang configurable sebelum Account Derivation Rule baru atau perubahan rule dapat digunakan pada transaksi. |
| Acceptance Criteria | Rule berstatus Pending Approval tidak dapat digunakan hingga disetujui oleh pihak berwenang.                     |

---

## BR-445 – Rule Usage Audit Trail & Traceability

| Attribute       | Description                                                                                                             |
| --------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-445                                                                                                                    |
| Requirement Name| Rule Usage Audit Trail & Traceability                                                                                    |
| Priority        | Critical                                                                                                                  |
| Description     | Sistem harus mencatat riwayat penggunaan, perubahan, dan hasil derivasi akun oleh setiap Account Derivation Rule untuk kebutuhan audit. |
| Business Value  | Mendukung transparansi dan akuntabilitas dalam proses posting akun otomatis lintas entitas.                              |

---

# 9.6.3.14.4 Business Rules

| Business Rule ID | Description                                                                                                                          |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-ADR-001     | Setiap Account Derivation Rule harus memiliki kombinasi kriteria kondisi yang unik dalam ruang lingkup yang sama untuk menghindari ambiguitas hasil derivasi. |
| BRule-ADR-002     | Rule dengan prioritas atau urutan lebih tinggi dievaluasi dan dieksekusi lebih dahulu dibandingkan rule dengan prioritas lebih rendah. |
| BRule-ADR-003     | Apabila tidak ada rule yang cocok dengan kondisi transaksi, sistem harus menggunakan default account yang telah dikonfigurasi.        |
| BRule-ADR-004     | Akun tujuan hasil derivasi harus merupakan akun yang valid dan berstatus aktif pada Chart of Accounts terkait.                        |
| BRule-ADR-005     | Perubahan pada rule tidak boleh mengubah hasil derivasi akun dari transaksi historis yang telah diposting.                            |
| BRule-ADR-006     | Rule hanya berlaku dan dievaluasi dalam periode efektif (Effective From/To) yang ditentukan.                                          |
| BRule-ADR-007     | Setiap pembuatan atau perubahan rule harus melalui approval workflow sebelum berstatus aktif dan dapat digunakan.                     |

---

# 9.6.3.14.5 Data Attributes

| Attribute                 | Description                                                             |
| -------------------------- | ------------------------------------------------------------------------ |
| Account Derivation Rule ID| Identitas unik sistem.                                                  |
| Rule Code / Name          | Kode dan nama rule.                                                     |
| Rule Scope                | Cakupan organisasi tempat rule berlaku (Organization Tree).             |
| Condition Criteria        | Kombinasi kondisi (transaction type, customer type, item category, dll).|
| Priority / Sequence       | Urutan prioritas eksekusi rule.                                         |
| Derived Account (Debit)   | Akun tujuan Debit hasil derivasi.                                       |
| Derived Account (Credit)  | Akun tujuan Kredit hasil derivasi.                                      |
| Default Account           | Akun fallback jika tidak ada rule yang cocok.                           |
| Effective From / To       | Periode berlaku.                                                        |
| Version                   | Nomor versi.                                                            |
| Approval Status           | Draft, Pending Approval, Approved, Published, Archived.                 |
| Status                    | Active, Suspended, Inactive, Archived.                                  |
| Created By / Created Date | Informasi pembuatan data.                                               |
| Updated By / Updated Date | Informasi perubahan terakhir.                                           |

---

# 9.6.3.14.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                        |
| --------------- | ------------------- | ----------- | -------------------------------------- |
| BR-436          | BO-009              | SH-003      | Account Derivation Rule Administration |
| BR-437          | BO-009              | SH-003      | Account Derivation Rule Administration |
| BR-438          | BO-009              | SH-003      | Account Derivation Rule Administration |
| BR-439          | BO-009              | SH-003      | Account Derivation Rule Administration |
| BR-440          | BO-009              | SH-003      | Account Derivation Rule Administration |
| BR-441          | BO-009              | SH-003      | Account Derivation Rule Administration |
| BR-442          | BO-009              | SH-003      | Account Derivation Rule Administration |
| BR-443          | BO-009              | SH-003      | Account Derivation Rule Administration |
| BR-444          | BO-009              | SH-003      | Account Derivation Rule Administration |
| BR-445          | BO-009              | SH-003      | Account Derivation Rule Administration |

---

# 9.6.3.14.7 Summary

Account Derivation Rule menyediakan mekanisme konfigurasi untuk menentukan akun tujuan secara otomatis berdasarkan kondisi transaksi, sehingga mengurangi input manual dan potensi kesalahan posting jurnal. Account Derivation Rule menjadi fondasi otomatisasi lintas departemen yang mendukung requirement lain yang terkait pada Financial Management Domain.
