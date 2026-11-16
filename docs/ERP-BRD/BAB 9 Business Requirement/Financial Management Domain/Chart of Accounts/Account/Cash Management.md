# 9.6.3.36 Cash Management Account Integration Requirements

---

# 9.6.3.36.1 Overview

Dokumen ini mengatur bagaimana Chart of Accounts (COA) dan Account Master diintegrasikan, direferensikan, dan dipakai secara spesifik oleh domain Cash & Bank. Berbeda dengan dokumen 9.5.9 Cash and Bank Requirements yang membahas proses bisnis kas dan bank secara menyeluruh (penerimaan, pengeluaran, petty cash, rekonsiliasi bank), dokumen ini fokus pada sudut pandang Account/COA: bagaimana setiap objek dan mutasi kas serta bank dipetakan ke akun buku besar (GL Account) yang benar, konsisten, dan tertelusur.

Setiap Bank Account Master dan Petty Cash Fund harus memiliki keterkaitan eksplisit ke Account Master pada Chart of Accounts agar saldo kas dan bank dapat disajikan secara akurat pada laporan keuangan. Demikian pula, proses rekonsiliasi bank memerlukan akun kliring (clearing account) khusus untuk menampung item yang belum tercocokkan (unreconciled items), sementara selisih hasil penghitungan fisik kas memerlukan akun selisih kas (cash over/short) yang terpisah dari akun kas operasional.

Dokumen ini menetapkan requirement agar seluruh objek Cash & Bank (Bank Account, Petty Cash Fund, Bank Statement Line, Reconciliation Item) memiliki mapping akun yang eksplisit dan terkendali, sehingga Single Source of Truth atas struktur akun tetap terjaga meskipun jumlah rekening bank dan unit organisasi yang mengelola kas kecil terus bertambah.

---

# 9.6.3.36.2 Business Capability

Integrasi Account pada domain Cash & Bank menyediakan kapabilitas berikut:

* Bank Account to GL Account One-to-One Mapping Framework.
* Petty Cash Fund Account Mapping per unit organisasi.
* Bank Reconciliation Clearing Account Management.
* Cash Over/Short Account Mapping.
* Bank Statement Transaction Auto-Mapping ke Account Master.
* Multi-Currency Bank Account GL Mapping.
* Bank Charges and Interest Account Mapping.
* Inter-Bank Transfer Suspense Account Management.
* Account Segment/Dimension Mapping untuk transaksi kas dan bank.
* Mapping Change Governance & Validation.
* Effective Dating pada seluruh mapping akun kas dan bank.
* Audit Trail atas seluruh perubahan mapping akun kas dan bank.

---

# 9.6.3.36.3 Business Requirement Catalogue

## BR-936 – Bank Account to GL Account One-to-One Mapping Framework

| Attribute           | Description                                                                                                                                                                                                       |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID       | BR-936                                                                                                                                                                                                           |
| Requirement Name     | Bank Account to GL Account One-to-One Mapping Framework                                                                                                                                                         |
| Category             | Cash Management Account Integration                                                                                                                                                                             |
| Priority             | Critical                                                                                                                                                                                                        |
| Business Objective   | BO-004, BO-022                                                                                                                                                                                                  |
| Stakeholder Owner    | SH-003 (CFO)                                                                                                                                                                                                    |
| Description          | Sistem harus memastikan setiap Bank Account Master dipetakan secara satu-ke-satu ke Account Master (Cash-in-Bank) tersendiri pada Chart of Accounts, tanpa berbagi akun GL dengan Bank Account Master lain.     |
| Business Rationale   | Mapping banyak-ke-satu antara Bank Account dan Account Master menyebabkan saldo per rekening tidak dapat dibedakan pada GL, sehingga menyulitkan rekonsiliasi dan pelaporan posisi kas per rekening.            |
| Business Value       | Menjamin bahwa saldo GL per Bank Account selalu dapat ditelusuri secara langsung sebagai Single Source of Truth posisi kas dan bank.                                                                            |
| Acceptance Criteria  | Sistem menolak pembuatan atau aktivasi Bank Account Master tanpa Account Master yang valid dan belum digunakan oleh Bank Account Master lain, serta memberikan validasi keunikan mapping pada saat konfigurasi. |

---

## BR-937 – Petty Cash Fund Account Mapping per Organization Unit

| Requirement ID | Requirement Name                                | Priority | Description                                                                                                                                                       |
| -------------- | -------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-937         | Petty Cash Fund Account Mapping per Organization Unit | Critical | Sistem harus memetakan setiap Petty Cash Fund ke Account Master kas kecil tersendiri berdasarkan unit organisasi dan/atau pemegang kas, agar saldo kas kecil per unit terpisah pada GL. |

---

## BR-938 – Bank Reconciliation Clearing Account

| Requirement ID | Requirement Name              | Priority | Description                                                                                                                                                                    |
| -------------- | -------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| BR-938         | Bank Reconciliation Clearing Account | Critical | Sistem harus menyediakan Account Master kliring rekonsiliasi bank yang menampung sementara transaksi belum tercocokkan (unreconciled/in-transit items) hingga proses rekonsiliasi selesai. |

---

## BR-939 – Cash Over/Short Account Mapping

| Requirement ID | Requirement Name       | Priority | Description                                                                                                                                                       |
| -------------- | -------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-939         | Cash Over/Short Account Mapping | High     | Sistem harus menyediakan Account Master terpisah untuk mencatat selisih lebih (cash over) dan selisih kurang (cash short) hasil penghitungan fisik kas kecil maupun kas laci, terpisah dari akun kas pokok. |

---

## BR-940 – Bank Statement Transaction Auto-Mapping to Account Master

| Requirement ID | Requirement Name                                    | Priority | Description                                                                                                                                                       |
| -------------- | ------------------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-940         | Bank Statement Transaction Auto-Mapping to Account Master | Critical | Sistem harus secara otomatis menurunkan Account Master yang sesuai dari kode/kategori transaksi pada mutasi bank statement yang diimpor, guna mengurangi entri manual saat proses rekonsiliasi. |

---

## BR-941 – Multi-Currency Bank Account GL Mapping

| Requirement ID | Requirement Name           | Priority | Description                                                                                                                                                       |
| -------------- | ------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-941         | Multi-Currency Bank Account GL Mapping | High     | Sistem harus memetakan Bank Account bervaluta asing ke Account Master utama serta akun selisih kurs realized/unrealized terkait untuk mendukung penyajian saldo dalam mata uang asal maupun base currency. |

---

## BR-942 – Bank Charges and Interest Account Mapping

| Requirement ID | Requirement Name              | Priority | Description                                                                                                                                                    |
| -------------- | ---------------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-942         | Bank Charges and Interest Account Mapping | High     | Sistem harus memetakan biaya administrasi bank (bank charges) dan pendapatan bunga rekening giro/tabungan ke Account Master yang berbeda dan konsisten per Bank Account. |

---

## BR-943 – Inter-Bank Transfer Suspense Account

| Requirement ID | Requirement Name        | Priority | Description                                                                                                                                                       |
| -------------- | ---------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-943         | Inter-Bank Transfer Suspense Account | High     | Sistem harus menyediakan Account Master suspense untuk menampung dana transfer antar rekening bank (inter-bank transfer) yang masih dalam proses (in-transit) sebelum diterima pada rekening tujuan. |

---

## BR-944 – Account Segment/Dimension Mapping for Cash and Bank Transactions

| Requirement ID | Requirement Name                                       | Priority | Description                                                                                                                                                       |
| -------------- | --------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-944         | Account Segment/Dimension Mapping for Cash and Bank Transactions | Medium   | Sistem harus mendukung mapping dimensi/segmen akun (entitas, cost center, cabang) pada transaksi kas dan bank agar pelaporan dapat disajikan per unit maupun konsolidasi. |

---

## BR-945 – Cash and Bank Account Mapping Governance

| Requirement ID | Requirement Name                     | Priority | Description                                                                                                                                                            |
| -------------- | ---------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-945         | Cash and Bank Account Mapping Governance | High     | Sistem harus menerapkan workflow persetujuan dan validasi dependensi sebelum perubahan mapping akun Bank Account atau Petty Cash Fund berlaku efektif, guna mencegah mapping yang tidak valid atau menggantung (orphaned). |

---

# 9.6.3.36.4 Business Rules

| Business Rule ID | Description                                                                                                                                    |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| BRule-CBAI-001     | Setiap Bank Account Master wajib memiliki tepat satu Account Master pada Chart of Accounts, dan satu Account Master hanya boleh digunakan oleh satu Bank Account Master. |
| BRule-CBAI-002     | Setiap Petty Cash Fund wajib memiliki mapping ke Account Master kas kecil yang berbeda per unit organisasi atau pemegang kas.                    |
| BRule-CBAI-003     | Item hasil rekonsiliasi bank yang belum tercocokkan wajib dicatat sementara pada akun kliring rekonsiliasi, bukan langsung pada akun kas/bank utama. |
| BRule-CBAI-004     | Selisih kas hasil penghitungan fisik wajib dicatat pada akun cash over/short, terpisah dari akun kas pokok.                                     |
| BRule-CBAI-005     | Mapping akun dari kategori transaksi bank statement ke Account Master harus divalidasi sebelum diterapkan pada proses auto-mapping.             |
| BRule-CBAI-006     | Account Master yang telah direferensikan oleh Bank Account Master atau Petty Cash Fund aktif tidak dapat dinonaktifkan atau dihapus secara permanen. |
| BRule-CBAI-007     | Setiap perubahan mapping akun kas dan bank harus melalui approval workflow dan tercatat dalam Audit Trail.                                       |

---

# 9.6.3.36.5 Data Attributes

| Attribute                  | Description                                                                          |
| --------------------------- | ------------------------------------------------------------------------------------- |
| Cash Bank Account Mapping ID | Identitas unik mapping akun kas dan bank.                                            |
| Object Type                 | Bank Account, Petty Cash Fund, Bank Statement Line, Reconciliation Item.              |
| Bank Account/Petty Cash Code | Referensi ke Bank Account Master atau Petty Cash Fund terkait.                       |
| Transaction Type             | Cash-in-Bank, Petty Cash, Clearing, Cash Over/Short, Bank Charges, Interest, Transfer Suspense, FX Gain/Loss. |
| Account ID                  | Referensi ke Account Master pada Chart of Accounts.                                   |
| Chart of Accounts ID        | Referensi ke struktur COA yang digunakan.                                            |
| Currency                    | Mata uang rekening bank atau kas kecil.                                              |
| Organization Node           | Scope entitas/unit organisasi terkait mapping.                                       |
| Segment/Dimension           | Cost Center, Branch, Entity, atau dimensi tambahan lain.                             |
| Effective From              | Tanggal mulai berlaku mapping.                                                        |
| Effective To                | Tanggal akhir berlaku mapping.                                                        |
| Mapping Status              | Draft, Pending Approval, Approved, Active, Inactive.                                  |
| Remarks                     | Catatan bisnis atas mapping.                                                          |
| Created By / Created Date   | Informasi pembuatan data.                                                             |
| Updated By / Updated Date   | Informasi perubahan terakhir.                                                         |

---

# 9.6.3.36.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                                    |
| -------------- | ------------------- | ----------- | ------------------------------------------------------ |
| BR-936         | BO-004, BO-022      | SH-003      | Bank Account to GL Mapping Administration               |
| BR-937         | BO-004, BO-020      | SH-003      | Petty Cash Fund Account Mapping Process                 |
| BR-938         | BO-004, BO-022      | SH-003      | Bank Reconciliation Clearing Account Process            |
| BR-939         | BO-022              | SH-003      | Cash Over/Short Account Mapping Process                 |
| BR-940         | BO-004, BO-022      | SH-003      | Bank Statement Auto-Mapping Process                     |
| BR-941         | BO-021, BO-022      | SH-003      | Multi-Currency Bank Account Mapping Process             |
| BR-942         | BO-022              | SH-003      | Bank Charges and Interest Account Mapping Process       |
| BR-943         | BO-004, BO-020      | SH-003      | Inter-Bank Transfer Suspense Account Process            |
| BR-944         | BO-020, BO-022      | SH-003      | Cash and Bank Segment/Dimension Mapping Process         |
| BR-945         | BO-004, BO-022      | SH-003      | Cash and Bank Account Mapping Governance Process        |

---

# 9.6.3.36.7 Summary

Cash Management Account Integration Requirements menetapkan bagaimana Chart of Accounts dan Account Master menjadi tulang punggung pencatatan akuntansi bagi seluruh objek dan mutasi kas dan bank. Melalui sepuluh requirement pada bagian ini, sistem memastikan setiap Bank Account Master, Petty Cash Fund, item rekonsiliasi, dan selisih kas dipetakan ke akun GL yang tepat, terkendali, dan tertelusur, tanpa menduplikasi requirement proses bisnis Cash & Bank yang telah dibahas pada dokumen 9.5.9.

Dokumen ini menjadi jembatan integrasi antara domain Cash & Bank dan Chart of Accounts, serta menjadi fondasi bagi keakuratan General Ledger, rekonsiliasi bank otomatis, dan penyajian posisi kas konsolidasi lintas entitas dan mata uang.
