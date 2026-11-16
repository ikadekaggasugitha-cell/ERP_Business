# 9.6.3.8 Segment Value Requirements

---

# 9.6.3.8.1 Overview

Segment Value merupakan daftar nilai valid yang sah digunakan untuk setiap Segment dalam struktur Chart of Accounts, misalnya daftar kode Cost Center, kode Branch, kode Project, atau kode Business Unit yang telah didefinisikan. Segment Value mencakup pengelolaan validasi nilai, hierarki nilai (parent-child), serta cross-validation rule antar segmen untuk memastikan hanya kombinasi segmen yang sah yang dapat digunakan pada transaksi keuangan. Segment Value menjadi lapisan kontrol yang memastikan integritas data pada setiap pencatatan transaksi lintas entitas organisasi.

---

# 9.6.3.8.2 Business Capability

Segment Value menyediakan kapabilitas berikut:

* Segment Value Definition & Registration.
* Segment Value Code & Description Management.
* Segment Value Hierarchy (Parent-Child Structure).
* Segment Value Validation Rule (Format, Panjang, Pola).
* Cross-Validation Rule Antar Segmen.
* Segment Value Mapping to Organization Scope.
* Effective Dating & Value Lifecycle Management.
* Approval Workflow untuk Perubahan Segment Value.
* Bulk Import & Export Segment Value.
* Duplicate Prevention & Uniqueness Enforcement.
* Segment Value Search & Lookup.
* Audit Trail & Change History.

---

# 9.6.3.8.3 Business Requirement Catalogue

## BR-376 – Segment Value Definition

| Attribute           | Description                                                                                                                                             |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-376                                                                                                                                                    |
| Requirement Name    | Segment Value Definition                                                                                                                                 |
| Category            | Segment Value                                                                                                                                            |
| Priority            | Critical                                                                                                                                                  |
| Business Objective  | BO-004                                                                                                                                                    |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                          |
| Description         | Sistem harus menyediakan Segment Value sebagai objek bisnis untuk mendefinisikan daftar nilai yang sah digunakan pada setiap Segment sesuai konfigurasi Chart of Accounts. |
| Business Rationale  | Setiap Segment memerlukan daftar nilai yang terkendali agar transaksi keuangan hanya menggunakan nilai yang telah divalidasi dan diakui oleh organisasi. |
| Business Value      | Menjadi satu sumber acuan (single source of truth) bagi seluruh nilai segmen yang digunakan lintas modul dan lintas entitas organisasi.                   |
| Acceptance Criteria | Administrator dapat membuat, mengubah, menonaktifkan, dan mengelola Segment Value sesuai kewenangan dan ruang lingkup organisasi.                        |

---

## BR-377 – Segment Value Hierarchy Structure

| Attribute        | Description                                                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-377                                                                                                                          |
| Requirement Name  | Segment Value Hierarchy Structure                                                                                              |
| Priority          | Critical                                                                                                                        |
| Description       | Sistem harus mendukung hierarki nilai segmen dalam struktur parent-child tanpa batas level, untuk kebutuhan pelaporan berjenjang (roll-up). |
| Business Value    | Memungkinkan konsolidasi dan agregasi laporan berdasarkan hierarki nilai segmen secara fleksibel.                              |

---

## BR-378 – Segment Value Format & Validation Rule

| Attribute           | Description                                                                                                                          |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID       | BR-378                                                                                                                              |
| Requirement Name     | Segment Value Format & Validation Rule                                                                                              |
| Priority             | Critical                                                                                                                            |
| Description          | Sistem harus memvalidasi setiap Segment Value terhadap aturan format Segment induknya (panjang karakter, tipe data, pola/regex).    |
| Acceptance Criteria  | Sistem menolak input Segment Value yang tidak sesuai dengan aturan validasi Segment terkait.                                        |

---

## BR-379 – Cross-Validation Rule Antar Segmen

| Attribute           | Description                                                                                                                                       |
| -------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID       | BR-379                                                                                                                                             |
| Requirement Name     | Cross-Validation Rule Antar Segmen                                                                                                                 |
| Priority             | Critical                                                                                                                                           |
| Description          | Sistem harus menyediakan mekanisme cross-validation rule untuk menentukan kombinasi Segment Value yang sah antar segmen (misalnya kombinasi Cost Center tertentu hanya sah dengan Branch tertentu). |
| Business Value       | Mencegah pembentukan kombinasi segmen yang tidak konsisten dengan struktur organisasi atau kebijakan bisnis.                                       |

---

## BR-380 – Segment Value Effective Dating

| Attribute        | Description                                                                                                          |
| ----------------- | --------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-380                                                                                                               |
| Requirement Name  | Segment Value Effective Dating                                                                                      |
| Priority          | High                                                                                                                 |
| Description       | Sistem harus mendukung periode berlaku (Effective From/To) untuk setiap Segment Value guna mengendalikan penggunaan nilai berdasarkan periode waktu tertentu. |

---

## BR-381 – Segment Value Status Lifecycle

| Attribute        | Description                                                                                                                          |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-381                                                                                                                              |
| Requirement Name  | Segment Value Status Lifecycle                                                                                                      |
| Priority          | High                                                                                                                                 |
| Description       | Sistem harus mengelola siklus status Segment Value (Draft, Active, Suspended, Inactive, Archived) sesuai kebijakan tata kelola data. |

---

## BR-382 – Segment Value Approval Workflow

| Attribute           | Description                                                                                                                |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID       | BR-382                                                                                                                     |
| Requirement Name     | Segment Value Approval Workflow                                                                                           |
| Priority             | High                                                                                                                       |
| Description          | Sistem harus menyediakan alur persetujuan configurable sebelum Segment Value baru atau perubahannya berstatus Active.     |
| Acceptance Criteria  | Segment Value tidak dapat digunakan pada transaksi sebelum melalui dan lolos alur persetujuan yang ditetapkan.            |

---

## BR-383 – Segment Value Uniqueness Enforcement

| Attribute        | Description                                                                                                             |
| ----------------- | ------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-383                                                                                                                 |
| Requirement Name  | Segment Value Uniqueness Enforcement                                                                                   |
| Priority          | Critical                                                                                                               |
| Description       | Sistem harus mencegah duplikasi kode Segment Value dalam ruang lingkup Segment dan Organization Scope yang sama.       |

---

## BR-384 – Segment Value Bulk Import & Export

| Attribute        | Description                                                                                                          |
| ----------------- | --------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-384                                                                                                               |
| Requirement Name  | Segment Value Bulk Import & Export                                                                                   |
| Priority          | Medium                                                                                                               |
| Description       | Sistem harus menyediakan fasilitas impor dan ekspor massal Segment Value dengan validasi otomatis terhadap format dan cross-validation rule. |

---

## BR-385 – Segment Value Audit Trail

| Attribute        | Description                                                                                                             |
| ----------------- | ------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-385                                                                                                                |
| Requirement Name  | Segment Value Audit Trail                                                                                            |
| Priority          | High                                                                                                                  |
| Description       | Sistem harus mencatat seluruh riwayat perubahan Segment Value, termasuk nilai sebelum dan sesudah perubahan, untuk kebutuhan audit. |

---

# 9.6.3.8.4 Business Rules

| Business Rule ID | Description                                                                                                                              |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-SEGV-001   | Setiap Segment Value harus memiliki kode unik dalam ruang lingkup Segment dan Organization Scope yang sama.                             |
| BRule-SEGV-002   | Segment Value tidak dapat dihapus apabila telah digunakan pada transaksi; perubahan hanya dapat dilakukan melalui perubahan status menjadi Inactive. |
| BRule-SEGV-003   | Perubahan status Segment Value dari Draft menjadi Active wajib melalui alur persetujuan sesuai kewenangan yang berlaku.                 |
| BRule-SEGV-004   | Kombinasi Segment Value antar segmen harus lolos cross-validation rule sebelum dapat digunakan sebagai kombinasi akun yang valid.        |
| BRule-SEGV-005   | Segment Value dengan struktur hierarki hanya dapat digunakan pada level anak (leaf node) untuk transaksi, kecuali dikonfigurasi sebaliknya. |
| BRule-SEGV-006   | Segment Value dengan Effective To Date yang telah lewat tidak dapat digunakan untuk transaksi baru.                                     |
| BRule-SEGV-007   | Setiap perubahan pada Segment Value wajib tercatat dalam audit trail termasuk pengguna, waktu, dan nilai sebelum/sesudah perubahan.      |

---

# 9.6.3.8.5 Data Attributes

| Attribute                 | Description                                                       |
| -------------------------- | ------------------------------------------------------------------ |
| Segment Value ID          | Identitas unik sistem.                                            |
| Segment ID                | Referensi ke Segment induk.                                       |
| Value Code                | Kode nilai segmen.                                                 |
| Value Name/Description    | Nama atau deskripsi nilai segmen.                                  |
| Parent Value ID           | Referensi ke nilai induk dalam hierarki (jika ada).                |
| Hierarchy Level           | Kedalaman level dalam struktur hierarki nilai.                     |
| Validation Pattern        | Pola atau aturan validasi format nilai.                            |
| Organization Scope        | Ruang lingkup organisasi tempat nilai berlaku.                     |
| Effective From / To       | Periode berlaku nilai segmen.                                      |
| Approval Status           | Draft, Pending Approval, Approved, Published, Archived.            |
| Status                    | Active, Suspended, Inactive, Archived.                             |
| Created By / Created Date | Informasi pembuatan data.                                          |
| Updated By / Updated Date | Informasi perubahan terakhir.                                      |

---

# 9.6.3.8.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                     |
| --------------- | ------------------- | ------------ | ------------------------------------ |
| BR-376         | BO-004              | SH-003      | Segment Value Administration        |
| BR-377         | BO-004              | SH-003      | Segment Value Hierarchy Configuration |
| BR-378         | BO-004              | SH-003      | Segment Value Format Validation     |
| BR-379         | BO-004              | SH-003      | Cross-Segment Validation Setup      |
| BR-380         | BO-004              | SH-003      | Segment Value Effective Date Management |
| BR-381         | BO-004              | SH-003      | Segment Value Status Management     |
| BR-382         | BO-004              | SH-003      | Segment Value Approval Processing   |
| BR-383         | BO-004              | SH-003      | Segment Value Uniqueness Control    |
| BR-384         | BO-004              | SH-003      | Segment Value Bulk Data Management  |
| BR-385         | BO-004              | SH-003      | Segment Value Audit & Compliance    |

---

# 9.6.3.8.7 Summary

Segment Value menyediakan daftar nilai yang sah dan terkendali bagi setiap Segment dalam Chart of Accounts, lengkap dengan hierarki, validasi format, serta cross-validation rule antar segmen. Segment Value menjadi mekanisme kontrol utama yang memastikan setiap kombinasi segmen pada transaksi keuangan konsisten, akurat, dan sesuai dengan kebijakan tata kelola organisasi pada Financial Management Domain.
