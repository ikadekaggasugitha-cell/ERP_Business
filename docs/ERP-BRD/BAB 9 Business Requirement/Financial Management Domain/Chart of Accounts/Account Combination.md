# 9.6.3.11 Account Combination Requirements

---

# 9.6.3.11.1 Overview

Account Combination merupakan kombinasi valid dari seluruh Segment (Company, Cost Center, Account, dan segmen tambahan lain sesuai konfigurasi Segment Structure) yang membentuk satu kode akun transaksi lengkap. Account Combination menjadi mekanisme kendali utama untuk memastikan bahwa setiap transaksi keuangan hanya dapat dicatat menggunakan kombinasi segmen yang telah tervalidasi, konsisten, dan sesuai dengan Cross-Validation Rule yang berlaku, sehingga integritas data keuangan tetap terjaga di seluruh entitas organisasi.

---

# 9.6.3.11.2 Business Capability

Account Combination menyediakan kapabilitas berikut:

* Definisi dan penyimpanan Account Combination sebagai objek bisnis tersendiri.
* Cross-Validation Rule antar Segment untuk mencegah kombinasi tidak valid.
* Validasi kombinasi akun secara real-time pada saat entri transaksi.
* Restricted/Disallowed Combination Rule untuk kombinasi yang dilarang.
* Dynamic Combination Generation (pembentukan kombinasi baru sesuai kebutuhan transaksi).
* Approval Workflow untuk pembentukan dan perubahan kombinasi akun.
* Effective Dating dan Versioning kombinasi akun.
* Audit Trail atas seluruh perubahan status dan penggunaan kombinasi.
* Usage Analytics dan pelaporan kombinasi akun yang aktif maupun tidak digunakan.
* Multi-Organization Scope pada validasi kombinasi akun.

---

# 9.6.3.11.3 Business Requirement Catalogue

## BR-406 – Account Combination Definition

| Attribute           | Description                                                                                                                                             |
| -------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-406                                                                                                                                                  |
| Requirement Name    | Account Combination Definition                                                                                                                          |
| Category            | Account Combination                                                                                                                                     |
| Priority            | Critical                                                                                                                                                |
| Business Objective  | BO-004, BO-014                                                                                                                                          |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                        |
| Description         | Sistem harus menyediakan Account Combination sebagai objek bisnis yang menggabungkan seluruh Segment (Company, Cost Center, Account, dan segmen lain) menjadi satu kode akun transaksi lengkap. |
| Business Rationale  | Setiap transaksi keuangan memerlukan satu kode akun lengkap yang merepresentasikan seluruh dimensi pencatatan secara konsisten.                        |
| Business Value      | Menjadi Single Source of Truth atas kombinasi segmen yang sah digunakan dalam pencatatan transaksi keuangan.                                            |
| Acceptance Criteria | Administrator dapat mendefinisikan, melihat, dan mengelola Account Combination beserta seluruh Segment penyusunnya sesuai kewenangan.                   |

---

## BR-407 – Cross-Validation Rule Engine

| Attribute      | Description                                                                                                                                     |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-407                                                                                                                                          |
| Requirement Name | Cross-Validation Rule Engine                                                                                                                  |
| Priority       | Critical                                                                                                                                        |
| Description    | Sistem harus menyediakan mesin aturan Cross-Validation Rule yang mendefinisikan kombinasi segmen mana yang diizinkan atau dilarang antar Segment. |
| Business Value | Mencegah pembentukan kombinasi akun yang tidak sesuai dengan kebijakan bisnis maupun struktur organisasi.                                       |

---

## BR-408 – Real-Time Combination Validation

| Attribute      | Description                                                                                                                   |
| --------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-408                                                                                                                        |
| Requirement Name | Real-Time Combination Validation                                                                                            |
| Priority       | Critical                                                                                                                      |
| Description    | Sistem harus memvalidasi kombinasi segmen secara real-time pada saat entri transaksi sebelum transaksi dapat disimpan.       |
| Acceptance Criteria | Sistem menolak entri transaksi yang menggunakan kombinasi akun tidak valid dan menampilkan pesan kesalahan yang jelas.   |

---

## BR-409 – Restricted / Disallowed Combination Rule

| Attribute      | Description                                                                                                                       |
| --------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-409                                                                                                                             |
| Requirement Name | Restricted / Disallowed Combination Rule                                                                                        |
| Priority       | High                                                                                                                               |
| Description    | Sistem harus mendukung penetapan kombinasi akun yang secara eksplisit dilarang (Disallowed) meskipun seluruh segmen penyusunnya valid secara individual. |
| Business Value | Melindungi organisasi dari pencatatan transaksi pada kombinasi yang tidak sesuai kebijakan akuntansi atau organisasi.               |

---

## BR-410 – Dynamic Combination Generation

| Attribute      | Description                                                                                                                     |
| --------------- | ----------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-410                                                                                                                           |
| Requirement Name | Dynamic Combination Generation                                                                                                 |
| Priority       | High                                                                                                                             |
| Description    | Sistem harus mampu membentuk kombinasi akun baru secara otomatis (on-demand) apabila kombinasi segmen yang diminta valid namun belum pernah dibuat sebelumnya. |
| Acceptance Criteria | Kombinasi baru yang terbentuk otomatis tercatat dengan status dan riwayat pembentukan yang jelas.                          |

---

## BR-411 – Combination Approval Workflow

| Attribute      | Description                                                                                                             |
| --------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-411                                                                                                                   |
| Requirement Name | Combination Approval Workflow                                                                                          |
| Priority       | High                                                                                                                     |
| Description    | Sistem harus menyediakan Approval Workflow yang configurable untuk pembentukan, perubahan, maupun penonaktifan Account Combination. |
| Business Value | Menjamin tata kelola dan kontrol atas setiap perubahan kombinasi akun yang berdampak pada pelaporan keuangan.           |

---

## BR-412 – Combination Effective Dating & Versioning

| Attribute      | Description                                                                                                       |
| --------------- | ----------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-412                                                                                                             |
| Requirement Name | Combination Effective Dating & Versioning                                                                       |
| Priority       | Medium                                                                                                             |
| Description    | Sistem harus mendukung Effective Dating dan Versioning pada Account Combination sehingga perubahan status atau aturan validasi tidak memengaruhi transaksi historis. |

---

## BR-413 – Combination Usage Analytics

| Attribute      | Description                                                                                                          |
| --------------- | ---------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-413                                                                                                               |
| Requirement Name | Combination Usage Analytics                                                                                       |
| Priority       | Medium                                                                                                              |
| Description    | Sistem harus menyediakan laporan penggunaan Account Combination, termasuk kombinasi aktif, tidak digunakan, maupun kombinasi yang mendekati batas Effective To. |
| Business Value | Mendukung housekeeping Chart of Accounts dan pengambilan keputusan berbasis data penggunaan aktual.                  |

---

## BR-414 – Combination Audit Trail

| Attribute      | Description                                                                                                     |
| --------------- | ----------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-414                                                                                                           |
| Requirement Name | Combination Audit Trail                                                                                        |
| Priority       | High                                                                                                             |
| Description    | Sistem harus mencatat audit trail atas seluruh pembentukan, perubahan, dan penonaktifan Account Combination, termasuk identitas pelaku dan waktu perubahan. |

---

## BR-415 – Multi-Organization Combination Scope

| Attribute      | Description                                                                                                               |
| --------------- | -------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-415                                                                                                                     |
| Requirement Name | Multi-Organization Combination Scope                                                                                    |
| Priority       | Critical                                                                                                                   |
| Description    | Sistem harus menerapkan validasi Account Combination sesuai ruang lingkup organisasi (Holding, Sub Holding, Company, Business Unit, dan seterusnya) tanpa batas level struktur. |
| Acceptance Criteria | Kombinasi akun yang valid pada satu scope organisasi tidak secara otomatis berlaku pada scope organisasi lain kecuali dikonfigurasikan sebagai shared. |

---

# 9.6.3.11.4 Business Rules

| Business Rule ID | Description                                                                                                                          |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| BRule-ACMB-001   | Setiap Account Combination harus terdiri dari seluruh Segment wajib sesuai Segment Structure yang berlaku pada scope terkait.       |
| BRule-ACMB-002   | Kombinasi akun yang tidak lolos Cross-Validation Rule tidak dapat digunakan dalam transaksi apapun.                                  |
| BRule-ACMB-003   | Sistem harus mencegah duplikasi Account Combination yang identik dalam ruang lingkup Chart of Accounts yang sama.                    |
| BRule-ACMB-004   | Perubahan Cross-Validation Rule tidak boleh mengubah status Account Combination yang telah digunakan pada transaksi historis.        |
| BRule-ACMB-005   | Account Combination berstatus Disallowed tidak dapat dipilih pada transaksi baru meskipun sebelumnya berstatus valid.                |
| BRule-ACMB-006   | Dynamic Combination Generation hanya berlaku apabila seluruh Segment penyusun berstatus Active dan lolos Cross-Validation Rule.      |
| BRule-ACMB-007   | Setiap perubahan status Account Combination harus melalui Approval Workflow dan tercatat dalam audit trail.                          |

---

# 9.6.3.11.5 Data Attributes

| Attribute                       | Description                                                              |
| -------------------------------- | ------------------------------------------------------------------------ |
| Account Combination ID          | Identitas unik sistem.                                                   |
| Combination Code                | Kode gabungan seluruh nilai Segment yang membentuk kombinasi.            |
| Segment Values                  | Nilai masing-masing Segment (Company, Cost Center, Account, dan lainnya). |
| Cross-Validation Rule Reference | Referensi aturan validasi yang digunakan untuk memvalidasi kombinasi.    |
| Validation Status               | Valid, Invalid, Pending Validation.                                       |
| Organization Scope              | Cakupan organisasi berlakunya kombinasi (Holding s.d. Project Site).      |
| Effective From / To             | Periode berlaku kombinasi.                                                |
| Version                         | Nomor versi kombinasi.                                                    |
| Approval Status                 | Draft, Pending Approval, Approved, Published, Archived.                  |
| Status                          | Active, Suspended, Inactive, Archived, Disallowed.                       |
| Usage Count / Last Used Date    | Statistik penggunaan kombinasi pada transaksi.                           |
| Created By / Created Date       | Informasi pembuatan data.                                                 |
| Updated By / Updated Date       | Informasi perubahan terakhir.                                             |

---

# 9.6.3.11.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                          |
| --------------- | ------------------- | ------------ | ---------------------------------------- |
| BR-406         | BO-004, BO-014      | SH-003       | Account Combination Administration       |
| BR-407         | BO-004              | SH-003       | Cross-Validation Rule Configuration      |
| BR-408         | BO-004              | SH-003       | Transaction Entry Validation             |
| BR-409         | BO-004              | SH-003       | Restricted Combination Management        |
| BR-410         | BO-004              | SH-003       | Dynamic Combination Generation           |
| BR-411         | BO-004              | SH-003       | Combination Approval Processing          |
| BR-412         | BO-004              | SH-003       | Combination Versioning Management        |
| BR-413         | BO-004              | SH-003       | Combination Usage Reporting              |
| BR-414         | BO-004              | SH-003       | Combination Audit Review                 |
| BR-415         | BO-004              | SH-003       | Multi-Organization Scope Administration  |

---

# 9.6.3.11.7 Summary

Account Combination memastikan bahwa setiap transaksi keuangan hanya dapat dicatat menggunakan kombinasi segmen yang telah tervalidasi melalui Cross-Validation Rule, konsisten dengan struktur organisasi, dan terkendali melalui Approval Workflow serta Audit Trail. Account Combination menjadi Single Source of Truth atas kode akun transaksi yang sah, sekaligus fondasi bagi requirement lain yang terkait pada Financial Management Domain.
