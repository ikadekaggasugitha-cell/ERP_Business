# 9.6.3.26 Account Structure Detail Requirements

---

# 9.6.3.26.1 Overview

Dokumen ini merupakan pendalaman teknis (*deep-dive*) dari Account Structure Requirements (9.6.3.1, BR-301 s.d. BR-310) dan tidak mengulang definisi dasar yang telah dijelaskan pada dokumen tersebut. Fokus dokumen ini adalah pada skenario struktur akun tingkat lanjut yang dihadapi grup perusahaan kompleks, yaitu: (1) kombinasi struktur multi-segmen pada organisasi berjenjang Holding–Sub Holding–Company dengan konfigurasi segmen yang berbeda pada tiap level; (2) aturan validasi lanjutan yang melibatkan lebih dari satu segmen sekaligus (*cross-segment validation*); (3) mekanisme perubahan struktur akun secara historis tanpa mengubah data transaksi yang telah diposting; dan (4) penanganan struktur akun pada skenario merger dan akuisisi anak perusahaan baru.

Kebutuhan ini penting karena pada praktiknya grup perusahaan multi-organisasi jarang menggunakan struktur akun yang seragam di seluruh entitas. Holding dapat menetapkan template struktur minimum yang wajib diikuti seluruh entitas, sementara Sub Holding dan Company dapat menambahkan segmen tambahan sesuai kebutuhan operasional lokal, selama tetap kompatibel untuk kebutuhan konsolidasi.

---

# 9.6.3.26.2 Business Capability

Account Structure Detail menyediakan kapabilitas lanjutan berikut:

* Hierarchical Structure Inheritance (Holding → Sub Holding → Company).
* Tier-Specific Segment Extension.
* Cross-Segment Validation Rule Engine.
* Segment Combination Exception Handling.
* Historical Structure Snapshot & Point-in-Time Reconstruction.
* Structure Change Impact Simulation.
* New Entity Onboarding Structure Template (M&A).
* Legacy-to-Target Segment Cross-Reference Mapping.
* Parallel Structure Coexistence during Transition.
* Structure Conflict Detection & Resolution Workflow.
* Multi-Segment Combination Governance Reporting.
* Structure Lineage Traceability.

---

# 9.6.3.26.3 Business Requirement Catalogue

## BR-836 – Hierarchical Structure Inheritance

| Attribute           | Description                                                                                                                                                                                                                                        |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-836                                                                                                                                                                                                                                            |
| Requirement Name    | Hierarchical Structure Inheritance                                                                                                                                                                                                                |
| Category            | Account Structure Detail                                                                                                                                                                                                                          |
| Priority            | Critical                                                                                                                                                                                                                                          |
| Business Objective  | BO-020 (Konsolidasi Lintas Perusahaan)                                                                                                                                                                                                            |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                                                                                                  |
| Description         | Sistem harus mendukung pewarisan (*inheritance*) segmen wajib dari Account Structure tingkat Holding ke Sub Holding dan Company, sehingga setiap entitas anak wajib mempertahankan segmen inti Holding sambil dapat menambahkan segmen tambahan sesuai kebutuhan lokal. |
| Business Rationale  | Grup perusahaan kompleks memerlukan konsistensi segmen inti (misalnya Company, Natural Account) untuk kebutuhan konsolidasi, namun tetap membutuhkan fleksibilitas segmen tambahan (misalnya Project, Region) pada level operasional yang berbeda. |
| Business Value      | Menjamin kompatibilitas struktur akun antarentitas untuk kebutuhan konsolidasi tanpa membatasi kebutuhan pelaporan lokal masing-masing entitas.                                                                                                   |
| Acceptance Criteria | Sistem menolak penyimpanan Account Structure Sub Holding/Company yang menghapus atau mengubah tipe data segmen inti yang diwariskan dari Holding; segmen tambahan dapat ditambahkan tanpa memengaruhi entitas lain.                              |

---

## BR-837 – Cross-Segment Validation Rule

| Attribute        | Description                                                                                                                                                                     |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-837                                                                                                                                                                          |
| Requirement Name | Cross-Segment Validation Rule                                                                                                                                                  |
| Priority         | Critical                                                                                                                                                                        |
| Description      | Sistem harus mendukung definisi aturan validasi yang melibatkan kombinasi lebih dari satu segmen sekaligus (misalnya kombinasi Segmen Company tertentu hanya boleh dipasangkan dengan rentang Segmen Natural Account tertentu), bukan hanya validasi per-segmen tunggal. |

---

## BR-838 – Segment Combination Exception List

| Attribute        | Description                                                                                                                                          |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-838                                                                                                                                               |
| Requirement Name | Segment Combination Exception List                                                                                                                   |
| Priority         | High                                                                                                                                                  |
| Description      | Sistem harus menyediakan daftar pengecualian (*exception list*) atas kombinasi segmen yang secara eksplisit tidak diperbolehkan maupun yang secara eksplisit diperbolehkan sebagai pengecualian dari aturan umum, lengkap dengan alasan bisnis dan masa berlaku. |

---

## BR-839 – Historical Structure Snapshot Preservation

| Attribute        | Description                                                                                                                                                                |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-839                                                                                                                                                                    |
| Requirement Name | Historical Structure Snapshot Preservation                                                                                                                               |
| Priority         | Critical                                                                                                                                                                  |
| Description      | Sistem harus menyimpan *snapshot* definisi Account Structure yang berlaku pada setiap periode historis, sehingga transaksi lama tetap dapat direkonstruksi dan ditampilkan sesuai struktur yang berlaku pada saat transaksi diposting, terlepas dari perubahan struktur di kemudian hari. |

---

## BR-840 – Structure Change Impact Simulation

| Attribute        | Description                                                                                                                                                       |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-840                                                                                                                                                            |
| Requirement Name | Structure Change Impact Simulation                                                                                                                               |
| Priority         | High                                                                                                                                                              |
| Description      | Sistem harus menyediakan mekanisme simulasi dampak sebelum perubahan Account Structure diterapkan, meliputi jumlah kombinasi akun aktif yang terdampak, entitas yang menggunakan struktur tersebut, serta potensi konflik dengan transaksi berjalan. |

---

## BR-841 – New Entity Onboarding Structure Template

| Attribute        | Description                                                                                                                                                                 |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-841                                                                                                                                                                      |
| Requirement Name | New Entity Onboarding Structure Template                                                                                                                                    |
| Priority         | Critical                                                                                                                                                                    |
| Description      | Sistem harus menyediakan templat Account Structure standar yang dapat digunakan untuk *onboarding* anak perusahaan baru hasil akuisisi, termasuk mekanisme validasi kesesuaian segmen inti terhadap struktur Holding sebelum entitas baru diaktifkan. |

---

## BR-842 – Legacy-to-Target Segment Cross-Reference Mapping

| Attribute        | Description                                                                                                                                                                |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-842                                                                                                                                                                    |
| Requirement Name | Legacy-to-Target Segment Cross-Reference Mapping                                                                                                                         |
| Priority         | Critical                                                                                                                                                                  |
| Description      | Sistem harus menyediakan tabel pemetaan silang (*cross-reference*) antara nilai segmen struktur akun lama milik entitas yang diakuisisi dengan nilai segmen struktur akun target Holding, untuk mendukung konversi data historis maupun pelaporan komparatif pasca-akuisisi. |

---

## BR-843 – Parallel Structure Coexistence during Transition

| Attribute        | Description                                                                                                                                                |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-843                                                                                                                                                     |
| Requirement Name | Parallel Structure Coexistence during Transition                                                                                                           |
| Priority         | High                                                                                                                                                        |
| Description      | Sistem harus mendukung penggunaan dua Account Structure (lama dan baru) secara paralel selama periode transisi migrasi struktur, dengan kontrol rekonsiliasi untuk memastikan tidak ada transaksi yang hilang atau terduplikasi. |

---

## BR-844 – Structure Conflict Resolution Workflow

| Attribute        | Description                                                                                                                                                |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-844                                                                                                                                                     |
| Requirement Name | Structure Conflict Resolution Workflow                                                                                                                     |
| Priority         | Critical                                                                                                                                                   |
| Description      | Sistem harus menyediakan alur kerja penyelesaian konflik struktur akun ketika dua entitas dengan Account Structure berbeda digabungkan (merger), termasuk penentuan struktur target, pemetaan segmen, dan persetujuan sebelum konsolidasi data. |

---

## BR-845 – Multi-Segment Combination Governance Reporting

| Attribute        | Description                                                                                                                                                          |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-845                                                                                                                                                                |
| Requirement Name | Multi-Segment Combination Governance Reporting                                                                                                                       |
| Priority         | High                                                                                                                                                                  |
| Description      | Sistem harus menyediakan laporan tata kelola atas penggunaan kombinasi segmen di seluruh entitas, mencakup kombinasi tidak terpakai (*orphaned*), kombinasi melanggar aturan validasi silang, serta kombinasi yang termasuk daftar pengecualian. |

---

# 9.6.3.26.4 Business Rules

| Business Rule ID | Description                                                                                                                                                                       |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-ASTD-001    | Setiap aturan cross-segment validation harus didefinisikan secara eksplisit; kombinasi segmen yang tidak memiliki aturan eksplisit dan bukan bagian dari exception list harus ditolak sistem.                     |
| BRule-ASTD-002    | Snapshot Account Structure historis harus disimpan permanen dan tertaut pada rentang tanggal transaksi yang menggunakannya, serta tidak boleh dihapus selama masih ada transaksi yang mereferensikannya.          |
| BRule-ASTD-003    | Pemetaan struktur akun untuk entitas baru hasil akuisisi harus diselesaikan dan disetujui sebelum entitas tersebut diperbolehkan memposting transaksi live.                                                        |
| BRule-ASTD-004    | Segmen tambahan yang ditetapkan pada level Sub Holding atau Company tidak boleh menghapus atau melonggarkan status wajib (*mandatory*) segmen inti yang diwariskan dari Holding.                                  |
| BRule-ASTD-005    | Periode Parallel Structure Coexistence harus memiliki tanggal mulai dan berakhir yang eksplisit, serta wajib melalui proses rekonsiliasi sebelum struktur lama dinonaktifkan.                                     |
| BRule-ASTD-006    | Konflik struktur akibat merger harus diselesaikan melalui alur persetujuan formal sebelum data kedua entitas dikonsolidasikan ke dalam satu Ledger atau Financial Book yang sama. |
| BRule-ASTD-007    | Seluruh perubahan aturan cross-segment validation, exception list, dan hasil resolusi konflik struktur harus dicatat lengkap dalam Audit Trail.                                                                    |

---

# 9.6.3.26.5 Data Attributes

| Attribute                        | Description                                                                                     |
| --------------------------------- | ------------------------------------------------------------------------------------------------ |
| Parent Structure ID               | Referensi ke Account Structure induk (Holding) yang diwarisi.                                     |
| Structure Tier                    | Holding, Sub Holding, atau Company.                                                                |
| Segment Inheritance Type          | Mandatory Inherited, Optional Inherited, atau Tier-Specific (tidak diwariskan).                   |
| Cross-Segment Validation Rule ID  | Identitas unik aturan validasi lintas segmen.                                                      |
| Rule Expression                   | Definisi logika kombinasi segmen yang diizinkan/dilarang.                                          |
| Combination Exception ID          | Identitas unik pengecualian kombinasi segmen.                                                      |
| Exception Reason                  | Alasan bisnis pengecualian kombinasi.                                                              |
| Structure Snapshot Date           | Tanggal snapshot definisi struktur historis diambil.                                               |
| Mapping Source Structure ID       | Referensi Account Structure asal (entitas yang diakuisisi/lama).                                   |
| Mapping Target Structure ID       | Referensi Account Structure tujuan (Holding/struktur baru).                                        |
| Segment Cross-Reference Value     | Pasangan nilai segmen lama terhadap nilai segmen target.                                           |
| Transition Start Date             | Tanggal mulai periode Parallel Structure Coexistence.                                              |
| Transition End Date               | Tanggal berakhir periode Parallel Structure Coexistence.                                           |
| Conflict Resolution Status        | Identified, Under Review, Resolved, Approved.                                                      |
| Onboarding Entity Reference       | Referensi entitas baru yang sedang menjalani proses onboarding struktur akun.                       |
| Created By / Created Date         | Informasi pembuatan data.                                                                          |
| Updated By / Updated Date         | Informasi perubahan terakhir.                                                                      |

---

# 9.6.3.26.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                          |
| -------------- | ------------------- | ----------- | ---------------------------------------- |
| BR-836         | BO-020              | SH-003      | Hierarchical Structure Governance         |
| BR-837         | BO-004              | SH-003      | Cross-Segment Validation Configuration    |
| BR-838         | BO-004              | SH-003      | Combination Exception Management          |
| BR-839         | BO-028              | SH-010      | Historical Structure Reconstruction       |
| BR-840         | BO-009              | SH-003      | Structure Change Impact Analysis          |
| BR-841         | BO-020              | SH-003      | M&A Entity Onboarding                     |
| BR-842         | BO-020              | SH-003      | Legacy Segment Cross-Reference Mapping    |
| BR-843         | BO-022              | SH-010      | Structure Migration & Reconciliation      |
| BR-844         | BO-020              | SH-003      | Merger Structure Conflict Resolution      |
| BR-845         | BO-028              | SH-010      | Segment Combination Governance Reporting  |

---

# 9.6.3.26.7 Summary

Dokumen ini memperdalam Account Structure Requirements dasar (9.6.3.1) dengan menambahkan kapabilitas yang dibutuhkan grup perusahaan kompleks: pewarisan struktur berjenjang Holding–Sub Holding–Company, validasi lintas segmen, preservasi struktur historis tanpa mengubah data transaksi lama, serta penanganan skenario merger dan akuisisi anak perusahaan baru melalui templat onboarding, pemetaan segmen silang, dan alur resolusi konflik struktur.

Dengan sepuluh kebutuhan bisnis tambahan (BR-836 s.d. BR-845) dan tujuh aturan bisnis (BRule-ASTD-001 s.d. BRule-ASTD-007), organisasi memperoleh kerangka kerja yang memastikan fleksibilitas struktur akun pada level operasional tetap kompatibel dengan kebutuhan konsolidasi Holding, sekaligus menjaga integritas data historis selama proses transformasi korporasi seperti merger dan akuisisi.
