# 9.6.3.29 Account Class Detail Requirements

---

# 9.6.3.29.1 Overview

Dokumen ini merupakan pendalaman (*deep-dive*) atas **Account Class** yang telah didefinisikan pada dokumen ringkas *Chart of Accounts / Account Class* (9.6.3.5, BR-341 s/d BR-350/352). Dokumen tersebut mengatur definisi, klasifikasi standar, dan tata kelola Account Class sebagai objek bisnis. Dokumen ini **tidak mengulang** ketentuan tersebut, melainkan memperdalam empat area teknis yang memiliki dampak langsung terhadap konsolidasi grup, kepatuhan regulasi, dan analisis manajerial lintas entitas, yaitu:

1. **Sub-klasifikasi lanjutan** di bawah Account Class untuk kebutuhan analisis manajerial yang lebih granular tanpa mengubah Financial Statement Mapping.
2. **Aturan konsolidasi laporan per Account Class**, termasuk penentuan metode konsolidasi (Full, Proportionate, Equity) dan eliminasi saldo intercompany.
3. **Pemetaan Account Class ke kebutuhan pelaporan regulator**, khususnya Otoritas Jasa Keuangan (OJK) bagi entitas yang diawasi, dan pelaporan perpajakan (SPT Tahunan/Masa).
4. **Penggunaan Account Class dalam analytical reporting lintas entitas**, sebagai dimensi analisis manajemen di luar struktur laporan keuangan statutori.

Sebagai holding dengan beberapa Legal Entity dan lini bisnis, kelompok usaha memerlukan Account Class yang tidak hanya menjadi label klasifikasi statis, tetapi juga menjadi penggerak (*driver*) keputusan konsolidasi, kepatuhan pelaporan, dan pelaporan manajemen yang konsisten di seluruh organisasi.

---

# 9.6.3.29.2 Business Capability

Account Class Detail menyediakan kapabilitas lanjutan berikut:

* Analytical Sub-Class Definition di bawah Account Class.
* Managerial Sub-Class Assignment pada Account Master.
* Consolidation Method Determination berbasis Account Class dan struktur kepemilikan.
* Intercompany Elimination Rule per Account Class.
* Minority Interest / Non-Controlling Interest Classification.
* Regulatory Reporting Mapping (OJK) per Account Class.
* Tax Reporting Mapping (SPT Tahunan/Masa) per Account Class.
* Regulatory Taxonomy Version Control.
* Cross-Entity Analytical Reporting Dimension.
* Account Class-Based Management Reporting Pack.
* Reconciliation antara Mapping Regulasi dan Saldo Buku Besar.

---

# 9.6.3.29.3 Business Requirement Catalogue

## BR-866 – Analytical Sub-Class Definition

| Attribute           | Description                                                                                                                                                                                                                       |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-866                                                                                                                                                                                                                           |
| Requirement Name    | Analytical Sub-Class Definition                                                                                                                                                                                                  |
| Category            | Account Class Detail                                                                                                                                                                                                            |
| Priority            | High                                                                                                                                                                                                                             |
| Business Objective  | BO-009                                                                                                                                                                                                                           |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                                                                                 |
| Description         | Sistem harus menyediakan **Analytical Sub-Class** sebagai lapisan klasifikasi opsional di bawah Account Class, digunakan untuk kebutuhan analisis manajerial (misalnya memecah Operating Expense menjadi Personnel Cost, Marketing Cost, IT Cost) tanpa mengubah Financial Statement Mapping induknya. |
| Business Rationale  | Kebutuhan manajemen untuk melihat detail biaya/pendapatan sering lebih granular daripada struktur laporan keuangan statutori; tanpa sub-klasifikasi, granularitas ini terpaksa dipaksakan pada level Account Class sehingga struktur laporan keuangan menjadi terlalu rinci dan sulit dipelihara. |
| Business Value      | Memberikan fleksibilitas analisis manajerial tanpa mengorbankan kesederhanaan struktur pelaporan statutori.                                                                                                                      |
| Acceptance Criteria | Administrator dapat membuat, mengubah, dan menonaktifkan Analytical Sub-Class di bawah Account Class tertentu; setiap Account Master dapat memiliki nol atau satu Analytical Sub-Class aktif.                                    |

---

## BR-867 – Managerial Sub-Class Assignment

| Attribute        | Description                                                                                                                                                                     |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-867                                                                                                                                                                         |
| Requirement Name | Managerial Sub-Class Assignment                                                                                                                                               |
| Priority         | Medium                                                                                                                                                                        |
| Description      | Sistem harus memungkinkan Analytical Sub-Class ditetapkan pada Account Master secara terpisah dari Financial Statement Mapping, sehingga perubahan sub-klasifikasi manajerial tidak memerlukan persetujuan pada level statutori. |

---

## BR-868 – Consolidation Method Determination

| Attribute        | Description                                                                                                                                                                                                     |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-868                                                                                                                                                                                                         |
| Requirement Name | Consolidation Method Determination                                                                                                                                                                             |
| Priority         | Critical                                                                                                                                                                                                       |
| Description      | Sistem harus menentukan metode konsolidasi (Full Consolidation, Proportionate Consolidation, Equity Method) berdasarkan kombinasi Account Class dan persentase kepemilikan Legal Entity terhadap entitas anak, sesuai kebijakan konsolidasi grup. |

---

## BR-869 – Intercompany Elimination Rule per Account Class

| Attribute        | Description                                                                                                                                                                             |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-869                                                                                                                                                                                  |
| Requirement Name | Intercompany Elimination Rule per Account Class                                                                                                                                        |
| Priority         | Critical                                                                                                                                                                                |
| Description      | Account Class yang ditandai sebagai Intercompany (misalnya Piutang/Utang Antar Perusahaan, Pendapatan/Beban Antar Perusahaan) harus memiliki aturan eliminasi otomatis yang diterapkan pada proses konsolidasi laporan keuangan grup. |

---

## BR-870 – Minority Interest Classification Rule

| Attribute        | Description                                                                                                                                                             |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-870                                                                                                                                                                 |
| Requirement Name | Minority Interest Classification Rule                                                                                                                                 |
| Priority         | High                                                                                                                                                                   |
| Description      | Account Class Ekuitas dan Laba Rugi harus mendukung penandaan Kepentingan Non-Pengendali (Non-Controlling Interest) agar proses konsolidasi dapat memisahkan porsi kepemilikan induk dan pihak minoritas secara otomatis. |

---

## BR-871 – Regulatory Reporting Mapping (OJK)

| Attribute        | Description                                                                                                                                                                                        |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-871                                                                                                                                                                                            |
| Requirement Name | Regulatory Reporting Mapping (OJK)                                                                                                                                                                |
| Priority         | Critical                                                                                                                                                                                          |
| Description      | Bagi entitas dalam grup yang diawasi Otoritas Jasa Keuangan (misalnya entitas pembiayaan atau sekuritas), Account Class harus dapat dipetakan ke pos pelaporan regulator sesuai taksonomi pelaporan OJK yang berlaku, termasuk mekanisme validasi saat taksonomi diperbarui. |

---

## BR-872 – Tax Reporting Mapping

| Attribute        | Description                                                                                                                                                                             |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-872                                                                                                                                                                                 |
| Requirement Name | Tax Reporting Mapping                                                                                                                                                                 |
| Priority         | High                                                                                                                                                                                   |
| Description      | Account Class harus dapat dipetakan ke pos pelaporan pajak (misalnya kelompok Beban yang Dapat Dikurangkan/Non-Dapat Dikurangkan pada rekonsiliasi fiskal, atau kelompok Objek Pajak Pertambahan Nilai) untuk mendukung penyusunan SPT Tahunan dan SPT Masa. |

---

## BR-873 – Cross-Entity Analytical Reporting Dimension

| Attribute        | Description                                                                                                                                                                       |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-873                                                                                                                                                                           |
| Requirement Name | Cross-Entity Analytical Reporting Dimension                                                                                                                                     |
| Priority         | High                                                                                                                                                                             |
| Description      | Account Class harus dapat digunakan sebagai dimensi analisis lintas Legal Entity pada analytical reporting (misalnya perbandingan komposisi Operating Expense antar anak perusahaan) tanpa memerlukan pemetaan ulang Chart of Accounts masing-masing entitas. |

---

## BR-874 – Account Class-Based Management Reporting Pack

| Attribute        | Description                                                                                                                                                              |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-874                                                                                                                                                                  |
| Requirement Name | Account Class-Based Management Reporting Pack                                                                                                                          |
| Priority         | Medium                                                                                                                                                                  |
| Description      | Sistem harus menyediakan paket pelaporan manajemen standar yang disusun berdasarkan Account Class dan Analytical Sub-Class, dapat digunakan lintas entitas untuk kebutuhan konsolidasi manajerial (management consolidation) di luar laporan statutori. |

---

## BR-875 – Regulatory Mapping Version Control & Reconciliation

| Attribute        | Description                                                                                                                                                                                     |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-875                                                                                                                                                                                         |
| Requirement Name | Regulatory Mapping Version Control & Reconciliation                                                                                                                                           |
| Priority         | Critical                                                                                                                                                                                       |
| Description      | Setiap perubahan pemetaan Account Class ke pelaporan regulator maupun pajak harus dikelola melalui versioning, dengan mekanisme rekonsiliasi otomatis antara saldo Account Class pada buku besar dan nilai yang dilaporkan ke regulator/otoritas pajak. |

---

# 9.6.3.29.4 Business Rules

| Business Rule ID | Description                                                                                                                                                                       |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-ACLD-001    | Setiap Analytical Sub-Class wajib dikaitkan dengan tepat satu Account Class induk dan tidak boleh mengubah Financial Statement Mapping Account Class tersebut.                    |
| BRule-ACLD-002    | Metode konsolidasi (Full, Proportionate, Equity) ditentukan oleh kombinasi Account Class dan persentase kepemilikan; perubahan persentase kepemilikan memicu evaluasi ulang metode konsolidasi. |
| BRule-ACLD-003    | Account Class yang ditandai Intercompany wajib memiliki pasangan akun lawan (counterpart account) pada entitas terkait sebelum proses eliminasi konsolidasi dapat dijalankan.     |
| BRule-ACLD-004    | Pemetaan Account Class ke pelaporan regulator (OJK) harus divalidasi terhadap versi taksonomi yang berlaku pada periode pelaporan yang bersangkutan.                              |
| BRule-ACLD-005    | Pemetaan Account Class ke pelaporan pajak harus disesuaikan mengikuti perubahan peraturan perpajakan yang berlaku efektif, dan tidak boleh diterapkan retroaktif pada periode yang telah dilaporkan. |
| BRule-ACLD-006    | Setiap perubahan mapping regulasi/pajak wajib melalui proses rekonsiliasi terhadap saldo Account Class pada buku besar sebelum dipublikasikan.                                     |
| BRule-ACLD-007    | Seluruh perubahan Analytical Sub-Class, metode konsolidasi, dan mapping regulasi/pajak harus tercatat lengkap dalam Audit Trail.                                                   |

---

# 9.6.3.29.5 Data Attributes

| Attribute                          | Description                                                                                     |
| ------------------------------------ | --------------------------------------------------------------------------------------------------- |
| Analytical Sub-Class Code           | Kode sub-klasifikasi analitis di bawah Account Class.                                                |
| Analytical Sub-Class Name           | Nama sub-klasifikasi analitis.                                                                       |
| Consolidation Method                | Full Consolidation, Proportionate Consolidation, Equity Method, None.                                |
| Ownership Threshold Reference       | Referensi ke ambang batas kepemilikan yang menentukan metode konsolidasi.                             |
| Intercompany Elimination Flag       | Menandakan apakah Account Class memerlukan eliminasi intercompany.                                    |
| Counterpart Account Class Reference | Referensi ke Account Class lawan pada entitas terkait untuk eliminasi.                                |
| Minority Interest Treatment         | Aturan perlakuan Kepentingan Non-Pengendali (jika berlaku).                                           |
| Regulatory Reporting Code (OJK)     | Kode pos pelaporan sesuai taksonomi OJK.                                                              |
| Regulatory Taxonomy Version         | Versi taksonomi pelaporan regulator yang berlaku.                                                     |
| Tax Reporting Code                  | Kode pos pelaporan pajak (rekonsiliasi fiskal/SPT).                                                   |
| Tax Regulation Effective Version    | Versi peraturan perpajakan yang berlaku efektif untuk mapping.                                        |
| Cross-Entity Analytical Dimension   | Referensi dimensi analisis lintas entitas.                                                            |
| Management Reporting Pack Reference | Referensi ke paket pelaporan manajemen yang menggunakan Account Class ini.                             |
| Reconciliation Status               | Status rekonsiliasi antara saldo buku besar dan nilai pelaporan regulator/pajak (Match/Mismatch/Pending). |

---

# 9.6.3.29.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                                |
| -------------- | ------------------- | ----------- | ------------------------------------------------ |
| BR-866         | BO-009              | SH-003      | Analytical Sub-Class Administration              |
| BR-867         | BO-009              | SH-003      | Managerial Classification Assignment             |
| BR-868         | BO-020              | SH-003      | Consolidation Method Governance                  |
| BR-869         | BO-020              | SH-003      | Intercompany Elimination Processing              |
| BR-870         | BO-020              | SH-003      | Minority Interest Consolidation                  |
| BR-871         | BO-022              | SH-010      | Regulatory Reporting (OJK) Compliance             |
| BR-872         | BO-022              | SH-010      | Tax Reporting Compliance                          |
| BR-873         | BO-004              | SH-003      | Cross-Entity Analytical Reporting                |
| BR-874         | BO-004              | SH-003      | Management Reporting Pack Preparation             |
| BR-875         | BO-028              | SH-010      | Regulatory Mapping Reconciliation & Audit         |

---

# 9.6.3.29.7 Summary

Dokumen ini memperdalam Account Class dari sekadar klasifikasi operasional menjadi penggerak keputusan konsolidasi grup, kepatuhan pelaporan regulator dan pajak, serta analisis manajerial lintas entitas. Dengan Analytical Sub-Class, organisasi memperoleh granularitas analisis yang dibutuhkan manajemen tanpa mengganggu kesederhanaan struktur laporan keuangan statutori.

Kapabilitas Consolidation Method Determination, Intercompany Elimination Rule, Regulatory Reporting Mapping, dan Regulatory Mapping Version Control & Reconciliation menjadikan Account Class sebagai titik kendali tunggal yang menghubungkan **Chart of Accounts**, **Financial Consolidation**, **Regulatory Compliance (OJK)**, **Tax Reporting**, dan **Cross-Entity Management Reporting** secara konsisten di seluruh organisasi multi-entitas.
