# 9.6.3.35 Treasury Account Integration Requirements

---

# 9.6.3.35.1 Overview

Dokumen ini mengatur bagaimana Chart of Accounts (COA) dan Account Master diintegrasikan, direferensikan, dan dipakai secara spesifik oleh domain Treasury. Berbeda dengan dokumen 9.5.8 Treasury Requirements yang membahas proses bisnis Treasury secara menyeluruh (cash position, forecasting, loan facility, investasi, hedging, bank pooling), dokumen ini fokus pada sudut pandang Account/COA: bagaimana setiap aktivitas Treasury dipetakan ke akun buku besar (GL Account) yang benar, konsisten, dan tertelusur.

Aktivitas Treasury menghasilkan berbagai jenis posisi keuangan yang harus tercermin secara akurat pada GL — mulai dari penempatan investasi jangka pendek per instrumen, kewajiban utang bank/fasilitas kredit, selisih kurs dari eksposur dan transaksi hedging, pendapatan/beban bunga, hingga instrumen lindung nilai (hedging instrument) itu sendiri. Tanpa kerangka mapping akun yang eksplisit dan terkendali, aktivitas Treasury berisiko menghasilkan pencatatan GL yang tidak konsisten antar entitas, mata uang, dan instrumen, sehingga mengganggu keandalan laporan keuangan konsolidasi.

Dokumen ini menetapkan requirement agar setiap objek Treasury (Investment Instrument, Loan/Credit Facility, FX Exposure, Hedging Instrument) memiliki keterkaitan eksplisit ke Account Master melalui mekanisme Treasury Account Mapping, sehingga Single Source of Truth atas struktur akun tetap terjaga meskipun aktivitas Treasury bersifat lintas entitas dan lintas mata uang.

---

# 9.6.3.35.2 Business Capability

Integrasi Account pada domain Treasury menyediakan kapabilitas berikut:

* Treasury Account Mapping Framework lintas jenis instrumen dan transaksi.
* Short-Term Investment Account Mapping per instrumen (deposito, money market, surat berharga).
* Loan Payable / Credit Facility Account Mapping per fasilitas dan per pemberi pinjaman.
* Foreign Exchange Gain/Loss Account Derivation (realized dan unrealized).
* Interest Income and Expense Account Mapping.
* Hedging Instrument Account Mapping (forward, swap, option).
* Treasury Sub-Ledger to GL Control Account Reconciliation.
* Multi-Currency Revaluation Account Mapping untuk saldo Treasury.
* Account Segment/Dimension Mapping untuk transaksi Treasury.
* Mapping Change Governance & Validation.
* Effective Dating pada seluruh mapping akun Treasury.
* Audit Trail atas seluruh perubahan mapping akun Treasury.

---

# 9.6.3.35.3 Business Requirement Catalogue

## BR-926 – Treasury Account Mapping Framework

| Attribute           | Description                                                                                                                                                                                                                                    |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID       | BR-926                                                                                                                                                                                                                                        |
| Requirement Name     | Treasury Account Mapping Framework                                                                                                                                                                                                            |
| Category             | Treasury Account Integration                                                                                                                                                                                                                  |
| Priority             | Critical                                                                                                                                                                                                                                      |
| Business Objective   | BO-004, BO-020                                                                                                                                                                                                                                |
| Stakeholder Owner    | SH-003 (CFO)                                                                                                                                                                                                                                  |
| Description          | Sistem harus menyediakan kerangka Treasury Account Mapping yang menghubungkan setiap jenis objek Treasury (Investment Instrument, Loan/Credit Facility, FX Exposure, Hedging Instrument) ke Account Master pada Chart of Accounts yang berlaku. |
| Business Rationale   | Tanpa kerangka mapping akun yang eksplisit, aktivitas Treasury berisiko dicatat pada akun yang tidak konsisten antar entitas dan instrumen, sehingga menyulitkan konsolidasi dan audit.                                                       |
| Business Value       | Menjamin bahwa setiap transaksi Treasury tercatat pada akun GL yang benar dan konsisten sebagai Single Source of Truth struktur akun.                                                                                                          |
| Acceptance Criteria  | Sistem menyediakan Treasury Account Mapping Master yang wajib diisi sebelum objek Treasury (instrumen investasi, fasilitas pinjaman, instrumen hedging) dapat digunakan untuk transaksi, dan menolak transaksi tanpa mapping akun yang valid. |

---

## BR-927 – Short-Term Investment Account Mapping per Instrument Type

| Requirement ID | Requirement Name                                       | Priority | Description                                                                                                                                                                        |
| -------------- | ------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-927         | Short-Term Investment Account Mapping per Instrument Type | Critical | Sistem harus memetakan setiap jenis instrumen investasi jangka pendek (deposito, money market, surat berharga) ke Account Master yang berbeda, sehingga saldo per instrumen dapat dibedakan pada GL. |

---

## BR-928 – Loan Payable / Credit Facility Account Mapping

| Requirement ID | Requirement Name                             | Priority | Description                                                                                                                                                       |
| -------------- | ---------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-928         | Loan Payable / Credit Facility Account Mapping | Critical | Sistem harus memetakan setiap fasilitas pinjaman/kredit (loan, credit line, overdraft) ke Account Master utang bank tersendiri, dipisahkan per fasilitas dan per pemberi pinjaman. |

---

## BR-929 – Foreign Exchange Gain/Loss Account Derivation

| Requirement ID | Requirement Name                          | Priority | Description                                                                                                                                                                       |
| -------------- | -------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-929         | Foreign Exchange Gain/Loss Account Derivation | Critical | Sistem harus menurunkan (derive) akun selisih kurs realized dan unrealized secara otomatis dari transaksi dan eksposur nilai tukar Treasury, terpisah dari akun selisih kurs operasional lainnya. |

---

## BR-930 – Interest Income and Expense Account Mapping

| Requirement ID | Requirement Name                        | Priority | Description                                                                                                                                                     |
| -------------- | ------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-930         | Interest Income and Expense Account Mapping | High     | Sistem harus memetakan pendapatan bunga (dari investasi/penempatan dana) dan beban bunga (dari fasilitas pinjaman) ke Account Master yang berbeda dan konsisten per mata uang. |

---

## BR-931 – Hedging Instrument Account Mapping

| Requirement ID | Requirement Name              | Priority | Description                                                                                                                                                              |
| -------------- | -------------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-931         | Hedging Instrument Account Mapping | High     | Sistem harus menyediakan mapping akun untuk setiap jenis instrumen hedging (forward, swap, option), mencakup akun perubahan nilai wajar (fair value), premi, dan penyelesaian (settlement). |

---

## BR-932 – Treasury Sub-Ledger to GL Control Account Reconciliation

| Requirement ID | Requirement Name                                    | Priority | Description                                                                                                                                                        |
| -------------- | ------------------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-932         | Treasury Sub-Ledger to GL Control Account Reconciliation | Critical | Sistem harus menetapkan Account Master sebagai control account bagi sub-ledger Treasury (investasi, pinjaman, hedging), dan memvalidasi kesesuaian saldo sub-ledger dengan saldo GL secara berkala. |

---

## BR-933 – Multi-Currency Revaluation Account Mapping for Treasury Balances

| Requirement ID | Requirement Name                                            | Priority | Description                                                                                                                                                            |
| -------------- | ---------------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-933         | Multi-Currency Revaluation Account Mapping for Treasury Balances | High     | Sistem harus memetakan akun revaluasi selisih kurs atas saldo Treasury bervaluta asing (pinjaman, investasi) yang direvaluasi pada akhir periode ke Account Master khusus revaluasi. |

---

## BR-934 – Account Segment/Dimension Mapping for Treasury Transactions

| Requirement ID | Requirement Name                                     | Priority | Description                                                                                                                                                       |
| -------------- | ------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-934         | Account Segment/Dimension Mapping for Treasury Transactions | Medium   | Sistem harus mendukung mapping dimensi/segmen akun (entitas, cost center, currency) pada transaksi Treasury agar pelaporan dapat disajikan per entitas maupun konsolidasi. |

---

## BR-935 – Treasury Account Mapping Change Governance

| Requirement ID | Requirement Name                        | Priority | Description                                                                                                                                                                  |
| -------------- | ------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| BR-935         | Treasury Account Mapping Change Governance | High     | Sistem harus menerapkan workflow persetujuan dan validasi dependensi sebelum perubahan Treasury Account Mapping berlaku efektif, guna mencegah mapping yang tidak valid atau menggantung (orphaned). |

---

# 9.6.3.35.4 Business Rules

| Business Rule ID | Description                                                                                                                                        |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-TRSAI-001    | Setiap objek Treasury (instrumen investasi, fasilitas pinjaman, instrumen hedging) wajib memiliki mapping ke Account Master yang aktif sebelum dapat digunakan untuk transaksi. |
| BRule-TRSAI-002    | Akun selisih kurs Treasury (realized dan unrealized) harus dipisahkan dari akun selisih kurs operasional lainnya pada Chart of Accounts.            |
| BRule-TRSAI-003    | Akun pendapatan bunga dan beban bunga tidak boleh menggunakan Account Master yang sama.                                                              |
| BRule-TRSAI-004    | Perubahan Treasury Account Mapping yang telah digunakan pada transaksi terposting tidak boleh mengubah pencatatan historis.                          |
| BRule-TRSAI-005    | Saldo sub-ledger Treasury per instrumen/fasilitas harus direkonsiliasi terhadap saldo Account Master control account secara berkala.                |
| BRule-TRSAI-006    | Account Master yang telah direferensikan oleh Treasury Account Mapping aktif tidak dapat dinonaktifkan atau dihapus secara permanen.                |
| BRule-TRSAI-007    | Setiap perubahan Treasury Account Mapping harus melalui approval workflow dan tercatat dalam Audit Trail.                                            |

---

# 9.6.3.35.5 Data Attributes

| Attribute                  | Description                                                                          |
| --------------------------- | ------------------------------------------------------------------------------------- |
| Treasury Account Mapping ID | Identitas unik mapping akun Treasury.                                                |
| Treasury Object Type        | Investment Instrument, Loan/Credit Facility, FX Exposure, Hedging Instrument.          |
| Instrument/Facility Code    | Referensi ke instrumen investasi atau fasilitas pinjaman terkait.                     |
| Transaction Type            | Principal, Interest Income, Interest Expense, FX Gain, FX Loss, Fair Value Change, Premium, Settlement. |
| Account ID                  | Referensi ke Account Master pada Chart of Accounts.                                   |
| Chart of Accounts ID        | Referensi ke struktur COA yang digunakan.                                            |
| Currency                    | Mata uang transaksi Treasury.                                                        |
| Organization Node           | Scope entitas/unit organisasi terkait mapping.                                       |
| Segment/Dimension           | Cost Center, Entity, Project, atau dimensi tambahan lain.                            |
| Effective From              | Tanggal mulai berlaku mapping.                                                        |
| Effective To                | Tanggal akhir berlaku mapping.                                                        |
| Mapping Status              | Draft, Pending Approval, Approved, Active, Inactive.                                  |
| Remarks                     | Catatan bisnis atas mapping.                                                          |
| Created By / Created Date   | Informasi pembuatan data.                                                             |
| Updated By / Updated Date   | Informasi perubahan terakhir.                                                         |

---

# 9.6.3.35.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                                  |
| -------------- | ------------------- | ----------- | -------------------------------------------------- |
| BR-926         | BO-004, BO-020      | SH-003      | Treasury Account Mapping Administration            |
| BR-927         | BO-004, BO-022      | SH-003      | Short-Term Investment Account Mapping Process      |
| BR-928         | BO-004, BO-020      | SH-003      | Loan Payable Account Mapping Process               |
| BR-929         | BO-021, BO-022      | SH-003      | FX Gain/Loss Account Derivation Process            |
| BR-930         | BO-004, BO-022      | SH-003      | Interest Account Mapping Process                   |
| BR-931         | BO-021, BO-022      | SH-003      | Hedging Instrument Account Mapping Process         |
| BR-932         | BO-004, BO-022      | SH-003      | Treasury Sub-Ledger Reconciliation Process         |
| BR-933         | BO-021, BO-022      | SH-003      | Treasury Revaluation Account Mapping Process       |
| BR-934         | BO-020, BO-022      | SH-003      | Treasury Segment/Dimension Mapping Process         |
| BR-935         | BO-004, BO-022      | SH-003      | Treasury Account Mapping Governance Process        |

---

# 9.6.3.35.7 Summary

Treasury Account Integration Requirements menetapkan bagaimana Chart of Accounts dan Account Master menjadi tulang punggung pencatatan akuntansi bagi seluruh aktivitas Treasury. Melalui sepuluh requirement pada bagian ini, sistem memastikan setiap instrumen investasi jangka pendek, fasilitas pinjaman, selisih kurs, bunga, dan instrumen hedging dipetakan ke akun GL yang tepat, terkendali, dan tertelusur, tanpa menduplikasi requirement proses bisnis Treasury yang telah dibahas pada dokumen 9.5.8.

Dokumen ini menjadi jembatan integrasi antara domain Treasury dan Chart of Accounts, serta menjadi fondasi bagi keakuratan General Ledger, Financial Consolidation, dan Currency Management dalam mencerminkan aktivitas Treasury secara konsolidatif lintas entitas dan mata uang.
