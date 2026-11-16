# 9.6.3.34 Fixed Assets Account Integration Requirements

---

# 9.6.3.34.1 Overview

Fixed Assets Account Integration Requirements mendefinisikan bagaimana struktur Chart of Accounts (COA) diintegrasikan, direferensikan, dan dipakai secara spesifik oleh domain Fixed Asset dari sudut pandang Account/COA, bukan dari sudut pandang proses bisnis aset tetap. Dokumen ini melengkapi requirement umum Fixed Asset (BR-586 s/d BR-595) dengan menjelaskan bagaimana setiap kategori aset, akumulasi penyusutan, beban penyusutan, aset dalam konstruksi, hingga pelepasan aset dipetakan ke akun GL yang tepat sesuai sifat akun (asset, contra-asset, expense, gain/loss) dan Scope organisasi.

Fokus dokumen ini adalah pada mekanisme mapping akun (account mapping/derivation) yang menghubungkan entitas non-COA (Asset Category, Cost Center/Department) dengan Account pada Chart of Accounts, termasuk akun aset tetap per kategori (Land, Building, Equipment, Vehicle, dan kategori lain), akun akumulasi penyusutan (Accumulated Depreciation) sebagai akun kontra-aset, akun beban penyusutan (Depreciation Expense), akun keuntungan/kerugian pelepasan aset (Gain/Loss on Disposal), serta akun aset dalam konstruksi (Construction in Progress/CIP). Seluruh mapping akun bersifat konfiguratif per Financial Organization dan Scope, sesuai prinsip Configuration over Customization, sehingga tidak memerlukan perubahan source code ketika kebijakan akuntansi aset tetap berubah.

---

# 9.6.3.34.2 Business Capability

Fixed Assets Account Integration menyediakan kapabilitas berikut:

* Asset Category to Asset Account Mapping.
* Accumulated Depreciation Contra-Asset Account Mapping.
* Depreciation Expense Account Mapping by Cost Center/Department.
* Construction in Progress (CIP) Account Mapping.
* CIP to Asset Account Capitalization Transfer Mapping.
* Gain/Loss on Disposal Account Mapping.
* Asset Revaluation Surplus/Reserve Account Mapping.
* Asset Impairment Loss Account Mapping.
* Intercompany Asset Transfer Account Mapping.
* Fixed Asset Sub-Ledger to GL Control Account Reconciliation Mapping.
* Effective Dating dan Versioning atas Fixed Asset Account Mapping.
* Approval Workflow dan Audit Trail atas perubahan Fixed Asset Account Mapping.

---

# 9.6.3.34.3 Business Requirement Catalogue

## BR-916 – Asset Category to Asset Account Mapping

| Attribute           | Description                                                                                                                                                                                                          |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-916                                                                                                                                                                                                              |
| Requirement Name    | Asset Category to Asset Account Mapping                                                                                                                                                                            |
| Category            | Fixed Asset Account Integration                                                                                                                                                                                    |
| Priority            | Critical                                                                                                                                                                                                            |
| Business Objective  | BO-004, BO-022                                                                                                                                                                                                      |
| Stakeholder Owner   | SH-003 (CFO), SH-038 (Asset Management Department)                                                                                                                                                                 |
| Description         | Sistem harus menyediakan mapping antara setiap Asset Category (Land, Building, Equipment, Vehicle, dan kategori lain yang dikonfigurasi) dengan Asset Account spesifik pada Chart of Accounts, sehingga setiap aset yang dikapitalisasi otomatis terhubung ke akun GL sesuai kategorinya. |
| Business Rationale  | Tanpa mapping eksplisit antara kategori aset dan Asset Account pada COA, kapitalisasi aset berisiko diposting ke akun yang tidak sesuai, sehingga menyulitkan klasifikasi aset tetap pada laporan posisi keuangan dan perhitungan penyusutan per kategori. |
| Business Value      | Menjadi single source of truth atas hubungan antara kategori aset dan akun GL, memastikan setiap aset tetap tercatat pada akun yang tepat sejak awal kapitalisasi.                                                |
| Acceptance Criteria | Setiap Asset Category wajib memiliki tepat satu Asset Account aktif per Financial Organization; sistem menolak kapitalisasi aset yang kategorinya belum memiliki mapping Asset Account valid; perubahan mapping tidak memengaruhi aset yang telah dikapitalisasi sebelumnya. |

---

## BR-917 – Accumulated Depreciation Contra-Asset Account Mapping

| Attribute        | Description                                                                                                                                                                    |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-917                                                                                                                                                                        |
| Requirement Name  | Accumulated Depreciation Contra-Asset Account Mapping                                                                                                                          |
| Priority          | Critical                                                                                                                                                                      |
| Description       | Sistem harus memetakan setiap Asset Category ke akun Accumulated Depreciation sebagai akun kontra-aset yang berpasangan langsung dengan Asset Account terkait pada COA.       |

---

## BR-918 – Depreciation Expense Account Mapping by Cost Center

| Attribute        | Description                                                                                                                                                            |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-918                                                                                                                                                                |
| Requirement Name  | Depreciation Expense Account Mapping by Cost Center                                                                                                                   |
| Priority          | Critical                                                                                                                                                              |
| Description       | Sistem harus memetakan beban penyusutan ke akun Depreciation Expense yang dapat bervariasi berdasarkan Cost Center/Department pemilik aset, namun tetap terkonsolidasi ke Accumulated Depreciation Account yang sama per kategori aset. |

---

## BR-919 – Construction in Progress (CIP) Account Mapping

| Attribute        | Description                                                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-919                                                                                                                                                            |
| Requirement Name  | Construction in Progress (CIP) Account Mapping                                                                                                                    |
| Priority          | Critical                                                                                                                                                          |
| Description       | Sistem harus menyediakan akun Construction in Progress pada COA, terpisah dari Asset Account definitif, untuk menampung akumulasi biaya aset yang masih dalam tahap pembangunan/konstruksi. |

---

## BR-920 – CIP to Asset Account Capitalization Transfer Mapping

| Attribute        | Description                                                                                                                                                          |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-920                                                                                                                                                              |
| Requirement Name  | CIP to Asset Account Capitalization Transfer Mapping                                                                                                                |
| Priority          | Critical                                                                                                                                                            |
| Description       | Sistem harus memetakan proses transfer saldo dari akun Construction in Progress ke Asset Account definitif sesuai Asset Category tujuan pada saat kapitalisasi, baik secara penuh maupun parsial. |

---

## BR-921 – Gain/Loss on Disposal Account Mapping

| Attribute        | Description                                                                                                                                                     |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-921                                                                                                                                                          |
| Requirement Name  | Gain/Loss on Disposal Account Mapping                                                                                                                            |
| Priority          | Critical                                                                                                                                                         |
| Description       | Sistem harus memetakan selisih antara nilai buku aset dan nilai pelepasan ke akun Gain/Loss on Disposal pada COA secara otomatis, terpisah dari akun pendapatan/beban operasional reguler. |

---

## BR-922 – Asset Revaluation Surplus/Reserve Account Mapping

| Attribute        | Description                                                                                                                                                   |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-922                                                                                                                                                        |
| Requirement Name  | Asset Revaluation Surplus/Reserve Account Mapping                                                                                                             |
| Priority          | High                                                                                                                                                          |
| Description       | Sistem harus memetakan hasil revaluasi aset tetap ke akun Revaluation Surplus/Reserve pada ekuitas atau akun penyesuaian nilai aset, sesuai kategori aset dan kebijakan akuntansi organisasi. |

---

## BR-923 – Asset Impairment Loss Account Mapping

| Attribute        | Description                                                                                                                                                  |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-923                                                                                                                                                       |
| Requirement Name  | Asset Impairment Loss Account Mapping                                                                                                                        |
| Priority          | Medium                                                                                                                                                       |
| Description       | Sistem harus memetakan kerugian penurunan nilai aset (impairment loss) ke akun beban impairment pada COA yang terpisah dari beban penyusutan reguler.        |

---

## BR-924 – Intercompany Asset Transfer Account Mapping

| Attribute        | Description                                                                                                                                                        |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-924                                                                                                                                                             |
| Requirement Name  | Intercompany Asset Transfer Account Mapping                                                                                                                        |
| Priority          | High                                                                                                                                                               |
| Description       | Sistem harus memetakan akun Due To/Due From atau akun eliminasi intercompany yang berpasangan pada saat transfer aset antar Financial Organization, memastikan jurnal double-entry seimbang pada kedua entitas. |

---

## BR-925 – Fixed Asset Sub-Ledger to GL Control Account Reconciliation Mapping

| Attribute        | Description                                                                                                                                                             |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-925                                                                                                                                                                 |
| Requirement Name  | Fixed Asset Sub-Ledger to GL Control Account Reconciliation Mapping                                                                                                    |
| Priority          | High                                                                                                                                                                   |
| Description       | Sistem harus menyediakan mapping dan laporan rekonsiliasi otomatis antara saldo Fixed Asset Register per aset dengan saldo Asset Account dan Accumulated Depreciation Account pada General Ledger, serta menampilkan selisih sebagai exception. |

---

# 9.6.3.34.4 Business Rules

| Business Rule ID | Description                                                                                                                                                        |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-FAAI-001    | Setiap Asset Category wajib memiliki kombinasi unik Asset Account, Accumulated Depreciation Account, dan Depreciation Expense Account sebelum aset dapat dikapitalisasi. |
| BRule-FAAI-002    | Accumulated Depreciation Account wajib berupa akun kontra-aset dan saldonya tidak boleh melebihi saldo Asset Account yang berpasangan.                          |
| BRule-FAAI-003    | Saldo akun Construction in Progress wajib ditransfer penuh ke Asset Account definitif pada saat kapitalisasi selesai; kapitalisasi parsial diperbolehkan namun penutupan CIP wajib tuntas pada akhir proyek. |
| BRule-FAAI-004    | Akun Gain/Loss on Disposal ditentukan secara otomatis oleh sistem berdasarkan Asset Category dan tidak dapat dipilih secara manual oleh pengguna pada saat entri disposal. |
| BRule-FAAI-005    | Perubahan mapping Asset Account, Accumulated Depreciation Account, atau Depreciation Expense Account pada Asset Category yang masih memiliki aset aktif dilarang tanpa proses migrasi/reklasifikasi, dan tidak boleh mengubah posting historis. |
| BRule-FAAI-006    | Depreciation Expense Account dapat bervariasi per Cost Center/Department, namun akumulasinya wajib bermuara pada satu Accumulated Depreciation Account yang sama per kategori aset. |
| BRule-FAAI-007    | Transfer aset antar Financial Organization wajib menggunakan pasangan akun Due To/Due From atau akun eliminasi intercompany yang telah dipetakan, dan menghasilkan jurnal seimbang pada entitas asal maupun tujuan. |

---

# 9.6.3.34.5 Data Attributes

| Attribute                       | Description                                                                    |
| --------------------------------- | -------------------------------------------------------------------------------- |
| Fixed Asset Account Mapping ID   | Identitas unik sistem atas konfigurasi mapping akun Fixed Asset.               |
| Financial Organization           | Referensi Financial Organization pemilik mapping akun.                          |
| Asset Category                    | Klasifikasi aset (Land, Building, Equipment, Vehicle, dan lain-lain).           |
| Asset Account                     | Referensi Account pada COA sebagai akun aset tetap definitif.                   |
| Accumulated Depreciation Account | Referensi Account kontra-aset akumulasi penyusutan.                             |
| Depreciation Expense Account     | Referensi Account beban penyusutan, dapat bervariasi per Cost Center.           |
| Construction in Progress Account | Referensi Account penampung biaya aset dalam konstruksi.                        |
| Gain/Loss on Disposal Account    | Referensi Account keuntungan/kerugian pelepasan aset.                           |
| Revaluation Surplus/Reserve Account | Referensi Account hasil revaluasi aset.                                       |
| Impairment Loss Account          | Referensi Account kerugian penurunan nilai aset.                                |
| Intercompany Due To/From Account | Referensi Account eliminasi intercompany untuk transfer aset antar entitas.     |
| Cost Center/Department           | Dimensi non-COA yang menjadi basis derivasi Depreciation Expense Account.       |
| Effective From/To                 | Periode berlaku mapping akun.                                                    |
| Version                           | Nomor versi mapping.                                                             |
| Approval Status                  | Draft, Pending Approval, Approved, Published, Archived.                        |
| Status                             | Active, Suspended, Inactive, Archived.                                          |

---

# 9.6.3.34.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder    | Future Process                                        |
| --------------- | ------------------- | -------------- | ---------------------------------------------------------- |
| BR-916          | BO-004, BO-022      | SH-003, SH-038 | Asset Category Account Administration                     |
| BR-917          | BO-022              | SH-003, SH-038 | Accumulated Depreciation Account Configuration             |
| BR-918          | BO-009, BO-022      | SH-038         | Depreciation Expense Account Derivation                    |
| BR-919          | BO-004              | SH-038         | Construction in Progress Account Administration            |
| BR-920          | BO-004, BO-022      | SH-003, SH-038 | CIP Capitalization Transfer Processing                     |
| BR-921          | BO-022              | SH-003, SH-038 | Asset Disposal Account Processing                          |
| BR-922          | BO-022              | SH-003         | Asset Revaluation Account Processing                        |
| BR-923          | BO-022              | SH-003, SH-038 | Asset Impairment Account Processing                         |
| BR-924          | BO-004, BO-022      | SH-003, SH-038 | Intercompany Asset Transfer Processing                      |
| BR-925          | BO-004, BO-022      | SH-003, SH-038 | Fixed Asset Sub-Ledger to GL Reconciliation                 |

---

# 9.6.3.34.7 Summary

Fixed Assets Account Integration Requirements melengkapi requirement umum Fixed Asset dengan menetapkan bagaimana Chart of Accounts diintegrasikan secara spesifik ke dalam pengelolaan aset tetap, mencakup mapping akun per kategori aset, akun akumulasi penyusutan sebagai kontra-aset, akun beban penyusutan per Cost Center, akun aset dalam konstruksi, akun keuntungan/kerugian pelepasan aset, akun revaluasi dan impairment, hingga rekonsiliasi subledger terhadap General Ledger. Dengan mapping akun yang konfiguratif dan dapat diaudit, domain ini memastikan akurasi laporan keuangan aset tetap di seluruh entitas organisasi tanpa memerlukan perubahan source code, serta menjadi dasar bagi desain teknis integrasi akun pada dokumen berikutnya.
