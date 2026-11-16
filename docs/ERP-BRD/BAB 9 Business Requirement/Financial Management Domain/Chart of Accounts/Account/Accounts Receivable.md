# 9.6.3.33 Accounts Receivable Account Integration Requirements

---

# 9.6.3.33.1 Overview

Accounts Receivable Account Integration Requirements mendefinisikan bagaimana struktur Chart of Accounts (COA) diintegrasikan, direferensikan, dan dipakai secara spesifik oleh domain Accounts Receivable dari sudut pandang Account/COA, bukan dari sudut pandang proses bisnis piutang usaha. Dokumen ini melengkapi requirement umum Accounts Receivable (BR-576 s/d BR-585) dengan menjelaskan bagaimana setiap transaksi piutang — invoice, penerimaan pembayaran, cadangan piutang tak tertagih, uang muka pelanggan, hingga selisih kurs — dipetakan ke akun GL yang tepat sesuai kategori, sifat kontra-aset, dan Scope organisasi.

Fokus dokumen ini adalah pada mekanisme mapping akun (account mapping/derivation) yang menghubungkan entitas non-COA (Customer, Product/Service Category, Currency) dengan Account pada Chart of Accounts, termasuk akun kontrol piutang usaha (AR Control Account), akun cadangan piutang tak tertagih (Allowance for Doubtful Accounts) beserta perhitungan otomatisnya, mapping invoice ke akun pendapatan sesuai kategori produk/jasa, serta akun uang muka pelanggan (Customer Advance). Seluruh mapping akun bersifat konfiguratif per Financial Organization dan Scope, sesuai prinsip Configuration over Customization, sehingga tidak memerlukan perubahan source code ketika kebijakan akuntansi piutang berubah.

---

# 9.6.3.33.2 Business Capability

Accounts Receivable Account Integration menyediakan kapabilitas berikut:

* AR Control Account Mapping per Financial Organization dan Customer Group.
* Allowance for Doubtful Accounts (Contra-Asset) Account Mapping.
* Automated Bad Debt Provision Calculation dan GL Posting Account.
* Revenue Account Derivation berdasarkan Product/Service Category.
* Customer Advance dan Deposit Liability Account Mapping.
* Unbilled/Accrued Receivable Account Mapping.
* Realized dan Unrealized FX Gain/Loss Account Segregation untuk transaksi AR.
* Credit Note dan Debit Note Contra-Revenue Account Mapping.
* Bad Debt Write-Off Expense Account Mapping.
* AR Sub-Ledger to GL Control Account Reconciliation Mapping.
* Effective Dating dan Versioning atas AR Account Mapping.
* Approval Workflow dan Audit Trail atas perubahan AR Account Mapping.

---

# 9.6.3.33.3 Business Requirement Catalogue

## BR-906 – AR Control Account Mapping

| Attribute           | Description                                                                                                                                                                                                          |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-906                                                                                                                                                                                                              |
| Requirement Name    | AR Control Account Mapping                                                                                                                                                                                         |
| Category            | AR Account Integration                                                                                                                                                                                             |
| Priority            | Critical                                                                                                                                                                                                            |
| Business Objective  | BO-004, BO-022                                                                                                                                                                                                      |
| Stakeholder Owner   | SH-003 (CFO), SH-028 (Accounting Department)                                                                                                                                                                       |
| Description         | Sistem harus menyediakan mapping AR Control Account pada Chart of Accounts yang menjadi satu-satunya akun ringkasan (summary account) atas seluruh saldo piutang usaha per Financial Organization dan Customer Group, sehingga setiap invoice, pembayaran, dan penyesuaian piutang wajib diposting melalui akun kontrol ini. |
| Business Rationale  | Tanpa AR Control Account yang terdefinisi secara eksplisit pada COA, saldo subledger piutang tidak memiliki titik rekonsiliasi tunggal terhadap General Ledger, sehingga berisiko menimbulkan selisih yang sulit ditelusuri antara subledger dan buku besar. |
| Business Value      | Menjadi single source of truth atas saldo piutang usaha pada level akun GL, serta menjadi dasar rekonsiliasi otomatis antara AR Sub-Ledger dan General Ledger.                                                    |
| Acceptance Criteria | Setiap Financial Organization/Customer Group wajib memiliki tepat satu AR Control Account aktif; sistem menolak posting transaksi piutang yang tidak memiliki AR Control Account valid; AR Control Account tidak dapat menerima jurnal manual langsung tanpa referensi subledger. |

---

## BR-907 – Allowance for Doubtful Accounts Configuration

| Attribute        | Description                                                                                                                                                                          |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-907                                                                                                                                                                              |
| Requirement Name  | Allowance for Doubtful Accounts Configuration                                                                                                                                        |
| Priority          | Critical                                                                                                                                                                            |
| Description       | Sistem harus menyediakan konfigurasi Allowance for Doubtful Accounts sebagai akun kontra-aset (contra-asset) pada COA yang berpasangan langsung dengan AR Control Account terkait, per Financial Organization. |

---

## BR-908 – Automated Bad Debt Provision Calculation and Posting

| Attribute        | Description                                                                                                                                                                              |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-908                                                                                                                                                                                    |
| Requirement Name  | Automated Bad Debt Provision Calculation and Posting                                                                                                                                     |
| Priority          | Critical                                                                                                                                                                                  |
| Description       | Sistem harus dapat menghitung estimasi cadangan piutang tak tertagih secara otomatis (persentase aging bucket atau metode lain yang dikonfigurasi) dan memposting jurnal penyesuaian ke Allowance for Doubtful Accounts secara periodik. |

---

## BR-909 – Revenue Account Mapping by Product/Service Category

| Attribute        | Description                                                                                                                                                        |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-909                                                                                                                                                              |
| Requirement Name  | Revenue Account Mapping by Product/Service Category                                                                                                                |
| Priority          | Critical                                                                                                                                                            |
| Description       | Sistem harus memetakan setiap baris invoice pelanggan ke akun pendapatan pada COA berdasarkan kategori produk/jasa, sehingga pendapatan tersegmentasi secara akurat pada laporan keuangan. |

---

## BR-910 – Customer Advance and Deposit Account Mapping

| Attribute        | Description                                                                                                                                                    |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Requirement ID    | BR-910                                                                                                                                                          |
| Requirement Name  | Customer Advance and Deposit Account Mapping                                                                                                                    |
| Priority          | Critical                                                                                                                                                        |
| Description       | Sistem harus memetakan penerimaan uang muka pelanggan (customer advance/deposit) ke akun liabilitas khusus pada COA yang terpisah dari AR Control Account, hingga uang muka tersebut dialokasikan ke invoice terkait. |

---

## BR-911 – Unbilled and Accrued Receivable Account Mapping

| Attribute        | Description                                                                                                                                                             |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-911                                                                                                                                                                   |
| Requirement Name  | Unbilled and Accrued Receivable Account Mapping                                                                                                                          |
| Priority          | High                                                                                                                                                                     |
| Description       | Sistem harus menyediakan akun Unbilled/Accrued Receivable pada COA untuk mencatat pengakuan pendapatan yang belum ditagihkan secara formal kepada pelanggan, terpisah dari AR Control Account. |

---

## BR-912 – FX Gain/Loss Account Mapping for Accounts Receivable

| Attribute        | Description                                                                                                                                                            |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-912                                                                                                                                                                 |
| Requirement Name  | FX Gain/Loss Account Mapping for Accounts Receivable                                                                                                                  |
| Priority          | High                                                                                                                                                                   |
| Description       | Sistem harus memetakan selisih kurs piutang dalam mata uang asing ke akun Realized FX Gain/Loss dan Unrealized FX Gain/Loss yang terpisah pada COA, per Financial Organization. |

---

## BR-913 – Credit Note and Debit Note Account Mapping

| Attribute        | Description                                                                                                                                       |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-913                                                                                                                                             |
| Requirement Name  | Credit Note and Debit Note Account Mapping                                                                                                        |
| Priority          | High                                                                                                                                               |
| Description       | Sistem harus memetakan credit note dan debit note pelanggan ke akun kontra-pendapatan (sales return/allowance) atau akun penyesuaian yang sesuai, konsisten dengan akun pendapatan invoice asal. |

---

## BR-914 – Bad Debt Write-Off Expense Account Mapping

| Attribute        | Description                                                                                                                                                       |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-914                                                                                                                                                            |
| Requirement Name  | Bad Debt Write-Off Expense Account Mapping                                                                                                                       |
| Priority          | Medium                                                                                                                                                            |
| Description       | Sistem harus memetakan penghapusan piutang (write-off) ke Allowance for Doubtful Accounts terlebih dahulu, dan hanya membebankan ke akun Bad Debt Expense apabila saldo cadangan tidak mencukupi. |

---

## BR-915 – AR Sub-Ledger to GL Control Account Reconciliation Mapping

| Attribute        | Description                                                                                                                                                 |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID    | BR-915                                                                                                                                                       |
| Requirement Name  | AR Sub-Ledger to GL Control Account Reconciliation Mapping                                                                                                  |
| Priority          | High                                                                                                                                                         |
| Description       | Sistem harus menyediakan mapping dan laporan rekonsiliasi otomatis antara saldo AR Sub-Ledger per pelanggan dengan saldo AR Control Account pada General Ledger, serta menampilkan selisih sebagai exception. |

---

# 9.6.3.33.4 Business Rules

| Business Rule ID | Description                                                                                                                                                       |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-ARAI-001    | Setiap transaksi piutang usaha wajib diposting melalui AR Control Account yang telah dipetakan pada Financial Organization terkait; posting langsung tanpa referensi subledger dilarang. |
| BRule-ARAI-002    | Allowance for Doubtful Accounts wajib berupa akun kontra-aset dan hanya dapat diposting melalui proses provisioning otomatis atau write-off yang disetujui, bukan jurnal manual bebas. |
| BRule-ARAI-003    | Mapping akun pendapatan berdasarkan kategori produk/jasa bersifat wajib pada setiap baris invoice; invoice tanpa mapping akun pendapatan valid tidak dapat diposting ke General Ledger. |
| BRule-ARAI-004    | Akun uang muka pelanggan diklasifikasikan sebagai liabilitas dan tidak boleh dinetting langsung terhadap saldo debit AR Control Account sebelum proses alokasi ke invoice dilakukan. |
| BRule-ARAI-005    | Selisih kurs piutang wajib dipisahkan antara akun Realized FX Gain/Loss dan Unrealized FX Gain/Loss, dan hanya dapat diposting melalui proses revaluasi otomatis pada akhir periode. |
| BRule-ARAI-006    | Perubahan atau penonaktifan mapping AR Control Account, Allowance Account, maupun Revenue Account yang masih memiliki saldo piutang terbuka dilarang tanpa proses migrasi/reklasifikasi terlebih dahulu. |
| BRule-ARAI-007    | Write-off piutang wajib mengurangi saldo Allowance for Doubtful Accounts terlebih dahulu sebelum membebankan selisihnya ke akun Bad Debt Expense pada periode berjalan. |

---

# 9.6.3.33.5 Data Attributes

| Attribute                     | Description                                                                    |
| ------------------------------ | -------------------------------------------------------------------------------- |
| AR Account Mapping ID         | Identitas unik sistem atas konfigurasi mapping akun AR.                        |
| Financial Organization        | Referensi Financial Organization pemilik mapping akun.                          |
| Customer Group/Category       | Klasifikasi pelanggan yang menentukan AR Control Account terkait.               |
| AR Control Account            | Referensi Account pada COA sebagai akun ringkasan piutang usaha.                |
| Allowance for Doubtful Account | Referensi Account kontra-aset cadangan piutang tak tertagih.                    |
| Revenue Account                | Referensi Account pendapatan berdasarkan Product/Service Category.             |
| Customer Advance Account       | Referensi Account liabilitas uang muka pelanggan.                               |
| Unbilled Receivable Account    | Referensi Account piutang yang belum ditagihkan.                                |
| FX Realized/Unrealized Account | Referensi Account selisih kurs piutang, terpisah realized dan unrealized.       |
| Bad Debt Expense Account       | Referensi Account beban piutang tak tertagih.                                   |
| Product/Service Category       | Dimensi non-COA yang menjadi basis derivasi Revenue Account.                    |
| Effective From/To              | Periode berlaku mapping akun.                                                    |
| Version                        | Nomor versi mapping.                                                             |
| Approval Status                | Draft, Pending Approval, Approved, Published, Archived.                        |
| Status                          | Active, Suspended, Inactive, Archived.                                          |

---

# 9.6.3.33.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder    | Future Process                                  |
| --------------- | ------------------- | -------------- | -------------------------------------------------- |
| BR-906          | BO-004, BO-022      | SH-003, SH-028 | AR Control Account Administration                  |
| BR-907          | BO-022              | SH-003, SH-028 | Allowance Account Configuration                     |
| BR-908          | BO-009, BO-022      | SH-003, SH-028 | Automated Bad Debt Provisioning                     |
| BR-909          | BO-004, BO-022      | SH-028         | Revenue Account Derivation Processing               |
| BR-910          | BO-004              | SH-028         | Customer Advance Account Administration             |
| BR-911          | BO-022              | SH-003, SH-028 | Unbilled Receivable Account Processing              |
| BR-912          | BO-022              | SH-003         | FX Gain/Loss Account Revaluation                    |
| BR-913          | BO-004              | SH-028         | Credit/Debit Note Account Processing                |
| BR-914          | BO-022              | SH-003, SH-028 | Bad Debt Write-Off Account Processing               |
| BR-915          | BO-004, BO-022      | SH-003, SH-028 | AR Sub-Ledger to GL Reconciliation                  |

---

# 9.6.3.33.7 Summary

Accounts Receivable Account Integration Requirements melengkapi requirement umum Accounts Receivable dengan menetapkan bagaimana Chart of Accounts diintegrasikan secara spesifik ke dalam proses piutang usaha, mencakup AR Control Account, Allowance for Doubtful Accounts, mapping akun pendapatan berbasis kategori produk/jasa, akun uang muka pelanggan, akun piutang belum ditagih, akun selisih kurs, hingga rekonsiliasi subledger terhadap General Ledger. Dengan mapping akun yang konfiguratif dan dapat diaudit, domain ini memastikan akurasi laporan keuangan piutang usaha di seluruh entitas organisasi tanpa memerlukan perubahan source code, serta menjadi dasar bagi desain teknis integrasi akun pada dokumen berikutnya.
