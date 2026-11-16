# 9.6.3.19 Account Version Requirements

---

# 9.6.3.19.1 Overview

Account Version mengelola histori perubahan struktur dan atribut akun dari waktu ke waktu (versioning) sehingga perubahan Chart of Accounts tidak memengaruhi pelaporan historis. Setiap perubahan signifikan terhadap atribut akun—seperti nama, kategori, normal balance, atau status—dikelola sebagai versi baru dengan periode keberlakuan (effective dating) yang jelas, tanpa menghapus atau menimpa versi sebelumnya. Dengan demikian, laporan keuangan pada suatu periode selalu merujuk pada versi akun yang berlaku pada periode tersebut, menjaga integritas dan akurasi pelaporan historis.

---

# 9.6.3.19.2 Business Capability

Account Version menyediakan kapabilitas berikut:

* Account Attribute Versioning.
* Effective Dated Version Management.
* Version Numbering & Sequencing.
* Historical Reporting Integrity.
* Version Comparison (Diff) antar versi.
* Version Approval Workflow.
* Version Rollback / Revert.
* Version Change Log & Notification.
* Period Lock Protection untuk versi historis.
* Version Audit Trail.
* Multi-Attribute Version Snapshot.
* Version Lookup berdasarkan tanggal transaksi.

---

# 9.6.3.19.3 Business Requirement Catalogue

## BR-486 – Account Version Definition

| Attribute           | Description                                                                                                                                        |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-486                                                                                                                                               |
| Requirement Name    | Account Version Definition                                                                                                                          |
| Category            | Account Version                                                                                                                                     |
| Priority            | Critical                                                                                                                                             |
| Business Objective  | BO-022, BO-028                                                                                                                                       |
| Stakeholder Owner   | SH-003 (CFO), SH-010 (Internal Audit)                                                                                                                |
| Description         | Sistem harus menyediakan Account Version sebagai objek bisnis untuk mengelola histori perubahan struktur dan atribut akun dari waktu ke waktu.       |
| Business Rationale  | Perubahan atribut akun (nama, kategori, normal balance, status) yang terjadi seiring waktu harus tercatat sebagai versi terpisah agar konteks historis setiap transaksi tetap dapat ditelusuri sesuai kondisi akun pada saat transaksi terjadi. |
| Business Value      | Menjaga akurasi dan integritas pelaporan keuangan historis serta menyediakan jejak audit yang lengkap atas perubahan Chart of Accounts.              |
| Acceptance Criteria | Sistem dapat membuat, menyimpan, dan menampilkan versi akun beserta atribut yang berubah tanpa menghapus versi sebelumnya.                           |

---

## BR-487 – Version Creation on Attribute Change

| Attribute      | Description                                                                                                                                   |
| -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-487                                                                                                                                         |
| Requirement Name | Version Creation on Attribute Change                                                                                                         |
| Priority       | Critical                                                                                                                                        |
| Description    | Sistem harus secara otomatis membuat versi baru ketika terjadi perubahan pada atribut akun yang bersifat material, tanpa menimpa versi yang ada. |
| Business Value | Memastikan setiap perubahan atribut akun tercatat sebagai riwayat yang dapat ditelusuri.                                                       |

---

## BR-488 – Effective Dating per Version

| Attribute      | Description                                                                                                                        |
| -------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-488                                                                                                                            |
| Requirement Name | Effective Dating per Version                                                                                                    |
| Priority       | Critical                                                                                                                          |
| Description    | Sistem harus mengelola periode keberlakuan (Effective From / Effective To) untuk setiap versi akun tanpa terjadi tumpang tindih tanggal. |
| Acceptance Criteria | Sistem menolak penyimpanan versi baru apabila periode keberlakuannya tumpang tindih dengan versi lain pada akun yang sama.  |

---

## BR-489 – Historical Reporting Integrity

| Attribute      | Description                                                                                                                              |
| -------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-489                                                                                                                                    |
| Requirement Name | Historical Reporting Integrity                                                                                                          |
| Priority       | Critical                                                                                                                                  |
| Description    | Sistem harus memastikan laporan keuangan historis merujuk pada versi akun yang berlaku pada tanggal transaksi, bukan versi akun terkini. |
| Business Value | Menjamin akurasi laporan keuangan lintas periode meskipun Chart of Accounts telah berubah.                                               |

---

## BR-490 – Version Comparison (Diff)

| Attribute      | Description                                                                                                        |
| -------------- | --------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-490                                                                                                            |
| Requirement Name | Version Comparison (Diff)                                                                                       |
| Priority       | High                                                                                                               |
| Description    | Sistem harus menyediakan fungsi perbandingan (diff) antara dua versi akun untuk menampilkan atribut yang berubah. |
| Acceptance Criteria | Pengguna dapat memilih dua versi akun dan melihat daftar perbedaan atribut secara berdampingan.               |

---

## BR-491 – Version Approval Workflow

| Attribute      | Description                                                                                                                  |
| -------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-491                                                                                                                        |
| Requirement Name | Version Approval Workflow                                                                                                   |
| Priority       | Critical                                                                                                                      |
| Description    | Sistem harus mengarahkan setiap versi akun baru yang mengubah atribut signifikan melalui alur persetujuan sebelum berlaku efektif. |
| Business Value | Menjamin tata kelola dan validasi perubahan Chart of Accounts sebelum berdampak pada pelaporan.                              |

---

## BR-492 – Version Rollback / Revert

| Attribute      | Description                                                                                                    |
| -------------- | -------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-492                                                                                                          |
| Requirement Name | Version Rollback / Revert                                                                                     |
| Priority       | High                                                                                                             |
| Description    | Sistem harus memungkinkan pembuatan versi baru yang mengembalikan atribut akun ke kondisi versi sebelumnya (revert), tanpa menghapus riwayat versi. |
| Acceptance Criteria | Proses rollback tercatat sebagai versi baru dengan referensi ke versi asal yang dipulihkan.                  |

---

## BR-493 – Version Audit Trail

| Attribute      | Description                                                                                                    |
| -------------- | -------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-493                                                                                                          |
| Requirement Name | Version Audit Trail                                                                                           |
| Priority       | Critical                                                                                                        |
| Description    | Sistem harus mencatat metadata lengkap setiap perubahan versi, meliputi pembuat, waktu, alasan perubahan, dan nilai sebelum/sesudah. |
| Business Value | Mendukung kebutuhan audit internal dan kepatuhan atas perubahan Chart of Accounts.                             |

---

## BR-494 – Period Lock Protection

| Attribute      | Description                                                                                                       |
| -------------- | ----------------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-494                                                                                                           |
| Requirement Name | Period Lock Protection                                                                                          |
| Priority       | High                                                                                                              |
| Description    | Sistem harus mencegah perubahan atau penghapusan versi akun yang berlaku pada periode akuntansi yang telah ditutup. |
| Acceptance Criteria | Sistem menampilkan penolakan eksplisit saat pengguna mencoba mengubah versi pada periode tertutup.           |

---

## BR-495 – Version Change Notification

| Attribute      | Description                                                                                                   |
| -------------- | ---------------------------------------------------------------------------------------------------------------- |
| Requirement ID | BR-495                                                                                                         |
| Requirement Name | Version Change Notification                                                                                  |
| Priority       | Medium                                                                                                          |
| Description    | Sistem harus mengirimkan notifikasi kepada pemangku kepentingan terkait saat versi akun baru dibuat, disetujui, atau berlaku efektif. |
| Business Value | Meningkatkan transparansi dan kesadaran pemangku kepentingan atas perubahan Chart of Accounts.                 |

---

# 9.6.3.19.4 Business Rules

| Business Rule ID | Description                                                                                                                                    |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-AVER-001    | Setiap perubahan atribut akun yang bersifat material (nama, kategori, normal balance, status) harus menghasilkan versi baru dan tidak menimpa versi sebelumnya. |
| BRule-AVER-002    | Versi akun harus memiliki Effective From dan tidak boleh tumpang tindih (overlap) dengan versi lain pada akun yang sama.                       |
| BRule-AVER-003    | Laporan keuangan historis harus merujuk pada versi akun yang berlaku pada tanggal transaksi, bukan versi akun terkini.                         |
| BRule-AVER-004    | Versi baru yang mengubah atribut signifikan wajib melalui Approval Workflow sebelum berlaku efektif.                                           |
| BRule-AVER-005    | Versi akun pada periode akuntansi yang telah ditutup tidak dapat diubah atau dihapus.                                                          |
| BRule-AVER-006    | Sistem harus menyimpan seluruh riwayat versi akun tanpa batas waktu untuk keperluan audit.                                                     |
| BRule-AVER-007    | Setiap perubahan versi harus mencatat log perubahan (nilai sebelum vs nilai sesudah) untuk keperluan telusur.                                  |

---

# 9.6.3.19.5 Data Attributes

| Attribute                 | Description                                                        |
| -------------------------- | ---------------------------------------------------------------------- |
| Account Version ID        | Identitas unik sistem untuk versi akun.                            |
| Account ID                | Referensi ke akun induk yang divisikan.                             |
| Version Number            | Nomor urut versi akun.                                              |
| Effective From / To       | Periode keberlakuan versi.                                          |
| Changed Attributes        | Snapshot atribut yang berubah pada versi tersebut.                  |
| Change Reason             | Alasan atau justifikasi perubahan versi.                            |
| Approval Status           | Draft, Pending Approval, Approved, Published, Archived.             |
| Status                    | Active, Suspended, Inactive, Archived.                              |
| Created By / Created Date | Informasi pembuatan data.                                           |
| Updated By / Updated Date | Informasi perubahan terakhir.                                       |

---

# 9.6.3.19.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder    | Future Process                    |
| -------------- | ------------------- | -------------- | ------------------------------------ |
| BR-486         | BO-022, BO-028      | SH-003, SH-010 | Account Version Administration      |
| BR-487         | BO-022              | SH-003         | Account Attribute Change Management |
| BR-488         | BO-022              | SH-003         | Effective Dating Management         |
| BR-489         | BO-022              | SH-003, SH-010 | Historical Financial Reporting      |
| BR-490         | BO-022              | SH-010         | Version Comparison Review           |
| BR-491         | BO-028              | SH-003, SH-010 | Account Version Approval Workflow   |
| BR-492         | BO-022              | SH-003         | Version Rollback Processing         |
| BR-493         | BO-028              | SH-010         | Version Audit Review                |
| BR-494         | BO-028              | SH-010         | Period Lock Enforcement             |
| BR-495         | BO-022              | SH-003         | Version Change Notification         |

---

# 9.6.3.19.7 Summary

Account Version menyediakan mekanisme pengelolaan histori perubahan struktur dan atribut akun dalam Chart of Accounts melalui pendekatan versioning berbasis effective dating. Dengan menjaga setiap versi akun sebagai catatan historis yang utuh, sistem memastikan bahwa perubahan Chart of Accounts tidak memengaruhi akurasi pelaporan keuangan historis. Account Version menjadi fondasi bagi requirement lain yang terkait pada Financial Management Domain, khususnya dalam mendukung integritas pelaporan dan kebutuhan audit.
