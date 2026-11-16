# 9.6.3.41 Financial Reporting Account Integration Requirements

---

# 9.6.3.41.1 Overview

Financial Reporting Account Integration Requirements mendefinisikan bagaimana setiap akun pada Chart of Accounts diintegrasikan, direferensikan, dan digunakan secara spesifik oleh proses Financial Reporting, dilihat dari sudut pandang Account/COA — bukan dari sudut pandang proses penyusunan laporan itu sendiri. Dokumen ini melengkapi Financial Statement Mapping (9.6.3.25) yang membahas mekanisme pemetaan akun ke baris laporan secara umum, dengan memperluas cakupan pada skenario multi-standar akuntansi, akun untuk laporan segmen usaha, penanganan akun baru yang belum dipetakan pada saat laporan digenerate, serta validasi kelengkapan mapping sebelum laporan final diterbitkan.

Fokus dokumen ini adalah pada perilaku Account itu sendiri ketika berinteraksi dengan proses Financial Reporting: status pemetaan akun (mapped/unmapped/suspense), atribut segmen usaha yang melekat pada akun, riwayat mapping akun antar periode untuk kebutuhan laporan komparatif, serta konsistensi pemetaan akun yang sama pada Chart of Accounts bersama (shared COA) lintas entitas saat konsolidasi. Dengan demikian, Account bukan sekadar objek pasif yang direferensikan, melainkan entitas yang memiliki siklus hidup integrasi tersendiri terhadap proses pelaporan keuangan.

Requirement pada dokumen ini tidak menduplikasi requirement umum Financial Reporting (BR-676 s/d BR-685) maupun requirement umum Financial Statement Mapping (BR-546 s/d BR-555), melainkan menjembatani keduanya dari perspektif akun sebagai unit terkecil integrasi data.

---

# 9.6.3.41.2 Business Capability

Financial Reporting Account Integration menyediakan kapabilitas berikut:

* Resolusi Mapping Akun secara Otomatis Berdasarkan Standar Akuntansi Aktif saat Laporan Digenerate.
* Penandaan Akun sebagai Segment Reporting Account untuk Kebutuhan Laporan Segmen Usaha.
* Deteksi Akun Baru yang Belum Dipetakan (Unmapped Account) secara Real-Time pada saat Runtime Pelaporan.
* Gerbang Validasi Kelengkapan Mapping (Completeness Gate) sebelum Laporan Difinalisasi/Diterbitkan.
* Rekonsiliasi Silang Mapping Akun antar Standar Akuntansi yang Berbeda.
* Penyimpanan Snapshot Mapping Historis Akun untuk Kebutuhan Laporan Komparatif Antar Periode.
* Ketertelusuran Kontribusi Saldo Akun terhadap Baris Laporan Keuangan (Line Item Contribution Traceability).
* Penampungan Sementara Akun yang Tidak Dapat Dipetakan pada Akun Suspense Pelaporan.
* Saran Mapping Otomatis Berdasarkan Atribut Akun (Account Type, Account Class, Account Category).
* Pemeriksaan Konsistensi Mapping Akun pada Shared Chart of Accounts Lintas Entitas untuk Konsolidasi.
* Pengelolaan Dampak Perubahan Struktur Akun terhadap Mapping Laporan yang Sudah Berjalan.
* Audit Trail Interaksi Akun terhadap Proses Pelaporan Keuangan.

---

# 9.6.3.41.3 Business Requirement Catalogue

## BR-986 – Multi-Standard Account Mapping Resolution at Report Generation

| Attribute           | Description                                                                                                                                                                                                    |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-986                                                                                                                                                                                                          |
| Requirement Name    | Multi-Standard Account Mapping Resolution at Report Generation                                                                                                                                                 |
| Category            | Financial Reporting Account Integration                                                                                                                                                                        |
| Priority            | Critical                                                                                                                                                                                                        |
| Business Objective  | BO-022                                                                                                                                                                                                          |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                                                               |
| Description         | Sistem harus mampu meresolusi mapping akun yang berbeda untuk setiap standar akuntansi yang aktif (PSAK, IFRS, Tax Basis, Management Reporting) secara otomatis pada saat laporan keuangan digenerate, berdasarkan standar akuntansi yang dipilih pengguna atau yang berlaku untuk Financial Organization tertentu. |
| Business Rationale  | Satu akun dapat memiliki mapping berbeda pada tiap standar akuntansi; tanpa resolusi otomatis, pengguna berisiko menggunakan mapping standar yang salah sehingga laporan tidak sesuai standar yang dipersyaratkan. |
| Business Value      | Menjamin laporan keuangan yang dihasilkan selalu menggunakan mapping akun yang benar sesuai standar akuntansi yang dipilih, tanpa intervensi manual.                                                          |
| Acceptance Criteria | Sistem secara otomatis memilih mapping akun yang sesuai dengan standar akuntansi yang dipilih pada saat laporan digenerate, dan menampilkan indikator standar akuntansi yang digunakan pada laporan yang dihasilkan. |

---

## BR-987 – Segment Reporting Account Tagging

| Requirement ID | Requirement Name                   | Priority | Description                                                                                                                                                       |
| -------------- | ------------------------------------ | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-987         | Segment Reporting Account Tagging   | High     | Sistem harus menyediakan atribut penanda pada akun (Segment Reporting Flag) beserta kode segmen usaha (Business Segment Code) agar akun dapat digunakan dalam penyusunan Laporan Segmen Usaha secara otomatis. |

---

## BR-988 – Unmapped New Account Detection at Report Runtime

| Requirement ID | Requirement Name                          | Priority | Description                                                                                                                                                     |
| -------------- | -------------------------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-988         | Unmapped New Account Detection at Report Runtime | Critical | Sistem harus mendeteksi dan menandai akun baru yang belum memiliki mapping ke baris laporan keuangan pada saat proses generate laporan dijalankan, serta menampilkan daftar akun tersebut kepada pengguna sebelum laporan diselesaikan. |

---

## BR-989 – Mapping Completeness Gate Prior to Report Finalization

| Requirement ID | Requirement Name                              | Priority | Description                                                                                                                                                 |
| -------------- | ------------------------------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-989         | Mapping Completeness Gate Prior to Report Finalization | Critical | Sistem harus menerapkan gerbang validasi yang mencegah laporan keuangan difinalisasi (published) apabila terdapat akun aktif bersaldo tidak nol yang belum dipetakan pada standar akuntansi yang digunakan. |

---

## BR-990 – Account-to-Line-Item Cross-Standard Reconciliation

| Requirement ID | Requirement Name                                  | Priority | Description                                                                                                                                        |
| -------------- | ---------------------------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-990         | Account-to-Line-Item Cross-Standard Reconciliation  | High     | Sistem harus menyediakan rekonsiliasi yang membandingkan hasil mapping akun ke baris laporan antar standar akuntansi yang berbeda, untuk memastikan tidak terdapat akun yang hilang atau tumpang tindih pemetaan antar standar. |

---

## BR-991 – Historical Mapping Snapshot for Comparative Statements

| Requirement ID | Requirement Name                              | Priority | Description                                                                                                                                       |
| -------------- | ------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-991         | Historical Mapping Snapshot for Comparative Statements | High     | Sistem harus menyimpan snapshot mapping akun pada setiap periode laporan yang telah diterbitkan, sehingga laporan komparatif antar periode tetap merujuk pada mapping historis yang berlaku pada periode terkait, bukan mapping terkini. |

---

## BR-992 – Account Balance Contribution Traceability per Line Item

| Requirement ID | Requirement Name                                | Priority | Description                                                                                                                            |
| -------------- | ---------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| BR-992         | Account Balance Contribution Traceability per Line Item | High     | Sistem harus mampu menampilkan rincian kontribusi saldo tiap akun terhadap suatu baris laporan keuangan (drill-down dari baris laporan ke akun sumber beserta nilainya). |

---

## BR-993 – Suspense/Unmapped Account Holding Treatment

| Requirement ID | Requirement Name                             | Priority | Description                                                                                                                                     |
| -------------- | ------------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-993         | Suspense/Unmapped Account Holding Treatment      | Medium   | Sistem harus menampung sementara saldo akun yang belum dapat dipetakan secara otomatis ke dalam Akun Suspense Pelaporan (Reporting Suspense Account), agar laporan tetap dapat disusun sambil menunggu penyelesaian mapping manual. |

---

## BR-994 – Account Attribute-Driven Automatic Mapping Suggestion

| Requirement ID | Requirement Name                                | Priority | Description                                                                                                                                 |
| -------------- | ---------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| BR-994         | Account Attribute-Driven Automatic Mapping Suggestion | Medium   | Sistem harus menyediakan saran mapping otomatis untuk akun baru berdasarkan kemiripan atribut akun (Account Type, Account Class, Account Category) dengan akun yang sudah dipetakan sebelumnya. |

---

## BR-995 – Cross-Entity Account Mapping Consistency Check for Consolidation

| Requirement ID | Requirement Name                                          | Priority | Description                                                                                                                                       |
| -------------- | -------------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-995         | Cross-Entity Account Mapping Consistency Check for Consolidation | Critical | Sistem harus memvalidasi konsistensi mapping akun yang sama pada Chart of Accounts bersama (shared COA) di seluruh entitas anak sebelum laporan konsolidasi disusun, dan menandai ketidaksesuaian mapping antar entitas. |

---

# 9.6.3.41.4 Business Rules

| Business Rule ID  | Description                                                                                                                                                       |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-FREPAI-001   | Setiap akun baru yang dibuat pada Chart of Accounts secara otomatis berstatus "Unmapped" hingga dipetakan ke minimal satu baris laporan pada setiap standar akuntansi yang aktif. |
| BRule-FREPAI-002   | Laporan keuangan final tidak dapat diterbitkan (published) apabila terdapat akun aktif dengan saldo tidak nol yang berstatus Unmapped pada standar akuntansi yang digunakan untuk laporan tersebut. |
| BRule-FREPAI-003   | Akun yang ditandai sebagai Segment Reporting Account wajib memiliki atribut Business Segment Code yang valid sebelum dapat digunakan dalam penyusunan Laporan Segmen Usaha. |
| BRule-FREPAI-004   | Perubahan mapping akun pada satu standar akuntansi tidak boleh memengaruhi mapping akun yang sama pada standar akuntansi lain, kecuali dikonfigurasi secara eksplisit sebagai shared mapping. |
| BRule-FREPAI-005   | Snapshot mapping akun pada saat laporan suatu periode diterbitkan wajib disimpan agar laporan komparatif periode berikutnya tetap merujuk pada mapping historis yang berlaku saat itu. |
| BRule-FREPAI-006   | Akun yang tidak dapat dipetakan secara otomatis wajib ditampung sementara pada Akun Suspense Pelaporan hingga dipetakan secara manual oleh pemilik proses (process owner). |
| BRule-FREPAI-007   | Konsolidasi laporan lintas entitas mensyaratkan kesesuaian pemetaan akun yang sama pada shared Chart of Accounts di seluruh entitas anak yang dikonsolidasi; ketidaksesuaian wajib diselesaikan sebelum laporan konsolidasi difinalisasi. |

---

# 9.6.3.41.5 Data Attributes

| Attribute                          | Description                                                                                     |
| ------------------------------------ | --------------------------------------------------------------------------------------------------- |
| Account Reference                    | Referensi akun pada Chart of Accounts yang berinteraksi dengan proses Financial Reporting.        |
| Accounting Standard Context          | Standar akuntansi yang menjadi konteks resolusi mapping akun (PSAK, IFRS, Tax Basis, Management). |
| Financial Statement Line Item(s)     | Baris laporan keuangan hasil resolusi mapping untuk akun pada standar akuntansi terkait.           |
| Mapping Status                      | Status mapping akun: Mapped, Unmapped, Suspense.                                                    |
| Segment Reporting Flag               | Penanda apakah akun digunakan dalam Laporan Segmen Usaha.                                          |
| Business Segment Code                | Kode segmen usaha yang melekat pada akun apabila Segment Reporting Flag aktif.                    |
| Suspense Account Reference           | Referensi Akun Suspense Pelaporan tempat saldo akun yang belum dipetakan ditampung sementara.       |
| Mapping Snapshot Period/Version      | Periode dan versi mapping yang tersimpan sebagai snapshot historis.                                |
| Cross-Standard Mapping Link          | Tautan antar mapping akun yang sama pada standar akuntansi yang berbeda untuk kebutuhan rekonsiliasi. |
| Contribution Amount to Line Item     | Nilai kontribusi saldo akun terhadap suatu baris laporan keuangan pada periode tertentu.            |
| Consolidation Mapping Consistency Flag | Penanda konsistensi mapping akun yang sama antar entitas anak dalam konsolidasi.                  |
| Auto-Mapping Suggestion Source       | Referensi akun acuan yang digunakan sebagai dasar saran mapping otomatis.                          |
| Last Mapping Validation Date         | Tanggal terakhir validasi kelengkapan mapping dilakukan terhadap akun.                              |
| Created By / Created Date            | Informasi pembuatan data integrasi mapping akun.                                                    |
| Updated By / Updated Date            | Informasi perubahan terakhir data integrasi mapping akun.                                            |

---

# 9.6.3.41.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                                    |
| -------------- | ------------------- | ----------- | -------------------------------------------------- |
| BR-986         | BO-022              | SH-003      | Multi-Standard Mapping Resolution Engine            |
| BR-987         | BO-022              | SH-003      | Segment Reporting Account Configuration              |
| BR-988         | BO-022              | SH-003      | Unmapped Account Detection at Runtime                |
| BR-989         | BO-022              | SH-003      | Report Finalization Completeness Gate                |
| BR-990         | BO-022              | SH-003      | Cross-Standard Mapping Reconciliation                |
| BR-991         | BO-004              | SH-003      | Historical Mapping Snapshot Management               |
| BR-992         | BO-022              | SH-003      | Line Item Contribution Drill-Down                    |
| BR-993         | BO-022              | SH-003      | Reporting Suspense Account Management                |
| BR-994         | BO-009              | SH-003      | Automatic Mapping Suggestion Processing              |
| BR-995         | BO-004              | SH-003      | Consolidation Mapping Consistency Validation          |

---

# 9.6.3.41.7 Summary

Financial Reporting Account Integration Requirements menetapkan bagaimana Account/Chart of Accounts berperan aktif dalam proses Financial Reporting, mencakup resolusi mapping multi-standar akuntansi, penandaan akun untuk laporan segmen usaha, deteksi dan penanganan akun baru yang belum dipetakan, gerbang validasi kelengkapan mapping sebelum laporan diterbitkan, penyimpanan snapshot mapping historis, serta pemeriksaan konsistensi mapping akun lintas entitas untuk konsolidasi. Requirement pada dokumen ini melengkapi, bukan menduplikasi, requirement umum Financial Reporting (9.5.16) dan Financial Statement Mapping (9.6.3.25), dengan fokus khusus pada perspektif integrasi Account.
