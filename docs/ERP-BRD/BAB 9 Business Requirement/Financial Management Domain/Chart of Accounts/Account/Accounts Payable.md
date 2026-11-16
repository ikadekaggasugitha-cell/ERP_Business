# 9.6.3.32 Accounts Payable Account Integration Requirements

---

# 9.6.3.32.1 Overview

Dokumen ini membahas bagaimana Account (struktur akun dalam Chart of Accounts) diintegrasikan, direferensikan, dan dipakai secara spesifik oleh modul Accounts Payable (AP), dilihat dari sudut pandang Account itu sendiri — bukan dari sudut pandang proses bisnis pengelolaan utang usaha sebagaimana telah dibahas secara menyeluruh pada dokumen 9.5.5 Accounts Payable Requirements (BR-566 s/d BR-575).

Fokus dokumen ini adalah pada peran akun kontrol (AP Control Account) sebagai satu-satunya titik posting resmi kewajiban utang usaha ke General Ledger, akun GR/IR (Goods Receipt/Invoice Receipt) clearing account sebagai akun perantara antara penerimaan barang dan penerimaan invoice, akun retensi vendor (retention payable) sebagai akun terpisah dari kewajiban utama, mekanisme mapping otomatis vendor invoice ke akun beban atau persediaan berdasarkan kategori vendor dan/atau kategori item, serta akun uang muka vendor (advance payment) sebagai akun terpisah dari AP Control Account hingga terjadi proses offsetting.

Chart of Accounts pada domain Accounts Payable berfungsi sebagai kerangka pemetaan (account mapping framework) yang menentukan ke akun mana setiap komponen transaksi utang usaha harus dibukukan, tanpa mengharuskan pengguna memilih akun secara manual pada setiap transaksi. Ketepatan konfigurasi mapping akun ini menjadi penentu utama keakuratan neraca utang usaha dan kesesuaian saldo AP sub-ledger dengan General Ledger.

Ruang lingkup dokumen ini mencakup lima area utama: (1) akun kontrol utang usaha, (2) akun GR/IR clearing, (3) akun retensi vendor, (4) mapping otomatis akun beban/persediaan berdasarkan kategori vendor/item, dan (5) akun uang muka vendor. Requirement umum mengenai pencatatan invoice, three-way matching, penjadwalan pembayaran, dan rekonsiliasi vendor statement tetap menjadi cakupan dokumen 9.5.5 dan tidak diulang di sini.

---

# 9.6.3.32.2 Business Capability

- Penetapan dan penegakan AP Control Account tunggal per scope organisasi sebagai satu-satunya akun posting kewajiban utang usaha.
- Pencegahan posting manual langsung ke AP Control Account di luar proses otomatis dari sub-ledger Accounts Payable.
- Pemetaan otomatis nilai Goods Receipt ke akun GR/IR clearing account pada saat penerimaan barang sebelum invoice diterima.
- Penyelesaian (clearing) otomatis akun GR/IR pada saat invoice vendor berhasil dicocokkan (matched) dengan Goods Receipt terkait.
- Pemantauan saldo outstanding akun GR/IR clearing sebagai indikator penerimaan barang yang belum ditagih atau invoice yang belum diterima.
- Pemetaan otomatis nilai retensi kontrak ke akun retention payable yang terpisah dari AP Control Account.
- Reklasifikasi otomatis saldo retention payable ke akun kewajiban terbayar pada saat syarat pelepasan retensi terpenuhi.
- Derivasi akun beban atau persediaan secara otomatis berdasarkan kategori item pada baris invoice vendor.
- Derivasi akun secara otomatis berdasarkan kategori atau golongan vendor apabila mapping berbasis item tidak tersedia.
- Pemetaan otomatis uang muka vendor ke akun advance payment yang terpisah dari AP Control Account.
- Reklasifikasi otomatis saldo advance payment ke AP Control Account pada saat terjadi offsetting terhadap invoice.
- Rekonsiliasi otomatis antara saldo AP sub-ledger per vendor dengan saldo AP Control Account pada General Ledger di setiap akhir periode.

---

# 9.6.3.32.3 Business Requirement Catalogue

## BR-896 – AP Control Account Assignment & Sub-Ledger Posting Restriction

| Attribute           | Description                                                                                                                                                                                                                       |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-896                                                                                                                                                                                                                             |
| Requirement Name    | AP Control Account Assignment & Sub-Ledger Posting Restriction                                                                                                                                                                    |
| Category            | Accounts Payable – Account Integration                                                                                                                                                                                            |
| Priority            | Critical                                                                                                                                                                                                                           |
| Business Objective  | BO-004, BO-022                                                                                                                                                                                                                     |
| Stakeholder Owner   | SH-003 (CFO), SH-028 (Accounting Department)                                                                                                                                                                                      |
| Description         | Sistem harus menetapkan satu akun kontrol (AP Control Account) per scope organisasi sebagai satu-satunya akun yang menampung posting kewajiban utang usaha dari sub-ledger Accounts Payable, dan mencegah posting manual langsung ke akun tersebut di luar proses otomatis sub-ledger. |
| Business Rationale  | Tanpa pembatasan posting hanya melalui sub-ledger, saldo AP Control Account berisiko tidak sinkron dengan rincian saldo per vendor pada Accounts Payable, sehingga sulit direkonsiliasi dan dipertanggungjawabkan.                 |
| Business Value      | Menjamin kesesuaian yang konsisten antara saldo utang usaha pada General Ledger dan rincian saldo per vendor pada AP sub-ledger sebagai dasar laporan keuangan yang akurat.                                                       |
| Acceptance Criteria | Sistem menolak setiap upaya posting manual langsung ke akun yang dikonfigurasi sebagai AP Control Account, dan seluruh posting ke akun tersebut hanya dapat terjadi melalui proses otomatis dari transaksi Accounts Payable.       |

---

## BR-897 – GR/IR Clearing Account Posting on Goods Receipt

| Requirement ID | Requirement Name                             | Priority | Description                                                                                                                                                     |
| -------------- | ----------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-897         | GR/IR Clearing Account Posting on Goods Receipt | Critical | Sistem harus memposting nilai penerimaan barang secara otomatis ke akun GR/IR clearing account pada saat Goods Receipt dicatat, sebelum vendor invoice diterima. |

---

## BR-898 – GR/IR Clearing Account Settlement upon Invoice Matching

| Requirement ID | Requirement Name                                      | Priority | Description                                                                                                                                                       |
| -------------- | -------------------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-898         | GR/IR Clearing Account Settlement upon Invoice Matching | Critical | Sistem harus menyelesaikan (clear) saldo akun GR/IR clearing account secara otomatis pada saat vendor invoice berhasil dicocokkan dengan Goods Receipt terkait melalui proses matching. |

---

## BR-899 – GR/IR Clearing Account Aging & Outstanding Monitoring

| Requirement ID | Requirement Name                              | Priority | Description                                                                                                                                                |
| -------------- | ------------------------------------------------ | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-899         | GR/IR Clearing Account Aging & Outstanding Monitoring | High     | Sistem harus menyediakan pemantauan saldo outstanding akun GR/IR clearing berdasarkan umur (aging), untuk mengidentifikasi penerimaan barang yang belum ditagih atau invoice yang belum diproses. |

---

## BR-900 – Retention Payable Account Mapping

| Requirement ID | Requirement Name          | Priority | Description                                                                                                                                                 |
| -------------- | --------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| BR-900         | Retention Payable Account Mapping | High     | Sistem harus memetakan nilai retensi kontrak secara otomatis ke akun retention payable yang terpisah dari AP Control Account pada saat invoice vendor diproses. |

---

## BR-901 – Retention Release Account Reclassification

| Requirement ID | Requirement Name                    | Priority | Description                                                                                                                                                    |
| -------------- | -------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-901         | Retention Release Account Reclassification | High     | Sistem harus melakukan reklasifikasi saldo dari akun retention payable ke akun kewajiban terbayar secara otomatis pada saat syarat pelepasan retensi disetujui dan terpenuhi. |

---

## BR-902 – Automatic Expense/Inventory Account Derivation by Item Category

| Requirement ID | Requirement Name                                          | Priority | Description                                                                                                                                                       |
| -------------- | ------------------------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-902         | Automatic Expense/Inventory Account Derivation by Item Category | Critical | Sistem harus menentukan akun beban atau akun persediaan secara otomatis pada baris vendor invoice berdasarkan kategori item yang direferensikan, sesuai konfigurasi Default Account Rule. |

---

## BR-903 – Account Derivation Fallback by Vendor Category

| Requirement ID | Requirement Name                       | Priority | Description                                                                                                                                                    |
| -------------- | ------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-903         | Account Derivation Fallback by Vendor Category | High     | Sistem harus menyediakan mekanisme fallback derivasi akun berdasarkan kategori atau golongan vendor apabila mapping akun berbasis kategori item tidak tersedia untuk baris invoice tertentu. |

---

## BR-904 – Vendor Advance Payment Account Mapping

| Requirement ID | Requirement Name              | Priority | Description                                                                                                                                             |
| -------------- | -------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-904         | Vendor Advance Payment Account Mapping | High     | Sistem harus memetakan pencatatan uang muka vendor secara otomatis ke akun advance payment yang terpisah dari AP Control Account pada saat pembayaran uang muka dicatat. |

---

## BR-905 – Advance Offsetting Account Reclassification & AP Sub-Ledger Reconciliation

| Requirement ID | Requirement Name                                              | Priority | Description                                                                                                                                                     |
| -------------- | ------------------------------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-905         | Advance Offsetting Account Reclassification & AP Sub-Ledger Reconciliation | Medium   | Sistem harus mereklasifikasikan saldo akun advance payment ke AP Control Account secara otomatis pada saat terjadi offsetting terhadap invoice, dan merekonsiliasi saldo AP sub-ledger per vendor dengan saldo AP Control Account pada General Ledger di setiap akhir periode. |

---

# 9.6.3.32.4 Business Rules

| Business Rule ID | Description                                                                                                                                                        |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-APAI-001    | Posting kewajiban utang usaha ke General Ledger hanya boleh dilakukan melalui AP Control Account yang dikonfigurasi pada scope organisasi terkait; posting manual langsung ke akun tersebut tidak diperkenankan. |
| BRule-APAI-002    | Saldo akun GR/IR clearing wajib bernilai nol per baris Purchase Order setelah proses matching invoice selesai; saldo yang belum nol ditandai sebagai eksepsi outstanding. |
| BRule-APAI-003    | Nilai retensi dicatat ke akun retention payable yang berbeda dari AP Control Account, dan tidak dapat direklasifikasi ke akun kewajiban terbayar sebelum syarat pelepasan retensi terpenuhi. |
| BRule-APAI-004    | Derivasi akun beban/persediaan mengikuti urutan hierarki konfigurasi: kategori item terlebih dahulu, kemudian kategori vendor, dan terakhir akun default; ketiadaan mapping pada seluruh hierarki akan menghentikan proses posting. |
| BRule-APAI-005    | Uang muka vendor dicatat ke akun advance payment dan tidak boleh langsung mengurangi saldo AP Control Account sebelum terjadi proses offsetting terhadap invoice.       |
| BRule-APAI-006    | Setiap offsetting uang muka terhadap invoice wajib menghasilkan entri reklasifikasi yang memindahkan saldo dari akun advance payment ke AP Control Account sebesar nilai yang di-offset. |
| BRule-APAI-007    | Saldo AP sub-ledger per vendor wajib direkonsiliasi terhadap saldo AP Control Account pada General Ledger di setiap akhir periode; selisih yang ditemukan wajib ditandai sebagai eksepsi rekonsiliasi. |

---

# 9.6.3.32.5 Data Attributes

| Attribute Name              | Description                                                                                       |
| ---------------------------- | ----------------------------------------------------------------------------------------------------- |
| AP Control Account Code      | Kode akun kontrol utang usaha yang dikonfigurasi pada scope organisasi tertentu.                       |
| GR/IR Clearing Account Code  | Kode akun perantara antara nilai Goods Receipt dan Invoice Receipt sebelum proses matching selesai.    |
| Retention Payable Account Code | Kode akun yang menampung nilai retensi kontrak yang belum dilepaskan.                                |
| Advance Payment Account Code  | Kode akun yang menampung saldo uang muka vendor sebelum terjadi offsetting.                            |
| Vendor Category               | Kategori atau golongan vendor yang menjadi dasar fallback derivasi akun.                               |
| Item Category                | Kategori item pada baris invoice yang menjadi dasar utama derivasi akun beban/persediaan.               |
| Account Derivation Rule ID    | Identitas aturan konfigurasi yang menentukan hierarki dan hasil derivasi akun.                          |
| Purchase Order Line Reference | Referensi baris Purchase Order terkait posting GR/IR clearing.                                        |
| Goods Receipt Reference       | Referensi dokumen Goods Receipt yang membentuk saldo GR/IR clearing.                                    |
| Vendor Invoice Reference      | Referensi vendor invoice yang menyelesaikan (clear) saldo GR/IR clearing atau advance payment.          |
| Offset Amount                 | Nilai saldo advance payment yang dipindahkan (di-offset) ke AP Control Account pada suatu transaksi.    |
| Retention Release Status      | Status pemenuhan syarat pelepasan retensi (pending/released).                                          |
| AP Sub-Ledger Balance         | Saldo agregat utang usaha per vendor pada AP sub-ledger untuk keperluan rekonsiliasi.                   |
| Reconciliation Status         | Status kesesuaian saldo AP sub-ledger terhadap saldo AP Control Account pada General Ledger.            |

---

# 9.6.3.32.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                                        |
| --------------- | ------------------- | ------------ | -------------------------------------------------------- |
| BR-896          | BO-004, BO-022       | SH-003       | AP Control Account Configuration & Posting Control Design  |
| BR-897          | BO-004               | SH-028       | GR/IR Clearing Posting Design                              |
| BR-898          | BO-004, BO-022       | SH-028       | GR/IR Clearing Settlement Design                           |
| BR-899          | BO-022               | SH-028       | GR/IR Aging Monitoring Design                              |
| BR-900          | BO-022               | SH-028       | Retention Payable Account Mapping Design                   |
| BR-901          | BO-022               | SH-003       | Retention Release Reclassification Design                  |
| BR-902          | BO-004, BO-009        | SH-028       | Item-Based Account Derivation Engine Design                |
| BR-903          | BO-009               | SH-028       | Vendor-Based Account Derivation Fallback Design             |
| BR-904          | BO-004               | SH-028       | Vendor Advance Payment Account Mapping Design               |
| BR-905          | BO-022               | SH-003       | Advance Offsetting & AP-GL Reconciliation Design             |

---

# 9.6.3.32.7 Summary

Dokumen ini menetapkan requirement spesifik mengenai bagaimana Account/Chart of Accounts diintegrasikan dengan Accounts Payable, mencakup peran AP Control Account sebagai satu-satunya titik posting utang usaha, akun GR/IR clearing sebagai perantara antara penerimaan barang dan penerimaan invoice, akun retensi vendor yang terpisah dari kewajiban utama, mapping otomatis akun beban/persediaan berdasarkan kategori vendor dan item, serta akun uang muka vendor beserta mekanisme offsetting-nya. Berbeda dengan dokumen 9.5.5 yang membahas proses bisnis Accounts Payable secara umum, dokumen ini berfokus pada sudut pandang Account sebagai kerangka mapping dan target posting, sehingga menjadi pelengkap yang tidak tumpang tindih dengan requirement BR-566 s/d BR-575. Seluruh requirement pada dokumen ini menjadi dasar bagi desain teknis integrasi antara modul Chart of Accounts dan Accounts Payable pada tahap berikutnya.
