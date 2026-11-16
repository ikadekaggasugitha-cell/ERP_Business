# ERP Database Migrations

Implementasi fisik **Database Design Document (DBDD)** — `docs/ERP-DB`. Skrip migrasi
PostgreSQL untuk arsitektur **database-per-service** (ADR-001) dengan strategi
**expand–contract** (SAD BAB 10.4) menggunakan [`golang-migrate`](https://github.com/golang-migrate/migrate).

## Cakupan (20 basis data)

**Fase 1 — Core + Financial (13 DB)**

| Kelompok        | Basis Data (schema)                                                                 |
| --------------- | ----------------------------------------------------------------------------------- |
| Core / Platform | `org_db` (`org`), `user_db` (`usr`), `security_db` (`sec`), `workflow_db` (`wf`), `masterdata_db` (`md`), `audit_db` (`audit`) |
| Financial       | `coa_db` (`fin`,`coa`), `currency_db` (`cur`), `gl_db` (`gl`), `ap_db` (`ap`), `ar_db` (`ar`), `tax_db` (`tax`), `treasury_db` (`trs`) |

**Fase 2 — Operational + HR/Payroll + Reporting (7 DB)**

| Kelompok        | Basis Data (schema)                                                                 |
| --------------- | ----------------------------------------------------------------------------------- |
| Operational     | `procurement_db` (`pr`,`po`,`gr`), `inventory_db` (`inv`), `sales_db` (`so`), `manufacturing_db` (`mfg`) |
| HR / Payroll    | `hr_db` (`hr`), `payroll_db` (`pay`)                                                 |
| Reporting       | `reporting_db` (`rpt`, CQRS read model — derived, rebuildable, consumer-only)        |

**Fase 3 — Extended Domains (10 DB, DBDD BAB 9)**

| Kelompok        | Basis Data (schema)                                                                 |
| --------------- | ----------------------------------------------------------------------------------- |
| Financial ext.  | `fixedasset_db` (`fa`), `dimension_db` (`cc`,`pc`), `budget_db` (`bud`), `financialclose_db` (`fcl`), `consolidation_db` (`cons`) |
| Operational ext.| `project_db` (`prj`), `asset_db` (`ast`), `maintenance_db` (`mnt`), `crm_db` (`crm`), `localization_db` (`loc`) |

> **Total 30 basis data** — seluruh domain DBDD (termasuk yang sebelumnya ditangguhkan di
> BAB 8.9) telah terimplementasi.

## Struktur Direktori

```
db/
├── README.md
└── migrations/
    ├── org_db/
    │   ├── 000001_init.up.sql
    │   └── 000001_init.down.sql
    ├── audit_db/
    │   ├── 000001_init.up.sql
    │   ├── 000001_init.down.sql
    │   ├── 000002_partitions_2026.up.sql   # tabel partisi RANGE
    │   └── 000002_partitions_2026.down.sql
    └── ... (satu folder per basis data)
```

Setiap basis data adalah **database PostgreSQL terpisah** dengan riwayat migrasi
`golang-migrate` sendiri (tabel `schema_migrations` per DB). **Tidak ada foreign key
lintas basis data** — relasi lintas layanan diwakili referensi logis (`*_id`/`*_ref`)
yang divalidasi via aplikasi/event (Saga/Outbox).

## Menjalankan Migrasi

### golang-migrate (per basis data)

```bash
# Contoh: gl_db
migrate -path db/migrations/gl_db \
        -database "postgres://app@localhost:5432/gl_db?sslmode=disable" up

# Rollback satu langkah
migrate -path db/migrations/gl_db -database "$GL_DSN" down 1
```

Jalankan seluruh basis data (buat DB terlebih dahulu, mis. `CREATE DATABASE gl_db;`):

```bash
for db in org_db user_db security_db workflow_db masterdata_db audit_db \
          coa_db currency_db gl_db ap_db ar_db tax_db treasury_db \
          procurement_db inventory_db sales_db manufacturing_db \
          hr_db payroll_db reporting_db \
          fixedasset_db dimension_db budget_db financialclose_db consolidation_db \
          project_db asset_db maintenance_db crm_db localization_db; do
  migrate -path "db/migrations/$db" -database "postgres://app@localhost:5432/$db?sslmode=disable" up
done
```

### Tanpa CLI (psql langsung)

```bash
psql -d gl_db -f db/migrations/gl_db/000001_init.up.sql
psql -d gl_db -f db/migrations/gl_db/000002_partitions_2026.up.sql
```

## Standar yang Diterapkan (DBS-001 … DBS-016, BAB 2)

| DBS      | Realisasi di migrasi                                                                              |
| -------- | -------------------------------------------------------------------------------------------------- |
| 001–007  | `snake_case`, tabel singular, PK `id`, FK `<entity>_id`, indeks `idx_/uq_/fk_`, suffix `_at/_date/_amount/_qty/_rate`. |
| 008      | **Common Columns** (`id, organization_id, created_at, created_by, updated_at, updated_by, deleted_at, row_version`) pada seluruh tabel domain/master/transaksi. |
| 009      | FK fisik **hanya intra-DB**; `ON DELETE CASCADE` untuk komposisi induk–anak (mis. `journal_entry`→`journal_line`), selain itu `RESTRICT`. |
| 010      | Soft delete (`deleted_at`) + optimistic locking (`row_version`) via trigger `tg_touch_row()`.       |
| 011–012  | RLS (`ENABLE ROW LEVEL SECURITY`) + policy `scope_isolation` berbasis `public.current_org_scope()`. |
| 013      | Partisi **RANGE** bulanan pada `audit_log`, `gl.journal_entry`, `ap.ap_invoice`, `ar.ar_invoice`.   |
| 014      | Tabel `outbox` (produsen) & `processed_event` (konsumen) per basis data.                            |
| 015      | Indeks: composite `organization_id`-first, partial `WHERE deleted_at IS NULL`, BRIN pada kolom waktu, GIN pada JSONB. |
| 016      | Kolom efektivitas `effective_from/effective_to` pada entitas temporal.                              |

### Trigger pemeliharaan baris (DBS-008/010)

Setiap basis data mendefinisikan `public.tg_touch_row()` yang meng-update `updated_at`
dan menaikkan `row_version` pada setiap `UPDATE`. Trigger `BEFORE UPDATE` dipasang pada
seluruh tabel yang memiliki Common Columns.

### Row-Level Security (DBS-011/012)

Aplikasi **wajib** menyuntikkan scope organisasi pengguna sebelum menjalankan query:

```sql
SET app.org_scope = '{11111111-1111-1111-1111-111111111111,2222...}';  -- uuid[] literal
```

Helper `public.current_org_scope()` mengembalikan `uuid[]` dari `current_setting('app.org_scope', true)`.
Bila **tidak di-set**, fungsi mengembalikan `NULL` sehingga policy **menolak seluruh baris**
(deny-by-default — aman). Baris dengan `organization_id IS NULL` (default global, mis.
`system_parameter`) tetap terlihat.

> **Enforcement:** RLS berlaku otomatis untuk role non-owner. Agar owner tabel juga
> tunduk RLS, jalankan aplikasi memakai role terpisah **tanpa** `BYPASSRLS` (bukan owner),
> atau tambahkan `FORCE ROW LEVEL SECURITY`. Pembuatan role/grant bersifat **operasional**
> (SAD BAB 6.9/8.8) dan **di luar cakupan** skrip DDL ini.

> **Partisi & RLS (penting):** policy RLS pada tabel **induk** berpartisi TIDAK berlaku saat
> **partisi diakses langsung**. Karena itu migrasi `000003_review_remediation` mengaktifkan RLS +
> policy `scope_isolation` pada **setiap partisi**, dan menyediakan
> `public.enable_partition_rls(regclass)` yang **wajib dipanggil job provisioning partisi**
> (pg_partman) untuk setiap partisi baru. **Strategi grant:** berikan privilege pada tabel
> **induk** (akses aplikasi selalu via induk); RLS per-partisi adalah defense-in-depth.

### Keunikan Nomor Dokumen (Document Number Registry)

Nomor dokumen kritikal wajib dijaga **database**, bukan aplikasi:
- **Tabel terpartisi** (`gl.journal_entry`, `ap/ar.*_invoice`): UNIQUE-nya menyertakan kunci
  partisi sehingga tidak menjamin keunikan lintas partisi. Keunikan dijaga oleh
  `<schema>.document_number_registry` non-partisi — `UNIQUE (organization_id, document_type,
  fiscal_year, document_number)` — yang diisi otomatis oleh trigger `BEFORE INSERT`.
- **Tabel non-partisi** (`trs.payment.payment_no`, `tax.tax_transaction.efaktur_no`): sudah
  dijaga langsung oleh `UNIQUE` — tanpa registry.

### Partisi (DBS-013)

Tabel bervolume tinggi dibuat sebagai `PARTITION BY RANGE`. Migrasi `000002_partitions_2026`
membuat partisi bulanan untuk tahun berjalan. Untuk produksi, otomatisasi partisi masa
depan menggunakan `pg_partman` atau job terjadwal (SAD BAB 6.8). PK partitioned table
menyertakan kunci partisi (mis. `PRIMARY KEY (id, posting_date)`), sesuai batasan PostgreSQL.

## Di Luar Cakupan (mengacu DBDD BAB 1.2)

- Role/grant, HA/backup/PITR/tuning (operasional).
- Data seed/master awal (artefak Data Migration).
- Kontrak API/event (artefak API Specification).
- Logika bisnis/validasi aplikatif (layanan Go) — mis. validasi balanced journal
  (`SUM(debit)=SUM(credit)`), period-lock, overlap efektivitas.

## Revisi

| Versi | Tanggal    | Deskripsi                                              |
| ----- | ---------- | ------------------------------------------------------ |
| 0.1   | 2026-08-05 | Baseline migrasi — Core/Platform + Financial (13 DB).  |
| 0.2   | 2026-08-05 | Fase 2 — Operational + HR/Payroll + Reporting (7 DB). Total 20 DB. |
| 0.3   | 2026-08-06 | Fase 3 — Extended Domains dari DBDD BAB 9 (10 DB: fixedasset, dimension, budget, financialclose, consolidation, project, asset, maintenance, crm, localization). Total 30 DB. |
