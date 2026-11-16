# 9.6.3.13 Account Validation Rule Requirements

---

# 9.6.3.13.1 Overview

Account Validation Rule merupakan komponen Chart of Accounts yang mendefinisikan aturan validasi otomatis atas kombinasi akun (account combination) maupun proses posting transaksi. Account Validation Rule memastikan bahwa setiap kombinasi segmen akun, kombinasi akun dengan dimensi analitik, serta kondisi posting transaksi sesuai dengan kebijakan akuntansi dan kebijakan bisnis perusahaan sebelum transaksi disimpan ke buku besar. Dengan Account Validation Rule, sistem dapat mencegah kesalahan pencatatan, kombinasi akun yang tidak valid, serta pelanggaran kebijakan keuangan secara real-time pada saat transaksi dibuat maupun diposting.

---

# 9.6.3.13.2 Business Capability

Account Validation Rule menyediakan kapabilitas berikut:

* Account Combination Validation.
* Posting-Time Validation.
* Mandatory Segment Validation.
* Restricted/Prohibited Account Combination.
* Conditional Validation Rule berbasis kondisi transaksi.
* Scope-Based Validation Rule sesuai struktur organisasi.
* Real-Time Validation Feedback.
* Validation Rule Priority & Sequencing.
* Validation Exception & Override Handling.
* Effective Dating.
* Versioning.
* Approval Workflow.
* Audit Trail.

---

# 9.6.3.13.3 Business Requirement Catalogue

## BR-426 – Account Validation Rule Definition

| Attribute           | Description                                                                                                                                          |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID       | BR-426                                                                                                                                                |
| Requirement Name     | Account Validation Rule Definition                                                                                                                   |
| Category             | Account Validation Rule                                                                                                                              |
| Priority             | Critical                                                                                                                                              |
| Business Objective   | BO-028                                                                                                                                                |
| Stakeholder Owner    | SH-003 (Chief Financial Officer)                                                                                                                     |
| Description          | Sistem harus menyediakan Account Validation Rule sebagai objek bisnis untuk mendefinisikan aturan validasi otomatis atas kombinasi akun dan posting transaksi. |
| Business Rationale   | Kombinasi akun yang tidak konsisten dapat menimbulkan kesalahan pelaporan keuangan dan pelanggaran kebijakan akuntansi apabila tidak divalidasi secara sistematis. |
| Business Value       | Menjamin integritas data akuntansi sejak titik input transaksi.                                                                                      |
| Acceptance Criteria  | Administrator dapat membuat, mengubah, menonaktifkan, dan mengelola Account Validation Rule sesuai kewenangan.                                       |

---

## BR-427 – Account Combination Validation

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-427                                                                                                                             |
| Requirement Name | Account Combination Validation                                                                                                    |
| Priority         | Critical                                                                                                                           |
| Description      | Sistem harus memvalidasi kombinasi segmen akun (misalnya akun, entitas, departemen, proyek) terhadap aturan kombinasi yang telah ditetapkan sebelum kombinasi akun disimpan sebagai valid. |
| Acceptance Criteria | Sistem menolak kombinasi akun yang tidak sesuai dengan aturan kombinasi yang aktif.                                             |

---

## BR-428 – Mandatory Segment Combination Rule

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-428                                                                                                                             |
| Requirement Name | Mandatory Segment Combination Rule                                                                                                |
| Priority         | Critical                                                                                                                           |
| Description      | Sistem harus mendukung aturan validasi yang mewajibkan pengisian segmen tertentu (misalnya Cost Center atau Project) untuk kombinasi akun tertentu sesuai kebijakan perusahaan. |
| Business Value   | Memastikan kelengkapan data dimensi analitik pada setiap transaksi.                                                               |

---

## BR-429 – Restricted/Prohibited Account Combination Rule

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-429                                                                                                                             |
| Requirement Name | Restricted/Prohibited Account Combination Rule                                                                                    |
| Priority         | Critical                                                                                                                           |
| Description      | Sistem harus memungkinkan definisi kombinasi akun yang dilarang atau dibatasi (restricted/prohibited combination), sehingga sistem menolak pembentukan kombinasi tersebut baik pada saat pembuatan master maupun pada saat posting transaksi. |
| Acceptance Criteria | Sistem menampilkan pesan penolakan yang jelas ketika kombinasi akun yang dilarang digunakan.                                     |

---

## BR-430 – Conditional Validation Rule

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-430                                                                                                                             |
| Requirement Name | Conditional Validation Rule                                                                                                       |
| Priority         | High                                                                                                                               |
| Description      | Sistem harus mendukung aturan validasi kondisional yang berlaku berdasarkan atribut transaksi seperti jenis transaksi, jumlah (amount), mata uang, atau periode akuntansi. |
| Business Value   | Memberikan fleksibilitas kebijakan validasi sesuai konteks transaksi tanpa perubahan source code.                                 |

---

## BR-431 – Scope-Based Validation Rule

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-431                                                                                                                             |
| Requirement Name | Scope-Based Validation Rule                                                                                                       |
| Priority         | Critical                                                                                                                           |
| Description      | Sistem harus mendukung penerapan Account Validation Rule berdasarkan lingkup organisasi (Holding, Sub Holding, Company, Business Unit, Branch, Department, Warehouse, Project Site) sesuai struktur pohon organisasi. |
| Acceptance Criteria | Aturan validasi yang berlaku pada suatu node organisasi dapat diwariskan atau dikecualikan pada node turunan sesuai konfigurasi.  |

---

## BR-432 – Real-Time Posting Validation

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-432                                                                                                                             |
| Requirement Name | Real-Time Posting Validation                                                                                                       |
| Priority         | Critical                                                                                                                           |
| Description      | Sistem harus melakukan validasi Account Validation Rule secara real-time pada saat transaksi dibuat maupun pada saat proses posting ke buku besar, sebelum transaksi memperoleh status final. |
| Business Value   | Mencegah transaksi tidak valid masuk ke buku besar dan mengurangi kebutuhan koreksi pasca posting.                                |

---

## BR-433 – Validation Rule Priority & Sequencing

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-433                                                                                                                             |
| Requirement Name | Validation Rule Priority & Sequencing                                                                                             |
| Priority         | High                                                                                                                               |
| Description      | Sistem harus mendukung penentuan prioritas dan urutan eksekusi antar Account Validation Rule ketika lebih dari satu aturan berlaku pada kombinasi akun atau transaksi yang sama. |
| Acceptance Criteria | Sistem mengeksekusi aturan validasi sesuai urutan prioritas yang dikonfigurasi dan menghentikan proses saat aturan bersifat blocking terpenuhi. |

---

## BR-434 – Validation Exception & Override Handling

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-434                                                                                                                             |
| Requirement Name | Validation Exception & Override Handling                                                                                          |
| Priority         | High                                                                                                                               |
| Description      | Sistem harus menyediakan mekanisme penanganan pengecualian (exception) atas hasil validasi, termasuk kemampuan override oleh pengguna berwenang disertai justifikasi dan approval, untuk kondisi bisnis tertentu yang bersifat mendesak. |
| Business Value   | Menyeimbangkan kepatuhan terhadap kebijakan validasi dengan kebutuhan operasional yang mendesak.                                  |

---

## BR-435 – Validation Rule Audit Trail & Versioning

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-435                                                                                                                             |
| Requirement Name | Validation Rule Audit Trail & Versioning                                                                                          |
| Priority         | High                                                                                                                               |
| Description      | Sistem harus mencatat riwayat perubahan Account Validation Rule secara lengkap (audit trail) serta mendukung versioning dan effective dating atas setiap aturan validasi yang diterapkan. |
| Acceptance Criteria | Setiap perubahan aturan validasi tercatat dengan informasi pelaku, waktu, dan nilai sebelum/sesudah perubahan.                    |

---

# 9.6.3.13.4 Business Rules

| Business Rule ID | Description                                                                                                                          |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-AVR-001    | Setiap Account Validation Rule harus memiliki kode dan nama yang unik dalam ruang lingkup Chart of Accounts yang sama.               |
| BRule-AVR-002    | Kombinasi akun yang melanggar aturan validasi bertipe blocking tidak dapat disimpan atau diposting ke buku besar.                    |
| BRule-AVR-003    | Aturan validasi hanya berlaku pada rentang tanggal efektif (effective from/to) yang ditetapkan.                                      |
| BRule-AVR-004    | Perubahan Account Validation Rule yang berdampak pada transaksi berjalan harus melalui alur persetujuan (approval workflow).         |
| BRule-AVR-005    | Aturan validasi yang bersifat warning tidak menghentikan proses posting, namun harus dicatat sebagai catatan pengecualian.           |
| BRule-AVR-006    | Override atas hasil validasi hanya dapat dilakukan oleh pengguna dengan kewenangan khusus dan wajib disertai alasan (justification). |
| BRule-AVR-007    | Aturan validasi yang diwariskan dari organisasi induk tidak dapat dilonggarkan oleh organisasi anak tanpa otorisasi eksplisit.       |

---

# 9.6.3.13.5 Data Attributes

| Attribute                  | Description                                                        |
| ---------------------------- | --------------------------------------------------------------------- |
| Validation Rule ID         | Identitas unik sistem.                                             |
| Rule Code / Rule Name      | Kode dan nama aturan validasi.                                     |
| Rule Type                  | Mandatory Segment, Restricted Combination, Conditional, Cross-Segment. |
| Rule Severity              | Blocking, Warning, Informational.                                  |
| Applicable Scope           | Organization node tempat aturan validasi berlaku.                  |
| Rule Condition/Expression  | Logika kondisi validasi (segmen, nilai, tipe transaksi, dsb.).      |
| Priority/Sequence          | Urutan eksekusi aturan validasi.                                   |
| Effective From / To        | Periode berlaku aturan validasi.                                   |
| Version                    | Nomor versi aturan validasi.                                       |
| Approval Status            | Draft, Pending Approval, Approved, Published, Archived.            |
| Status                     | Active, Suspended, Inactive, Archived.                             |
| Override Allowed           | Penanda apakah aturan dapat dioverride beserta kewenangannya.      |
| Created By / Created Date  | Informasi pembuatan data.                                          |
| Updated By / Updated Date  | Informasi perubahan terakhir.                                      |

---

# 9.6.3.13.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                          |
| --------------- | -------------------- | ------------- | ------------------------------------------ |
| BR-426         | BO-028               | SH-003        | Account Validation Rule Administration    |
| BR-427         | BO-028               | SH-003        | Account Combination Setup                 |
| BR-428         | BO-028               | SH-003        | Account Combination Setup                 |
| BR-429         | BO-028               | SH-003        | Account Combination Setup                 |
| BR-430         | BO-028               | SH-003        | Transaction Posting Validation            |
| BR-431         | BO-028               | SH-003        | Organization-Based Rule Configuration     |
| BR-432         | BO-028               | SH-003        | Transaction Posting Validation            |
| BR-433         | BO-028               | SH-003        | Validation Rule Configuration             |
| BR-434         | BO-028               | SH-003        | Validation Exception Management           |
| BR-435         | BO-028               | SH-003        | Validation Rule Audit & Version Control   |

---

# 9.6.3.13.7 Summary

Account Validation Rule menyediakan mekanisme validasi otomatis atas kombinasi akun dan proses posting transaksi dalam Chart of Accounts. Account Validation Rule menjadi lapisan pengendalian utama yang menjamin integritas dan kepatuhan data akuntansi sejak titik input transaksi, serta menjadi fondasi bagi requirement lain yang terkait pada Financial Management Domain.
