# 9.6.3.40 Tax Account Integration Requirements

## 9.6.3.40.1 Overview

Dokumen ini membahas bagaimana Chart of Accounts (COA) — khususnya struktur Account (akun General Ledger) — direferensikan, dipetakan, dan dipakai secara teknis oleh proses Tax Management. Fokus dokumen ini BUKAN pada proses bisnis perpajakan secara umum (perhitungan pajak, jenis pajak, kalender pelaporan, dan kepatuhan telah dibahas tuntas pada BR-646 s/d BR-655), melainkan pada sudut pandang Account/COA: bagaimana akun PPN Keluaran dan PPN Masukan dipetakan, bagaimana akun utang PPh dipisahkan per jenis pasal, bagaimana akun beban pajak ditentukan, bagaimana mapping otomatis transaksi kena pajak ke akun terkait dijalankan, serta bagaimana akun pajak direkonsiliasi terhadap laporan SPT.

Chart of Accounts adalah titik integrasi wajib bagi seluruh transaksi kena pajak. Setiap Tax Code yang berlaku pada suatu transaksi harus memiliki pemetaan eksplisit ke Account tertentu, sehingga saldo akun PPN Keluaran, PPN Masukan, utang PPh per pasal, dan beban pajak dapat direkonsiliasi secara akurat dengan Surat Pemberitahuan (SPT) Masa maupun Tahunan. Tanpa pemetaan akun pajak yang konsisten dan terkelola melalui konfigurasi (bukan hard-code), risiko selisih antara saldo buku besar dan laporan pajak akan meningkat, serta rekonsiliasi akan menjadi proses manual yang rawan kesalahan.

Dokumen ini disusun agar tim desain data model, tim konfigurasi COA, dan tim Tax Management memiliki acuan bersama mengenai kontrak integrasi antara struktur akun dan proses perpajakan, sejalan dengan prinsip Single Source of Truth, Configuration over Customization, dan kepatuhan terhadap regulasi perpajakan Indonesia.

---

## 9.6.3.40.2 Business Capability

- Memetakan setiap Tax Code aktif ke Account GL yang valid dan spesifik, tanpa penggabungan lintas jenis pajak.
- Memisahkan akun PPN Keluaran dan akun PPN Masukan secara eksplisit pada COA.
- Memisahkan akun utang PPh per jenis pasal (PPh 21, 22, 23, 25, 26, 4 ayat 2, dan pasal lain sesuai regulasi berlaku).
- Menentukan akun beban pajak (tax expense) untuk pajak final maupun non-final yang berbeda dari akun utang pajak.
- Menjalankan mapping otomatis transaksi kena pajak ke akun terkait berdasarkan Tax Code, tanpa entri manual jurnal pajak.
- Mendukung segmentasi akun PPN berdasarkan tarif berlaku (mis. 11%, 12%, PPN dibebaskan, PPN tidak dipungut).
- Merekonsiliasi saldo akun pajak pada General Ledger terhadap nilai yang dilaporkan pada SPT Masa dan SPT Tahunan.
- Menyediakan akun pajak dibayar di muka (prepaid tax) untuk skema seperti PPh 25 dan kredit pajak.
- Mengonsolidasikan akun pajak lintas organisasi/entitas dengan NPWP berbeda tanpa mencampur saldo antar-entitas.
- Menyediakan akun suspense/exception untuk transaksi yang datanya tidak lengkap secara pajak sebelum direklasifikasi ke akun definitif.
- Menjaga konsistensi pemetaan akun pajak ketika terjadi perubahan struktur COA atau perubahan regulasi tarif pajak.
- Menjamin jejak audit atas seluruh perubahan pemetaan akun pajak.

---

## 9.6.3.40.3 Business Requirement Catalogue

## BR-976 – VAT Output/Input Account Mapping

| Attribute           | Description                                                                                                                                                     |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID      | BR-976                                                                                                                                                          |
| Requirement Name    | VAT Output/Input Account Mapping                                                                                                                                |
| Category            | Chart of Accounts – Tax Integration                                                                                                                             |
| Priority            | Critical                                                                                                                                                        |
| Business Objective  | BO-004, BO-022                                                                                                                                                   |
| Stakeholder Owner   | SH-003 (CFO)                                                                                                                                                    |
| Description         | Sistem harus menyediakan pemetaan eksplisit dan terpisah antara Account PPN Keluaran (VAT Output) dan Account PPN Masukan (VAT Input) untuk setiap Tax Code PPN yang aktif, tanpa penggabungan saldo. |
| Business Rationale  | PPN Keluaran dan PPN Masukan memiliki perlakuan pelaporan dan rekonsiliasi SPT yang berbeda, sehingga wajib dicatat pada akun terpisah untuk menjamin keakuratan pelaporan. |
| Business Value      | Menjamin Single Source of Truth atas saldo PPN yang siap direkonsiliasi dengan SPT Masa PPN tanpa penyesuaian manual.                                          |
| Acceptance Criteria | Setiap Tax Code PPN memiliki pemetaan Account VAT Output dan VAT Input yang tervalidasi; saldo kedua akun dapat ditelusuri terpisah pada laporan Trial Balance. |

---

## BR-977 – Withholding Tax Payable Account Mapping per Pasal

| Attribute        | Description                                                                                                              |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-977                                                                                                                    |
| Requirement Name | Withholding Tax Payable Account Mapping per Pasal                                                                        |
| Priority         | Critical                                                                                                                  |
| Description      | Sistem harus menyediakan Account utang PPh terpisah untuk setiap jenis pasal (PPh 21, 22, 23, 25, 26, 4 ayat 2, dan pasal lain sesuai regulasi), tanpa penggabungan saldo antar pasal. |

---

## BR-978 – Tax Expense Account Mapping

| Attribute        | Description                                                                                                       |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-978                                                                                                             |
| Requirement Name | Tax Expense Account Mapping                                                                                       |
| Priority         | Critical                                                                                                           |
| Description      | Sistem harus menyediakan Account beban pajak (tax expense) yang terpisah dari Account utang pajak, mencakup pajak final dan non-final, untuk mencerminkan beban periode berjalan secara akurat. |

---

## BR-979 – Automatic Tax Transaction-to-Account Posting Rule

| Attribute        | Description                                                                                                          |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-979                                                                                                                |
| Requirement Name | Automatic Tax Transaction-to-Account Posting Rule                                                                    |
| Priority         | Critical                                                                                                              |
| Description      | Sistem harus menjalankan mapping otomatis dari transaksi kena pajak ke Account terkait berdasarkan Tax Code yang melekat pada transaksi, tanpa memerlukan entri jurnal pajak manual. |

---

## BR-980 – Tax Account Reconciliation with SPT

| Attribute        | Description                                                                                                          |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-980                                                                                                                |
| Requirement Name | Tax Account Reconciliation with SPT                                                                                  |
| Priority         | Critical                                                                                                              |
| Description      | Sistem harus menyediakan fungsi rekonsiliasi antara saldo Account pajak pada General Ledger dengan nilai yang dilaporkan pada SPT Masa dan SPT Tahunan, termasuk identifikasi selisih. |

---

## BR-981 – Multi-Rate VAT Account Segmentation

| Attribute        | Description                                                                                                    |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-981                                                                                                          |
| Requirement Name | Multi-Rate VAT Account Segmentation                                                                            |
| Priority         | High                                                                                                            |
| Description      | Sistem harus mendukung segmentasi Account PPN berdasarkan tarif yang berlaku (mis. 11%, 12%, PPN dibebaskan, PPN tidak dipungut) agar saldo per tarif dapat ditelusuri terpisah. |

---

## BR-982 – Prepaid Tax Account Mapping and Rollforward

| Attribute        | Description                                                                                                       |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-982                                                                                                             |
| Requirement Name | Prepaid Tax Account Mapping and Rollforward                                                                       |
| Priority         | High                                                                                                                |
| Description      | Sistem harus menyediakan Account pajak dibayar di muka (mis. PPh 25, kredit pajak luar negeri) dengan mekanisme rollforward saldo antar periode fiskal hingga dikompensasikan. |

---

## BR-983 – Cross-Organization Tax Account Consolidation

| Attribute        | Description                                                                                                     |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-983                                                                                                           |
| Requirement Name | Cross-Organization Tax Account Consolidation                                                                    |
| Priority         | High                                                                                                              |
| Description      | Sistem harus mengonsolidasikan pelaporan Account pajak lintas organisasi/entitas dengan NPWP berbeda pada struktur multi-organisasi, tanpa mencampur saldo antar-entitas pada satu Account. |

---

## BR-984 – Tax Account Exception and Suspense Handling

| Attribute        | Description                                                                                                    |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-984                                                                                                          |
| Requirement Name | Tax Account Exception and Suspense Handling                                                                    |
| Priority         | Medium                                                                                                          |
| Description      | Sistem harus menyediakan Account suspense pajak untuk menampung transaksi dengan data pajak tidak lengkap, sebelum direklasifikasi ke Account pajak definitif setelah data dilengkapi. |

---

## BR-985 – Audit Trail of Tax Account Mapping Changes

| Attribute        | Description                                                                                                 |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-985                                                                                                       |
| Requirement Name | Audit Trail of Tax Account Mapping Changes                                                                   |
| Priority         | Critical                                                                                                      |
| Description      | Sistem harus mencatat seluruh perubahan pemetaan Tax Code-ke-Account pada Audit Trail, termasuk pengguna, waktu, nilai sebelum dan sesudah perubahan, serta alasan perubahan. |

---

## 9.6.3.40.4 Business Rules

| Business Rule ID  | Description                                                                                                                            |
| ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-TAXAI-001    | Setiap Tax Code aktif wajib memiliki pemetaan Account GL yang valid sebelum dapat digunakan pada transaksi.                              |
| BRule-TAXAI-002    | Account PPN Keluaran dan Account PPN Masukan wajib dipisahkan dan tidak boleh digabung dalam satu Account.                                |
| BRule-TAXAI-003    | Account utang PPh wajib dipisah per jenis pasal dan tidak boleh digabung antar pasal.                                                     |
| BRule-TAXAI-004    | Selisih hasil rekonsiliasi antara saldo Account pajak dan nilai SPT wajib direview dan disetujui sebelum periode pajak ditutup.           |
| BRule-TAXAI-005    | Perubahan pemetaan Account pajak tidak berlaku surut terhadap transaksi yang telah diposting pada periode sebelumnya.                    |
| BRule-TAXAI-006    | Transaksi lintas organisasi dengan NPWP berbeda tidak boleh digabung dalam satu Account pajak konsolidasi.                                |
| BRule-TAXAI-007    | Seluruh perubahan pemetaan Tax Code-ke-Account wajib tercatat pada Audit Trail dengan referensi pengguna, waktu, dan alasan perubahan.    |

---

## 9.6.3.40.5 Data Attributes

| Attribute                    | Description                                                                                          |
| ------------------------------ | ------------------------------------------------------------------------------------------------------ |
| Tax Account Mapping ID          | Identitas unik pemetaan antara Tax Code dan Account.                                                  |
| Tax Code                        | Kode pajak yang melekat pada transaksi (mis. PPN 11%, PPh 23, PPh 4 ayat 2).                          |
| Tax Type                        | Jenis pajak: PPN atau PPh.                                                                             |
| Pasal/Category                  | Kategori atau nomor pasal PPh yang berlaku pada mapping.                                                |
| GL Account Code/ID              | Referensi ke Account pada Chart of Accounts.                                                          |
| Account Direction               | Sifat normal saldo Account (Debit/Kredit) sesuai jenis pajak.                                         |
| Tax Rate                        | Tarif pajak yang berlaku pada mapping (mis. 11%, 12%, 2%, 0,5%).                                       |
| Organization/Entity ID          | Referensi entitas/organisasi tempat transaksi pajak tercatat.                                          |
| NPWP                             | Nomor Pokok Wajib Pajak entitas terkait transaksi.                                                     |
| Fiscal Period                    | Periode pajak (bulanan/tahunan) yang relevan dengan mapping.                                            |
| Reconciliation Status           | Status rekonsiliasi Account pajak terhadap SPT: Matched, Unmatched, Under Review.                       |
| Difference Amount               | Nilai selisih antara saldo Account pajak dan nilai SPT hasil rekonsiliasi.                              |
| Last Reconciled Date            | Tanggal terakhir rekonsiliasi Account pajak dilakukan.                                                  |
| Mapping Effective Date          | Tanggal mulai berlaku pemetaan Tax Code-Account.                                                        |
| Mapping End Date                | Tanggal berakhirnya pemetaan Tax Code-Account.                                                          |
| Last Modified By / Timestamp    | Pengguna dan waktu perubahan terakhir atas pemetaan.                                                    |

---

## 9.6.3.40.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                            |
| -------------- | ------------------- | ----------- | -------------------------------------------- |
| BR-976         | BO-004               | SH-003      | VAT Account Mapping Configuration             |
| BR-977         | BO-022               | SH-003      | Withholding Tax Account Configuration         |
| BR-978         | BO-022               | SH-003      | Tax Expense Recognition                        |
| BR-979         | BO-009               | SH-003      | Automated Tax Posting Engine                   |
| BR-980         | BO-007               | SH-011      | Tax Account and SPT Reconciliation             |
| BR-981         | BO-004               | SH-003      | VAT Rate Segmentation Configuration            |
| BR-982         | BO-022               | SH-003      | Prepaid Tax Rollforward Management             |
| BR-983         | BO-004               | SH-003      | Multi-Entity Tax Consolidation                 |
| BR-984         | BO-007               | SH-011      | Tax Exception and Suspense Review              |
| BR-985         | BO-007               | SH-011      | Tax Account Mapping Audit Review               |

---

## 9.6.3.40.7 Summary

Dokumen ini menetapkan requirement integrasi antara Chart of Accounts dan Tax Management dari sudut pandang Account: pemetaan akun PPN Keluaran dan Masukan, pemisahan akun utang PPh per pasal, penentuan akun beban pajak, mapping otomatis transaksi kena pajak ke akun terkait, dan rekonsiliasi akun pajak dengan SPT. Sepuluh requirement (BR-976 s/d BR-985) dan tujuh Business Rule (BRule-TAXAI-001 s/d BRule-TAXAI-007) memastikan bahwa struktur akun menjadi fondasi pencatatan dan pelaporan pajak yang akurat, dapat direkonsiliasi, dan tidak duplikatif terhadap requirement proses bisnis perpajakan umum yang telah dibahas pada BR-646 s/d BR-655.

Seluruh requirement pada bagian ini akan diturunkan lebih lanjut menjadi Functional Requirements, desain model data pemetaan akun-pajak, spesifikasi API integrasi, dan skenario pengujian pada dokumen lanjutan.
