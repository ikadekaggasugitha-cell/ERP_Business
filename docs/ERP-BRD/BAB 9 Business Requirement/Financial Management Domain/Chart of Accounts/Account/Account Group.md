# 9.6.3.30 Account Group Detail Requirements

---

# 9.6.3.30.1 Overview

Dokumen ini merupakan pendalaman (deep-dive) dari building block Account Group yang telah didefinisikan secara ringkas pada dokumen **9.6.3.6 Account Group Requirements (BR-351 s.d. BR-365)**. Dokumen 9.6.3.6 berfokus pada Account Group sebagai template konfigurasi administratif (number range, default attribute, field configuration, validation rule) untuk pembuatan Account Master. Dokumen ini **tidak mengulang** cakupan tersebut, melainkan memperluas kapabilitas Account Group ke ranah **dynamic grouping** yang bersifat rule-based, lintas entitas, dan berorientasi pada kebutuhan analisis manajemen.

Dalam praktik operasional grup perusahaan dengan banyak Legal Entity, Ledger, dan Chart of Accounts yang berbeda struktur, pengelompokan akun secara statis (fixed group per akun) tidak lagi memadai. Manajemen membutuhkan kemampuan untuk:

* Membentuk grup akun secara **dinamis** berdasarkan kondisi/rule (misalnya seluruh akun dengan Account Category = Operating Expense DAN Cost Center Type = Production), bukan hanya assignment statis satu akun ke satu grup.
* Membentuk grup akun **lintas Legal Entity/Chart of Accounts** untuk kebutuhan **konsolidasi manajerial** (management consolidation) yang berbeda dari konsolidasi statutori.
* Memungkinkan manajemen (CFO, COO, Finance Business Partner) membuat **custom group ad-hoc** untuk analisis sementara tanpa memerlukan perubahan Chart of Accounts atau permintaan ke tim IT/Finance Systems.
* Mengelola **hierarki grup akun berlapis (nested/multi-level group hierarchy)** yang dapat berbeda dari hierarki Account Structure formal, sehingga satu akun dapat berpartisipasi dalam banyak hierarki analitis sekaligus (multi-parent grouping).

Dengan kapabilitas ini, Account Group berevolusi dari sekadar objek administratif menjadi instrumen analitis strategis yang mendukung pelaporan manajerial real-time dan konsolidasi lintas entitas tanpa bergantung pada perubahan struktural pada Chart of Accounts dasar.

---

# 9.6.3.30.2 Business Capability

Account Group Detail menyediakan kapabilitas lanjutan berikut:

* Dynamic/Rule-Based Account Grouping (grouping berbasis kondisi, bukan assignment statis).
* Rule Engine dengan operator logika (AND/OR/NOT) atas atribut Account Master, Segment Value, dan Financial Dimension.
* Real-Time Rule Evaluation dan re-kalkulasi keanggotaan grup saat atribut akun berubah.
* Cross-Entity Account Grouping untuk konsolidasi manajerial lintas Legal Entity, Sub Holding, dan Chart of Accounts.
* Cross-Chart Mapping Resolution (pemetaan akun antar-Chart of Accounts yang berbeda struktur ke dalam satu grup manajerial).
* Ad-Hoc Custom Group Authoring oleh pengguna manajemen (self-service grouping) tanpa perubahan Chart of Accounts.
* Personal/Private Group vs Shared/Published Group (kontrol visibilitas dan siklus hidup custom group).
* Multi-Level Nested Group Hierarchy (grup di dalam grup, dengan kedalaman level yang dapat dikonfigurasi).
* Multi-Parent Membership (satu akun dapat menjadi anggota lebih dari satu grup analitis sekaligus).
* Group Simulation & Preview (pratinjau anggota grup sebelum publikasi/aktivasi rule).
* Group-Based Reporting Integration (penggunaan dynamic group sebagai dimensi pada laporan manajerial dan dashboard).
* Group Versioning & Point-in-Time Membership Reconstruction untuk analisis historis.

---

# 9.6.3.30.3 Business Requirement Catalogue

## BR-876 – Dynamic Rule-Based Account Grouping

| Attribute           | Description                                                                                                                                                                                                              |
| ------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requirement ID      | BR-876                                                                                                                                                                                                                   |
| Requirement Name    | Dynamic Rule-Based Account Grouping                                                                                                                                                                                     |
| Category            | Account Group Detail                                                                                                                                                                                                    |
| Priority            | Critical                                                                                                                                                                                                                 |
| Business Objective  | BO-008 (Keputusan berbasis data real-time)                                                                                                                                                                              |
| Stakeholder Owner   | SH-003 (Chief Financial Officer)                                                                                                                                                                                        |
| Description         | Sistem harus memungkinkan pembentukan Account Group secara dinamis berdasarkan rule/kondisi atas atribut Account Master (Account Category, Account Class, Account Type, Segment Value, Financial Dimension), bukan melalui assignment statis akun per akun. |
| Business Rationale  | Grouping statis membutuhkan pemeliharaan manual setiap kali ada akun baru atau perubahan atribut, sehingga rawan tertinggal (stale) dan tidak scalable untuk Chart of Accounts berskala besar.                          |
| Business Value      | Keanggotaan grup selalu konsisten dengan kondisi bisnis terkini tanpa intervensi manual, mempercepat penyediaan data untuk pengambilan keputusan.                                                                       |
| Acceptance Criteria | Pengguna berwenang dapat mendefinisikan rule kondisi menggunakan operator logika; sistem menghasilkan daftar akun anggota grup secara otomatis dan memperbarui keanggotaan setiap kali atribut akun yang relevan berubah. |

---

## BR-877 – Rule Condition Builder & Logical Operators

| Requirement ID   | Requirement Name                     | Priority | Description                                                                                                                                                                          |
| ---------------- | ------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-877           | Rule Condition Builder & Logical Operators | Critical | Sistem harus menyediakan rule condition builder yang mendukung operator logika AND, OR, NOT, serta operator perbandingan (equals, contains, range, wildcard) atas kombinasi atribut Account Master, Segment Value, dan Financial Dimension untuk membentuk kondisi keanggotaan grup. |

---

## BR-878 – Real-Time Group Membership Recalculation

| Requirement ID   | Requirement Name                        | Priority | Description                                                                                                                                                                    |
| ---------------- | ----------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-878           | Real-Time Group Membership Recalculation  | Critical | Sistem harus melakukan evaluasi ulang keanggotaan Dynamic Account Group secara real-time atau near real-time setiap kali atribut Account Master yang menjadi dasar rule mengalami perubahan, penambahan, atau penonaktifan akun. |

---

## BR-879 – Cross-Entity Account Grouping for Managerial Consolidation

| Requirement ID   | Requirement Name                                     | Priority | Description                                                                                                                                                                                     |
| ---------------- | ------------------------------------------------------ | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-879           | Cross-Entity Account Grouping for Managerial Consolidation | Critical | Sistem harus memungkinkan pembentukan Account Group yang mencakup akun dari lebih dari satu Legal Entity, Sub Holding, atau Chart of Accounts untuk kebutuhan konsolidasi manajerial, terpisah dari struktur konsolidasi statutori. |

---

## BR-880 – Cross-Chart Account Mapping Resolution

| Requirement ID   | Requirement Name                     | Priority | Description                                                                                                                                                          |
| ---------------- | ------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-880           | Cross-Chart Account Mapping Resolution | High     | Sistem harus menyediakan mekanisme pemetaan (mapping resolution) yang menyelaraskan akun dari Chart of Accounts berbeda struktur/penomoran ke dalam satu Account Group manajerial yang konsisten. |

---

## BR-881 – Ad-Hoc Custom Group Authoring (Self-Service)

| Requirement ID   | Requirement Name                          | Priority | Description                                                                                                                                                                    |
| ---------------- | -------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-881           | Ad-Hoc Custom Group Authoring (Self-Service)  | High     | Sistem harus memungkinkan pengguna manajemen (CFO, COO, Finance Business Partner) membuat custom Account Group ad-hoc secara mandiri untuk kebutuhan analisis sementara, tanpa memerlukan perubahan Chart of Accounts atau keterlibatan tim Finance Systems/IT. |

---

## BR-882 – Custom Group Visibility & Lifecycle Control

| Requirement ID   | Requirement Name                       | Priority | Description                                                                                                                                                     |
| ---------------- | ----------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-882           | Custom Group Visibility & Lifecycle Control | High     | Sistem harus membedakan status visibilitas custom group antara Personal/Private (hanya pembuat), Shared (tim tertentu), dan Published (organisasi), beserta siklus hidup Draft, Active, Expired, dan Retired untuk masing-masing status. |

---

## BR-883 – Multi-Level Nested Account Group Hierarchy

| Requirement ID   | Requirement Name                          | Priority | Description                                                                                                                                                              |
| ---------------- | -------------------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-883           | Multi-Level Nested Account Group Hierarchy    | Critical | Sistem harus mendukung hierarki Account Group berlapis (group of groups) dengan kedalaman level yang dapat dikonfigurasi, di mana satu grup induk dapat menaungi satu atau lebih grup anak untuk keperluan agregasi bertingkat. |

---

## BR-884 – Multi-Parent Group Membership

| Requirement ID   | Requirement Name              | Priority | Description                                                                                                                                                                     |
| ---------------- | -------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-884           | Multi-Parent Group Membership     | High     | Sistem harus memungkinkan satu Account Master menjadi anggota dari lebih dari satu Account Group analitis secara bersamaan (multi-parent), tanpa membatasi akun hanya pada satu grup tunggal seperti pada struktur Account Structure formal. |

---

## BR-885 – Group Membership Simulation, Preview, and Point-in-Time Reconstruction

| Requirement ID   | Requirement Name                                                   | Priority | Description                                                                                                                                                                          |
| ---------------- | ---------------------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BR-885           | Group Membership Simulation, Preview, and Point-in-Time Reconstruction | High     | Sistem harus menyediakan fungsi simulasi/pratinjau anggota grup sebelum rule dipublikasikan, serta kemampuan merekonstruksi keanggotaan Dynamic Account Group pada titik waktu tertentu di masa lalu (point-in-time) untuk kebutuhan analisis historis dan audit. |

---

# 9.6.3.30.4 Business Rules

| Business Rule ID | Description                                                                                                                                                                                              |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRule-AGD-001     | Rule kondisi Dynamic Account Group harus dievaluasi menggunakan snapshot atribut Account Master terkini; hasil evaluasi tidak boleh mengubah data akun itu sendiri, hanya keanggotaan grup.               |
| BRule-AGD-002     | Cross-Entity Account Group untuk konsolidasi manajerial harus secara eksplisit ditandai terpisah dari struktur konsolidasi statutori dan tidak boleh digunakan sebagai dasar pelaporan keuangan resmi ke regulator. |
| BRule-AGD-003     | Custom group ad-hoc berstatus Personal/Private hanya dapat dilihat dan diubah oleh pembuatnya; publikasi menjadi Shared atau Published wajib melalui persetujuan pemilik data (data owner) yang berwenang. |
| BRule-AGD-004     | Hierarki grup berlapis tidak boleh membentuk siklus (circular reference); grup anak tidak boleh secara langsung maupun tidak langsung menjadi induk dari grup leluhurnya sendiri.                          |
| BRule-AGD-005     | Satu akun dapat menjadi anggota banyak grup analitis (multi-parent), namun agregasi nilai pada laporan konsolidasi manajerial harus menyertakan mekanisme untuk mencegah penghitungan ganda (double counting) apabila grup-grup tersebut digabungkan dalam satu laporan. |
| BRule-AGD-006     | Perubahan rule pada Dynamic Account Group yang telah dipublikasikan dan digunakan pada laporan manajerial aktif harus melalui versi baru (new version), bukan menimpa versi yang sedang digunakan, agar laporan historis tetap dapat direproduksi. |
| BRule-AGD-007     | Setiap evaluasi ulang keanggotaan grup, publikasi custom group, dan perubahan hierarki berlapis harus dicatat dalam Audit Trail beserta identitas pengguna, waktu, dan cakupan perubahan.                  |

---

# 9.6.3.30.5 Data Attributes

| Attribute                     | Description                                                                                     |
| ------------------------------ | ------------------------------------------------------------------------------------------------- |
| Dynamic Group ID               | Identitas unik grup dinamis.                                                                     |
| Group Type                     | Static, Dynamic (Rule-Based), Cross-Entity, Custom Ad-Hoc.                                       |
| Rule Definition                | Struktur kondisi logika (AND/OR/NOT) atas atribut Account Master, Segment Value, Financial Dimension. |
| Rule Expression (Raw)          | Representasi teknis rule (mis. JSON/DSL) untuk keperluan eksekusi mesin evaluasi.                 |
| Scope Entities                 | Daftar Legal Entity/Sub Holding/Chart of Accounts yang tercakup dalam grup lintas entitas.         |
| Cross-Chart Mapping Reference  | Referensi pemetaan akun antar Chart of Accounts yang berbeda struktur.                             |
| Owner/Creator                  | Pengguna yang membuat custom group.                                                                |
| Visibility Level               | Personal/Private, Shared, Published.                                                              |
| Parent Group ID                | Referensi ke grup induk pada hierarki berlapis.                                                    |
| Hierarchy Level/Depth          | Level kedalaman grup dalam hierarki berlapis.                                                      |
| Membership Snapshot Date       | Tanggal snapshot evaluasi keanggotaan untuk rekonstruksi point-in-time.                             |
| Member Account List (Derived)  | Daftar akun anggota hasil evaluasi rule (data turunan, tidak diinput manual).                      |
| Double-Count Prevention Flag   | Penanda mekanisme pencegahan penghitungan ganda saat agregasi lintas grup.                          |
| Simulation Status              | Draft/Simulated, Published, Retired.                                                              |
| Version                        | Nomor versi rule/grup.                                                                             |
| Effective From / Effective To  | Periode berlaku rule atau grup.                                                                    |
| Status                         | Draft, Simulated, Active, Expired, Retired, Archived.                                              |
| Created By / Created Date      | Informasi pembuatan data.                                                                          |
| Updated By / Updated Date      | Informasi perubahan terakhir.                                                                      |

---

# 9.6.3.30.6 Requirement Traceability

| Requirement ID | Business Objective | Stakeholder | Future Process                             |
| -------------- | ------------------- | ----------- | ------------------------------------------- |
| BR-876         | BO-008              | SH-003      | Dynamic Grouping Rule Management            |
| BR-877         | BO-008              | SH-003      | Rule Condition Authoring                    |
| BR-878         | BO-008              | SH-003      | Real-Time Membership Recalculation          |
| BR-879         | BO-020              | SH-003      | Managerial Consolidation Grouping           |
| BR-880         | BO-020              | SH-003      | Cross-Chart Mapping Resolution              |
| BR-881         | BO-008              | SH-004      | Self-Service Ad-Hoc Analysis                |
| BR-882         | BO-004              | SH-004      | Custom Group Governance                     |
| BR-883         | BO-004              | SH-003      | Nested Group Hierarchy Administration       |
| BR-884         | BO-004              | SH-003      | Multi-Parent Group Assignment               |
| BR-885         | BO-008              | SH-003      | Group Simulation & Historical Reconstruction |

---

# 9.6.3.30.7 Summary

Account Group Detail Requirements memperluas fondasi Account Group dari sekadar template administratif menjadi instrumen analitis strategis yang mendukung kebutuhan manajemen modern. Melalui **dynamic rule-based grouping**, keanggotaan akun dalam grup selalu mencerminkan kondisi bisnis terkini tanpa pemeliharaan manual. Melalui **cross-entity grouping**, CFO dan tim finance dapat menyusun konsolidasi manajerial lintas Legal Entity dan Chart of Accounts yang berbeda struktur tanpa bergantung pada konsolidasi statutori. Melalui **ad-hoc custom group**, COO dan Finance Business Partner memperoleh kemandirian (self-service) untuk membentuk sudut pandang analitis baru secara cepat. Melalui **hierarki berlapis dan multi-parent membership**, satu akun dapat berpartisipasi dalam berbagai dimensi analisis sekaligus tanpa mendistorsi struktur Chart of Accounts formal.

Kapabilitas ini secara langsung mendukung **BO-004 (Single Source of Truth)** dengan menjaga rule dan hasil grouping sebagai satu sumber kebenaran yang dapat ditelusuri, **BO-008 (Keputusan berbasis data real-time)** melalui evaluasi keanggotaan grup yang selalu terkini, serta **BO-020 (Konsolidasi lintas perusahaan)** melalui kapabilitas cross-entity dan cross-chart grouping. Dengan demikian, Account Group Detail menjadi enabler penting bagi pelaporan manajerial yang gesit, akurat, dan dapat diaudit pada ERP multi-organisasi.
