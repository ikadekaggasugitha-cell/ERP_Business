# Cortex Framework V7 & SAP Coexistence Specification

Dokumen ini merupakan panduan arsitektur dan spesifikasi teknis untuk pengembangan **Enterprise Data Products** menggunakan **Cortex Framework V7** dan **Dataform (.js)** pada klaster analitik BigQuery / Lakehouse, serta panduan koeksistensi data dengan sistem **SAP ECC 6.0 & SAP S/4HANA** (ADR-013).

---

## 1. Core Operating Behaviors (Non-Negotiable)

1. **Surface Assumptions**:
   - Selalu paparkan asumsi mengenai definisi field data, mata uang dasar, atau logika konsolidasi sebelum membuat model transformasi.
2. **Manage Confusion Actively**:
   - Jangan pernah menebak struktur SAP. Pastikan apakah data product menargetkan **SAP ECC 6.0**, **SAP S/4HANA**, atau **Keduanya**:
     - *ECC 6.0*: Menggunakan tabel akuntansi terpisah `BKPF` (Header) dan `BSEG` (Line Items).
     - *S/4HANA*: Menggunakan tabel jurnal universal terpadu `ACDOCA` (*Universal Journal*).
3. **Strict Namespace Governance**:
   - Semua modul transformasi dan konfigurasi data product kustom **wajib** menggunakan namespace kustom: `src/data_modules/custom/` atau `src/data_modules/sinergi/`.
   - Dilarang keras menimpa (*overwrite*) namespace bawaan `cortex` untuk menjaga kompatibilitas upgrade framework upstream.

---

## 2. Layering Architecture

```text
┌────────────────────────────────────────────────────────────────────────┐
│  Lapisan 3: Data Products (Star Schema)                                 │
│  - fact_gl_transaction, fact_sales_order, fact_inventory_snapshot      │
│  - dim_organization, dim_currency, dim_fiscal_period, dim_coa          │
└───────────────────────────────────▲────────────────────────────────────┘
                                    │ Dataform Transformations (.js)
┌───────────────────────────────────┴────────────────────────────────────┐
│  Lapisan 2: Annotations & Semantic Layer                               │
│  - annotations/ecc/ (Field mapping untuk ECC: BKPF + BSEG)             │
│  - annotations/s4/  (Field mapping untuk S/4HANA: ACDOCA)              │
│  - Standardisasi nama kolom dan tipe data bisnis                       │
└───────────────────────────────────▲────────────────────────────────────┘
                                    │ Raw Ingestion / CDC
┌───────────────────────────────────┴────────────────────────────────────┐
│  Lapisan 1: Data Foundations (Raw SAP DDIC & ERP Kafka Streams)        │
│  - SAP DDIC Replikasi: DD02L, DD03L, BKPF, BSEG, ACDOCA, MARC, EKKO    │
│  - ERP New Transaksional: erp.gl.journal.posted.v1, erp.sales.*.v1    │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Conformed Dimensions Specification

### 3.1 `dim_organization` (Hierarchical Closure Table)
Menyatukan struktur organisasi pohon tanpa batas level dari ERP baru (`org.organization`) dan SAP *Company Code* / *Plant* / *Cost Center*:
- `organization_id` (UUID / Key)
- `ancestor_id` & `descendant_id` (Hierarki multi-level untuk rollup holding instan)
- `depth_level` (Holding, Sub Holding, Company, Branch, Unit)
- `sap_company_code` (Pemetaan kode perusahaan SAP, mis. `1000`, `2000`)
- `country_code` (ISO 3166-1 alpha-2, mis. `ID`, `SG`, `MY`)
- `functional_currency` (Mata uang statutori entitas)

### 3.2 `dim_currency` & Tri-Currency Translation
Menangani translasi jutaan transaksi valas secara sub-detik:
- `currency_code` (ISO 4217, mis. `IDR`, `USD`, `SGD`, `EUR`)
- `rate_date` & `rate_type` (Spot, Pajak/KMK, Average, Closing)
- `exchange_rate_to_usd` (Kurs terhadap anchor currency dengan presisi `numeric(18,8)`)
- `exchange_rate_to_idr` (Kurs terhadap presentation currency holding)

### 3.3 `dim_fiscal_period`
Penyelaras kalender fiskal internasional multi-negara:
- `fiscal_period_id` (Key)
- `calendar_code` (Standar Jan–Des, April–Maret untuk JP/IN, Juli–Juni untuk AU)
- `period_number` (1 s.d. 12, plus periode 13/14 untuk audit closing)
- `group_period_number` (Periode konsolidasi terpadu grup holding)

---

## 4. Star Schema Facts Specification

### 4.1 `fact_gl_transaction`
Tabel fakta transaksi buku besar terpartisi bulanan (Partisi: `posting_date`, Klaster: `organization_id`, `account_code`):
- `transaction_id` (UUID)
- `organization_key` (FK ke `dim_organization`)
- `fiscal_period_key` (FK ke `dim_fiscal_period`)
- `account_key` (FK ke `dim_chart_of_accounts`)
- `posting_date` (Tanggal posting transaksi)
- **Monetary Triplet**:
  - `amount_tc` + `currency_tc` (Transaction Currency)
  - `amount_fc` + `currency_fc` + `rate_fc` (Functional Currency)
  - `amount_pc` + `currency_pc` + `rate_pc` (Presentation Currency Grup)
- `source_system` (`NEW_ERP`, `SAP_ECC`, `SAP_S4HANA`)
- `sap_document_number` (Nomor dokumen SAP referensi)

