# 9.6.3.12 Account Control Requirements

---

# 9.6.3.12.1 Overview

Account Control merupakan mekanisme pengendalian penggunaan akun dalam Chart of Accounts yang membatasi bagaimana, oleh siapa, dan dalam kondisi apa suatu akun dapat digunakan untuk posting transaksi. Account Control mencakup pembatasan akun yang hanya boleh diposting oleh sistem (system-generated), akun yang memerlukan approval khusus sebelum posting, akun kontrol sub-ledger yang tidak boleh diposting langsung secara manual, hingga penguncian sementara suatu akun. Account Control berfungsi sebagai lapisan pengendalian internal yang melengkapi struktur Chart of Accounts guna menjaga integritas data keuangan, mendukung segregation of duties, dan memperkuat audit trail pada seluruh entitas organisasi.

---

# 9.6.3.12.2 Business Capability

Account Control menyediakan kapabilitas berikut:

* System-Only Account Restriction.
* Manual Posting Restriction.
* Approval-Required Posting Control.
* Control Account Enforcement (Sub-Ledger Only Posting).
* Account Locking & Suspension.
* Reconciliation Requirement Flagging.
* Role-Based / Segregation of Duties Posting Restriction.
* Scope-Based Account Control Configuration.
* Violation Attempt Logging & Alerting.
* Maker-Checker untuk Perubahan Account Control.
* Audit Trail.
* Effective Dating & Versioning.

---

# 9.6.3.12.3 Business Requirement Catalogue

## BR-416 – Account Control Definition

| Attribute           | Description                                                                                                                                        |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-416                                                                                                                                              |
| Requirement Name    | Account Control Definition                                                                                                                          |
| Category            | Account Control                                                                                                                                      |
| Priority            | Critical                                                                                                                                             |
| Business Objective  | BO-028                                                                                                                                               |
| Stakeholder Owner   | SH-003 (Chief Financial Officer), SH-010 (Internal Audit)                                                                                           |
| Description         | Sistem harus menyediakan Account Control sebagai objek bisnis untuk mendefinisikan pembatasan penggunaan suatu akun, termasuk jenis kontrol, ruang lingkup organisasi yang berlaku, dan status kontrol tersebut. |
| Business Rationale  | Tanpa mekanisme pengendalian akun yang eksplisit, akun-akun sensitif dapat diposting secara tidak sah sehingga meningkatkan risiko kesalahan pencatatan dan fraud. |
| Business Value      | Menyediakan fondasi pengendalian internal yang konsisten atas penggunaan akun pada seluruh entitas organisasi.                                       |
| Acceptance Criteria | Administrator dengan kewenangan dapat membuat, mengubah, dan mengelola Account Control atas akun tertentu sesuai ruang lingkup organisasi yang berlaku. |

---

## BR-417 – System-Only Posting Restriction

| Attribute        | Description                                                                                                                     |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-417                                                                                                                           |
| Requirement Name  | System-Only Posting Restriction                                                                                                 |
| Priority          | Critical                                                                                                                         |
| Description       | Sistem harus mendukung penandaan akun sebagai System-Only Account sehingga akun tersebut hanya dapat menerima posting yang dihasilkan secara otomatis oleh sistem, seperti realisasi selisih kurs, penutupan buku, atau eliminasi antar entitas. |
| Business Value    | Mencegah intervensi manual pada akun yang seharusnya hanya diisi melalui proses sistem yang telah tervalidasi.                   |

---

## BR-418 – Manual Posting Restriction

| Attribute        | Description                                                                                                                          |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-418                                                                                                                               |
| Requirement Name  | Manual Posting Restriction                                                                                                           |
| Priority          | Critical                                                                                                                             |
| Description       | Sistem harus mendukung penandaan akun sebagai No Manual Posting sehingga akun tersebut tidak dapat diposting melalui entri jurnal manual pada modul mana pun. |
| Acceptance Criteria | Sistem menolak setiap upaya posting manual pada akun yang ditandai No Manual Posting dan menampilkan pesan penolakan yang jelas.   |

---

## BR-419 – Approval-Required Posting Control

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-419                                                                                                                            |
| Requirement Name  | Approval-Required Posting Control                                                                                                 |
| Priority          | High                                                                                                                              |
| Description       | Sistem harus mendukung penandaan akun sebagai Approval Required sehingga setiap jurnal manual yang menyentuh akun tersebut wajib melalui alur persetujuan configurable sebelum diposting final. |
| Business Value    | Menambahkan lapisan validasi tambahan pada akun-akun yang berisiko tinggi terhadap kesalahan atau penyalahgunaan.                 |

---

## BR-420 – Control Account Restriction (Sub-Ledger Only Posting)

| Attribute        | Description                                                                                                                                    |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-420                                                                                                                                          |
| Requirement Name  | Control Account Restriction (Sub-Ledger Only Posting)                                                                                          |
| Priority          | Critical                                                                                                                                        |
| Description       | Sistem harus mendukung penandaan akun sebagai Control Account, yaitu akun ringkasan sub-ledger (piutang, utang, persediaan, aset tetap) yang hanya dapat diposting melalui transaksi sub-ledger terkait dan tidak dapat menerima entri jurnal manual langsung. |
| Acceptance Criteria | Sistem mencegah entri jurnal manual langsung pada akun yang ditandai sebagai Control Account dan mengarahkan pengguna ke modul sub-ledger yang sesuai. |

---

## BR-421 – Account Locking & Suspension

| Attribute        | Description                                                                                                                    |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-421                                                                                                                         |
| Requirement Name  | Account Locking & Suspension                                                                                                   |
| Priority          | High                                                                                                                           |
| Description       | Sistem harus mendukung penguncian sementara suatu akun (Locked/Suspended) sehingga akun tersebut tidak dapat menerima posting apa pun selama periode tertentu, misalnya selama proses rekonsiliasi atau restrukturisasi akun. |
| Business Value    | Mencegah perubahan saldo akun pada periode kritis yang memerlukan stabilitas data.                                            |

---

## BR-422 – Reconciliation Requirement Control

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-422                                                                                                                            |
| Requirement Name  | Reconciliation Requirement Control                                                                                                |
| Priority          | High                                                                                                                              |
| Description       | Sistem harus mendukung penandaan akun sebagai Reconciliation Required sehingga akun tersebut wajib direkonsiliasi sesuai frekuensi yang ditentukan sebelum periode akuntansi dapat ditutup. |
| Acceptance Criteria | Sistem mencegah penutupan periode apabila akun yang ditandai Reconciliation Required belum direkonsiliasi sesuai jadwal yang berlaku. |

---

## BR-423 – Role-Based Posting Restriction (Segregation of Duties)

| Attribute        | Description                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-423                                                                                                                            |
| Requirement Name  | Role-Based Posting Restriction (Segregation of Duties)                                                                           |
| Priority          | Critical                                                                                                                          |
| Description       | Sistem harus mendukung pembatasan posting akun berdasarkan peran atau pengguna tertentu sehingga hanya peran atau pengguna yang berwenang dapat memposting ke akun yang ditandai restricted, guna mendukung segregation of duties. |
| Business Value    | Mengurangi risiko konflik kepentingan dan penyalahgunaan wewenang atas akun-akun sensitif.                                       |

---

## BR-424 – Violation Attempt Logging & Alert

| Attribute        | Description                                                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-424                                                                                                                          |
| Requirement Name  | Violation Attempt Logging & Alert                                                                                              |
| Priority          | High                                                                                                                            |
| Description       | Sistem harus mencatat dan mengirimkan notifikasi setiap kali terjadi upaya posting yang melanggar pengaturan Account Control, termasuk identitas pengguna, waktu kejadian, dan jenis pelanggaran. |
| Business Value    | Meningkatkan kemampuan deteksi dini atas upaya pelanggaran pengendalian akun untuk kepentingan audit dan investigasi.          |

---

## BR-425 – Account Control Change Approval Workflow

| Attribute        | Description                                                                                                                    |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-425                                                                                                                        |
| Requirement Name  | Account Control Change Approval Workflow                                                                                     |
| Priority          | High                                                                                                                          |
| Description       | Sistem harus mewajibkan setiap perubahan pengaturan Account Control atas suatu akun melalui alur persetujuan maker-checker sebelum perubahan tersebut berlaku efektif. |
| Acceptance Criteria | Sistem menolak penerapan langsung perubahan Account Control tanpa persetujuan dari pihak yang berwenang sesuai workflow yang dikonfigurasi. |

---

# 9.6.3.12.4 Business Rules

| Business Rule ID | Description                                                                                                                                                                     |
| ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-ACTL-001    | Setiap akun yang ditandai sebagai System-Only harus menolak seluruh upaya posting manual dari modul jurnal umum, kecuali oleh proses otomatis sistem yang telah terotorisasi. |
| BRule-ACTL-002    | Perubahan pengaturan Account Control (System-Only, No Manual Posting, Approval Required, Control Account, Locked/Suspended) wajib melalui alur persetujuan maker-checker dan tidak dapat diterapkan langsung oleh pembuat perubahan. |
| BRule-ACTL-003    | Akun yang dikonfigurasi sebagai Control Account hanya dapat menerima posting melalui transaksi sub-ledger (piutang, utang, persediaan, aset tetap) dan tidak dapat menerima entri jurnal manual langsung ke General Ledger. |
| BRule-ACTL-004    | Setiap upaya posting yang melanggar pengaturan Account Control harus ditolak oleh sistem dan dicatat dalam log audit termasuk identitas pengguna, waktu, dan jenis pelanggaran. |
| BRule-ACTL-005    | Akun yang berstatus Locked/Suspended tidak dapat menerima posting apa pun sampai statusnya dikembalikan menjadi aktif oleh pihak yang berwenang.                              |
| BRule-ACTL-006    | Akun yang ditandai membutuhkan approval khusus wajib melalui workflow persetujuan sebelum jurnal dapat diposting final, sesuai threshold dan hierarki approval yang berlaku pada ruang lingkup organisasi terkait. |
| BRule-ACTL-007    | Pengaturan Account Control berlaku sesuai ruang lingkup organisasi (scope-based) sehingga suatu akun dapat memiliki kontrol berbeda antar entitas organisasi sesuai konfigurasi yang ditetapkan. |

---

# 9.6.3.12.5 Data Attributes

| Attribute                     | Description                                                                                    |
| ------------------------------ | ------------------------------------------------------------------------------------------------ |
| Account Control ID            | Identitas unik sistem.                                                                          |
| Account Reference              | Referensi ke akun pada Chart of Accounts yang dikenakan kontrol.                                |
| Control Type                   | System-Only, No Manual Posting, Approval Required, Control Account, Locked/Suspended, Reconciliation Required, Role Restricted. |
| Organization Scope             | Ruang lingkup organisasi tempat pengaturan kontrol berlaku.                                     |
| Restricted Role / User List    | Daftar peran atau pengguna yang diizinkan atau dikecualikan dari pembatasan.                     |
| Approval Threshold             | Ambang batas nilai transaksi yang memicu kebutuhan approval.                                    |
| Lock Start Date / End Date     | Periode berlakunya penguncian akun.                                                              |
| Reconciliation Frequency       | Frekuensi rekonsiliasi yang dipersyaratkan (harian, mingguan, bulanan).                          |
| Effective From / To            | Periode berlaku pengaturan Account Control.                                                       |
| Version                        | Nomor versi.                                                                                     |
| Approval Status                | Draft, Pending Approval, Approved, Published, Archived.                                          |
| Status                         | Active, Suspended, Inactive, Archived.                                                            |
| Created By / Created Date      | Informasi pembuatan data.                                                                        |
| Updated By / Updated Date      | Informasi perubahan terakhir.                                                                    |

---

# 9.6.3.12.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder      | Future Process                              |
| --------------- | ------------------- | ---------------- | -------------------------------------------- |
| BR-416          | BO-028              | SH-003, SH-010    | Account Control Administration               |
| BR-417          | BO-028              | SH-010            | System Posting Enforcement                   |
| BR-418          | BO-028              | SH-010            | Manual Journal Restriction                    |
| BR-419          | BO-028              | SH-003            | Approval-Based Posting                        |
| BR-420          | BO-028              | SH-003, SH-010    | Control Account Enforcement                   |
| BR-421          | BO-028              | SH-010            | Account Locking Management                    |
| BR-422          | BO-028              | SH-003            | Reconciliation Compliance                     |
| BR-423          | BO-028              | SH-010            | Segregation of Duties Enforcement             |
| BR-424          | BO-028              | SH-010            | Audit Log & Alert Monitoring                  |
| BR-425          | BO-028              | SH-003, SH-010    | Account Control Change Governance             |

---

# 9.6.3.12.7 Summary

Account Control menyediakan lapisan pengendalian yang membatasi penggunaan akun dalam Chart of Accounts, mencakup pembatasan posting sistem-only, larangan posting manual, kebutuhan approval khusus, penguncian akun, kewajiban rekonsiliasi, hingga pembatasan berbasis peran. Account Control menjadi mekanisme kunci untuk memperkuat pengendalian internal, mendukung segregation of duties, dan menjaga integritas audit trail pada seluruh entitas organisasi dalam Financial Management Domain.
