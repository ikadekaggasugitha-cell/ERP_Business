# 9.6.3.20 Account Security Requirements

---

# 9.6.3.20.1 Overview

Account Security mengatur hak akses terhadap akun atau kelompok akun tertentu dalam Chart of Accounts berdasarkan Role dan Organization Scope. Kapabilitas ini membatasi siapa yang dapat melihat, menggunakan, memposting, atau menyetujui transaksi pada akun sensitif, sehingga integritas data keuangan tetap terjaga dan risiko penyalahgunaan akun dapat diminimalkan. Account Security bekerja secara terintegrasi dengan struktur organisasi berjenjang dan model Scope-Based Everything, sehingga hak akses dapat dikonfigurasi mengikuti posisi setiap entitas dalam Organization Tree tanpa memerlukan perubahan source code.

---

# 9.6.3.20.2 Business Capability

Account Security menyediakan kapabilitas berikut:

* Role-Based Account Access Control.
* Organization Scope-Based Account Restriction.
* Sensitive Account Classification.
* Restricted Account Group Management.
* Account-Level Permission Matrix (View, Post, Approve, Modify).
* Segregation of Duties Enforcement.
* Time-Bound / Temporary Access Grant.
* Emergency (Break-Glass) Access.
* Access Request & Approval Workflow.
* Access Recertification / Periodic Review.
* Sensitive Balance Masking.
* Audit Trail of Access Changes.

---

# 9.6.3.20.3 Business Requirement Catalogue

## BR-496 – Account Security Policy Definition

| Attribute           | Description                                                                                                                                                                       |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID       | BR-496                                                                                                                                                                             |
| Requirement Name     | Account Security Policy Definition                                                                                                                                                |
| Category             | Account Security                                                                                                                                                                   |
| Priority             | Critical                                                                                                                                                                           |
| Business Objective   | BO-018, BO-028                                                                                                                                                                     |
| Stakeholder Owner    | SH-026 (Security Engineer), SH-003 (Chief Financial Officer)                                                                                                                       |
| Description          | Sistem harus menyediakan Account Security Policy sebagai objek konfigurasi untuk mendefinisikan hak akses terhadap akun atau kelompok akun tertentu berdasarkan Role dan Organization Scope. |
| Business Rationale   | Akun-akun tertentu (misalnya payroll, intercompany, suspense, executive compensation) bersifat sensitif dan memerlukan pembatasan akses yang tidak cukup diatur melalui hak akses modul secara umum. |
| Business Value       | Menyediakan mekanisme kontrol akses granular pada level akun yang mendukung tata kelola keuangan dan kepatuhan terhadap kebijakan internal maupun regulasi eksternal.              |
| Acceptance Criteria  | Security Engineer atau Administrator yang berwenang dapat membuat, mengubah, mengaktifkan, dan menonaktifkan Account Security Policy sesuai kewenangan yang dimiliki.             |

---

## BR-497 – Role-Based Account Access Assignment

| Attribute        | Description                                                                                                                          |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-497                                                                                                                               |
| Requirement Name  | Role-Based Account Access Assignment                                                                                                 |
| Priority          | Critical                                                                                                                             |
| Description       | Sistem harus dapat menetapkan hak akses akun atau kelompok akun kepada Role tertentu, sehingga akses hanya diberikan kepada pengguna dengan Role yang sesuai. |
| Business Value    | Memastikan akses terhadap akun sensitif hanya diberikan berdasarkan fungsi jabatan, bukan berdasarkan identitas pengguna secara individual. |

---

## BR-498 – Organization Scope-Based Account Restriction

| Attribute        | Description                                                                                                                                         |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-498                                                                                                                                              |
| Requirement Name  | Organization Scope-Based Account Restriction                                                                                                       |
| Priority          | Critical                                                                                                                                            |
| Description       | Sistem harus membatasi akses akun berdasarkan Organization Scope, sehingga pengguna hanya dapat mengakses akun pada entitas organisasi sesuai kewenangannya. |
| Acceptance Criteria | Pengguna yang berada di luar Organization Scope yang ditetapkan tidak dapat melihat, memposting, atau menyetujui transaksi pada akun terkait.        |

---

## BR-499 – Sensitive Account Classification and Restricted Account Group

| Attribute        | Description                                                                                                                             |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-499                                                                                                                                  |
| Requirement Name  | Sensitive Account Classification and Restricted Account Group                                                                          |
| Priority          | High                                                                                                                                    |
| Description       | Sistem harus dapat mengklasifikasikan akun sebagai Sensitive Account dan mengelompokkannya ke dalam Restricted Account Group untuk penerapan kebijakan keamanan secara massal. |
| Business Value    | Mempermudah pengelolaan kebijakan keamanan pada sekelompok akun dengan karakteristik risiko yang serupa.                                |

---

## BR-500 – Account-Level Permission Matrix

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-500                                                                                                                            |
| Requirement Name  | Account-Level Permission Matrix                                                                                                   |
| Priority          | Critical                                                                                                                          |
| Description       | Sistem harus menyediakan matriks hak akses pada level akun yang membedakan jenis akses View, Post, Approve, dan Modify secara terpisah. |
| Acceptance Criteria | Administrator dapat mengonfigurasi kombinasi hak akses yang berbeda untuk Role yang berbeda pada akun yang sama.                  |

---

## BR-501 – Segregation of Duties Enforcement on Account Posting

| Attribute        | Description                                                                                                                                |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-501                                                                                                                                     |
| Requirement Name  | Segregation of Duties Enforcement on Account Posting                                                                                      |
| Priority          | Critical                                                                                                                                   |
| Description       | Sistem harus mencegah pengguna yang sama membuat sekaligus menyetujui transaksi pada akun sensitif yang sama, sesuai prinsip segregation of duties. |
| Business Value    | Mengurangi risiko kecurangan (fraud) dan kesalahan pencatatan pada akun-akun yang bersifat kritis.                                          |

---

## BR-502 – Time-Bound / Temporary Access Grant

| Attribute        | Description                                                                                                                          |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-502                                                                                                                              |
| Requirement Name  | Time-Bound / Temporary Access Grant                                                                                                 |
| Priority          | Medium                                                                                                                              |
| Description       | Sistem harus mendukung pemberian akses sementara terhadap akun sensitif dengan periode berlaku tertentu yang otomatis berakhir.      |
| Acceptance Criteria | Akses sementara secara otomatis dicabut oleh sistem setelah tanggal berakhir tanpa memerlukan intervensi manual.                     |

---

## BR-503 – Emergency (Break-Glass) Access

| Attribute        | Description                                                                                                                                    |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-503                                                                                                                                         |
| Requirement Name  | Emergency (Break-Glass) Access                                                                                                                 |
| Priority          | High                                                                                                                                            |
| Description       | Sistem harus menyediakan mekanisme akses darurat yang memungkinkan pengguna tertentu mengakses akun sensitif di luar kebijakan normal dalam kondisi mendesak, dengan pencatatan dan persetujuan ulang wajib. |
| Business Value    | Menjamin kelangsungan proses bisnis kritis tanpa mengorbankan akuntabilitas dan jejak audit.                                                   |

---

## BR-504 – Access Request and Approval Workflow

| Attribute        | Description                                                                                                                    |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-504                                                                                                                         |
| Requirement Name  | Access Request and Approval Workflow                                                                                           |
| Priority          | High                                                                                                                            |
| Description       | Sistem harus menyediakan alur permintaan dan persetujuan (approval workflow) yang configurable untuk setiap perubahan hak akses terhadap akun sensitif. |
| Acceptance Criteria | Setiap permintaan akses baru, perubahan, atau pencabutan hak akses tidak berlaku efektif sebelum melalui approval sesuai konfigurasi. |

---

## BR-505 – Access Recertification and Audit Trail

| Attribute        | Description                                                                                                                                       |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-505                                                                                                                                             |
| Requirement Name  | Access Recertification and Audit Trail                                                                                                             |
| Priority          | Critical                                                                                                                                           |
| Description       | Sistem harus mendukung peninjauan ulang (recertification) hak akses akun secara berkala serta mencatat seluruh riwayat perubahan akses dalam audit trail yang tidak dapat diubah. |
| Business Value    | Memberikan jaminan bahwa hak akses yang berjalan senantiasa relevan dan dapat ditelusuri untuk kebutuhan audit internal maupun eksternal.          |

---

# 9.6.3.20.4 Business Rules

| Business Rule ID | Description                                                                                                                                                     |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-ASEC-001    | Setiap akun atau kelompok akun sensitif harus memiliki minimal satu Account Security Policy yang mendefinisikan Role dan Organization Scope yang diizinkan mengakses. |
| BRule-ASEC-002    | Hak akses posting ke akun sensitif hanya dapat diberikan kepada Role yang terdaftar secara eksplisit dalam Account Security Policy dan berada dalam Organization Scope yang sesuai. |
| BRule-ASEC-003    | Sistem harus menerapkan segregation of duties sehingga pengguna yang membuat jurnal tidak dapat menyetujui jurnal yang sama pada akun sensitif yang sama.       |
| BRule-ASEC-004    | Akses sementara (time-bound access) harus memiliki tanggal mulai dan tanggal berakhir, dan dicabut secara otomatis oleh sistem ketika periode berakhir.        |
| BRule-ASEC-005    | Setiap penggunaan akses darurat (break-glass) wajib dicatat dalam audit trail dan disetujui ulang (post-approval) oleh Security Engineer atau CFO dalam batas waktu yang ditentukan. |
| BRule-ASEC-006    | Perubahan konfigurasi Account Security, baik penambahan, modifikasi, maupun pencabutan hak akses, wajib melalui approval workflow dan tercatat dalam audit trail. |
| BRule-ASEC-007    | Account Security Policy wajib ditinjau ulang (recertification) secara berkala sesuai jadwal yang dikonfigurasi, dan hak akses yang tidak lagi relevan harus dicabut. |

---

# 9.6.3.20.5 Data Attributes

| Attribute                     | Description                                                              |
| ------------------------------ | ------------------------------------------------------------------------- |
| Account Security Policy ID    | Identitas unik sistem.                                                   |
| Applicable Account/Account Group | Akun atau kelompok akun yang menjadi target kebijakan keamanan.       |
| Role                           | Role yang diberikan hak akses.                                           |
| Organization Scope            | Cakupan entitas organisasi yang berlaku bagi kebijakan akses.            |
| Permission Type               | View, Post, Approve, Modify.                                             |
| Access Type                   | Standard, Temporary, Emergency (Break-Glass).                            |
| Effective From / To           | Periode berlaku hak akses.                                               |
| Approval Status               | Draft, Pending Approval, Approved, Published, Archived.                  |
| Status                        | Active, Suspended, Inactive, Archived.                                   |
| Last Recertification Date     | Tanggal terakhir peninjauan ulang hak akses.                             |
| Created By / Created Date     | Informasi pembuatan data.                                                |
| Updated By / Updated Date     | Informasi perubahan terakhir.                                            |

---

# 9.6.3.20.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                          |
| --------------- | ------------------- | ------------ | ----------------------------------------- |
| BR-496          | BO-018, BO-028      | SH-026, SH-003 | Account Security Policy Administration  |
| BR-497          | BO-018              | SH-026       | Role-Based Account Access Management    |
| BR-498          | BO-018              | SH-026       | Organization Scope Access Enforcement   |
| BR-499          | BO-018              | SH-003       | Sensitive Account Classification        |
| BR-500          | BO-018              | SH-026       | Account Permission Configuration        |
| BR-501          | BO-028              | SH-003       | Segregation of Duties Monitoring        |
| BR-502          | BO-018              | SH-026       | Temporary Access Provisioning           |
| BR-503          | BO-028              | SH-026       | Emergency Access Handling               |
| BR-504          | BO-018              | SH-026       | Access Request & Approval Administration |
| BR-505          | BO-028              | SH-003, SH-026 | Access Recertification & Audit Review  |

---

# 9.6.3.20.7 Summary

Account Security menyediakan lapisan kontrol akses granular pada level akun dalam Chart of Accounts, memastikan hanya pengguna dengan Role dan Organization Scope yang sesuai yang dapat melihat, memposting, atau menyetujui transaksi pada akun sensitif. Melalui kombinasi klasifikasi akun sensitif, permission matrix, segregation of duties, akses sementara maupun darurat, serta audit trail dan recertification berkala, Account Security menjadi fondasi tata kelola keamanan yang mendukung integritas data keuangan pada Financial Management Domain.
