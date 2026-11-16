# 9.6.3.37 Cost Accounting Account Integration Requirements

---

# 9.6.3.37.1 Overview

Dokumen ini menetapkan persyaratan integrasi antara struktur akun (Chart of Accounts/Account) dengan kebutuhan Cost Accounting, khususnya dari sudut pandang bagaimana Account digunakan, direferensikan, dan dikendalikan pada proses akuntansi biaya. Dokumen ini bukan merupakan requirement umum modul Cost Center maupun Profit Center — hal tersebut telah dibahas secara menyeluruh pada dokumen 9.5.10 (BR-616–BR-625) dan 9.5.11 (BR-626–BR-635). Fokus dokumen ini adalah sisi Account: bagaimana kombinasi Account dengan Cost Center dan Profit Center dikonfigurasi sebagai kombinasi valid (account-dimension combination), bagaimana Account digunakan untuk mencatat alokasi biaya overhead antar unit, bagaimana Account WIP (Work In Progress) direferensikan dalam siklus produksi/proyek, serta bagaimana Account varians biaya (cost variance) dipetakan untuk mengukur selisih antara biaya standar dan biaya aktual.

Chart of Accounts menyediakan fondasi pengelompokan akun (Account Category, Account Class, Account Group, Account Type) yang harus dapat dikombinasikan secara konsisten dengan dimensi Cost Center dan Profit Center pada saat pencatatan biaya, sehingga setiap transaksi cost accounting dapat ditelusuri baik dari sisi akun (apa yang dicatat) maupun dari sisi dimensi (di mana biaya tersebut terjadi). Sejalan dengan prinsip Configuration over Customization, seluruh kombinasi Account-Cost Center, Account-Profit Center, penentuan Account WIP, dan Account varians biaya harus dapat dikonfigurasi tanpa perubahan source code, serta konsisten digunakan sebagai satu sumber kebenaran (single source of truth) oleh seluruh modul yang terkait dengan biaya produksi maupun proyek.

---

# 9.6.3.37.2 Business Capability

- Konfigurasi kombinasi valid (valid combination) antara Account biaya dengan Cost Center dan Profit Center untuk mencegah pencatatan biaya pada kombinasi yang tidak sah.
- Klasifikasi Account biaya berdasarkan sifat biaya (direct cost, indirect cost, overhead) yang terhubung ke Account Category dan Account Class pada Chart of Accounts.
- Penentuan Account alokasi biaya overhead (cost allocation account) sebagai jembatan pencatatan antara Cost Center sumber dan Cost Center/Profit Center tujuan.
- Penentuan Account WIP (Work In Progress) yang merepresentasikan akumulasi biaya produksi atau proyek yang belum selesai pada neraca.
- Mapping Account WIP terhadap Account Finished Goods atau Account Cost of Goods Sold pada saat penyelesaian produksi/proyek (WIP clearing).
- Penentuan Account varians biaya (cost variance account) untuk mencatat selisih antara biaya standar (standard cost) dan biaya aktual (actual cost).
- Segregasi Account varians berdasarkan jenis varians (material variance, labor variance, overhead variance) sebagai sub-akun analitis di bawah Account Class yang relevan.
- Validasi otomatis bahwa Account biaya yang digunakan pada transaksi cost accounting hanya Account berstatus Active dan berada pada Account Group yang diperbolehkan untuk posting biaya.
- Penyediaan mapping Account ke Cost Element sebagai jembatan antara struktur akun keuangan dan struktur elemen biaya pada cost accounting.
- Ketertelusuran (traceability) saldo Account WIP dan Account varians biaya terhadap General Ledger secara real-time sebagai bagian dari single source of truth.
- Dukungan konfigurasi Account biaya lintas entitas dalam struktur multi-organisasi tanpa duplikasi struktur akun antar Company atau Business Unit.
- Penyediaan audit trail atas setiap perubahan mapping Account terhadap Cost Center, Profit Center, WIP, maupun varians biaya untuk kebutuhan kepatuhan dan pengendalian internal.

---

# 9.6.3.37.3 Business Requirement Catalogue

## BR-946 – Account-Cost Center Valid Combination Configuration

| Attribute           | Description                                                                                                                                                                                                        |
| ------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-946                                                                                                                                                                                                             |
| Requirement Name    | Account-Cost Center Valid Combination Configuration                                                                                                                                                               |
| Category            | Cost Accounting Account Integration                                                                                                                                                                               |
| Priority            | Critical                                                                                                                                                                                                           |
| Business Objective  | BO-004, BO-008                                                                                                                                                                                                     |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                                                                  |
| Description         | Sistem harus menyediakan konfigurasi kombinasi valid (valid combination) antara Account biaya dengan Cost Center dan Profit Center, sehingga hanya kombinasi yang telah ditentukan yang dapat digunakan pada posting transaksi biaya. |
| Business Rationale  | Tanpa validasi kombinasi Account-Cost Center, pengguna dapat mencatat biaya pada kombinasi akun dan dimensi yang tidak relevan, sehingga menurunkan keandalan data biaya per unit organisasi.                     |
| Business Value      | Menjamin integritas data cost accounting sebagai satu sumber kebenaran (single source of truth) melalui pencegahan kombinasi Account-dimensi yang tidak sah sejak titik input transaksi.                         |
| Acceptance Criteria | Sistem menolak posting transaksi biaya apabila kombinasi Account dengan Cost Center atau Profit Center yang dipilih tidak terdaftar pada tabel kombinasi valid yang telah dikonfigurasi.                        |

---

## BR-947 – Account Classification for Direct, Indirect & Overhead Cost

| Attribute       | Description                                                                                                                                                        |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-947                                                                                                                                                            |
| Requirement Name| Account Classification for Direct, Indirect & Overhead Cost                                                                                                       |
| Priority        | Critical                                                                                                                                                          |
| Description     | Sistem harus mengklasifikasikan Account biaya ke dalam kategori direct cost, indirect cost, dan overhead cost berdasarkan Account Category dan Account Class.  |

---

## BR-948 – Overhead Cost Allocation Account Mapping

| Attribute       | Description                                                                                                                                                            |
| --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-948                                                                                                                                                              |
| Requirement Name| Overhead Cost Allocation Account Mapping                                                                                                                            |
| Priority        | Critical                                                                                                                                                            |
| Description     | Sistem harus menyediakan Account alokasi biaya overhead yang digunakan sebagai jembatan pencatatan antara Cost Center sumber dan Cost Center/Profit Center tujuan. |

---

## BR-949 – Allocation Account Balancing Validation

| Attribute       | Description                                                                                                                                                |
| --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-949                                                                                                                                                    |
| Requirement Name| Allocation Account Balancing Validation                                                                                                                   |
| Priority        | High                                                                                                                                                       |
| Description     | Sistem harus memvalidasi bahwa nilai debit dan kredit pada Account alokasi biaya overhead selalu seimbang pada setiap proses alokasi antar unit.        |

---

## BR-950 – WIP Account Designation for Production & Project

| Attribute       | Description                                                                                                                                                         |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-950                                                                                                                                                            |
| Requirement Name| WIP Account Designation for Production & Project                                                                                                                  |
| Priority        | Critical                                                                                                                                                          |
| Description     | Sistem harus menyediakan penetapan Account WIP (Work In Progress) yang merepresentasikan akumulasi biaya produksi atau proyek yang belum selesai pada neraca.   |

---

## BR-951 – WIP to Finished Goods/COGS Clearing Mapping

| Attribute       | Description                                                                                                                                                          |
| --------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-951                                                                                                                                                             |
| Requirement Name| WIP to Finished Goods/COGS Clearing Mapping                                                                                                                        |
| Priority        | Critical                                                                                                                                                           |
| Description     | Sistem harus menyediakan mapping Account WIP terhadap Account Finished Goods atau Account Cost of Goods Sold pada saat proses penyelesaian produksi/proyek.      |

---

## BR-952 – Cost Variance Account Designation

| Attribute       | Description                                                                                                                                                          |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-952                                                                                                                                                            |
| Requirement Name| Cost Variance Account Designation                                                                                                                                 |
| Priority        | Critical                                                                                                                                                          |
| Description     | Sistem harus menyediakan Account varians biaya (cost variance account) untuk mencatat selisih antara biaya standar dan biaya aktual pada level Account tertentu. |

---

## BR-953 – Cost Variance Account Segregation by Variance Type

| Attribute       | Description                                                                                                                                                              |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-953                                                                                                                                                                |
| Requirement Name| Cost Variance Account Segregation by Variance Type                                                                                                                    |
| Priority        | High                                                                                                                                                                   |
| Description     | Sistem harus memisahkan Account varians biaya berdasarkan jenis varians (material variance, labor variance, overhead variance) sebagai sub-akun analitis.          |

---

## BR-954 – Account-Cost Element Mapping

| Attribute       | Description                                                                                                                                                     |
| --------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-954                                                                                                                                                       |
| Requirement Name| Account-Cost Element Mapping                                                                                                                                 |
| Priority        | High                                                                                                                                                          |
| Description     | Sistem harus menyediakan mapping antara Account pada Chart of Accounts dengan Cost Element pada modul Cost Accounting sebagai jembatan struktur biaya.     |

---

## BR-955 – Real-Time WIP & Variance Account Reconciliation to GL

| Attribute       | Description                                                                                                                                                       |
| --------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID  | BR-955                                                                                                                                                          |
| Requirement Name| Real-Time WIP & Variance Account Reconciliation to GL                                                                                                          |
| Priority        | Critical                                                                                                                                                        |
| Description     | Sistem harus menyediakan rekonsiliasi saldo Account WIP dan Account varians biaya terhadap General Ledger secara real-time untuk menjamin keakuratan pelaporan. |

---

# 9.6.3.37.4 Business Rules

| Business Rule ID   | Description                                                                                                                                                     |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-CCAI-001      | Setiap transaksi cost accounting wajib menggunakan kombinasi Account, Cost Center, dan Profit Center yang terdaftar pada tabel kombinasi valid.               |
| BRule-CCAI-002      | Account biaya hanya dapat digunakan untuk posting apabila berstatus Active dan berada pada Account Group yang diperbolehkan untuk transaksi biaya.           |
| BRule-CCAI-003      | Account alokasi biaya overhead wajib menghasilkan entri debit dan kredit yang seimbang pada setiap proses alokasi antar Cost Center atau Profit Center.       |
| BRule-CCAI-004      | Saldo Account WIP hanya dapat dipindahkan (clearing) ke Account Finished Goods atau Account Cost of Goods Sold melalui proses penyelesaian yang tervalidasi.  |
| BRule-CCAI-005      | Account varians biaya wajib disegregasi berdasarkan jenis varians (material, labor, overhead) dan tidak dapat digabungkan pada satu Account yang sama.        |
| BRule-CCAI-006      | Perubahan mapping Account terhadap Cost Element, WIP, atau varians biaya tidak boleh mengubah nilai transaksi historis yang telah tercatat pada GL.           |
| BRule-CCAI-007      | Saldo Account WIP dan Account varians biaya wajib terekonsiliasi dengan General Ledger pada setiap penutupan periode fiskal.                                  |

---

# 9.6.3.37.5 Data Attributes

| Attribute Name           | Description                                                                                     | Data Type | Mandatory |
| ------------------------ | ------------------------------------------------------------------------------------------------ | --------- | --------- |
| Account Code             | Kode akun pada Chart of Accounts yang digunakan pada transaksi cost accounting.                  | String    | Yes       |
| Cost Center Code         | Kode Cost Center yang dikombinasikan dengan Account pada transaksi biaya.                        | String    | Yes       |
| Profit Center Code       | Kode Profit Center yang dikombinasikan dengan Account pada transaksi biaya.                      | String    | No        |
| Cost Classification       | Klasifikasi biaya pada Account (Direct, Indirect, Overhead).                                     | Enum      | Yes       |
| Allocation Account Flag   | Penanda bahwa Account merupakan Account alokasi biaya overhead.                                  | Boolean   | Yes       |
| WIP Account Flag          | Penanda bahwa Account merupakan Account Work In Progress.                                        | Boolean   | Yes       |
| Variance Type             | Jenis varians biaya yang dipetakan pada Account (Material, Labor, Overhead).                     | Enum      | No        |
| Cost Element Code         | Kode Cost Element yang dipetakan terhadap Account pada Chart of Accounts.                        | String    | Yes       |
| WIP Clearing Account Code | Kode Account tujuan clearing dari Account WIP (Finished Goods/COGS).                              | String    | Conditional |
| Reconciliation Status     | Status rekonsiliasi saldo Account WIP/varians terhadap General Ledger pada periode tertentu.      | Enum      | Yes       |

---

# 9.6.3.37.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                                       |
| -------------- | ------------------- | ----------- | ----------------------------------------------------- |
| BR-946         | BO-004, BO-008       | SH-003      | Account-Cost Center Valid Combination Setup            |
| BR-947         | BO-004               | SH-003      | Cost Account Classification Configuration              |
| BR-948         | BO-004, BO-008       | SH-003      | Overhead Allocation Account Configuration              |
| BR-949         | BO-004               | SH-003      | Allocation Account Balancing Validation                |
| BR-950         | BO-004, BO-008       | SH-004      | WIP Account Setup for Production & Project             |
| BR-951         | BO-004, BO-008       | SH-004      | WIP to Finished Goods/COGS Clearing Processing         |
| BR-952         | BO-004, BO-008       | SH-003      | Cost Variance Account Setup                            |
| BR-953         | BO-004               | SH-003      | Cost Variance Account Segregation                      |
| BR-954         | BO-004               | SH-003      | Account-Cost Element Mapping Configuration             |
| BR-955         | BO-004, BO-008       | SH-003      | WIP & Variance Account Reconciliation to GL             |

---

# 9.6.3.37.7 Summary

Cost Accounting Account Integration Requirements menetapkan bagaimana struktur akun (Chart of Accounts/Account) diintegrasikan secara spesifik dengan kebutuhan akuntansi biaya, mencakup kombinasi valid Account dengan Cost Center dan Profit Center, penetapan Account alokasi biaya overhead, Account WIP untuk produksi dan proyek, serta Account varians biaya antara standar dan aktual. Dokumen ini melengkapi requirement umum Cost Center (9.5.10) dan Profit Center (9.5.11) dengan sudut pandang Account, memastikan bahwa setiap transaksi biaya tercatat pada kombinasi akun dan dimensi yang sah, tertelusur, dan terekonsiliasi secara real-time terhadap General Ledger sebagai satu sumber kebenaran (single source of truth) bagi seluruh modul cost accounting.
