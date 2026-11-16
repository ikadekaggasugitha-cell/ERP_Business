# 9.6.3.28 Account Type Detail Requirements

---

# 9.6.3.28.1 Overview

Dokumen ini merupakan pendalaman (*deep-dive*) atas **Account Type** yang telah didefinisikan pada dokumen ringkas *Chart of Accounts / Account Type* (9.6.3.4, BR-331 s/d BR-340). Dokumen tersebut mengatur definisi, siklus hidup, dan tata kelola Account Type sebagai objek bisnis. Dokumen ini **tidak mengulang** ketentuan tersebut, melainkan memperdalam tiga area teknis yang menjadi sumber risiko terbesar pada implementasi Chart of Accounts, yaitu:

1. Perilaku **normal balance** (posisi saldo normal debit/kredit) per Account Type dan bagaimana sistem memvalidasinya pada saat posting jurnal.
2. Dampak Account Type terhadap **struktur laporan keuangan**, khususnya pemisahan akun antara *Statement of Financial Position* (Neraca) dan *Statement of Profit or Loss* (Laba Rugi).
3. Aturan validasi Account Type pada saat **kombinasi akun** (account combination) dibentuk, serta penanganan **akun kontra** (contra account) seperti akumulasi penyusutan, cadangan kerugian piutang, dan retur penjualan.

Tanpa aturan yang eksplisit pada area ini, organisasi berisiko menerima jurnal dengan saldo berlawanan arah tanpa terdeteksi, salah menempatkan akun kontra pada sisi laporan yang keliru, atau membentuk kombinasi akun yang secara akuntansi tidak valid namun tetap lolos validasi struktural Chart of Accounts.

---

# 9.6.3.28.2 Business Capability

Account Type Detail menyediakan kapabilitas lanjutan berikut:

* Normal Balance Assignment per Account Type (Debit/Kredit).
* Real-Time Posting Validation terhadap Normal Balance.
* Abnormal Balance Detection & Exception Workflow.
* Balance Sheet vs Income Statement Behavior Mapping.
* Statement Section Auto-Classification pada saat pembuatan akun.
* Contra Account Designation & Host Account Linking.
* Contra Account Sign Convention pada Penyajian Laporan.
* Account Type Compatibility Matrix untuk Account Combination.
* Validasi Kombinasi Akun Berbasis Account Type saat Pembuatan Kode Akun Baru.
* Reclassification Impact Simulation sebelum perubahan Account Type diberlakukan.
* Financial Statement Line Auto-Mapping berbasis Account Type.
* Cross-Module Consistency Check (Fixed Assets, Accounts Receivable, Inventory) untuk akun kontra.

---

# 9.6.3.28.3 Business Requirement Catalogue

## BR-856 – Normal Balance Assignment

| Attribute           | Description                                                                                                                                                                                                                        |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID      | BR-856                                                                                                                                                                                                                            |
| Requirement Name    | Normal Balance Assignment                                                                                                                                                                                                         |
| Category            | Account Type Detail                                                                                                                                                                                                              |
| Priority            | Critical                                                                                                                                                                                                                          |
| Business Objective  | BO-022                                                                                                                                                                                                                            |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                                                                                  |
| Description         | Sistem harus mewajibkan setiap Account Type memiliki atribut **Normal Balance** (Debit atau Kredit) yang eksplisit, dan atribut ini diwariskan secara otomatis oleh setiap Account Master yang menggunakan Account Type tersebut. |
| Business Rationale  | Tanpa Normal Balance yang eksplisit pada level Account Type, validasi posting jurnal harus bergantung pada logika aplikasi yang tersebar dan rawan inkonsistensi antar modul (General Ledger, Accounts Payable, Fixed Assets).   |
| Business Value      | Menjamin konsistensi arah saldo di seluruh transaksi dan mencegah kesalahan input jurnal sejak titik entri.                                                                                                                       |
| Acceptance Criteria | Setiap Account Type memiliki Normal Balance yang wajib diisi sebelum Account Type dapat diaktifkan; seluruh Account Master mewarisi nilai tersebut secara read-only.                                                              |

---

## BR-857 – Posting Validation Against Normal Balance

| Attribute        | Description                                                                                                                                                                                                                                                          |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-857                                                                                                                                                                                                                                                              |
| Requirement Name | Posting Validation Against Normal Balance                                                                                                                                                                                                                           |
| Priority         | Critical                                                                                                                                                                                                                                                            |
| Description      | Pada saat jurnal diposting, sistem harus membandingkan arah saldo baris jurnal (debit/kredit) terhadap Normal Balance Account Type. Contoh: baris jurnal kredit pada akun bertipe Asset (normal balance debit) menghasilkan saldo negatif dan harus ditandai sebagai potensi anomali. |

---

## BR-858 – Balance Sheet vs Income Statement Behavior Mapping

| Attribute        | Description                                                                                                                                                                                                                    |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-858                                                                                                                                                                                                                        |
| Requirement Name | Balance Sheet vs Income Statement Behavior Mapping                                                                                                                                                                            |
| Priority         | Critical                                                                                                                                                                                                                      |
| Description      | Setiap Account Type harus memiliki atribut **Statement Behavior** yang menentukan apakah saldo akun tersebut bersifat kumulatif sepanjang umur entitas (Balance Sheet: Asset, Liability, Equity) atau di-*reset* setiap penutupan periode (Income Statement: Revenue, Expense). |

---

## BR-859 – Contra Account Designation

| Attribute        | Description                                                                                                                                                                                                                                                        |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-859                                                                                                                                                                                                                                                            |
| Requirement Name | Contra Account Designation                                                                                                                                                                                                                                       |
| Priority         | Critical                                                                                                                                                                                                                                                          |
| Description      | Sistem harus menyediakan penanda **Contra Account** pada Account Type sehingga akun seperti Akumulasi Penyusutan (contra-asset), Cadangan Kerugian Penurunan Nilai Piutang (contra-asset), Retur & Potongan Penjualan (contra-revenue) memiliki Normal Balance berlawanan dari kelompok induknya namun tetap disajikan pada bagian laporan yang sama. |

---

## BR-860 – Contra Account Presentation Sign Convention

| Attribute        | Description                                                                                                                                                                     |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-860                                                                                                                                                                        |
| Requirement Name | Contra Account Presentation Sign Convention                                                                                                                                  |
| Priority         | Critical                                                                                                                                                                     |
| Description      | Sistem harus menampilkan saldo Contra Account sebagai pengurang (nilai negatif atau tanda kurung) terhadap akun induknya pada laporan keuangan, tanpa mengubah tanda saldo pada buku besar. |

---

## BR-861 – Account Type Validation on Account Combination Creation

| Attribute        | Description                                                                                                                                                                                          |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-861                                                                                                                                                                                              |
| Requirement Name | Account Type Validation on Account Combination Creation                                                                                                                                            |
| Priority         | Critical                                                                                                                                                                                            |
| Description      | Pada saat kombinasi akun (Segment Value + Account Master) dibentuk, sistem harus memvalidasi bahwa Account Type akun tersebut kompatibel dengan dimensi lain (misalnya akun Header tidak boleh dikombinasikan dengan Cost Center transaksi operasional). |

---

## BR-862 – Account Type Compatibility Matrix

| Attribute        | Description                                                                                                                                                                       |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-862                                                                                                                                                                           |
| Requirement Name | Account Type Compatibility Matrix                                                                                                                                               |
| Priority         | High                                                                                                                                                                             |
| Description      | Sistem harus menyediakan matriks kompatibilitas antara Account Type dan atribut kombinasi akun lain (Segment, Cost Center, Intercompany, Project) yang dapat dikonfigurasi oleh administrator. |

---

## BR-863 – Abnormal Balance Detection & Exception Handling

| Attribute        | Description                                                                                                                                                                                     |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-863                                                                                                                                                                                         |
| Requirement Name | Abnormal Balance Detection & Exception Handling                                                                                                                                               |
| Priority         | High                                                                                                                                                                                           |
| Description      | Sistem harus mendeteksi saldo akun yang berlawanan dari Normal Balance pada akhir periode (misalnya akun Kas dengan saldo kredit) dan menampilkannya pada laporan eksepsi untuk ditinjau oleh Finance sebelum penutupan periode. |

---

## BR-864 – Account Type-Driven Financial Statement Line Mapping

| Attribute        | Description                                                                                                                                                                     |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-864                                                                                                                                                                          |
| Requirement Name | Account Type-Driven Financial Statement Line Mapping                                                                                                                           |
| Priority         | Critical                                                                                                                                                                       |
| Description      | Account Type harus digunakan sebagai salah satu kriteria default dalam pemetaan otomatis akun ke baris laporan keuangan (Financial Statement Mapping), termasuk penentuan bagian (Aset Lancar, Liabilitas Jangka Panjang, Beban Operasional, dsb). |

---

## BR-865 – Reclassification Impact Simulation

| Attribute        | Description                                                                                                                                                                                          |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-865                                                                                                                                                                                              |
| Requirement Name | Reclassification Impact Simulation                                                                                                                                                                  |
| Priority         | High                                                                                                                                                                                                |
| Description      | Sebelum Account Type suatu akun diubah (misalnya dari Posting menjadi Control, atau perubahan status Contra Account), sistem harus menyediakan simulasi dampak terhadap laporan keuangan historis dan saldo berjalan sebagai bahan persetujuan. |

---

# 9.6.3.28.4 Business Rules

| Business Rule ID | Description                                                                                                                                                          |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-ATD-001     | Setiap Account Type wajib memiliki Normal Balance (Debit atau Kredit) yang tidak boleh kosong.                                                                       |
| BRule-ATD-002     | Jurnal yang menghasilkan saldo akun berlawanan dari Normal Balance harus memicu peringatan sistem, kecuali akun ditandai sebagai Contra Account atau Clearing Account. |
| BRule-ATD-003     | Setiap Contra Account wajib direferensikan ke akun induk (host account) dan mewarisi bagian laporan keuangan induknya, namun dengan Normal Balance berlawanan.        |
| BRule-ATD-004     | Account Type dengan Statement Behavior Balance Sheet tidak boleh direklasifikasi langsung menjadi Statement Behavior Income Statement tanpa melalui Reclassification Impact Simulation dan persetujuan. |
| BRule-ATD-005     | Kombinasi akun (account combination) harus divalidasi terhadap Account Type Compatibility Matrix sebelum kombinasi tersebut dapat digunakan pada transaksi.          |
| BRule-ATD-006     | Perubahan Normal Balance pada Account Type yang telah memiliki transaksi historis tidak diperbolehkan; perubahan harus dilakukan melalui Account Type baru dan migrasi terkendali. |
| BRule-ATD-007     | Seluruh deteksi saldo abnormal, penetapan Contra Account, dan hasil simulasi reklasifikasi harus tercatat lengkap dalam Audit Trail.                                  |

---

# 9.6.3.28.5 Data Attributes

| Attribute                             | Description                                                                                     |
| -------------------------------------- | ------------------------------------------------------------------------------------------------- |
| Normal Balance                         | Posisi saldo normal Account Type: Debit atau Kredit.                                              |
| Statement Behavior                     | Balance Sheet (kumulatif) atau Income Statement (di-reset per periode).                           |
| Balance Sheet Section                  | Aset Lancar, Aset Tidak Lancar, Liabilitas Jangka Pendek, Liabilitas Jangka Panjang, Ekuitas, N/A. |
| Income Statement Section               | Pendapatan Operasional, Pendapatan Non-Operasional, Beban Pokok, Beban Operasional, Beban Non-Operasional, N/A. |
| Contra Account Flag                    | Menandakan apakah Account Type berperan sebagai akun kontra.                                      |
| Host Account Type Reference            | Referensi ke Account Type induk bagi Contra Account.                                              |
| Presentation Sign Convention           | Aturan penyajian tanda saldo pada laporan (Normal, Negatif, Dalam Kurung).                         |
| Abnormal Balance Tolerance             | Menentukan apakah saldo berlawanan diperbolehkan sementara (misalnya Clearing Account).            |
| Abnormal Balance Notification Rule     | Aturan eskalasi notifikasi saat saldo abnormal terdeteksi.                                        |
| Account Combination Compatibility Ref  | Referensi ke matriks kompatibilitas kombinasi akun.                                                |
| Financial Statement Line Mapping Ref   | Referensi ke pemetaan baris laporan keuangan.                                                      |
| Reclassification Simulation Reference  | Referensi ke hasil simulasi dampak reklasifikasi terakhir.                                        |
| Reclassification Approval Reference    | Referensi ke persetujuan perubahan Account Type.                                                  |

---

# 9.6.3.28.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                                  |
| -------------- | ------------------- | ----------- | ------------------------------------------------ |
| BR-856         | BO-022              | SH-003      | Normal Balance Governance                        |
| BR-857         | BO-028              | SH-010      | Journal Posting Validation                       |
| BR-858         | BO-022              | SH-003      | Financial Statement Structuring                  |
| BR-859         | BO-022              | SH-003      | Contra Account Management                        |
| BR-860         | BO-004              | SH-003      | Financial Statement Presentation                 |
| BR-861         | BO-009              | SH-003      | Account Combination Validation                   |
| BR-862         | BO-009              | SH-003      | Account Combination Governance                   |
| BR-863         | BO-028              | SH-010      | Period-End Exception Review                      |
| BR-864         | BO-004              | SH-003      | Financial Statement Line Mapping                 |
| BR-865         | BO-028              | SH-010      | Account Type Change Impact Assessment            |

---

# 9.6.3.28.7 Summary

Dokumen ini memperdalam Account Type dari sekadar klasifikasi perilaku posting menjadi kerangka kerja teknis yang mengendalikan arah saldo normal, penempatan akun pada struktur laporan keuangan, penanganan akun kontra, dan validasi kombinasi akun. Dengan aturan Normal Balance yang eksplisit dan validasi posting yang konsisten, organisasi memperoleh jaminan bahwa setiap jurnal yang diposting selaras dengan karakteristik akuntansi akun bersangkutan sejak titik entri, bukan hanya pada saat rekonsiliasi akhir periode.

Kapabilitas Contra Account Designation, Account Type Compatibility Matrix, dan Reclassification Impact Simulation menjadikan Account Type sebagai kontrol preventif yang melindungi integritas Neraca dan Laba Rugi, sekaligus mendukung kebutuhan **General Ledger**, **Fixed Assets**, **Accounts Receivable**, **Financial Statement Mapping**, dan **Audit Trail** secara terintegrasi.
