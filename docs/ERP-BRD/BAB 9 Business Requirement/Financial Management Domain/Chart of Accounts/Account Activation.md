# 9.6.3.18 Account Activation Requirements

---

# 9.6.3.18.1 Overview

Account Activation mengatur siklus hidup status akun dalam Chart of Accounts, mencakup status Draft, Pending Approval, Active, Suspended, dan Inactive, beserta proses aktivasi dan deaktivasi akun secara terkendali. Kapabilitas ini memastikan bahwa setiap akun hanya dapat digunakan untuk transaksi setelah melalui proses validasi dan persetujuan yang sesuai, serta dapat dinonaktifkan secara aman tanpa mengganggu integritas data historis. Account Activation menjadi mekanisme kontrol governance yang menjaga akuntabilitas perubahan status akun di seluruh struktur organisasi.

---

# 9.6.3.18.2 Business Capability

Account Activation menyediakan kapabilitas berikut:

* Account Status Lifecycle Management (Draft, Pending Approval, Active, Suspended, Inactive).
* Draft Submission untuk Persetujuan Aktivasi.
* Approval Workflow untuk Aktivasi Akun.
* Activation Execution & Effective Dating.
* Account Suspension dengan Alasan Terdokumentasi.
* Account Reactivation dari Status Suspended.
* Account Deactivation (Inactive) dengan Validasi Ketergantungan.
* Restriksi Aktivasi/Deaktivasi Berbasis Saldo dan Transaksi Berjalan.
* Bulk Activation & Bulk Deactivation.
* Notifikasi Perubahan Status Akun.
* Audit Trail Perubahan Status Akun.
* Scope-Based Authorization untuk Aktivasi/Deaktivasi.

---

# 9.6.3.18.3 Business Requirement Catalogue

## BR-476 – Account Status Lifecycle Definition

| Attribute           | Description                                                                                                                                                          |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-476                                                                                                                                                              |
| Requirement Name    | Account Status Lifecycle Definition                                                                                                                                |
| Category            | Account Activation                                                                                                                                                 |
| Priority            | Critical                                                                                                                                                           |
| Business Objective  | BO-028                                                                                                                                                              |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                   |
| Description         | Sistem harus menyediakan status akun berbasis siklus hidup yang terdiri dari Draft, Pending Approval, Active, Suspended, dan Inactive untuk setiap akun dalam Chart of Accounts. |
| Business Rationale  | Akun memerlukan mekanisme kontrol status yang jelas agar penggunaan akun dalam transaksi keuangan hanya terjadi setelah melalui tahapan validasi yang sah.         |
| Business Value      | Menjaga integritas data keuangan dan mencegah penggunaan akun yang belum atau tidak lagi layak digunakan.                                                          |
| Acceptance Criteria | Sistem dapat menampilkan dan mengelola status akun sesuai siklus hidup yang ditetapkan, dengan transisi status yang tervalidasi dan tercatat.                      |

---

## BR-477 – Draft Submission for Activation Approval

| Attribute        | Description                                                                                                                       |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-477                                                                                                                             |
| Requirement Name  | Draft Submission for Activation Approval                                                                                          |
| Priority          | Critical                                                                                                                           |
| Description       | Sistem harus memungkinkan akun berstatus Draft diajukan ke status Pending Approval oleh pengguna yang memiliki otorisasi sesuai scope organisasi. |
| Acceptance Criteria | Pengguna berwenang dapat mengajukan akun Draft ke Pending Approval, dan sistem mencatat waktu serta pelaku pengajuan.           |

---

## BR-478 – Activation Approval Workflow

| Attribute        | Description                                                                                                                        |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-478                                                                                                                              |
| Requirement Name  | Activation Approval Workflow                                                                                                        |
| Priority          | Critical                                                                                                                            |
| Description       | Sistem harus menyediakan alur persetujuan (approval workflow) configurable untuk mengubah status akun dari Pending Approval menjadi Active atau dikembalikan ke Draft dengan catatan penolakan. |
| Business Value    | Memastikan setiap aktivasi akun melalui kontrol persetujuan berjenjang sesuai kebijakan organisasi.                                |

---

## BR-479 – Account Activation Execution

| Attribute        | Description                                                                                                                    |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-479                                                                                                                          |
| Requirement Name  | Account Activation Execution                                                                                                    |
| Priority          | Critical                                                                                                                        |
| Description       | Sistem harus mengaktifkan akun secara otomatis pada tanggal efektif setelah status Approved tercapai, sehingga akun dapat digunakan dalam transaksi. |
| Acceptance Criteria | Akun berubah menjadi status Active pada tanggal efektif yang ditetapkan dan tersedia untuk digunakan dalam transaksi jurnal. |

---

## BR-480 – Account Suspension

| Attribute        | Description                                                                                                                     |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-480                                                                                                                          |
| Requirement Name  | Account Suspension                                                                                                              |
| Priority          | High                                                                                                                            |
| Description       | Sistem harus memungkinkan akun berstatus Active diubah menjadi Suspended secara sementara, disertai alasan yang didokumentasikan, untuk membatasi penggunaan akun tanpa menghapus akun secara permanen. |
| Business Value    | Memberikan mekanisme kontrol sementara atas akun bermasalah tanpa mengganggu riwayat transaksi.                                |

---

## BR-481 – Account Reactivation from Suspended

| Attribute        | Description                                                                                                                    |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-481                                                                                                                        |
| Requirement Name  | Account Reactivation from Suspended                                                                                            |
| Priority          | High                                                                                                                          |
| Description       | Sistem harus memungkinkan akun berstatus Suspended dikembalikan ke status Active melalui persetujuan pengguna berwenang, setelah kondisi penyebab suspensi terselesaikan. |
| Acceptance Criteria | Akun Suspended dapat direaktivasi menjadi Active dengan pencatatan alasan dan pelaku reaktivasi.                             |

---

## BR-482 – Account Deactivation

| Attribute        | Description                                                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-482                                                                                                                          |
| Requirement Name  | Account Deactivation                                                                                                            |
| Priority          | Critical                                                                                                                        |
| Description       | Sistem harus memungkinkan akun diubah menjadi status Inactive secara permanen ketika akun tidak lagi digunakan, dengan tetap mempertahankan riwayat transaksi historis untuk kebutuhan pelaporan dan audit. |
| Business Value    | Menjaga kebersihan daftar akun aktif tanpa menghilangkan jejak historis untuk kepentingan audit dan pelaporan.                  |

---

## BR-483 – Activation and Deactivation Dependency Validation

| Attribute        | Description                                                                                                                         |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-483                                                                                                                              |
| Requirement Name  | Activation and Deactivation Dependency Validation                                                                                   |
| Priority          | Critical                                                                                                                            |
| Description       | Sistem harus memvalidasi keberadaan saldo, transaksi berjalan, atau ketergantungan pada objek lain sebelum akun diizinkan untuk dinonaktifkan atau disuspensi. |
| Acceptance Criteria | Sistem menolak permintaan deaktivasi/suspensi apabila akun masih memiliki saldo aktif atau keterkaitan transaksi yang belum diselesaikan, dan menampilkan pesan peringatan yang jelas. |

---

## BR-484 – Bulk Activation and Deactivation

| Attribute        | Description                                                                                                                  |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-484                                                                                                                        |
| Requirement Name  | Bulk Activation and Deactivation                                                                                              |
| Priority          | Medium                                                                                                                        |
| Description       | Sistem harus menyediakan fungsi aktivasi dan deaktivasi akun secara massal (bulk) berdasarkan kriteria seleksi tertentu, dengan tetap melalui validasi dan approval workflow yang berlaku. |
| Business Value    | Meningkatkan efisiensi pengelolaan status akun dalam volume besar, khususnya saat implementasi awal atau restrukturisasi Chart of Accounts. |

---

## BR-485 – Account Activation Audit Trail and Notification

| Attribute        | Description                                                                                                                    |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-485                                                                                                                        |
| Requirement Name  | Account Activation Audit Trail and Notification                                                                              |
| Priority          | Critical                                                                                                                      |
| Description       | Sistem harus mencatat seluruh perubahan status akun secara lengkap (pelaku, waktu, status sebelum dan sesudah, alasan) serta mengirimkan notifikasi kepada pihak terkait atas setiap perubahan status. |
| Acceptance Criteria | Setiap perubahan status akun tercatat dalam audit trail yang dapat ditelusuri, dan notifikasi terkirim kepada pemilik proses dan pihak berwenang terkait. |

---

# 9.6.3.18.4 Business Rules

| Business Rule ID | Description                                                                                                                   |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| BRule-AACT-001    | Akun hanya dapat digunakan dalam transaksi jurnal apabila berstatus Active.                                                  |
| BRule-AACT-002    | Transisi status akun harus mengikuti alur yang telah ditetapkan: Draft → Pending Approval → Active → Suspended/Inactive, dan tidak diperbolehkan melompati tahapan tanpa otorisasi yang sesuai. |
| BRule-AACT-003    | Akun tidak dapat dinonaktifkan (Inactive) apabila masih memiliki saldo berjalan atau keterkaitan transaksi yang belum diselesaikan. |
| BRule-AACT-004    | Perubahan status akun dari dan ke status Active wajib melalui approval workflow sesuai konfigurasi organisasi.               |
| BRule-AACT-005    | Setiap perubahan status akun wajib disertai alasan (reason) yang dapat diaudit, khususnya untuk status Suspended dan Inactive. |
| BRule-AACT-006    | Akun berstatus Inactive tidak dapat dihapus dari sistem; riwayat transaksi historis harus tetap tersimpan dan dapat diakses untuk pelaporan dan audit. |
| BRule-AACT-007    | Wewenang untuk mengaktifkan, menyuspensi, atau menonaktifkan akun ditentukan berdasarkan scope organisasi dan peran pengguna yang telah dikonfigurasi. |

---

# 9.6.3.18.5 Data Attributes

| Attribute                  | Description                                                                 |
| --------------------------- | ---------------------------------------------------------------------------- |
| Account Activation ID      | Identitas unik sistem untuk pencatatan perubahan status akun.               |
| Account ID                 | Referensi ke akun yang mengalami perubahan status.                          |
| Previous Status            | Status akun sebelum perubahan (Draft, Pending Approval, Active, Suspended, Inactive). |
| New Status                 | Status akun setelah perubahan.                                              |
| Effective Date             | Tanggal berlakunya perubahan status.                                        |
| Reason / Remarks           | Alasan perubahan status, khususnya untuk suspensi dan deaktivasi.           |
| Approval Status            | Draft, Pending Approval, Approved, Rejected.                                |
| Approved By / Approved Date | Informasi pihak dan waktu persetujuan perubahan status.                     |
| Status                     | Status akun terkini (Draft, Pending Approval, Active, Suspended, Inactive). |
| Created By / Created Date  | Informasi pembuatan data.                                                    |
| Updated By / Updated Date  | Informasi perubahan terakhir.                                                |

---

# 9.6.3.18.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                          |
| --------------- | ------------------- | ----------- | ----------------------------------------- |
| BR-476          | BO-028              | SH-003      | Account Status Lifecycle Administration |
| BR-477          | BO-011              | SH-028      | Account Draft Submission                |
| BR-478          | BO-012              | SH-003      | Account Activation Approval             |
| BR-479          | BO-028              | SH-028      | Account Activation Execution            |
| BR-480          | BO-007              | SH-011      | Account Suspension Management           |
| BR-481          | BO-011              | SH-028      | Account Reactivation                    |
| BR-482          | BO-028              | SH-003      | Account Deactivation                    |
| BR-483          | BO-022              | SH-010      | Account Deactivation Validation         |
| BR-484          | BO-029              | SH-028      | Bulk Account Status Management          |
| BR-485          | BO-028              | SH-010      | Account Status Audit & Notification     |

---

# 9.6.3.18.7 Summary

Account Activation menyediakan kontrol siklus hidup status akun yang terstruktur, mulai dari Draft, Pending Approval, Active, Suspended, hingga Inactive, disertai alur persetujuan dan validasi ketergantungan yang memadai. Kapabilitas ini menjaga integritas dan akuntabilitas Chart of Accounts, memastikan hanya akun yang telah tervalidasi dan disetujui yang dapat digunakan dalam transaksi keuangan, serta mendukung governance dan audit trail yang menyeluruh atas seluruh perubahan status akun pada Financial Management Domain.
