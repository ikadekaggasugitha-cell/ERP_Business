# 9.6.3.9 Financial Dimension Requirements

---

# 9.6.3.9.1 Overview

Financial Dimension merupakan atribut analitik tambahan di luar segmen wajib Chart of Accounts (COA), seperti Project, Department, Region, atau Product Line, yang digunakan untuk mendukung pelaporan multidimensi tanpa mengubah struktur akun utama. Financial Dimension memungkinkan organisasi melakukan analisis kinerja keuangan secara fleksibel berdasarkan berbagai sudut pandang bisnis, sejalan dengan prinsip Configuration over Customization dan Scope-Based Everything yang dianut oleh platform ERP.

---

# 9.6.3.9.2 Business Capability

Financial Dimension menyediakan kapabilitas berikut:

* Dimension Definition dan Configuration.
* Dimension Value Master Data Management.
* Mandatory dan Optional Dimension per Scope Organisasi.
* Dimension Combination Validation.
* Multi Level Dimension Hierarchy.
* Dimension Mapping ke Segmen COA.
* Dimension-Based Financial Reporting.
* Default Dimension Rule per Transaction Type.
* Effective Dating dan Versioning Dimension Value.
* Approval Workflow untuk perubahan Dimension.
* Scope-Based Dimension Access Control.
* Audit Trail atas seluruh perubahan Dimension.

---

# 9.6.3.9.3 Business Requirement Catalogue

## BR-386 – Financial Dimension Definition

| Attribute           | Description                                                                                                                                                       |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID      | BR-386                                                                                                                                                            |
| Requirement Name    | Financial Dimension Definition                                                                                                                                    |
| Category            | Financial Dimension                                                                                                                                               |
| Priority            | Critical                                                                                                                                                          |
| Business Objective  | BO-008, BO-014                                                                                                                                                    |
| Stakeholder Owner   | SH-003 (CFO)                                                                                                                                                      |
| Description         | Sistem harus menyediakan Financial Dimension sebagai objek bisnis analitik tambahan di luar segmen wajib COA, yang dapat dikonfigurasi sesuai kebutuhan pelaporan organisasi. |
| Business Rationale  | Organisasi membutuhkan atribut analitik yang fleksibel untuk menganalisis kinerja keuangan tanpa harus mengubah struktur akun utama.                              |
| Business Value      | Menyediakan fondasi pelaporan multidimensi yang konsisten dan real-time bagi seluruh entitas organisasi.                                                          |
| Acceptance Criteria | Administrator dapat membuat, mengubah, dan mengelola Financial Dimension beserta nilainya sesuai kewenangan.                                                      |

---

## BR-387 – Dimension Value Master Data Management

| Attribute       | Description                                                                                                        |
| ---------------- | ------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-387                                                                                                              |
| Requirement Name  | Dimension Value Master Data Management                                                                              |
| Priority          | Critical                                                                                                            |
| Description       | Sistem harus menyediakan pengelolaan Dimension Value sebagai master data yang dapat ditambahkan, diubah, dinonaktifkan, dan dipetakan ke berbagai Financial Dimension. |
| Business Value    | Menjamin konsistensi dan akurasi nilai dimensi yang digunakan di seluruh transaksi keuangan.                       |

---

## BR-388 – Mandatory dan Optional Dimension per Scope

| Attribute       | Description                                                                                                                          |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-388                                                                                                                                |
| Requirement Name  | Mandatory dan Optional Dimension per Scope                                                                                           |
| Priority          | Critical                                                                                                                              |
| Description       | Sistem harus mendukung konfigurasi Financial Dimension sebagai mandatory atau optional berdasarkan Scope Organisasi (Holding, Sub Holding, Company, Business Unit, Branch, Department, Warehouse, Project Site). |
| Acceptance Criteria | Setiap Scope Organisasi dapat mengatur kombinasi Dimension yang wajib maupun opsional tanpa perubahan source code.                 |

---

## BR-389 – Dimension Combination Validation

| Attribute       | Description                                                                                                            |
| ---------------- | -------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-389                                                                                                                |
| Requirement Name  | Dimension Combination Validation                                                                                     |
| Priority          | High                                                                                                                  |
| Description       | Sistem harus melakukan validasi kombinasi nilai Financial Dimension pada saat input transaksi untuk mencegah kombinasi yang tidak valid atau tidak diizinkan. |
| Business Value    | Mengurangi kesalahan input transaksi dan menjaga integritas data pelaporan multidimensi.                             |

---

## BR-390 – Financial Dimension Hierarchy

| Attribute       | Description                                                                                                   |
| ---------------- | -------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-390                                                                                                         |
| Requirement Name  | Financial Dimension Hierarchy                                                                                  |
| Priority          | High                                                                                                           |
| Description       | Sistem harus mendukung struktur hierarki bertingkat untuk nilai Dimension tertentu (misalnya Region atau Product Line) guna keperluan agregasi dan roll-up pelaporan. |
| Business Value    | Memungkinkan analisis keuangan dari level detail hingga level konsolidasi sesuai kebutuhan manajemen.         |

---

## BR-391 – Dimension Mapping ke Segmen COA

| Attribute       | Description                                                                                                                        |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-391                                                                                                                            |
| Requirement Name  | Dimension Mapping ke Segmen COA                                                                                                   |
| Priority          | High                                                                                                                              |
| Description       | Sistem harus menyediakan mekanisme pemetaan antara Financial Dimension dengan segmen Chart of Accounts guna mendukung integrasi pelaporan keuangan dan analitik. |
| Acceptance Criteria | Pemetaan Dimension ke Segmen COA dapat dikonfigurasi dan diaudit oleh Administrator.                                          |

---

## BR-392 – Default Dimension Rule per Transaction Type

| Attribute       | Description                                                                                                          |
| ---------------- | ---------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-392                                                                                                             |
| Requirement Name  | Default Dimension Rule per Transaction Type                                                                       |
| Priority          | Medium                                                                                                             |
| Description       | Sistem harus mendukung konfigurasi aturan default nilai Financial Dimension berdasarkan jenis transaksi tertentu untuk mempercepat proses input. |
| Business Value    | Mempercepat proses transaksi dan mengurangi risiko kesalahan input dimensi secara manual.                         |

---

## BR-393 – Dimension-Based Financial Reporting

| Attribute       | Description                                                                                                              |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-393                                                                                                                  |
| Requirement Name  | Dimension-Based Financial Reporting                                                                                    |
| Priority          | Critical                                                                                                               |
| Description       | Sistem harus mendukung penyusunan laporan keuangan dan analitik berdasarkan kombinasi satu atau lebih Financial Dimension secara real-time. |
| Business Value    | Mendukung pengambilan keputusan berbasis data real-time dari berbagai sudut pandang analitik bisnis.                  |

---

## BR-394 – Effective Dating dan Versioning Dimension Value

| Attribute       | Description                                                                                                       |
| ---------------- | ------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-394                                                                                                         |
| Requirement Name  | Effective Dating dan Versioning Dimension Value                                                                |
| Priority          | Medium                                                                                                         |
| Description       | Sistem harus mendukung periode berlaku dan versi atas setiap Dimension Value untuk menjaga keakuratan historis pelaporan. |
| Acceptance Criteria | Perubahan Dimension Value tidak menghapus riwayat versi sebelumnya dan tetap dapat ditelusuri.                |

---

## BR-395 – Approval Workflow dan Audit Trail Financial Dimension

| Attribute       | Description                                                                                                              |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-395                                                                                                                  |
| Requirement Name  | Approval Workflow dan Audit Trail Financial Dimension                                                                  |
| Priority          | High                                                                                                                    |
| Description       | Sistem harus menyediakan alur persetujuan (approval workflow) yang configurable serta audit trail atas setiap perubahan definisi dan nilai Financial Dimension. |
| Business Value    | Menjamin tata kelola, transparansi, dan integritas data atas pengelolaan Financial Dimension.                          |

---

# 9.6.3.9.4 Business Rules

| Business Rule ID | Description                                                                                                                  |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| BRule-FDIM-001    | Setiap Financial Dimension harus memiliki kode dan nama yang unik dalam ruang lingkup organisasi yang sama.                 |
| BRule-FDIM-002    | Nilai Financial Dimension yang berstatus Inactive tidak dapat digunakan pada transaksi baru namun tetap tampil pada laporan historis. |
| BRule-FDIM-003    | Kombinasi Financial Dimension yang bersifat mandatory pada suatu Scope Organisasi wajib diisi sebelum transaksi dapat diposting. |
| BRule-FDIM-004    | Perubahan status Approval Workflow atas Financial Dimension harus tercatat dalam Audit Trail beserta identitas pengguna dan waktu perubahan. |
| BRule-FDIM-005    | Hierarki Financial Dimension tidak boleh membentuk referensi siklikal (circular reference).                                |
| BRule-FDIM-006    | Dimension Value yang telah digunakan dalam transaksi tidak dapat dihapus, hanya dapat dinonaktifkan.                        |
| BRule-FDIM-007    | Pemetaan Financial Dimension ke Segmen COA harus konsisten dengan struktur Chart of Accounts yang berlaku pada Scope terkait. |

---

# 9.6.3.9.5 Data Attributes

| Attribute                  | Description                                                     |
| --------------------------- | ---------------------------------------------------------------- |
| Dimension ID                | Identitas unik sistem untuk Financial Dimension.                |
| Dimension Code              | Kode Financial Dimension.                                       |
| Dimension Name              | Nama Financial Dimension.                                       |
| Dimension Type              | Klasifikasi dimensi, misalnya Project, Department, Region, Product Line. |
| Mandatory Flag              | Penanda apakah Dimension bersifat wajib pada Scope tertentu.     |
| Dimension Value Code / Name | Kode dan nama nilai Dimension.                                   |
| Parent Dimension Value      | Referensi hierarki nilai Dimension induk.                        |
| Scope Organisasi            | Scope penerapan Dimension (Holding, Company, Business Unit, dst). |
| Effective From / To         | Periode berlaku nilai Dimension.                                 |
| Version                     | Nomor versi Dimension Value.                                     |
| Approval Status             | Draft, Pending Approval, Approved, Published, Archived.          |
| Status                      | Active, Suspended, Inactive, Archived.                           |
| Created By / Created Date   | Informasi pembuatan data.                                        |
| Updated By / Updated Date   | Informasi perubahan terakhir.                                    |

---

# 9.6.3.9.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                         |
| -------------- | ------------------- | ----------- | ---------------------------------------- |
| BR-386         | BO-008, BO-014       | SH-003      | Financial Dimension Administration      |
| BR-387         | BO-008               | SH-003      | Dimension Value Master Data Management  |
| BR-388         | BO-008               | SH-003      | Scope-Based Dimension Configuration     |
| BR-389         | BO-014               | SH-003      | Transaction Input Validation            |
| BR-390         | BO-008               | SH-003      | Dimension Hierarchy Management          |
| BR-391         | BO-014               | SH-003      | COA-Dimension Mapping Administration    |
| BR-392         | BO-008               | SH-003      | Default Dimension Rule Configuration    |
| BR-393         | BO-008               | SH-003      | Multidimensional Financial Reporting    |
| BR-394         | BO-014               | SH-003      | Dimension Value Versioning              |
| BR-395         | BO-008               | SH-003      | Dimension Approval and Audit Governance |

---

# 9.6.3.9.7 Summary

Financial Dimension menyediakan atribut analitik tambahan di luar segmen wajib Chart of Accounts yang memungkinkan pelaporan multidimensi secara fleksibel tanpa mengubah struktur akun utama. Financial Dimension menjadi fondasi bagi kebutuhan analisis kinerja keuangan berbasis data real-time serta menjadi elemen pendukung requirement lain yang terkait pada Financial Management Domain.
