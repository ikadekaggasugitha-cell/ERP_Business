# 9.6.3.24 Tax Mapping Requirements

---

# 9.6.3.24.1 Overview

Tax Mapping mendefinisikan pemetaan antara kode/jenis pajak dengan akun Chart of Accounts yang sesuai, sehingga pencatatan pajak keluaran (output tax), pajak masukan (input tax), dan pajak pemotongan/pemungutan (withholding tax) dapat dilakukan secara otomatis, konsisten, dan dapat ditelusuri pada setiap transaksi keuangan lintas organisasi.

---

# 9.6.3.24.2 Business Capability

Tax Mapping menyediakan kapabilitas berikut:

* Tax Code & Tax Type Master Data.
* Output Tax (PPN Keluaran) Account Mapping.
* Input Tax (PPN Masukan) Account Mapping.
* Withholding Tax (PPh) Account Mapping.
* Transaction-Type Based Tax Mapping.
* Scope-Based Tax Mapping Configuration.
* Multi Tax Rate & Effective Dating.
* Automatic Tax Posting ke General Ledger.
* Multi-Organization & Multi-Currency Tax Support.
* Approval Workflow.
* Versioning.
* Audit Trail.

---

# 9.6.3.24.3 Business Requirement Catalogue

## BR-536 – Tax Code and Account Mapping Definition

| Attribute           | Description                                                                                                                                                     |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-536                                                                                                                                                          |
| Requirement Name    | Tax Code and Account Mapping Definition                                                                                                                        |
| Category            | Tax Mapping                                                                                                                                                     |
| Priority            | Critical                                                                                                                                                        |
| Business Objective  | BO-007                                                                                                                                                          |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                |
| Description         | Sistem harus menyediakan Tax Mapping sebagai objek bisnis untuk memetakan setiap Tax Code/Tax Type ke akun Chart of Accounts yang sesuai secara terpusat.       |
| Business Rationale  | Pemetaan pajak yang tidak konsisten menimbulkan risiko salah posting, ketidaksesuaian pelaporan pajak, dan potensi temuan audit.                                |
| Business Value      | Menjamin konsistensi pencatatan pajak keluaran dan masukan pada seluruh transaksi keuangan.                                                                     |
| Acceptance Criteria | Administrator dapat membuat, mengubah, dan mengelola Tax Mapping antara Tax Code dan akun Chart of Accounts sesuai kewenangan.                                  |

---

## BR-537 – Tax Type Classification (Output/Input/Withholding)

| Attribute        | Description                                                                                                                            |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-537                                                                                                                                  |
| Requirement Name | Tax Type Classification (Output/Input/Withholding)                                                                                      |
| Priority         | Critical                                                                                                                                |
| Description      | Sistem harus mengklasifikasikan Tax Code ke dalam jenis Output Tax, Input Tax, atau Withholding Tax sebagai dasar penentuan akun mapping. |
| Business Value   | Memastikan setiap jenis pajak diarahkan ke akun yang tepat sesuai sifat kewajiban perpajakannya.                                        |

---

## BR-538 – Output Tax Account Mapping

| Attribute        | Description                                                                                                                    |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-538                                                                                                                        |
| Requirement Name | Output Tax Account Mapping                                                                                                    |
| Priority         | Critical                                                                                                                       |
| Description      | Sistem harus memetakan Tax Code kategori Output Tax (misal PPN Keluaran) ke akun kewajiban pajak (tax payable) pada Chart of Accounts. |
| Acceptance Criteria | Transaksi penjualan yang mengandung pajak keluaran otomatis mem-posting nilai pajak ke akun mapping yang telah ditentukan. |

---

## BR-539 – Input Tax Account Mapping

| Attribute        | Description                                                                                                                   |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-539                                                                                                                       |
| Requirement Name | Input Tax Account Mapping                                                                                                    |
| Priority         | Critical                                                                                                                      |
| Description      | Sistem harus memetakan Tax Code kategori Input Tax (misal PPN Masukan) ke akun piutang pajak (tax receivable) pada Chart of Accounts. |
| Acceptance Criteria | Transaksi pembelian yang mengandung pajak masukan otomatis mem-posting nilai pajak ke akun mapping yang telah ditentukan. |

---

## BR-540 – Withholding Tax (PPh) Account Mapping

| Attribute        | Description                                                                                                                       |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-540                                                                                                                            |
| Requirement Name | Withholding Tax (PPh) Account Mapping                                                                                             |
| Priority         | High                                                                                                                               |
| Description      | Sistem harus memetakan setiap jenis pajak penghasilan yang dipotong/dipungut (PPh Pasal 21/23/4(2), dsb.) ke akun terkait pada Chart of Accounts. |
| Business Value   | Memastikan kewajiban pemotongan/pemungutan pajak penghasilan tercatat secara akurat dan sesuai jenisnya.                            |

---

## BR-541 – Transaction-Type Based Tax Mapping

| Attribute        | Description                                                                                                                                        |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-541                                                                                                                                             |
| Requirement Name | Transaction-Type Based Tax Mapping                                                                                                                 |
| Priority         | High                                                                                                                                                |
| Description      | Sistem harus mendukung pemetaan pajak berbeda berdasarkan jenis transaksi (Sales, Purchase, Import, Export, Asset Disposal, dan sejenisnya).       |
| Business Value   | Memungkinkan penerapan aturan pajak yang berbeda sesuai karakteristik transaksi tanpa perubahan source code.                                       |

---

## BR-542 – Scope-Based Tax Mapping Configuration

| Attribute        | Description                                                                                                                                                  |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-542                                                                                                                                                       |
| Requirement Name | Scope-Based Tax Mapping Configuration                                                                                                                        |
| Priority         | High                                                                                                                                                          |
| Description      | Sistem harus mendukung konfigurasi Tax Mapping yang berbeda pada level organisasi tertentu (Company, Business Unit, Branch) sesuai kebutuhan lokal/regulasi. |
| Acceptance Criteria | Setiap entitas organisasi dapat memiliki Tax Mapping tersendiri atau mewarisi (inherit) dari level organisasi di atasnya. |

---

## BR-543 – Multi Tax Rate and Effective Dating

| Attribute        | Description                                                                                                                        |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-543                                                                                                                             |
| Requirement Name | Multi Tax Rate and Effective Dating                                                                                                |
| Priority         | High                                                                                                                               |
| Description      | Sistem harus mendukung penerapan tarif pajak berbeda berdasarkan periode berlaku (effective dating) mengikuti perubahan regulasi. |
| Business Value   | Menjamin transaksi historis tetap menggunakan tarif pajak yang berlaku pada saat transaksi terjadi.                               |

---

## BR-544 – Automatic Tax Posting Integration

| Attribute        | Description                                                                                                                             |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-544                                                                                                                                   |
| Requirement Name | Automatic Tax Posting Integration                                                                                                        |
| Priority         | Critical                                                                                                                                 |
| Description      | Sistem harus secara otomatis mem-posting nilai pajak ke General Ledger berdasarkan Tax Mapping pada saat transaksi AP/AR/GL diproses.   |
| Acceptance Criteria | Setiap transaksi yang mengandung komponen pajak menghasilkan jurnal otomatis sesuai akun mapping tanpa intervensi manual. |

---

## BR-545 – Tax Mapping Approval Workflow and Audit Trail

| Attribute        | Description                                                                                                                                       |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID   | BR-545                                                                                                                                             |
| Requirement Name | Tax Mapping Approval Workflow and Audit Trail                                                                                                     |
| Priority         | High                                                                                                                                               |
| Description      | Sistem harus menyediakan approval workflow untuk setiap pembuatan/perubahan Tax Mapping serta mencatat seluruh riwayat perubahan dalam audit trail. |
| Business Value   | Meningkatkan tata kelola dan transparansi atas perubahan pemetaan pajak yang berdampak pada kepatuhan perpajakan.                                  |

---

# 9.6.3.24.4 Business Rules

| Business Rule ID | Description                                                                                                                          |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-TXM-001    | Setiap Tax Code harus dipetakan ke minimal satu akun Chart of Accounts sebelum dapat digunakan pada transaksi.                        |
| BRule-TXM-002    | Kombinasi Tax Code, Transaction Type, dan Organization Scope harus unik dalam satu Tax Mapping yang aktif.                            |
| BRule-TXM-003    | Perubahan tarif pajak tunduk pada Effective Dating dan tidak boleh mengubah nilai pajak pada transaksi historis yang sudah diposting. |
| BRule-TXM-004    | Tax Mapping yang telah digunakan dalam transaksi tidak dapat dihapus, hanya dapat dinonaktifkan dan digantikan dengan versi baru.     |
| BRule-TXM-005    | Setiap pembuatan atau perubahan Tax Mapping wajib melalui Approval Workflow sebelum berlaku efektif.                                  |
| BRule-TXM-006    | Sistem harus mencegah adanya lebih dari satu Tax Mapping aktif yang menimbulkan ambiguitas posting pada satu transaksi yang sama.     |
| BRule-TXM-007    | Setiap perubahan Tax Mapping harus tercatat dalam Audit Trail, mencakup pengguna, waktu, dan nilai sebelum/sesudah perubahan.         |

---

# 9.6.3.24.5 Data Attributes

| Attribute                 | Description                                                          |
| -------------------------- | --------------------------------------------------------------------- |
| Tax Mapping ID             | Identitas unik sistem.                                                |
| Tax Code / Tax Type        | Kode dan jenis pajak (Output Tax, Input Tax, Withholding Tax).        |
| Transaction Type           | Jenis transaksi terkait (Sales, Purchase, Import, Export, dsb.).      |
| Mapped Account             | Akun Chart of Accounts tujuan posting pajak.                          |
| Organization Scope         | Cakupan organisasi tempat mapping berlaku.                            |
| Tax Rate                   | Persentase atau nilai tarif pajak yang berlaku.                       |
| Effective From / To        | Periode berlaku mapping dan tarif pajak.                              |
| Version                    | Nomor versi Tax Mapping.                                              |
| Approval Status            | Draft, Pending Approval, Approved, Published, Archived.               |
| Status                     | Active, Suspended, Inactive, Archived.                                |
| Created By / Created Date  | Informasi pembuatan data.                                             |
| Updated By / Updated Date  | Informasi perubahan terakhir.                                         |

---

# 9.6.3.24.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                          |
| -------------- | ------------------- | ----------- | ---------------------------------------- |
| BR-536         | BO-007              | SH-003      | Tax Mapping Administration               |
| BR-537         | BO-007              | SH-011      | Tax Type Classification                  |
| BR-538         | BO-007              | SH-003      | Output Tax Posting                       |
| BR-539         | BO-007              | SH-003      | Input Tax Posting                        |
| BR-540         | BO-007              | SH-011      | Withholding Tax Posting                  |
| BR-541         | BO-007              | SH-011      | Transaction-Type Tax Configuration       |
| BR-542         | BO-007              | SH-003      | Scope-Based Tax Configuration            |
| BR-543         | BO-007              | SH-011      | Tax Rate Maintenance                     |
| BR-544         | BO-007              | SH-003      | Automatic Tax Posting                    |
| BR-545         | BO-007              | SH-011      | Tax Mapping Governance & Audit           |

---

# 9.6.3.24.7 Summary

Tax Mapping menyediakan mekanisme pemetaan antara Tax Code dan akun Chart of Accounts yang menjamin pencatatan pajak keluaran, pajak masukan, dan withholding tax dilakukan secara otomatis, konsisten, dan sesuai regulasi pada seluruh lini organisasi. Tax Mapping menjadi fondasi tata kelola perpajakan yang terintegrasi dengan requirement lain pada Financial Management Domain.
