# 9.6.3.38 Consolidation Account Integration Requirements

---

# 9.6.3.38.1 Overview

Dokumen ini menetapkan persyaratan integrasi antara struktur akun (Chart of Accounts/Account) dengan kebutuhan Financial Consolidation, khususnya dari sudut pandang bagaimana Account digunakan, direferensikan, dan dipetakan dalam proses konsolidasi laporan keuangan lintas entitas. Dokumen ini bukan merupakan requirement umum modul Financial Consolidation — hal tersebut telah dibahas secara menyeluruh pada dokumen 9.5.15 (BR-666–BR-675). Fokus dokumen ini adalah sisi Account: bagaimana Account eliminasi transaksi intercompany ditetapkan, bagaimana Account investasi pada entitas anak (investment in subsidiary) dipetakan terhadap ekuitas entitas anak, bagaimana Account kepentingan non-pengendali (Non-Controlling Interest/NCI) dipisahkan, bagaimana Account selisih translasi mata uang konsolidasi (cumulative translation adjustment) diperlakukan, serta bagaimana Account entitas anak dipetakan (account mapping) ke Account standar grup.

Chart of Accounts pada setiap entitas anak dalam struktur multi-organisasi dapat memiliki variasi struktur akun lokal sesuai kebutuhan regulasi maupun operasional masing-masing entitas. Oleh karena itu, proses konsolidasi memerlukan mapping Account yang eksplisit dan terkonfigurasi antara Account entitas anak dengan Account standar grup, agar laporan konsolidasi dapat dihasilkan secara konsisten tanpa kehilangan makna akun asal. Sejalan dengan prinsip Configuration over Customization dan Single Source of Truth, seluruh Account eliminasi, Account investasi, Account NCI, Account translasi mata uang, dan mapping Account entitas anak harus dapat dikonfigurasi tanpa perubahan source code, serta digunakan secara konsisten sebagai satu sumber kebenaran oleh seluruh proses konsolidasi lintas perusahaan.

---

# 9.6.3.38.2 Business Capability

- Konfigurasi Account eliminasi transaksi intercompany untuk mengeliminasi saldo dan transaksi antar entitas dalam grup konsolidasi.
- Penentuan Account investasi pada entitas anak (investment in subsidiary) yang dipetakan terhadap ekuitas entitas anak pada saat konsolidasi.
- Eliminasi resiprokal antara Account investasi entitas induk dengan Account ekuitas entitas anak sesuai metode konsolidasi (full consolidation, proportional, equity method).
- Penentuan Account kepentingan non-pengendali (Non-Controlling Interest/NCI) untuk memisahkan porsi kepemilikan pihak di luar entitas induk.
- Perhitungan dan pencatatan Account NCI pada laporan posisi keuangan dan laporan laba rugi konsolidasi berdasarkan persentase kepemilikan.
- Penentuan Account selisih translasi mata uang konsolidasi (cumulative translation adjustment/CTA) untuk entitas anak dengan mata uang fungsional berbeda.
- Mapping Account entitas anak (local chart of accounts) ke Account standar grup (group chart of accounts) sebagai bagian dari proses harmonisasi struktur akun konsolidasi.
- Validasi kelengkapan mapping Account seluruh entitas anak sebelum proses konsolidasi dijalankan, untuk mencegah akun yang tidak terpetakan (unmapped account).
- Penyediaan jejak audit (audit trail) atas setiap perubahan mapping Account, Account eliminasi, dan Account NCI antar periode konsolidasi.
- Ketertelusuran (traceability) saldo Account hasil konsolidasi kembali ke Account asal pada masing-masing entitas anak (drill-down).
- Dukungan multi-currency pada Account translasi, termasuk penerapan kurs penutupan (closing rate), kurs rata-rata (average rate), dan kurs historis sesuai jenis akun.
- Rekonsiliasi otomatis antara saldo Account eliminasi, Account investasi, dan Account NCI terhadap General Ledger masing-masing entitas pada setiap siklus konsolidasi.

---

# 9.6.3.38.3 Business Requirement Catalogue

## BR-956 – Intercompany Elimination Account Configuration

| Attribute           | Description                                                                                                                                                                                                    |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-956                                                                                                                                                                                                        |
| Requirement Name    | Intercompany Elimination Account Configuration                                                                                                                                                               |
| Category            | Consolidation Account Integration                                                                                                                                                                            |
| Priority            | Critical                                                                                                                                                                                                      |
| Business Objective  | BO-004, BO-020                                                                                                                                                                                                |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                                                             |
| Description         | Sistem harus menyediakan konfigurasi Account eliminasi transaksi intercompany yang digunakan untuk mengeliminasi saldo dan transaksi antar entitas dalam grup konsolidasi.                                 |
| Business Rationale  | Tanpa Account eliminasi yang terkonfigurasi secara eksplisit, saldo dan transaksi intercompany berpotensi tercatat ganda pada laporan konsolidasi, sehingga menurunkan keakuratan laporan keuangan grup.    |
| Business Value      | Menjamin laporan konsolidasi mencerminkan posisi keuangan grup secara akurat sebagai satu sumber kebenaran (single source of truth) tanpa distorsi akibat transaksi antar entitas.                          |
| Acceptance Criteria | Sistem dapat mengidentifikasi dan mengeliminasi saldo Account intercompany secara otomatis berdasarkan pasangan Account eliminasi yang telah dikonfigurasi antar entitas anak dan entitas induk.            |

---

## BR-957 – Investment in Subsidiary Account Mapping

| Attribute       | Description                                                                                                                                                       |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-957                                                                                                                                                            |
| Requirement Name| Investment in Subsidiary Account Mapping                                                                                                                          |
| Priority        | Critical                                                                                                                                                          |
| Description     | Sistem harus menyediakan penetapan Account investasi pada entitas anak (investment in subsidiary) yang dipetakan terhadap ekuitas entitas anak saat konsolidasi. |

---

## BR-958 – Reciprocal Elimination of Investment & Subsidiary Equity Account

| Attribute       | Description                                                                                                                                                          |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-958                                                                                                                                                            |
| Requirement Name| Reciprocal Elimination of Investment & Subsidiary Equity Account                                                                                                   |
| Priority        | Critical                                                                                                                                                          |
| Description     | Sistem harus mengeliminasi secara resiprokal Account investasi entitas induk dengan Account ekuitas entitas anak sesuai metode konsolidasi yang berlaku.        |

---

## BR-959 – Non-Controlling Interest (NCI) Account Designation

| Attribute       | Description                                                                                                                                                    |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-959                                                                                                                                                        |
| Requirement Name| Non-Controlling Interest (NCI) Account Designation                                                                                                            |
| Priority        | Critical                                                                                                                                                      |
| Description     | Sistem harus menyediakan Account kepentingan non-pengendali (NCI) untuk memisahkan porsi kepemilikan pihak di luar entitas induk pada laporan konsolidasi.  |

---

## BR-960 – NCI Account Calculation by Ownership Percentage

| Attribute       | Description                                                                                                                                                       |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID  | BR-960                                                                                                                                                          |
| Requirement Name| NCI Account Calculation by Ownership Percentage                                                                                                                  |
| Priority        | High                                                                                                                                                             |
| Description     | Sistem harus menghitung dan mencatat nilai Account NCI pada laporan posisi keuangan dan laba rugi konsolidasi berdasarkan persentase kepemilikan entitas anak. |

---

## BR-961 – Cumulative Translation Adjustment (CTA) Account Configuration

| Attribute       | Description                                                                                                                                                          |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-961                                                                                                                                                             |
| Requirement Name| Cumulative Translation Adjustment (CTA) Account Configuration                                                                                                     |
| Priority        | Critical                                                                                                                                                           |
| Description     | Sistem harus menyediakan Account selisih translasi mata uang konsolidasi (CTA) untuk entitas anak dengan mata uang fungsional berbeda dari mata uang pelaporan. |

---

## BR-962 – Multi-Rate Translation Application per Account Type

| Attribute       | Description                                                                                                                                                       |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID  | BR-962                                                                                                                                                          |
| Requirement Name| Multi-Rate Translation Application per Account Type                                                                                                             |
| Priority        | High                                                                                                                                                             |
| Description     | Sistem harus menerapkan kurs penutupan, kurs rata-rata, dan kurs historis pada Account translasi sesuai jenis Account (neraca, laba rugi, ekuitas).           |

---

## BR-963 – Subsidiary Account to Group Standard Account Mapping

| Attribute       | Description                                                                                                                                                    |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-963                                                                                                                                                       |
| Requirement Name| Subsidiary Account to Group Standard Account Mapping                                                                                                          |
| Priority        | Critical                                                                                                                                                      |
| Description     | Sistem harus menyediakan mapping Account entitas anak (local chart of accounts) ke Account standar grup (group chart of accounts) untuk kebutuhan konsolidasi. |

---

## BR-964 – Unmapped Account Validation Before Consolidation Run

| Attribute       | Description                                                                                                                                                    |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-964                                                                                                                                                       |
| Requirement Name| Unmapped Account Validation Before Consolidation Run                                                                                                          |
| Priority        | High                                                                                                                                                          |
| Description     | Sistem harus memvalidasi kelengkapan mapping Account seluruh entitas anak dan mencegah proses konsolidasi berjalan apabila terdapat Account yang belum terpetakan. |

---

## BR-965 – Consolidated Account Balance Drill-Down to Source Entity

| Attribute       | Description                                                                                                                                                    |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-965                                                                                                                                                       |
| Requirement Name| Consolidated Account Balance Drill-Down to Source Entity                                                                                                      |
| Priority        | Critical                                                                                                                                                      |
| Description     | Sistem harus menyediakan ketertelusuran (drill-down) saldo Account hasil konsolidasi kembali ke Account asal pada masing-masing entitas anak secara real-time. |

---

# 9.6.3.38.4 Business Rules

| Business Rule ID   | Description                                                                                                                                               |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-FCONAI-001    | Setiap pasangan Account eliminasi intercompany wajib dikonfigurasi secara eksplisit antar entitas anak dan entitas induk sebelum konsolidasi dijalankan. |
| BRule-FCONAI-002    | Account investasi pada entitas anak wajib dieliminasi secara resiprokal dengan Account ekuitas entitas anak sesuai metode konsolidasi yang berlaku.     |
| BRule-FCONAI-003    | Account NCI wajib dihitung berdasarkan persentase kepemilikan yang berlaku pada periode konsolidasi terkait dan tidak dapat dicatat secara manual.       |
| BRule-FCONAI-004    | Account CTA wajib digunakan untuk mencatat selisih translasi mata uang bagi entitas anak dengan mata uang fungsional berbeda dari mata uang pelaporan.  |
| BRule-FCONAI-005    | Seluruh Account entitas anak wajib memiliki mapping ke Account standar grup sebelum proses konsolidasi dapat dijalankan; Account tanpa mapping akan diblokir. |
| BRule-FCONAI-006    | Perubahan mapping Account entitas anak ke Account standar grup tidak boleh mengubah hasil konsolidasi periode sebelumnya yang telah difinalisasi.       |
| BRule-FCONAI-007    | Saldo Account hasil konsolidasi wajib dapat ditelusuri (drill-down) kembali ke Account asal pada entitas anak sampai level transaksi.                    |

---

# 9.6.3.38.5 Data Attributes

| Attribute Name              | Description                                                                                    | Data Type | Mandatory |
| ---------------------------- | ------------------------------------------------------------------------------------------------ | --------- | --------- |
| Source Account Code          | Kode Account pada entitas anak (local chart of accounts) sebelum dipetakan ke akun grup.         | String    | Yes       |
| Group Account Code           | Kode Account standar grup (group chart of accounts) hasil mapping.                               | String    | Yes       |
| Elimination Account Code     | Kode Account eliminasi yang digunakan untuk mengeliminasi transaksi intercompany.                | String    | Yes       |
| Investment Account Code      | Kode Account investasi pada entitas anak (investment in subsidiary).                             | String    | Yes       |
| NCI Account Code             | Kode Account kepentingan non-pengendali (Non-Controlling Interest).                               | String    | Yes       |
| Ownership Percentage         | Persentase kepemilikan entitas induk terhadap entitas anak yang digunakan untuk perhitungan NCI. | Numeric   | Yes       |
| CTA Account Code             | Kode Account selisih translasi mata uang konsolidasi (cumulative translation adjustment).        | String    | Yes       |
| Translation Rate Type        | Jenis kurs yang diterapkan pada Account (Closing Rate, Average Rate, Historical Rate).            | Enum      | Yes       |
| Mapping Status               | Status kelengkapan mapping Account entitas anak terhadap Account standar grup.                    | Enum      | Yes       |
| Consolidation Method          | Metode konsolidasi yang diterapkan pada entitas anak (Full, Proportional, Equity Method).         | Enum      | Yes       |

---

# 9.6.3.38.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                                          |
| -------------- | ------------------- | ----------- | -------------------------------------------------------- |
| BR-956         | BO-004, BO-020       | SH-003      | Intercompany Elimination Account Setup                    |
| BR-957         | BO-004, BO-020       | SH-003      | Investment in Subsidiary Account Mapping Setup             |
| BR-958         | BO-020               | SH-003      | Investment & Equity Reciprocal Elimination Processing      |
| BR-959         | BO-004, BO-020       | SH-003      | NCI Account Setup                                          |
| BR-960         | BO-020               | SH-003      | NCI Calculation & Reporting                                |
| BR-961         | BO-004, BO-020       | SH-003      | CTA Account Setup for Currency Translation                 |
| BR-962         | BO-020               | SH-003      | Multi-Rate Currency Translation Processing                 |
| BR-963         | BO-004, BO-020       | SH-004      | Subsidiary to Group Account Mapping Setup                  |
| BR-964         | BO-004               | SH-003      | Unmapped Account Validation Prior to Consolidation Run      |
| BR-965         | BO-004, BO-008       | SH-003      | Consolidated Account Drill-Down Reporting                  |

---

# 9.6.3.38.7 Summary

Consolidation Account Integration Requirements menetapkan bagaimana struktur akun (Chart of Accounts/Account) diintegrasikan secara spesifik dengan kebutuhan konsolidasi laporan keuangan lintas entitas, mencakup Account eliminasi transaksi intercompany, Account investasi pada entitas anak, Account kepentingan non-pengendali (NCI), Account selisih translasi mata uang konsolidasi (CTA), serta mapping Account entitas anak ke Account standar grup. Dokumen ini melengkapi requirement umum Financial Consolidation (9.5.15) dengan sudut pandang Account, memastikan bahwa setiap Account yang terlibat dalam proses konsolidasi terpetakan secara lengkap, tereliminasi secara akurat, dan tertelusur kembali ke entitas asal, sehingga laporan konsolidasi grup dapat diandalkan sebagai satu sumber kebenaran (single source of truth) bagi pengambilan keputusan lintas perusahaan.
