# Database Migration Review — ERP (30 databases, 70 SQL files)

Review menyeluruh seluruh migrasi (`db/migrations/**`) terhadap DBDD (`docs/ERP-DB`),
BRD (`docs/ERP-BRD`), dan SAD (`docs/ERP-SAD`). Temuan diverifikasi secara otomatis
(parsing 70 file) dan **empiris terhadap PostgreSQL 16** untuk item yang berisiko.

**Ringkasan status:** struktur solid — up+down 30/30 lulus, tidak ada cross-database FK,
RLS lengkap, trigger lengkap.

> **UPDATE 2026-08-06 — Seluruh temuan M & N telah DISELESAIKAN** via migrasi
> `*_review_remediation` (lihat `db/CHANGELOG.md`) dan diverifikasi dengan regression test.
> Item **O** (operasional) tetap sebagai catatan runbook, di luar cakupan DDL.

| Sev | # | Temuan | Status |
| --- | - | ------ | ------ |
| 🟠 Medium | M1 | RLS di-*bypass* saat mengakses partisi secara langsung. | ✅ Fixed — RLS+policy per-partisi + `enable_partition_rls()`; regression PASS. |
| 🟠 Medium | M2 | Keunikan nomor dokumen melemah karena kunci partisi masuk ke UNIQUE. | ✅ Fixed — `document_number_registry` (gl/ap/ar) + auto-trigger; regression PASS. |
| 🟡 Minor | N1 | 5 kolom FK tanpa indeks pendukung (DBS-015). | ✅ Fixed — 5 indeks ditambahkan. |
| 🟡 Minor | N2 | Penamaan sebagian FK/index disingkat (DBS-005). | ✅ Fixed — di-rename di coa_db & dimension_db. |
| 🟡 Minor | N3 | JSONB queryable belum ber-GIN. | ✅ Fixed — GIN pada `criteria` & `specification`. |
| 🟡 Minor | N4 | Indeks redundan. | ✅ Fixed — 32 indeks single-kolom redundan di-drop; indeks parsial `_active` dipertahankan (terbukti bermanfaat). |
| 🔵 Ops | O1 | Model enforcement RLS (role non-owner tanpa BYPASSRLS). | 📋 Runbook — strategi grant didokumentasikan (README + CHANGELOG). |
| 🔵 Ops | O2 | `reporting_db` projector perlu scope org. | 📋 Runbook. |
| 🔵 Ops | O3 | Auto-provisioning partisi (pg_partman) & housekeeping. | 📋 Runbook — job pg_partman wajib memanggil `public.enable_partition_rls()`. |
| 🔵 Ops | O4 | Panduan Expand–Contract migrasi lanjutan. | 📋 Terdokumentasi di CHANGELOG. |

---

## 1. Konsistensi terhadap BRD / SAD / Database Design

**Sesuai.** Setiap basis data memetakan 1:1 ke service SAD (SVC-001…029) dan bab DBDD
(BAB 3–9). Kolom & entitas mengikuti Data Dictionary; enum status mengikuti Appendix A
(BAB 8.8) dan BRD tiap domain. Prinsip SAD ditegakkan: **database-per-service** (ADR-001),
**Hybrid Multi-Tenancy + RLS** (ADR-002), **outbox/idempotency** (BAB 5.5/5.6), **partisi
RANGE** (DBS-013), **CQRS read model** (`reporting_db`, ADR-009), **PII** (BAB 7.4).

Catatan konsistensi kecil (by design, terdokumentasi):
* `dimension_db` menggabungkan Cost Center + Profit Center sesuai pengelompokan BAB 1.4 ("Cost/Profit Center").
* Entitas "Case" (BR-780) direalisasikan sebagai `crm.support_case` untuk menghindari keyword SQL `CASE`.
* Kolom `*_line`/anak menyimpan `organization_id` sendiri (denormalisasi) demi RLS & indeks — sesuai pengecualian BAB 2.4.

## 2. Naming Convention (schema/table/column/constraint/index/trigger/function/policy)

**Valid** (seluruh migrasi apply tanpa bentrok nama). DBS-001..007 dipatuhi: `snake_case`,
tabel singular, PK `id`, FK logis `*_ref`/`*_id`, boolean `is_`/`has_`, waktu `_at`, tanggal
`_date`, uang `_amount`, qty `_qty`, kurs `_rate`. Fungsi bersama `public.tg_touch_row()` /
`public.current_org_scope()`, trigger `touch_row`, policy `scope_isolation` — konsisten di 30 DB.
Tidak ada identifier > 63 karakter.

**N2 (Minor):** beberapa nama FK/index disingkat dan menyimpang dari pola DBS-005 penuh, mis.
`fk_finorg_coa`, `fk_cca_cost_center`, `fk_car_cost_center`, `fk_pca_profit_center`,
`idx_finorg_coa`. Jelas namun tidak seragam. Rekomendasi: standardкан ke `fk_<table>_<ref_table>`.

## 3. Foreign Key / Unique / Check / Referential Integrity

* **FK ON DELETE:** 58 CASCADE (komposisi induk–anak: `journal_line`, `*_invoice_line`,
  `wbs_element`, `work_order_part`, dst.), 34 RESTRICT (data transaksional/master),
  4 SET NULL (`elimination_entry.elimination_rule_id`, `crm.opportunity.lead_id`,
  `crm.interaction.lead_id`, `trs.bank_statement_line.matched_payment_id`). **Sesuai DBS-009.**
* **FK partisi (gl/ap/ar):** komposit `(id, <date>) → parent(id, <date>)` — divalidasi:
  `posting_date`/`invoice_date` tidak cocok → FK menolak; cascade delete induk → baris anak terhapus. ✔
* **Check constraint:** enum status seluruh domain sesuai Appendix A; range constraint
  (`effective_to >= effective_from`, `debit/credit >= 0`, `pct 0–100`, `period 1–12`) terpasang.
* **Referential integrity lintas layanan** dijaga via referensi logis + Saga/outbox (bukan FK).

**M2 (Medium) — keunikan nomor dokumen melemah.** Karena PostgreSQL mewajibkan kunci
partisi masuk ke UNIQUE, constraint menjadi:
`uq_journal_entry_no (organization_id, journal_no, posting_date)` dan
`uq_ap/ar_invoice_no (organization_id, invoice_no, invoice_date)`.
**Terbukti empiris:** dua baris `(org, journal_no='JE-1')` dengan `posting_date` berbeda
**diterima** (2 baris). Artinya keunikan `journal_no`/`invoice_no` per organisasi TIDAK
dijamin lintas tanggal/partisi.
*Rekomendasi:* tegakkan keunikan nomor dokumen di layer aplikasi (service Go), atau simpan
tabel "sequence/registry nomor dokumen" non-partisi ber-`UNIQUE (organization_id, doc_no)`
sebagai penjaga. (Ini konsekuensi wajar partisi; perlu keputusan eksplisit.)

## 4. Row-Level Security (RLS) & Policy

* **Cakupan lengkap:** 130/130 tabel domain ber-scope memiliki `ENABLE ROW LEVEL SECURITY`
  + policy `scope_isolation` (USING + WITH CHECK) berbasis `public.current_org_scope()`.
  Tabel teknis `outbox`/`processed_event` sengaja tanpa RLS (diakses relay/worker).
* **Deny-by-default** terverifikasi: tanpa `app.org_scope` → 0 baris; scope org A → hanya org A.

**M1 (Medium, keamanan) — RLS di-bypass pada akses partisi langsung.** Policy pada tabel
induk berpartisi **tidak** otomatis berlaku untuk query yang menyasar partisi anak secara
langsung. **Terbukti empiris:** `app_role` (scope=org1) membaca `gl.journal_entry` → 1 baris
(benar), namun `gl.journal_entry_y2026m08` langsung → **2 baris** (bocor lintas-org).
*Rekomendasi (pilih satu):*
  1. **Grant hanya pada tabel induk**, tidak pada partisi (skrip grant operasional) — aplikasi
     memang selalu mengakses induk; ini cara paling bersih.
  2. Aktifkan `ENABLE ROW LEVEL SECURITY` + policy identik pada **setiap partisi** (bisa
     di-generate di migrasi `000002_partitions`).
  3. Tegakkan lewat model peran (aplikasi tak pernah menyentuh nama partisi).
Direkomendasikan #1 + dokumentasikan di README (bagian RLS).

**O1 (Ops):** enforcement RLS mensyaratkan aplikasi konek sebagai **role non-owner tanpa
BYPASSRLS** (pemilik tabel & superuser melewati RLS). Pembuatan role/grant berada di artefak
operasional (di luar cakupan DDL, BAB 1.2) — perlu dipastikan ada di runbook deploy.

**O2 (Ops):** `reporting_db` mengaktifkan RLS pada read model; **projector** (konsumen event)
harus menyetel `app.org_scope` yang sesuai (atau memakai role khusus) agar upsert tidak tertolak.

## 5. Optimistic Locking (`row_version`) & Audit Columns

* **Cakupan lengkap:** 124/124 tabel ber-`row_version` memiliki trigger `touch_row`
  (`BEFORE UPDATE`) yang menaikkan `row_version` dan menyetel `updated_at`. Tidak ada tabel
  ber-`row_version` tanpa trigger.
* **Sengaja tanpa `row_version`/trigger (benar):** `org.organization_closure` &
  `sec.role_permission` (junction, di-rebuild), `audit.audit_log` (append-only + guard
  immutability), 3 read model `reporting_db` (derived/rebuildable).
* **Common Columns** (`created_at/by`, `updated_at/by`, `deleted_at`, `row_version`) konsisten;
  `created_by NOT NULL` (seed/migrasi wajib menyuplai). Audit lengkap (before/after) di
  `audit_db` via event (BAB 3.7).
* Catatan: enforcement optimistic lock bersifat *app-side* (`UPDATE ... WHERE row_version = :v`);
  trigger hanya menaikkan versi — desain yang benar.

## 6. Outbox Pattern & processed_event (DBS-014)

* Setiap DB **produsen** memiliki `outbox` (id, aggregate_type/id, event_type, payload JSONB,
  organization_id, correlation_id, created_at, published_at). Konsumen murni
  (`audit_db`, `reporting_db`) hanya `processed_event`. **Sesuai peran.**
* Indeks outbox: parsial `(created_at) WHERE published_at IS NULL` (antrian publish),
  `aggregate_id`, `correlation_id` — tepat untuk relay & korelasi Saga.
* `processed_event` PK `event_id` → idempotensi konsumen. ✔
* Catatan: pembersihan berkala baris `published_at IS NOT NULL` (BAB 8.6) adalah housekeeping
  operasional (O3), bukan DDL.

## 7. Index Strategy (B-Tree / Composite / Partial / BRIN / GIN)

* **Composite org-first:** `UNIQUE (organization_id, code)` di seluruh master; query periodik
  `(organization_id, fiscal_period_id)`, `(organization_id, status)`. ✔
* **Partial:** `WHERE deleted_at IS NULL` (master aktif), `WHERE published_at IS NULL` (outbox),
  `WHERE status='Pending'` (task antrian). ✔
* **BRIN** pada kolom waktu monoton tabel terpartisi: `journal_entry.posting_date`,
  `stock_movement.movement_date`, `audit_log.occurred_at`, `ap/ar_invoice.invoice_date`. ✔
* **GIN** pada JSONB rule/atribut: 8 indeks (`structure`, `posting_rule` debit/credit,
  `workflow.approver_rule`, `recurring template_lines`, `org attr_value`,
  `cost_allocation targets`, `elimination config`). ✔ (payload outbox sengaja tidak di-GIN.)

**N1 (Minor) — 5 kolom FK tanpa indeks pendukung** (DBS-015 "setiap FK punya indeks"):
  * `cons.elimination_entry.elimination_rule_id`
  * `inv.stock_balance.warehouse_id`, `inv.stock_movement.warehouse_id`, `inv.stock_reservation.warehouse_id`
    (hanya tercakup indeks komposit ber-*leading* `organization_id`, bukan `warehouse_id`)
  * `trs.bank_statement_line.matched_payment_id`
  Dampak: `DELETE/UPDATE` pada tabel induk (warehouse/payment/rule) memicu *seq scan* pengecekan
  FK; rendah karena RESTRICT/SET NULL & induk jarang dihapus. *Rekomendasi:* tambah indeks
  pada kolom FK tsb (mis. `idx_stock_movement_wh (warehouse_id)`).

**N3 (Minor):** `crm.customer_segment.criteria` & `ast.asset.specification` (JSONB yang mungkin
di-query) belum ber-GIN — tambahkan bila menjadi predikat query.

**N4 (Minor):** indeks parsial `idx_<t>_active (organization_id) WHERE deleted_at IS NULL`
sebagian tumpang-tindih dengan UNIQUE `(organization_id, code)` untuk lookup by-org; simpan
hanya bila query "list aktif per org tanpa code" memang panas, jika tidak → redundan.

## 8. Kesiapan Expand–Contract (SAD BAB 10.4)

* Baseline `000001_init` bersifat **additive** (hanya CREATE) → aman sebagai fase *expand*
  awal; seluruh statement transaksional (tanpa `CREATE INDEX CONCURRENTLY`) sehingga
  golang-migrate menjalankannya atomik per file. ✔
* **O4 (guidance) untuk migrasi berikutnya:** ikuti expand→contract — tambah kolom *nullable*
  → backfill → set `NOT NULL`/constraint di migrasi terpisah; buat indeks besar dengan
  `CREATE INDEX CONCURRENTLY` (di luar transaksi, file terpisah tanpa statement lain);
  hindari `DROP`/rename destruktif pada fase expand.

## 9. Rollback Safety

* **Terverifikasi:** down 30/30 bersih; seluruh schema domain hilang, fungsi bersama di-drop
  setelah schema (urutan benar: `DROP SCHEMA ... CASCADE` men-drop trigger lebih dulu).
* Partisi (gl/ap/ar) memakai `DETACH PARTITION` sebelum `DROP` (dependensi FK per-partisi);
  audit/inventory memakai `DROP` biasa (tanpa FK masuk). **Asumsi:** partisi kosong saat
  rollback (baseline) — didokumentasikan; rollback dengan data historis bukan skenario yang
  didukung (memang tidak lazim).
* Catatan minor: `DROP FUNCTION IF EXISTS public.tg_touch_row()/current_org_scope()` aman
  karena tiap DB hanya mendefinisikannya sekali (migrasi `000001`).

## 10. Audit Dependency Antar-Service (No Cross-Database FK)

* **Terverifikasi otomatis:** setiap `REFERENCES <schema>.<table>` menunjuk schema yang
  memang dibuat di DB yang sama → **NOL foreign key lintas basis data.** Relasi lintas
  layanan seluruhnya berupa referensi logis (`*_ref`, atau `*_id` yang divalidasi via
  aplikasi/event). Konsisten penuh dengan ADR-001.

---

## Prioritas Perbaikan (rekomendasi)

1. **M1** — putuskan strategi RLS partisi (disarankan: grant hanya pada tabel induk + catatan
   README; atau generate policy per-partisi di `000002`).
2. **M2** — tegakkan keunikan `journal_no`/`invoice_no` di aplikasi atau via tabel registry
   nomor dokumen non-partisi.
3. **N1** — tambah 5 indeks FK (migrasi additive `000003_fk_indexes` per DB terkait).
4. **N2/N3/N4** — rapikan penamaan FK/index, tambah GIN bila perlu, tinjau indeks parsial redundan.
5. **O1–O4** — masukkan ke runbook operasional & konvensi migrasi lanjutan.

> Tidak ada temuan yang memblokir; seluruh migrasi valid & reversible. M1 & M2 sebaiknya
> ditindaklanjuti sebelum go-live karena berdampak pada isolasi data dan integritas nomor dokumen.
