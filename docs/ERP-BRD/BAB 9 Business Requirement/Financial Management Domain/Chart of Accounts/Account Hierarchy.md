# 9.6.3.10 Account Hierarchy Requirements

---

# 9.6.3.10.1 Overview

Account Hierarchy mendefinisikan struktur berjenjang (parent-child, rollup) antar akun dalam Chart of Accounts untuk kebutuhan pelaporan ringkas (summary) dan konsolidasi laporan keuangan pada berbagai tingkat detail. Account Hierarchy memungkinkan saldo akun anak (child account) diakumulasikan secara otomatis ke akun induk (parent account) sehingga organisasi dapat menyajikan laporan keuangan mulai dari tingkat paling rinci (transaksional) hingga tingkat paling ringkas (statutory/konsolidasi), tanpa mengubah struktur data transaksi yang mendasarinya. Account Hierarchy dikelola secara configurable dan mendukung kedalaman level tanpa batas, sejalan dengan prinsip Organization Tree First dan Configuration over Customization.

---

# 9.6.3.10.2 Business Capability

Account Hierarchy menyediakan kapabilitas berikut:

* Parent-Child Account Relationship Management.
* Multi-Level Rollup / Summary Balance Calculation.
* Unlimited Hierarchy Depth.
* Multiple Hierarchy Trees (Alternate Hierarchy View).
* Cross-Organization Hierarchy Consolidation.
* Header/Summary Account Posting Control.
* Hierarchy Structural Validation (Circular Reference & Orphan Prevention).
* Hierarchy Effective Dating.
* Hierarchy Versioning.
* Approval Workflow atas Perubahan Struktur.
* Audit Trail dan Change History.
* Hierarchy Visualization (Tree View).

---

# 9.6.3.10.3 Business Requirement Catalogue

## BR-396 – Account Hierarchy Definition

| Attribute           | Description                                                                                                                                                          |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-396                                                                                                                                                                |
| Requirement Name    | Account Hierarchy Definition                                                                                                                                          |
| Category            | Account Hierarchy                                                                                                                                                     |
| Priority            | Critical                                                                                                                                                              |
| Business Objective  | BO-004, BO-020                                                                                                                                                        |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                      |
| Description         | Sistem harus menyediakan Account Hierarchy sebagai objek bisnis yang mendefinisikan struktur berjenjang (parent-child) antar akun dalam Chart of Accounts.            |
| Business Rationale  | Struktur akun yang konsisten dan terpusat diperlukan agar seluruh entitas organisasi memiliki satu acuan hierarki dalam menyusun laporan keuangan pada tiap level.     |
| Business Value      | Menyediakan satu sumber kebenaran (single source of truth) atas struktur hierarki akun bagi seluruh entitas organisasi.                                              |
| Acceptance Criteria | Administrator dapat mendefinisikan, mengubah, dan mempublikasikan Account Hierarchy sesuai kewenangan; sistem menampilkan struktur hierarki secara visual (tree view). |

---

## BR-397 – Parent-Child Account Relationship

| Attribute         | Description                                                                                                                                     |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-397                                                                                                                                          |
| Requirement Name  | Parent-Child Account Relationship                                                                                                              |
| Priority          | Critical                                                                                                                                        |
| Description       | Sistem harus mendukung relasi parent-child antar akun, di mana satu akun induk (parent) dapat memiliki satu atau lebih akun anak (child account). |
| Acceptance Criteria | Pengguna dapat menetapkan dan mengubah relasi parent-child pada akun sesuai kewenangan yang dimiliki.                                          |

---

## BR-398 – Multi-Level Rollup dan Kontrol Posting Akun Summary

| Attribute         | Description                                                                                                                                            |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-398                                                                                                                                                 |
| Requirement Name  | Multi-Level Rollup dan Kontrol Posting Akun Summary                                                                                                    |
| Priority          | Critical                                                                                                                                                |
| Description       | Sistem harus menjumlahkan (rollup) saldo akun anak ke akun induk secara otomatis pada seluruh tingkat hierarki, serta mencegah posting transaksi langsung ke akun header/summary. |
| Business Value    | Menjamin akurasi saldo akun ringkas tanpa risiko kesalahan posting manual.                                                                            |

---

## BR-399 – Unlimited Hierarchy Depth

| Attribute         | Description                                                                                                                                 |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-399                                                                                                                                       |
| Requirement Name  | Unlimited Hierarchy Depth                                                                                                                    |
| Priority          | High                                                                                                                                          |
| Description       | Sistem harus mendukung kedalaman hierarki akun tanpa batas jumlah level, dikonfigurasi tanpa perubahan source code.                          |
| Acceptance Criteria | Administrator dapat menambahkan level hierarki baru pada struktur akun tanpa keterbatasan jumlah level.                                    |

---

## BR-400 – Multiple Hierarchy Trees (Alternate Hierarchy View)

| Attribute         | Description                                                                                                                                                  |
| ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-400                                                                                                                                                        |
| Requirement Name  | Multiple Hierarchy Trees (Alternate Hierarchy View)                                                                                                          |
| Priority          | High                                                                                                                                                          |
| Description       | Sistem harus mendukung lebih dari satu struktur hierarki (alternate hierarchy) atas Chart of Accounts yang sama untuk kebutuhan pelaporan yang berbeda-beda, seperti statutory reporting, management reporting, dan tax reporting. |
| Business Value    | Memberikan fleksibilitas penyajian laporan sesuai kebutuhan masing-masing pemangku kepentingan tanpa mengubah struktur akun dasar.                            |

---

## BR-401 – Cross-Organization Hierarchy Consolidation

| Attribute         | Description                                                                                                                                              |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-401                                                                                                                                                    |
| Requirement Name  | Cross-Organization Hierarchy Consolidation                                                                                                                |
| Priority          | Critical                                                                                                                                                  |
| Description       | Sistem harus mendukung konsolidasi hierarki akun lintas entitas organisasi (Company, Sub Holding, Holding) untuk menghasilkan laporan keuangan konsolidasian. |
| Acceptance Criteria | Sistem dapat menghasilkan laporan konsolidasi berdasarkan hierarki akun yang telah dipetakan pada seluruh entitas terkait.                              |

---

## BR-402 – Hierarchy Effective Dating and Versioning

| Attribute         | Description                                                                                                                                    |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-402                                                                                                                                          |
| Requirement Name  | Hierarchy Effective Dating and Versioning                                                                                                      |
| Priority          | High                                                                                                                                            |
| Description       | Sistem harus mendukung effective dating dan versioning atas struktur hierarki akun agar perubahan struktur dapat ditelusuri berdasarkan periode berlaku. |
| Business Value    | Menjaga integritas historis laporan keuangan meskipun struktur hierarki mengalami perubahan dari waktu ke waktu.                               |

---

## BR-403 – Hierarchy Structural Validation

| Attribute         | Description                                                                                                                                                  |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-403                                                                                                                                                        |
| Requirement Name  | Hierarchy Structural Validation                                                                                                                              |
| Priority          | High                                                                                                                                                          |
| Description       | Sistem harus melakukan validasi struktural terhadap hierarki akun, termasuk pencegahan referensi sirkular (circular reference), akun tanpa induk yang tidak valid (orphan), dan duplikasi relasi. |
| Acceptance Criteria | Sistem menolak konfigurasi hierarki yang menghasilkan referensi sirkular atau relasi tidak valid disertai pesan kesalahan yang jelas.                     |

---

## BR-404 – Hierarchy Change Approval Workflow

| Attribute         | Description                                                                                                    |
| ------------------ | ---------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-404                                                                                                          |
| Requirement Name  | Hierarchy Change Approval Workflow                                                                              |
| Priority          | Medium                                                                                                          |
| Description       | Sistem harus menyediakan approval workflow configurable atas perubahan struktur Account Hierarchy sebelum perubahan tersebut dipublikasikan dan berlaku efektif. |
| Business Value    | Memastikan setiap perubahan struktur hierarki melalui kontrol tata kelola yang memadai sebelum digunakan.        |

---

## BR-405 – Hierarchy Audit Trail and Change History

| Attribute         | Description                                                                                                                                       |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-405                                                                                                                                              |
| Requirement Name  | Hierarchy Audit Trail and Change History                                                                                                            |
| Priority          | High                                                                                                                                                |
| Description       | Sistem harus mencatat audit trail atas seluruh perubahan struktur Account Hierarchy, termasuk penambahan, perpindahan (re-parenting), dan penghapusan relasi akun. |
| Acceptance Criteria | Sistem menampilkan riwayat perubahan hierarki lengkap dengan informasi pengguna, waktu, dan nilai sebelum/sesudah perubahan.                       |

---

# 9.6.3.10.4 Business Rules

| Business Rule ID | Description                                                                                                                     |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| BRule-AH-001      | Setiap akun anak (child account) hanya dapat memiliki satu akun induk (parent) langsung dalam satu hierarchy tree yang sama.    |
| BRule-AH-002      | Sistem harus mencegah terbentuknya referensi sirkular antar akun dalam struktur hierarki.                                       |
| BRule-AH-003      | Akun berstatus header/summary tidak dapat menjadi target posting transaksi jurnal secara langsung.                              |
| BRule-AH-004      | Perubahan struktur hierarki yang mempengaruhi saldo rollup harus melalui approval workflow sebelum berlaku efektif.             |
| BRule-AH-005      | Rollup saldo mengikuti sifat normal balance akun induk dan disesuaikan dengan sign convention akun anak.                        |
| BRule-AH-006      | Penghapusan akun yang masih memiliki child aktif tidak diperbolehkan; re-parenting harus dilakukan terlebih dahulu.             |
| BRule-AH-007      | Setiap perubahan struktur hierarki dicatat dalam audit trail dengan mencantumkan pengguna, waktu, serta nilai sebelum/sesudah.   |

---

# 9.6.3.10.5 Data Attributes

| Attribute                  | Description                                                            |
| --------------------------- | ------------------------------------------------------------------------ |
| Account Hierarchy ID       | Identitas unik sistem.                                                 |
| Hierarchy Name/Code        | Nama atau kode struktur hierarki.                                       |
| Hierarchy Type             | Statutory, Management, Tax, Custom.                                    |
| Parent Account Reference   | Referensi akun induk.                                                   |
| Child Account Reference    | Referensi akun anak.                                                    |
| Level/Depth                | Tingkat kedalaman akun dalam struktur hierarki.                         |
| Rollup Sign                | Tanda penjumlahan saldo (+/-) terhadap akun induk.                      |
| Effective From / To        | Periode berlaku struktur hierarki.                                      |
| Version                    | Nomor versi struktur hierarki.                                          |
| Approval Status            | Draft, Pending Approval, Approved, Published, Archived.                |
| Status                     | Active, Suspended, Inactive, Archived.                                  |
| Created By / Created Date  | Informasi pembuatan data.                                               |
| Updated By / Updated Date  | Informasi perubahan terakhir.                                           |

---

# 9.6.3.10.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                          |
| -------------- | ------------------- | ----------- | ----------------------------------------- |
| BR-396         | BO-004, BO-020       | SH-003      | Account Hierarchy Administration          |
| BR-397         | BO-004               | SH-003      | Account Hierarchy Administration          |
| BR-398         | BO-004               | SH-028      | Financial Statement Summary Reporting     |
| BR-399         | BO-004               | SH-003      | Account Hierarchy Administration          |
| BR-400         | BO-020               | SH-003      | Alternate Financial Reporting             |
| BR-401         | BO-020               | SH-003      | Consolidated Financial Reporting          |
| BR-402         | BO-004               | SH-003      | Account Hierarchy Administration          |
| BR-403         | BO-004               | SH-003      | Account Hierarchy Administration          |
| BR-404         | BO-020               | SH-003      | Account Hierarchy Approval                |
| BR-405         | BO-004               | SH-010      | Financial Audit & Compliance Review       |

---

# 9.6.3.10.7 Summary

Account Hierarchy menyediakan struktur berjenjang parent-child antar akun yang menjadi dasar pelaporan ringkas dan konsolidasi laporan keuangan pada berbagai tingkat detail organisasi. Melalui dukungan multi-level rollup, kedalaman hierarki tanpa batas, alternate hierarchy view, serta konsolidasi lintas entitas organisasi, Account Hierarchy menjadi fondasi bagi requirement lain yang terkait pada Financial Management Domain, khususnya dalam penyusunan laporan keuangan konsolidasian yang akurat dan konsisten.
