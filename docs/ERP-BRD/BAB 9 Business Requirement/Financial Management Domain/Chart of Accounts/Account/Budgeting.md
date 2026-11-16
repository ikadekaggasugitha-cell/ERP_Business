# 9.6.3.39 Budgeting Account Integration Requirements

## 9.6.3.39.1 Overview

Dokumen ini membahas bagaimana Chart of Accounts (COA) — khususnya struktur Account (akun General Ledger) — direferensikan, dipetakan, dan dipakai secara teknis oleh proses Budget Management. Fokus dokumen ini BUKAN pada proses bisnis penganggaran secara umum (perencanaan, alokasi, revisi, dan persetujuan anggaran telah dibahas tuntas pada BR-636 s/d BR-645), melainkan pada sudut pandang Account/COA: bagaimana setiap akun GL berperan sebagai unit terkecil pengendalian anggaran, bagaimana akun dipetakan ke Budget Line Item, bagaimana akun kontrol anggaran (budgetary control account) mencegah transaksi melampaui sisa anggaran, serta bagaimana akun varians dan granularitas anggaran per kombinasi akun ditentukan.

Chart of Accounts adalah fondasi dari seluruh mekanisme budgetary control. Tanpa pemetaan akun yang eksplisit dan terkelola dengan baik, sistem tidak dapat menentukan transaksi mana yang harus divalidasi terhadap sisa anggaran, akun mana yang menjadi titik akumulasi komitmen (encumbrance), maupun akun mana yang digunakan untuk mencatat selisih (varians) antara nilai anggaran dan realisasi. Setiap Account pada COA yang bersifat Budget-Controlled harus memiliki metadata tambahan berupa jenis kontrol, granularitas, dan referensi Budget Line Item yang berlaku pada periode fiskal tertentu.

Dokumen ini disusun agar tim desain data model, tim konfigurasi COA, dan tim Budget Management memiliki acuan bersama mengenai kontrak integrasi antara struktur akun dan proses anggaran, sejalan dengan prinsip Single Source of Truth dan Configuration over Customization yang dianut ERP ini.

---

## 9.6.3.39.2 Business Capability

- Memetakan setiap Account pada COA ke satu atau lebih Budget Line Item pada periode fiskal aktif.
- Menandai Account tertentu sebagai Budgetary Control Account, dengan jenis kontrol Hard Stop, Warning, atau None.
- Menghitung sisa anggaran (available budget) secara real-time per kombinasi Account-Cost Center saat transaksi diposting.
- Mencegah pemostingan transaksi pada akun yang telah melampaui sisa anggaran, kecuali melalui override berotorisasi.
- Menghasilkan akun varians realisasi vs anggaran secara otomatis per Account tanpa input manual.
- Mendukung granularitas anggaran minimum pada kombinasi Account x Cost Center x Periode Fiskal, dengan opsi dimensi tambahan (Project, Organization Unit).
- Menggulung (roll-up) saldo anggaran dan realisasi sepanjang hierarki Account (parent-child) untuk keperluan konsolidasi.
- Menjaga konsistensi pemetaan akun-anggaran ketika terjadi perubahan struktur COA (reklasifikasi, penonaktifan, penggabungan akun).
- Mencatat komitmen (encumbrance/commitment) pada akun terkait sebelum realisasi aktual tercatat.
- Menyediakan mekanisme exclusion/inclusion agar jenis akun tertentu (mis. akun off-balance sheet, akun statistik) tidak tunduk pada kontrol anggaran.
- Menjamin jejak audit atas seluruh perubahan pemetaan akun-ke-anggaran.
- Mendukung konsolidasi anggaran lintas organisasi pada akun yang sama dalam struktur multi-organisasi.

---

## 9.6.3.39.3 Business Requirement Catalogue

## BR-966 – Account-to-Budget-Line Mapping

| Attribute           | Description                                                                                                                                                          |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-966                                                                                                                                                                |
| Requirement Name    | Account-to-Budget-Line Mapping                                                                                                                                       |
| Category            | Chart of Accounts – Budgeting Integration                                                                                                                            |
| Priority            | Critical                                                                                                                                                              |
| Business Objective  | BO-004, BO-022                                                                                                                                                        |
| Stakeholder Owner   | SH-003 (CFO)                                                                                                                                                          |
| Description         | Sistem harus menyediakan pemetaan eksplisit antara setiap Account pada COA dengan satu atau lebih Budget Line Item pada periode fiskal aktif, termasuk metadata dimensi tambahan (Cost Center, Project, Organization Unit). |
| Business Rationale  | Tanpa pemetaan akun-ke-budget yang eksplisit, sistem tidak dapat menentukan akun mana yang harus divalidasi terhadap anggaran maupun menyusun laporan realisasi per akun secara konsisten. |
| Business Value      | Menjamin Single Source of Truth atas hubungan struktur akun dan struktur anggaran, mengurangi risiko salah alokasi anggaran antar akun.                              |
| Acceptance Criteria | Setiap Account berstatus Budget-Controlled memiliki minimal satu mapping Budget Line Item aktif yang tervalidasi sebelum periode fiskal dibuka untuk transaksi.      |

---

## BR-967 – Budgetary Control Account Designation

| Attribute        | Description                                                                                                                        |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-967                                                                                                                             |
| Requirement Name | Budgetary Control Account Designation                                                                                             |
| Priority         | Critical                                                                                                                           |
| Description      | Sistem harus menyediakan atribut pada Account untuk menandai akun sebagai Budgetary Control Account dengan jenis kontrol (Hard Stop, Warning, None) yang dapat dikonfigurasi per Account Class atau per Account individual. |

---

## BR-968 – Account-Cost Center Budget Granularity

| Attribute        | Description                                                                                                                 |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-968                                                                                                                      |
| Requirement Name | Account-Cost Center Budget Granularity                                                                                     |
| Priority         | Critical                                                                                                                    |
| Description      | Sistem harus mendukung granularitas anggaran minimum pada kombinasi Account x Cost Center x Periode Fiskal, dengan opsi memperluas granularitas ke dimensi Project atau Organization Unit sesuai konfigurasi organisasi. |

---

## BR-969 – Real-Time Available Budget Validation per Account

| Attribute        | Description                                                                                                        |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-969                                                                                                              |
| Requirement Name | Real-Time Available Budget Validation per Account                                                                  |
| Priority         | Critical                                                                                                            |
| Description      | Sistem harus menghitung sisa anggaran (available budget) secara real-time pada kombinasi akun-dimensi terkait setiap kali transaksi menyentuh akun Budget-Controlled, sebelum transaksi diposting ke General Ledger. |

---

## BR-970 – Account-Level Budget Variance Generation

| Attribute        | Description                                                                                                     |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-970                                                                                                           |
| Requirement Name | Account-Level Budget Variance Generation                                                                        |
| Priority         | Critical                                                                                                         |
| Description      | Sistem harus menghasilkan nilai varians (realisasi dikurangi anggaran) secara otomatis per Account pada setiap periode pelaporan, tanpa memerlukan input atau perhitungan manual. |

---

## BR-971 – Account Hierarchy Roll-Up for Budget Consolidation

| Attribute        | Description                                                                                                          |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-971                                                                                                                |
| Requirement Name | Account Hierarchy Roll-Up for Budget Consolidation                                                                   |
| Priority         | High                                                                                                                  |
| Description      | Sistem harus menggulung (roll-up) nilai anggaran, komitmen, realisasi, dan varians sepanjang hierarki parent-child Account Structure untuk mendukung pelaporan anggaran pada level akun ringkasan. |

---

## BR-972 – Multi-Dimensional Budget Account Segmentation

| Attribute        | Description                                                                                                     |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-972                                                                                                           |
| Requirement Name | Multi-Dimensional Budget Account Segmentation                                                                    |
| Priority         | Medium                                                                                                            |
| Description      | Sistem harus mendukung segmentasi anggaran pada akun yang sama berdasarkan kombinasi dimensi Organization Unit, Project, dan Cost Center secara bersamaan, sesuai konfigurasi Organization Tree. |

---

## BR-973 – Account Inclusion/Exclusion Rule for Budgetary Control

| Attribute        | Description                                                                                                        |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-973                                                                                                              |
| Requirement Name | Account Inclusion/Exclusion Rule for Budgetary Control                                                             |
| Priority         | High                                                                                                                |
| Description      | Sistem harus menyediakan mekanisme pengecualian agar Account Class tertentu (mis. akun off-balance sheet, akun statistik, akun antar-perusahaan) dapat dikecualikan dari kontrol anggaran secara eksplisit. |

---

## BR-974 – Budget Mapping Impact of Account Restructuring

| Attribute        | Description                                                                                                            |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-974                                                                                                                  |
| Requirement Name | Budget Mapping Impact of Account Restructuring                                                                         |
| Priority         | High                                                                                                                    |
| Description      | Sistem harus mendeteksi dan menandai dampak perubahan struktur Account (reklasifikasi, penonaktifan, penggabungan) terhadap pemetaan Budget Line Item yang sudah aktif, serta memerlukan persetujuan Budget Owner sebelum perubahan diterapkan. |

---

## BR-975 – Audit Trail of Account-Budget Mapping Changes

| Attribute        | Description                                                                                                  |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-975                                                                                                        |
| Requirement Name | Audit Trail of Account-Budget Mapping Changes                                                                |
| Priority         | Critical                                                                                                      |
| Description      | Sistem harus mencatat seluruh perubahan pemetaan Account-ke-Budget Line Item pada Audit Trail, termasuk pengguna, waktu, nilai sebelum dan sesudah perubahan, serta alasan perubahan. |

---

## 9.6.3.39.4 Business Rules

| Business Rule ID  | Description                                                                                                                          |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------ |
| BRule-BUDAI-001    | Setiap Account berstatus Budget-Controlled wajib memiliki minimal satu Budget Line Item aktif pada periode fiskal berjalan.          |
| BRule-BUDAI-002    | Transaksi yang menyentuh Account berstatus Hard Stop tidak dapat diposting apabila melebihi sisa anggaran, kecuali melalui override berotorisasi dengan jejak audit. |
| BRule-BUDAI-003    | Sisa anggaran (available budget) dihitung sebagai Anggaran dikurangi (Actual ditambah Commitment/Encumbrance) pada kombinasi Account-Cost Center yang sama. |
| BRule-BUDAI-004    | Perubahan struktur Account yang memiliki mapping Budget Line Item aktif memerlukan persetujuan Budget Owner sebelum diterapkan pada COA. |
| BRule-BUDAI-005    | Akun varians anggaran-realisasi dihasilkan otomatis oleh sistem per Account dan tidak dapat diinput atau diubah secara manual.       |
| BRule-BUDAI-006    | Granularitas anggaran minimum adalah kombinasi Account x Cost Center x Periode Fiskal; dimensi tambahan bersifat opsional per organisasi. |
| BRule-BUDAI-007    | Seluruh perubahan pemetaan Account-ke-Budget wajib tercatat pada Audit Trail dengan referensi pengguna, waktu, dan alasan perubahan.  |

---

## 9.6.3.39.5 Data Attributes

| Attribute                     | Description                                                                                     |
| ------------------------------ | ------------------------------------------------------------------------------------------------ |
| Budget Account Mapping ID      | Identitas unik pemetaan antara Account dan Budget Line Item.                                     |
| GL Account Code/ID             | Referensi ke Account pada Chart of Accounts.                                                     |
| Cost Center ID                 | Referensi dimensi Cost Center yang membentuk granularitas anggaran bersama Account.               |
| Organization Unit ID           | Referensi opsional dimensi organisasi sesuai Organization Tree.                                  |
| Project ID                     | Referensi opsional dimensi proyek apabila granularitas diperluas.                                 |
| Fiscal Period/Year             | Periode anggaran yang berlaku atas pemetaan.                                                     |
| Budget Line Item ID            | Referensi ke item anggaran spesifik pada modul Budget Management.                                |
| Control Type                   | Jenis kontrol anggaran pada Account: Hard Stop, Warning, atau None.                               |
| Original Budget Amount         | Nilai anggaran awal pada kombinasi Account-dimensi.                                               |
| Revised Budget Amount          | Nilai anggaran setelah revisi disetujui.                                                          |
| Committed Amount (Encumbrance) | Nilai komitmen yang telah dicatat namun belum menjadi realisasi aktual.                          |
| Actual Amount Posted           | Nilai realisasi aktual yang telah diposting ke Account.                                          |
| Available Budget Amount        | Nilai sisa anggaran hasil kalkulasi otomatis sistem.                                              |
| Variance Amount / Variance %   | Selisih nilai dan persentase antara realisasi dan anggaran pada Account.                          |
| Mapping Effective Date         | Tanggal mulai berlaku pemetaan Account-Budget.                                                    |
| Mapping End Date               | Tanggal berakhirnya pemetaan Account-Budget.                                                      |
| Last Modified By / Timestamp   | Pengguna dan waktu perubahan terakhir atas pemetaan.                                              |

---

## 9.6.3.39.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                              |
| -------------- | ------------------- | ----------- | -------------------------------------------- |
| BR-966         | BO-004               | SH-003      | Account-Budget Mapping Configuration          |
| BR-967         | BO-022               | SH-003      | Budgetary Control Account Setup               |
| BR-968         | BO-004               | SH-003      | Budget Granularity Configuration               |
| BR-969         | BO-022               | SH-003      | Real-Time Budget Validation Engine            |
| BR-970         | BO-022               | SH-003      | Budget Variance Reporting                      |
| BR-971         | BO-004               | SH-003      | Budget Consolidation Roll-Up                   |
| BR-972         | BO-009               | SH-003      | Multi-Dimensional Budget Configuration         |
| BR-973         | BO-007               | SH-011      | Budgetary Control Exception Handling           |
| BR-974         | BO-007               | SH-003      | COA Restructuring Impact Assessment            |
| BR-975         | BO-007               | SH-011      | Account-Budget Mapping Audit Review            |

---

## 9.6.3.39.7 Summary

Dokumen ini menetapkan requirement integrasi antara Chart of Accounts dan Budget Management dari sudut pandang Account: pemetaan akun ke Budget Line Item, penetapan akun kontrol anggaran, validasi sisa anggaran secara real-time, pembentukan akun varians otomatis, serta granularitas anggaran per kombinasi akun-cost center. Sepuluh requirement (BR-966 s/d BR-975) dan tujuh Business Rule (BRule-BUDAI-001 s/d BRule-BUDAI-007) memastikan bahwa struktur akun menjadi fondasi pengendalian anggaran yang konsisten, dapat ditelusuri, dan tidak duplikatif terhadap requirement proses bisnis penganggaran umum yang telah dibahas pada BR-636 s/d BR-645.

Seluruh requirement pada bagian ini akan diturunkan lebih lanjut menjadi Functional Requirements, desain model data pemetaan akun-anggaran, spesifikasi API integrasi, dan skenario pengujian pada dokumen lanjutan.
