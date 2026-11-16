# 9.6.3.27 Account Category Detail Requirements

---

# 9.6.3.27.1 Overview

Dokumen ini merupakan pendalaman teknis (*deep-dive*) dari Account Category Requirements (9.6.3.3, BR-321 s.d. BR-330) dan tidak mengulang definisi dasar yang telah dijelaskan pada dokumen tersebut. Fokus dokumen ini adalah pada: (1) pemetaan kategori akun ke sub-kategori yang lebih granular (misalnya Current Asset dan Non-Current Asset di bawah kategori Asset); (2) penanganan perbedaan kategorisasi antarstandar akuntansi (PSAK, IFRS, US GAAP) dalam satu Chart of Accounts yang sama; (3) skenario reklasifikasi kategori akun akibat perubahan kebijakan akuntansi; serta (4) dampak reklasifikasi terhadap laporan keuangan komparatif.

Kebutuhan ini penting bagi grup perusahaan multi-organisasi yang beroperasi lintas yurisdiksi dan wajib menyusun laporan keuangan berdasarkan lebih dari satu standar akuntansi sekaligus (misalnya laporan statutori PSAK untuk entitas lokal dan laporan konsolidasi IFRS untuk Holding), tanpa harus memelihara Chart of Accounts yang terpisah untuk setiap standar.

---

# 9.6.3.27.2 Business Capability

Account Category Detail menyediakan kapabilitas lanjutan berikut:

* Granular Sub-Category Hierarchy (Current vs Non-Current, dsb.).
* Sub-Category Rollup Consistency.
* Multi Accounting Standard Category Mapping Matrix.
* Standard-Specific Category Variance Handling.
* Reporting Standard Designation per Ledger/Financial Book.
* Accounting Policy Change Reclassification Workflow.
* Reclassification Effective Date Control.
* Comparative Period Restatement Handling.
* Reclassification Disclosure & Audit Trail.
* Multi-Standard Category Comparison Reporting.
* Category Mapping Completeness Validation.

---

# 9.6.3.27.3 Business Requirement Catalogue

## BR-846 – Granular Sub-Category Hierarchy

| Attribute           | Description                                                                                                                                                                                                                                        |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-846                                                                                                                                                                                                                                              |
| Requirement Name    | Granular Sub-Category Hierarchy                                                                                                                                                                                                                     |
| Category            | Account Category Detail                                                                                                                                                                                                                             |
| Priority            | Critical                                                                                                                                                                                                                                             |
| Business Objective  | BO-022 (Akurasi Laporan Keuangan)                                                                                                                                                                                                                    |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                                                                                                     |
| Description         | Sistem harus mendukung sub-kategori granular di bawah Account Category tingkat tertinggi, misalnya Current Asset dan Non-Current Asset di bawah Asset, atau Current Liability dan Non-Current Liability di bawah Liability, dengan kedalaman hierarki yang dapat dikonfigurasi. |
| Business Rationale  | Penyajian laporan posisi keuangan yang sesuai standar akuntansi (misalnya klasifikasi lancar/tidak lancar) membutuhkan granularitas di bawah kategori utama yang tidak cukup ditangani oleh kategori tingkat tertinggi saja.                        |
| Business Value      | Memungkinkan penyusunan laporan keuangan yang sesuai format standar (*classified balance sheet*) secara otomatis tanpa rekonfigurasi manual pada setiap periode pelaporan.                                                                          |
| Acceptance Criteria | Administrator dapat mendefinisikan sub-kategori di bawah kategori induk; total saldo seluruh sub-kategori harus sama dengan total saldo kategori induk pada setiap laporan yang dihasilkan.                                                          |

---

## BR-847 – Multi-Standard Category Mapping Matrix

| Attribute        | Description                                                                                                                                                                       |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-847                                                                                                                                                                             |
| Requirement Name | Multi-Standard Category Mapping Matrix                                                                                                                                            |
| Priority         | Critical                                                                                                                                                                           |
| Description      | Sistem harus menyediakan matriks pemetaan yang memungkinkan satu Account Master dipetakan ke kategori berbeda untuk setiap standar akuntansi yang aktif (PSAK, IFRS, US GAAP), tanpa mengubah kategori dasar pada Chart of Accounts. |

---

## BR-848 – Standard-Specific Category Variance Handling

| Attribute        | Description                                                                                                                                                                 |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-848                                                                                                                                                                       |
| Requirement Name | Standard-Specific Category Variance Handling                                                                                                                                |
| Priority         | Critical                                                                                                                                                                     |
| Description      | Sistem harus menangani kasus ketika satu akun dikategorikan berbeda antarstandar (misalnya suatu instrumen dikategorikan sebagai Liability menurut PSAK namun sebagian sebagai Equity menurut IFRS), termasuk validasi bahwa perbedaan tersebut terdokumentasi dan tidak menimbulkan saldo yang tidak seimbang pada laporan masing-masing standar. |

---

## BR-849 – Reporting Standard Designation per Ledger/Book

| Attribute        | Description                                                                                                                                                    |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-849                                                                                                                                                          |
| Requirement Name | Reporting Standard Designation per Ledger/Book                                                                                                                 |
| Priority         | High                                                                                                                                                            |
| Description      | Sistem harus memungkinkan setiap Ledger atau Financial Book menetapkan standar akuntansi acuan yang digunakan untuk menentukan kategori akun mana yang ditampilkan sebagai kategori utama dalam pelaporan. |

---

## BR-850 – Accounting Policy Change Reclassification Request

| Attribute        | Description                                                                                                                                                     |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-850                                                                                                                                                            |
| Requirement Name | Accounting Policy Change Reclassification Request                                                                                                                |
| Priority         | Critical                                                                                                                                                          |
| Description      | Sistem harus menyediakan mekanisme permintaan reklasifikasi kategori akun secara formal ketika terjadi perubahan kebijakan akuntansi, lengkap dengan alasan, dasar kebijakan, dan dampak yang diusulkan. |

---

## BR-851 – Reclassification Approval and Effective Date Control

| Attribute        | Description                                                                                                                     |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-851                                                                                                                          |
| Requirement Name | Reclassification Approval and Effective Date Control                                                                           |
| Priority         | Critical                                                                                                                        |
| Description      | Setiap permintaan reklasifikasi kategori akun harus melalui persetujuan berjenjang dan hanya berlaku efektif mulai tanggal yang ditetapkan, tanpa mengubah klasifikasi transaksi yang telah diposting sebelum tanggal tersebut. |

---

## BR-852 – Comparative Period Restatement Impact

| Attribute        | Description                                                                                                                                                      |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-852                                                                                                                                                             |
| Requirement Name | Comparative Period Restatement Impact                                                                                                                              |
| Priority         | Critical                                                                                                                                                            |
| Description      | Sistem harus mendukung penyajian ulang (*restatement*) saldo komparatif periode sebelumnya sesuai kategori baru hasil reklasifikasi, dengan penandaan yang jelas bahwa angka tersebut merupakan angka yang disajikan kembali. |

---

## BR-853 – Reclassification Disclosure & Audit Trail

| Attribute        | Description                                                                                                                                              |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-853                                                                                                                                                     |
| Requirement Name | Reclassification Disclosure & Audit Trail                                                                                                                  |
| Priority         | High                                                                                                                                                        |
| Description      | Sistem harus mencatat setiap reklasifikasi kategori akun secara lengkap dalam Audit Trail, termasuk kategori sebelum dan sesudah perubahan, serta menyediakan data pendukung untuk kebutuhan catatan atas laporan keuangan (*disclosure notes*). |

---

## BR-854 – Sub-Category Rollup Consistency Validation

| Attribute        | Description                                                                                                                                          |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-854                                                                                                                                                 |
| Requirement Name | Sub-Category Rollup Consistency Validation                                                                                                             |
| Priority         | High                                                                                                                                                    |
| Description      | Sistem harus memvalidasi bahwa total nilai seluruh sub-kategori senantiasa konsisten dengan total kategori induknya pada setiap standar akuntansi yang digunakan, dan mencegah publikasi laporan apabila terjadi ketidaksesuaian. |

---

## BR-855 – Multi-Standard Category Comparison Reporting

| Attribute        | Description                                                                                                                                                |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID   | BR-855                                                                                                                                                       |
| Requirement Name | Multi-Standard Category Comparison Reporting                                                                                                                 |
| Priority         | High                                                                                                                                                          |
| Description      | Sistem harus menyediakan laporan pembanding yang menampilkan klasifikasi akun berdampingan antarstandar akuntansi (PSAK, IFRS, US GAAP) untuk mendukung analisis perbedaan dan proses audit. |

---

# 9.6.3.27.4 Business Rules

| Business Rule ID | Description                                                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| BRule-ACD-001      | Setiap sub-kategori harus tepat memiliki satu kategori induk yang terdaftar pada Account Category Master; sub-kategori tanpa induk tidak diperbolehkan.           |
| BRule-ACD-002      | Setiap akun yang tercakup dalam standar akuntansi aktif wajib memiliki pemetaan kategori untuk masing-masing standar tersebut sebelum periode pelaporan dibuka.    |
| BRule-ACD-003      | Reklasifikasi kategori akun tidak boleh mengubah kategori pada jurnal historis yang telah diposting; reklasifikasi hanya berlaku maju (*prospective*) kecuali disertai instruksi restatement resmi. |
| BRule-ACD-004      | Nilai komparatif yang disajikan ulang akibat reklasifikasi wajib ditandai sebagai "Restated" dan disertai referensi ke permintaan reklasifikasi terkait.          |
| BRule-ACD-005      | Reklasifikasi kategori hanya berlaku efektif setelah melalui persetujuan berjenjang dan memiliki tanggal efektif yang telah ditetapkan.                            |
| BRule-ACD-006      | Perbedaan pemetaan kategori antarstandar akuntansi harus terdokumentasi; kombinasi akun-standar yang belum memiliki pemetaan tidak boleh digunakan dalam pelaporan resmi. |
| BRule-ACD-007      | Seluruh aktivitas reklasifikasi, termasuk kategori sebelum dan sesudah perubahan, harus dicatat lengkap dalam Audit Trail.                                         |

---

# 9.6.3.27.5 Data Attributes

| Attribute                        | Description                                                                                     |
| --------------------------------- | ------------------------------------------------------------------------------------------------ |
| Sub-Category Code                 | Kode sub-kategori akun.                                                                            |
| Sub-Category Name                 | Nama sub-kategori akun (misalnya Current Asset, Non-Current Asset).                                |
| Parent Category Reference         | Referensi ke Account Category induk.                                                               |
| Accounting Standard Code          | PSAK, IFRS, US GAAP, Local GAAP, atau standar lainnya.                                              |
| Standard-Specific Mapping ID      | Identitas unik pemetaan kategori untuk kombinasi akun dan standar tertentu.                        |
| Mapping Completeness Status       | Complete, Incomplete, Pending Review.                                                               |
| Reclassification Request ID       | Identitas unik permintaan reklasifikasi kategori.                                                   |
| Reclassification Reason           | Alasan bisnis/kebijakan reklasifikasi.                                                              |
| Prior Category (Before Reclass)   | Kategori/sub-kategori sebelum reklasifikasi.                                                        |
| New Category (After Reclass)      | Kategori/sub-kategori setelah reklasifikasi.                                                        |
| Effective Reclassification Date   | Tanggal reklasifikasi mulai berlaku.                                                                 |
| Restatement Flag                  | Penanda bahwa data komparatif merupakan hasil penyajian kembali.                                     |
| Comparative Impact Description    | Ringkasan dampak reklasifikasi terhadap laporan komparatif.                                          |
| Disclosure Reference              | Referensi ke catatan atas laporan keuangan terkait reklasifikasi.                                   |
| Approval Status                   | Draft, Pending Approval, Approved, Rejected.                                                        |
| Created By / Created Date         | Informasi pembuatan data.                                                                             |
| Updated By / Updated Date         | Informasi perubahan terakhir.                                                                        |

---

# 9.6.3.27.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                            |
| -------------- | ------------------- | ----------- | ------------------------------------------- |
| BR-846         | BO-022              | SH-003      | Classified Financial Statement Preparation  |
| BR-847         | BO-020              | SH-003      | Multi-Standard Category Mapping             |
| BR-848         | BO-022              | SH-003      | Standard Variance Reconciliation            |
| BR-849         | BO-020              | SH-003      | Reporting Standard Governance               |
| BR-850         | BO-004              | SH-003      | Accounting Policy Change Management         |
| BR-851         | BO-028              | SH-010      | Reclassification Approval Governance        |
| BR-852         | BO-022              | SH-003      | Comparative Report Restatement              |
| BR-853         | BO-028              | SH-010      | Reclassification Audit & Disclosure         |
| BR-854         | BO-022              | SH-003      | Category Rollup Validation                  |
| BR-855         | BO-020              | SH-010      | Multi-Standard Comparative Reporting        |

---

# 9.6.3.27.7 Summary

Dokumen ini memperdalam Account Category Requirements dasar (9.6.3.3) dengan menambahkan kapabilitas granularitas sub-kategori, pemetaan kategori lintas standar akuntansi (PSAK, IFRS, US GAAP), serta tata kelola reklasifikasi kategori akun akibat perubahan kebijakan akuntansi berikut dampaknya terhadap laporan komparatif.

Dengan sepuluh kebutuhan bisnis tambahan (BR-846 s.d. BR-855) dan tujuh aturan bisnis (BRule-ACD-001 s.d. BRule-ACD-007), organisasi dapat menyusun laporan keuangan yang akurat dan konsisten di berbagai standar akuntansi sekaligus, sambil menjaga integritas data historis, kejelasan penyajian ulang komparatif, dan kelengkapan jejak audit atas setiap perubahan klasifikasi akun.
