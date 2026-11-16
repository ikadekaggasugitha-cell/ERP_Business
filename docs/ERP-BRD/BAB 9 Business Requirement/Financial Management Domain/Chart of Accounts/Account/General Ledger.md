# 9.6.3.31 General Ledger Account Integration Requirements

---

# 9.6.3.31.1 Overview

Dokumen ini membahas bagaimana Account (struktur akun dalam Chart of Accounts) diintegrasikan, direferensikan, dan digunakan secara spesifik oleh modul General Ledger (GL), dilihat dari sudut pandang Account itu sendiri — bukan dari sudut pandang proses bisnis jurnal sebagaimana telah dibahas pada dokumen 9.5.4 General Ledger Requirements (BR-556 s/d BR-565).

Fokus dokumen ini adalah pada mekanisme validasi kombinasi segmen akun (Account Combination) pada saat posting jurnal, proses perhitungan dan penyimpanan saldo per akun (account balance) per periode fiskal, kemampuan penelusuran (drill-down) dari saldo akun menuju detail jurnal pembentuknya, serta penanganan akun yang berstatus suspended atau inactive pada saat proses posting berlangsung.

Chart of Accounts berperan sebagai kerangka acuan (reference framework) yang menentukan apakah sebuah jurnal dapat diterima oleh General Ledger. Setiap baris jurnal yang masuk ke GL, baik berasal dari input manual maupun dari sub-ledger seperti Accounts Payable dan Accounts Receivable, wajib merujuk pada kombinasi akun yang valid dan berstatus aktif. Selain sebagai gatekeeper pada saat posting, Account juga menjadi unit agregasi utama bagi seluruh saldo yang disajikan dalam laporan keuangan, sehingga akurasi perhitungan saldo per akun per periode menjadi prasyarat mutlak bagi keandalan laporan keuangan organisasi.

Ruang lingkup dokumen ini mencakup empat area utama: (1) validasi Account Combination saat posting jurnal, (2) perhitungan dan penyimpanan saldo akun per periode, (3) drill-down saldo akun ke jurnal dan dokumen sumber, dan (4) penanganan akun suspended/inactive dalam alur posting. Requirement umum mengenai proses jurnal, closing periode, dan approval workflow tetap menjadi cakupan dokumen 9.5.4 dan tidak diulang di sini.

---

# 9.6.3.31.2 Business Capability

- Validasi otomatis kombinasi segmen akun (Account Combination) terhadap Chart of Accounts sebelum jurnal diterima untuk posting.
- Penolakan otomatis atas baris jurnal yang mereferensikan akun berstatus suspended, inactive, atau non-postable.
- Perhitungan saldo akun (debit total, kredit total, dan saldo bersih) secara real-time pada setiap periode fiskal yang berjalan.
- Penyimpanan saldo awal (opening balance) dan saldo akhir (closing balance) per akun, per periode, dan per scope organisasi.
- Carry-forward saldo akhir periode menjadi saldo awal periode berikutnya secara otomatis pada akun neraca (balance sheet account).
- Reset saldo ke nol pada awal tahun fiskal untuk akun laba-rugi (income statement account) sesuai kebijakan Account Type.
- Penelusuran (drill-down) dari saldo akun pada tingkat ringkasan menuju rincian jurnal pembentuk saldo tersebut.
- Penelusuran lanjutan (drill-through) dari baris jurnal menuju dokumen sumber transaksi di modul asal.
- Penyimpanan saldo akun dalam mata uang transaksi dan mata uang fungsional secara paralel untuk akun bertipe monetary.
- Pencegahan reaktivasi otomatis akun suspended tanpa proses otorisasi, meskipun terdapat kebutuhan posting mendesak.
- Penyediaan status keterisian saldo (populated/zero balance) akun sebagai dasar proses rekonsiliasi dan audit.
- Pencatatan jejak akses terhadap fitur drill-down sebagai bagian dari audit trail penggunaan data akun.

---

# 9.6.3.31.3 Business Requirement Catalogue

## BR-886 – Account Combination Validation at Journal Posting

| Attribute           | Description                                                                                                                                                                                                                                  |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-886                                                                                                                                                                                                                                       |
| Requirement Name    | Account Combination Validation at Journal Posting                                                                                                                                                                                           |
| Category            | General Ledger – Account Integration                                                                                                                                                                                                        |
| Priority            | Critical                                                                                                                                                                                                                                     |
| Business Objective  | BO-004, BO-022                                                                                                                                                                                                                               |
| Stakeholder Owner   | SH-003 (CFO), SH-028 (Accounting Department)                                                                                                                                                                                                |
| Description         | Sistem harus memvalidasi setiap baris jurnal yang akan diposting ke General Ledger terhadap Account Combination yang terdaftar dan berstatus aktif pada Chart of Accounts, mencakup kesesuaian seluruh segmen (natural account, cost center, dan segmen lain yang dikonfigurasi). |
| Business Rationale  | Tanpa validasi kombinasi akun yang ketat, jurnal dapat diposting ke kombinasi segmen yang tidak valid atau tidak dikenal, sehingga merusak integritas struktur pelaporan keuangan yang bergantung pada Chart of Accounts.                    |
| Business Value      | Menjamin bahwa setiap saldo yang terbentuk di General Ledger selalu dapat dipetakan pada struktur akun yang sah, sehingga laporan keuangan dapat diandalkan sebagai single source of truth.                                                |
| Acceptance Criteria | Jurnal dengan kombinasi segmen yang tidak terdaftar pada Account Combination ditolak sistem sebelum status posted, disertai pesan kesalahan yang mengidentifikasi segmen yang tidak valid.                                                 |

---

## BR-887 – Segment Value Existence & Cross-Validation Rule Enforcement

| Requirement ID | Requirement Name                                          | Priority | Description                                                                                                                                                     |
| -------------- | ----------------------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-887         | Segment Value Existence & Cross-Validation Rule Enforcement | High     | Sistem harus memastikan setiap nilai segmen pada baris jurnal terdaftar pada Segment Value yang valid, serta menerapkan aturan cross-validation antar segmen sesuai konfigurasi Account Validation Rule sebelum kombinasi diterima. |

---

## BR-888 – Inactive/Suspended Account Rejection at Posting

| Requirement ID | Requirement Name                             | Priority | Description                                                                                                                                                     |
| -------------- | ---------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-888         | Inactive/Suspended Account Rejection at Posting | Critical | Sistem harus menolak posting jurnal yang mereferensikan akun berstatus suspended, inactive, atau non-postable sesuai Account Activation, meskipun kombinasi segmen sebelumnya pernah valid. |

---

## BR-889 – Account Balance Calculation & Period Accumulation

| Requirement ID | Requirement Name                             | Priority | Description                                                                                                                                                    |
| -------------- | ----------------------------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-889         | Account Balance Calculation & Period Accumulation | Critical | Sistem harus menghitung dan menyimpan akumulasi debit, akumulasi kredit, serta saldo bersih per akun untuk setiap periode fiskal secara otomatis pada saat jurnal diposting. |

---

## BR-890 – Opening Balance Carry-Forward per Fiscal Year

| Requirement ID | Requirement Name                        | Priority | Description                                                                                                                                                          |
| -------------- | ------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-890         | Opening Balance Carry-Forward per Fiscal Year | High     | Sistem harus melakukan carry-forward saldo akhir periode sebelumnya menjadi saldo awal periode berjalan secara otomatis, dengan penanganan berbeda untuk akun neraca dan akun laba-rugi sesuai Account Type. |

---

## BR-891 – Multi-Currency Account Balance Storage

| Requirement ID | Requirement Name                    | Priority | Description                                                                                                                                                 |
| -------------- | -------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-891         | Multi-Currency Account Balance Storage | High     | Sistem harus menyimpan saldo akun bertipe monetary/foreign currency dalam mata uang transaksi asal dan mata uang fungsional secara paralel untuk setiap periode. |

---

## BR-892 – Account Balance Drill-Down to Journal Detail

| Requirement ID | Requirement Name                       | Priority | Description                                                                                                                                                   |
| -------------- | ----------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-892         | Account Balance Drill-Down to Journal Detail | High     | Sistem harus menyediakan kemampuan drill-down dari saldo akun pada tingkat ringkasan periode menuju daftar rinci jurnal yang membentuk saldo tersebut, lengkap dengan referensi dokumen dan tanggal posting. |

---

## BR-893 – Account Balance Drill-Through to Source Document

| Requirement ID | Requirement Name                          | Priority | Description                                                                                                                                              |
| -------------- | -------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-893         | Account Balance Drill-Through to Source Document | High     | Sistem harus mendukung penelusuran lanjutan (drill-through) dari baris jurnal hasil drill-down menuju dokumen sumber transaksi pada modul asal, seperti invoice, PO, atau Goods Receipt. |

---

## BR-894 – Real-Time Account Balance Update on Posting

| Requirement ID | Requirement Name                    | Priority | Description                                                                                                                             |
| -------------- | -------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| BR-894         | Real-Time Account Balance Update on Posting | Medium   | Sistem harus memperbarui saldo akun secara real-time setiap kali jurnal berhasil diposting, tanpa memerlukan proses batch terpisah untuk tampilan saldo terkini. |

---

## BR-895 – Account Balance Reconciliation & Control Total Validation

| Requirement ID | Requirement Name                                  | Priority | Description                                                                                                                                                    |
| -------------- | ---------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-895         | Account Balance Reconciliation & Control Total Validation | Medium   | Sistem harus menyediakan mekanisme validasi bahwa total saldo ringkasan per akun senantiasa sama dengan total nilai jurnal detail pembentuknya, dan menandai selisih sebagai eksepsi rekonsiliasi. |

---

# 9.6.3.31.4 Business Rules

| Business Rule ID | Description                                                                                                                                                     |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-GLAI-001    | Setiap baris jurnal wajib mereferensikan Account Combination yang terdaftar dan berstatus aktif pada Chart of Accounts sebelum dapat diposting ke General Ledger. |
| BRule-GLAI-002    | Akun berstatus suspended atau inactive tidak dapat menerima posting baru, terlepas dari validitas kombinasi segmen yang pernah tercatat sebelumnya.              |
| BRule-GLAI-003    | Saldo akun per periode dihitung secara otomatis oleh sistem pada setiap posting jurnal dan tidak dapat diubah secara manual di luar mekanisme posting/reversal.  |
| BRule-GLAI-004    | Saldo awal suatu periode wajib sama dengan saldo akhir periode sebelumnya untuk akun dan scope organisasi yang sama.                                             |
| BRule-GLAI-005    | Akses fitur drill-down dan drill-through atas saldo akun tunduk pada kebijakan Account Security yang berlaku bagi pengguna terkait.                              |
| BRule-GLAI-006    | Akun bertipe monetary/foreign currency wajib menyimpan representasi saldo ganda, yaitu dalam mata uang transaksi dan mata uang fungsional.                       |
| BRule-GLAI-007    | Selisih antara saldo ringkasan akun dan total jurnal detail pembentuknya wajib ditandai secara otomatis sebagai eksepsi rekonsiliasi untuk ditindaklanjuti.       |

---

# 9.6.3.31.5 Data Attributes

| Attribute Name              | Description                                                                                  |
| ---------------------------- | ---------------------------------------------------------------------------------------------- |
| Account Combination ID       | Identitas unik kombinasi segmen akun yang direferensikan oleh baris jurnal.                    |
| Natural Account Segment      | Segmen akun utama yang menentukan klasifikasi akuntansi (aset, liabilitas, pendapatan, biaya). |
| Fiscal Period                | Periode akuntansi tempat saldo akun dihitung dan disimpan.                                     |
| Fiscal Year                  | Tahun fiskal yang menjadi acuan carry-forward saldo akun.                                      |
| Opening Balance               | Saldo awal akun pada periode berjalan, hasil carry-forward dari periode sebelumnya.            |
| Period Debit Total            | Akumulasi nilai debit akun pada periode berjalan.                                              |
| Period Credit Total           | Akumulasi nilai kredit akun pada periode berjalan.                                             |
| Closing Balance               | Saldo akhir akun pada periode berjalan, hasil dari opening balance ditambah aktivitas periode. |
| Transaction Currency Balance  | Saldo akun dalam mata uang transaksi asal.                                                    |
| Functional Currency Balance   | Saldo akun dalam mata uang fungsional organisasi.                                              |
| Account Status               | Status akun (active, suspended, inactive) yang menentukan kelayakan menerima posting.          |
| Journal Reference ID          | Referensi ke jurnal yang membentuk komponen saldo akun tertentu.                               |
| Source Document Reference     | Referensi ke dokumen sumber transaksi pada modul asal untuk keperluan drill-through.            |
| Last Posting Timestamp        | Waktu posting terakhir yang memengaruhi saldo akun.                                            |
| Reconciliation Status         | Status kesesuaian saldo ringkasan akun terhadap total jurnal detail (matched/exception).        |

---

# 9.6.3.31.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                                     |
| --------------- | ------------------- | ------------ | ----------------------------------------------------- |
| BR-886          | BO-004, BO-022       | SH-003       | Account Combination Validation Engine Design           |
| BR-887          | BO-004               | SH-028       | Segment Cross-Validation Rule Configuration            |
| BR-888          | BO-004, BO-022       | SH-028       | Suspended Account Posting Control Design                |
| BR-889          | BO-022               | SH-028       | Account Balance Calculation Engine Design               |
| BR-890          | BO-022               | SH-028       | Fiscal Year Carry-Forward Process Design                |
| BR-891          | BO-022               | SH-003       | Multi-Currency Balance Storage Design                    |
| BR-892          | BO-004, BO-009        | SH-028       | Account Balance Drill-Down Feature Design                |
| BR-893          | BO-009               | SH-028       | Drill-Through to Source Document Design                  |
| BR-894          | BO-009               | SH-003       | Real-Time Balance Update Process Design                  |
| BR-895          | BO-022               | SH-028       | Account Balance Reconciliation Process Design            |

---

# 9.6.3.31.7 Summary

Dokumen ini menetapkan requirement spesifik mengenai bagaimana Account/Chart of Accounts diintegrasikan dengan General Ledger, mencakup validasi Account Combination pada saat posting, perhitungan dan penyimpanan saldo akun per periode, kemampuan drill-down dan drill-through dari saldo menuju jurnal dan dokumen sumber, serta penanganan akun suspended/inactive dalam alur posting. Berbeda dengan dokumen 9.5.4 yang membahas proses bisnis General Ledger secara umum, dokumen ini berfokus pada sudut pandang Account sebagai kerangka acuan dan unit agregasi saldo, sehingga menjadi pelengkap yang tidak tumpang tindih dengan requirement BR-556 s/d BR-565. Seluruh requirement pada dokumen ini menjadi dasar bagi desain teknis integrasi antara modul Chart of Accounts dan General Ledger pada tahap berikutnya.
