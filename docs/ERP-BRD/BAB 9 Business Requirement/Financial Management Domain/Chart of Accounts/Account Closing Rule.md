# 9.6.3.17 Account Closing Rule Requirements

---

# 9.6.3.17.1 Overview

Account Closing Rule merupakan komponen konfigurasi dalam Chart of Accounts yang mendefinisikan perlakuan khusus terhadap suatu akun atau kelompok akun pada saat proses penutupan periode maupun tahun buku, misalnya penutupan akun Laba Rugi (Revenue, Expense, Cost of Goods Sold) ke akun Retained Earnings pada akhir tahun fiskal. Account Closing Rule memastikan proses tutup buku (period-end closing dan year-end closing) berjalan secara konsisten, terotomasi, dan sesuai dengan standar akuntansi yang berlaku di seluruh entitas organisasi.

---

# 9.6.3.17.2 Business Capability

Account Closing Rule menyediakan kapabilitas berikut:

* Standard Closing Rule (Profit & Loss to Retained Earnings).
* Custom Closing Rule Configuration per Account atau Account Category.
* Multi-Level Closing Support (Period-End Closing dan Year-End Closing).
* Closing Sequence/Execution Order Management.
* Automatic Closing Journal Entry Generation.
* Reversing Entry Generation untuk Saldo Awal Periode Berikutnya.
* Multi Accounting Standard/Ledger Closing Support.
* Closing Simulation dan Preview sebelum Posting Final.
* Effective Dating.
* Versioning.
* Approval Workflow.
* Audit Trail.

---

# 9.6.3.17.3 Business Requirement Catalogue

## BR-466 – Account Closing Rule Definition

| Attribute           | Description                                                                                                                                                                        |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-466                                                                                                                                                                             |
| Requirement Name    | Account Closing Rule Definition                                                                                                                                                   |
| Category            | Account Closing Rule                                                                                                                                                              |
| Priority            | Critical                                                                                                                                                                           |
| Business Objective  | BO-022                                                                                                                                                                             |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                                  |
| Description         | Sistem harus menyediakan Account Closing Rule sebagai objek konfigurasi untuk mendefinisikan perlakuan khusus suatu akun atau kelompok akun pada saat proses penutupan periode maupun tahun buku, termasuk penutupan akun Laba Rugi ke akun Retained Earnings. |
| Business Rationale  | Penutupan akun yang dilakukan secara manual atau tidak konsisten berisiko menimbulkan kesalahan saldo awal periode berikutnya serta ketidaksesuaian laporan keuangan antar entitas. |
| Business Value      | Menjamin proses penutupan buku besar berjalan konsisten, akurat, dan sesuai standar akuntansi yang berlaku di seluruh organisasi.                                                 |
| Acceptance Criteria | Administrator dapat membuat, mengubah, menonaktifkan, dan mengelola Account Closing Rule sesuai kewenangan yang dimiliki.                                                         |

---

## BR-467 – Standard Closing to Retained Earnings

| Attribute        | Description                                                                                                                                                  |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-467                                                                                                                                                       |
| Requirement Name  | Standard Closing to Retained Earnings                                                                                                                        |
| Priority          | Critical                                                                                                                                                     |
| Description       | Sistem harus menyediakan closing rule standar yang menutup seluruh akun Laba Rugi (Revenue, Expense, Cost of Goods Sold) ke akun Retained Earnings secara otomatis pada akhir tahun fiskal. |
| Business Value    | Mengeliminasi risiko kesalahan manual dalam penutupan akun Laba Rugi pada setiap akhir tahun buku.                                                          |

---

## BR-468 – Closing Rule per Account Category

| Attribute        | Description                                                                                                                                                          |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-468                                                                                                                                                              |
| Requirement Name  | Closing Rule per Account Category                                                                                                                                   |
| Priority          | High                                                                                                                                                                 |
| Description       | Sistem harus memungkinkan Account Closing Rule dikonfigurasi pada level Account Category maupun level Account individual, sehingga perlakuan penutupan dapat diterapkan secara massal maupun spesifik. |
| Acceptance Criteria | Pengguna dapat mengaitkan satu closing rule ke satu Account Category sehingga berlaku otomatis untuk seluruh akun anggotanya.                                     |

---

## BR-469 – Multi-Level Closing Period Support

| Attribute        | Description                                                                                                                                                                    |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-469                                                                                                                                                                         |
| Requirement Name  | Multi-Level Closing Period Support                                                                                                                                            |
| Priority          | High                                                                                                                                                                           |
| Description       | Sistem harus mendukung penerapan closing rule pada berbagai level penutupan, yaitu Period-End Closing (bulanan) dan Year-End Closing (tahunan), dengan perlakuan yang dapat berbeda pada masing-masing level. |

---

## BR-470 – Automatic Closing Journal Entry Generation

| Attribute        | Description                                                                                                                                          |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-470                                                                                                                                               |
| Requirement Name  | Automatic Closing Journal Entry Generation                                                                                                           |
| Priority          | Critical                                                                                                                                              |
| Description       | Sistem harus secara otomatis membuat jurnal penutup (closing journal entry) berdasarkan closing rule yang berlaku pada saat proses tutup periode/tahun buku dijalankan. |
| Business Value    | Mempercepat proses tutup buku dan mengurangi ketergantungan pada input jurnal manual oleh staf akuntansi.                                            |

---

## BR-471 – Reversing Entry for Opening Balance

| Attribute        | Description                                                                                                                                                 |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-471                                                                                                                                                      |
| Requirement Name  | Reversing Entry for Opening Balance                                                                                                                        |
| Priority          | High                                                                                                                                                        |
| Description       | Sistem harus mampu membuat jurnal pembalik (reversing entry) atau saldo awal (opening balance) pada periode berikutnya sesuai dengan jenis akun dan closing rule yang diterapkan. |

---

## BR-472 – Closing Sequence Configuration

| Attribute        | Description                                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-472                                                                                                                                            |
| Requirement Name  | Closing Sequence Configuration                                                                                                                    |
| Priority          | Medium                                                                                                                                            |
| Description       | Sistem harus menyediakan pengaturan urutan (sequence) eksekusi closing rule ketika lebih dari satu rule berlaku pada satu proses penutupan, guna menghindari konflik hasil penutupan. |

---

## BR-473 – Multi Accounting Standard Closing

| Attribute        | Description                                                                                                                                                        |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-473                                                                                                                                                             |
| Requirement Name  | Multi Accounting Standard Closing                                                                                                                                  |
| Priority          | High                                                                                                                                                               |
| Description       | Sistem harus mendukung konfigurasi closing rule yang berbeda untuk masing-masing accounting standard/ledger, contoh statutory ledger dan group reporting ledger, dalam satu Chart of Accounts. |
| Business Value    | Menjamin kepatuhan penutupan buku terhadap standar akuntansi lokal maupun standar pelaporan konsolidasi grup.                                                     |

---

## BR-474 – Closing Simulation and Preview

| Attribute        | Description                                                                                                                                          |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-474                                                                                                                                               |
| Requirement Name  | Closing Simulation and Preview                                                                                                                       |
| Priority          | Medium                                                                                                                                               |
| Description       | Sistem harus menyediakan fitur simulasi/preview hasil jurnal penutup sebelum proses posting final dilakukan, agar pengguna dapat melakukan validasi terlebih dahulu. |

---

## BR-475 – Closing Rule Approval Workflow

| Attribute        | Description                                                                                                                                             |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-475                                                                                                                                                  |
| Requirement Name  | Closing Rule Approval Workflow                                                                                                                          |
| Priority          | High                                                                                                                                                    |
| Description       | Sistem harus menerapkan approval workflow pada perubahan Account Closing Rule sebelum rule tersebut aktif digunakan dalam proses penutupan periode/tahun buku. |
| Acceptance Criteria | Closing rule berstatus Draft atau Pending Approval tidak dapat dieksekusi pada proses tutup buku.                                                    |

---

# 9.6.3.17.4 Business Rules

| Business Rule ID | Description                                                                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-ACR-001     | Setiap Account Closing Rule harus dikaitkan dengan minimal satu akun tujuan penutupan (target closing account), seperti akun Retained Earnings. |
| BRule-ACR-002     | Account Closing Rule hanya dapat diterapkan pada akun bertipe Revenue, Expense, atau tipe akun lain yang secara eksplisit dikonfigurasi sebagai temporary account. |
| BRule-ACR-003     | Perubahan Account Closing Rule yang telah dieksekusi pada suatu periode tidak dapat mengubah hasil penutupan yang telah diposting, kecuali melalui mekanisme reversal. |
| BRule-ACR-004     | Proses tutup buku tidak dapat diselesaikan apabila terdapat akun yang wajib memiliki closing rule namun belum memiliki closing rule aktif yang berlaku. |
| BRule-ACR-005     | Closing rule dengan status Draft atau Pending Approval tidak dapat digunakan dalam eksekusi proses penutupan periode/tahun buku.                |
| BRule-ACR-006     | Setiap eksekusi closing rule harus menghasilkan audit trail yang mencatat akun sumber, akun tujuan, nominal, dan waktu eksekusi.                 |
| BRule-ACR-007     | Closing rule berlaku sesuai Effective From/To dan scope organisasi (Company/Business Unit) tempat rule tersebut dikonfigurasi.                   |

---

# 9.6.3.17.5 Data Attributes

| Attribute                          | Description                                                    |
| ----------------------------------- | --------------------------------------------------------------- |
| Account Closing Rule ID            | Identitas unik sistem.                                          |
| Rule Code                          | Kode unik Account Closing Rule.                                |
| Rule Name                          | Nama Account Closing Rule.                                     |
| Source Account/Account Category    | Akun atau kategori akun sumber yang akan ditutup.              |
| Target Closing Account             | Akun tujuan penutupan (contoh: Retained Earnings).             |
| Closing Type                       | Period-End Closing, Year-End Closing.                          |
| Accounting Standard/Ledger         | Standar akuntansi atau ledger tempat rule berlaku.             |
| Execution Sequence                 | Urutan eksekusi closing rule.                                  |
| Reversing Entry Flag               | Penanda apakah rule menghasilkan jurnal pembalik.              |
| Effective From / To                | Periode berlaku rule.                                          |
| Version                            | Nomor versi.                                                    |
| Approval Status                    | Draft, Pending Approval, Approved, Published, Archived.        |
| Status                             | Active, Suspended, Inactive, Archived.                          |
| Created By / Created Date          | Informasi pembuatan data.                                       |
| Updated By / Updated Date          | Informasi perubahan terakhir.                                   |

---

# 9.6.3.17.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                          |
| -------------- | ------------------- | ----------- | ---------------------------------------- |
| BR-466         | BO-022              | SH-003      | Account Closing Rule Administration     |
| BR-467         | BO-022              | SH-003      | Year-End Closing Automation             |
| BR-468         | BO-022              | SH-028      | Account Category Closing Configuration  |
| BR-469         | BO-022              | SH-028      | Period-End Closing Automation           |
| BR-470         | BO-022              | SH-003      | Closing Journal Entry Generation        |
| BR-471         | BO-022              | SH-028      | Reversing Entry Processing              |
| BR-472         | BO-022              | SH-028      | Closing Sequence Management             |
| BR-473         | BO-021              | SH-003      | Multi-Ledger Closing Configuration      |
| BR-474         | BO-008              | SH-028      | Closing Simulation & Validation         |
| BR-475         | BO-011              | SH-010      | Closing Rule Approval Management        |

---

# 9.6.3.17.7 Summary

Account Closing Rule menyediakan mekanisme konfigurasi untuk memastikan proses penutupan periode dan tahun buku berjalan secara konsisten, otomatis, dan sesuai standar akuntansi, termasuk penutupan akun Laba Rugi ke akun Retained Earnings. Account Closing Rule menjadi fondasi bagi requirement lain yang terkait pada Financial Management Domain, khususnya proses Period Closing dan Year-End Closing.
