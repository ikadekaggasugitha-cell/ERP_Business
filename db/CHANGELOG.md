# Database Migration CHANGELOG

Semua perubahan skema signifikan. Format: keputusan + alasan + verifikasi.

## Production-Readiness Program — 2026-08-06

Roadmap 6 langkah menuju production-ready (bukan sekadar migrasi berhasil).

### Step 1 — Hardening: roles/grants (menutup O1) — DONE & verified
- `db/ops/roles.sql`: role global `erp_migrator` (owner), `erp_app_rw`, `erp_app_ro` — semua
  `NOSUPERUSER NOBYPASSRLS`, dengan `statement_timeout`/`lock_timeout`/`idle_in_transaction_session_timeout`
  dan `default_transaction_read_only` untuk RO.
- `db/ops/apply_grants.sql`: least-privilege per-DB — grant pada **parent/regular tables saja**
  (bukan partisi), `REVOKE ... FROM PUBLIC`, default privileges untuk tabel baru.
- **Verified:** app role bukan owner ⇒ RLS ditegakkan (parent visible=1, isolasi); akses langsung
  ke partisi **permission denied**; `erp_app_ro` INSERT ditolak (read-only).

### Step 2 — Seed Data Framework (reference/master data) — DONE & verified
- `db/seeds/<db>/seed.sql` idempoten (`ON CONFLICT DO NOTHING`), dapat di-scope per tenant via
  `-v org_id=`. Cakupan: currency+rate types, language+locale, org types, RBAC sistem
  (permission/role/role_permission), tax code Indonesia, fiscal calendar + FY2026.
- Konvensi: reference diseed di **system org** `0000…0000`; aplikasi menyertakan id ini pada
  `app.org_scope`. **Verified:** 2× run idempoten; counts stabil (currency 9, permission 70,
  role_permission 113, fiscal_period 12, dst.); tanpa pelanggaran FK/enum.

### Step 3 — Partition strategy — DONE & verified
- Migrasi `*_partition_hardening` (29 DB) menambah fungsi maintenance reusable di `public`:
  `create_monthly_partition` (auto-RLS untuk parent ber-RLS), `ensure_future_partitions`,
  `detach_old_partitions` (arsip via DETACH).
- **`gl.journal_line`** dikonversi menjadi RANGE-partitioned by `posting_date` (13 partisi;
  PK `(id, posting_date)`, uq `(journal_entry_id, line_no, posting_date)`, FK komposit + CASCADE, RLS).
- **`<schema>.outbox`** di 28 DB produsen dikonversi menjadi RANGE-partitioned by `created_at`
  (PK `(id, created_at)`; tanpa RLS — tabel teknis). audit_db & reporting_db tidak punya outbox.
- Runbook: `db/ops/partition_maintenance.md` (jadwal pg_cron provision+archive, tabel retensi).
- **Verified:** UP 30/30, DOWN 30/30 (reversible). Fungsional: `ensure_future_partitions` membuat
  partisi baru dgn RLS otomatis; journal_line & outbox routing benar; cascade FK utuh;
  `detach_old_partitions` mengarsipkan partisi lama tanpa drop.

### Step 4 — Performance benchmark at-scale — DONE & verified
- `db/benchmark/benchmark_at_scale.sql` (100k journal entries + 200k lines, 50k AP/AR/outbox) +
  `db/benchmark/RESULTS.md` (baseline plans + timings).
- **Terbukti:** index scan aktif pada lookup selektif (`source_document_ref`, `vendor_id`,
  `customer_id`, partial outbox), partition pruning ke 1 partisi untuk query ber-predikat tanggal
  (GL-B, AP-B), dan partial index `idx_outbox_unpublished` melayani relay poll.
- **Guidance:** sertakan kunci partisi pada query pelaporan untuk pruning; pertimbangkan composite
  `(customer_id,status)`/`(vendor_id,status)` bila laporan aging dominan.

### Step 5 — Backup/Restore + PITR — DONE & verified (lokal)
- `db/backup/`: `logical_backup.sh` (pg_dump -Fc semua DB + globals + checksum), `logical_restore.sh`
  (createdb + pg_restore --clean -j), `verify_restore.sh` (restore→bandingkan count semua tabel),
  `pitr_demo.sh` (base backup→disaster→recover-to-target), `README.md` (RPO/RTO + runbook).
- **Verified:** logical dump `gl_db` (500 entri + registry) → drop → restore = **count cocok,
  13 partisi utuh**; `verify_restore.sh` = PASS. **PITR** (WAL archiving + pg_basebackup + recovery
  ke `recovery_target_time`) = data pra-target **A** selamat, **B**+DROP tidak ter-replay
  (`m2_restored=true, rows=1:A-before-target`).
- Target: RPO ≤ 60s (archive_timeout) / ~0 dgn streaming replica; RTO menit/DB (logical) atau
  base-restore+replay (PITR). Rekomendasi produksi: pgBackRest/wal-g + standby.

### Step 6 — Go repository layer + migration pipeline — DONE & verified
- `services/gl-service/` (reference untuk 29 service lain): **pgx v5 + sqlc + golang-migrate**.
  - `db/queries/journal.sql` (input sqlc) + `db/schema/schema.sql` (katalog) → `internal/gldb/*`
    (kode ter-generate sqlc: models, Queries, params/rows type-safe).
  - `internal/platform/postgres.go` — pool + `WithOrgScope(ctx, scope, fn)` yang menyuntik RLS
    per-request via `set_config('app.org_scope', $1, true)` (transaction-local, aman untuk pooling).
  - `internal/repository/journal_repository.go` — repo (CreateEntryWithLines atomik, ListByPeriod,
    Totals, Post optimistic-lock, Drain/MarkOutbox relay tanpa scope).
  - `cmd/server/main.go`, `Makefile` (sqlc/migrate/grants/seed/build), `README.md`,
    `.github/workflows/gl-service.yml` (CI: postgres service → migrate up → sqlc diff → build/test → down).
- **Verified (Docker):** `sqlc generate` bersih; **golang-migrate asli** up/version/down gl_db
  (1..4, dirty=false, 13+13 partisi, down→0 schema); pola `WithOrgScope` (set_config tx-local)
  isolasi RLS benar & tidak bocor lintas-koneksi; `go mod tidy && go vet && go build` = **BUILD_OK**.

## Production-readiness roadmap: SELESAI (6/6)
Hardening → Seed → Partition → Benchmark → Backup/DR → Go repo layer. Semua terverifikasi.

---

## [000003 / 000002] Review Remediation — 2026-08-06

Menindaklanjuti `db/REVIEW.md`. Ditambahkan sebagai migrasi additive `*_review_remediation`
(nomor **000003** untuk DB berpartisi yang sudah punya 000002; **000002** untuk DB lain —
golang-migrate mengizinkan gap versi). Seluruhnya reversible dan terverifikasi di PostgreSQL 16.

### M1 — Kebocoran RLS pada partisi (Keamanan) — RESOLVED
- **Masalah:** policy RLS pada tabel induk berpartisi tidak berlaku saat partisi diakses langsung
  (terbukti: `app_role` scope org1 membaca partisi langsung → 2 baris lintas-org).
- **Perbaikan:** `ENABLE ROW LEVEL SECURITY` + policy `scope_isolation` pada **setiap partisi**
  (gl.journal_entry, ap.ap_invoice, ar.ar_invoice, audit.audit_log, inv.stock_movement).
  Ditambah fungsi reusable `public.enable_partition_rls(regclass)` yang **wajib dipanggil oleh
  job provisioning partisi** (pg_partman) untuk setiap partisi baru.
- **Strategi grant (WAJIB di runbook operasional):** aplikasi konek sebagai **role non-owner
  tanpa BYPASSRLS**. Berikan privilege pada tabel **induk** (akses normal selalu via induk).
  RLS per-partisi menjadi defense-in-depth bila ada akses langsung ke partisi.
- **Regression:** `db/tests/rls_regression.sql` → parent 1/0 & direct-partition 1/0
  (visible/cross-tenant). **PASS.**

### M2 — Keunikan nomor dokumen dijaga database (bukan aplikasi) — RESOLVED
- **Keputusan:** pendekatan **Document Number Registry** (bukan enforcement di aplikasi).
- **Implementasi:** tabel non-partisi `gl.document_number_registry`, `ap.document_number_registry`,
  `ar.document_number_registry` dengan `UNIQUE (organization_id, document_type, fiscal_year,
  document_number)` + trigger `BEFORE INSERT` auto-populate (`tg_register_journal_no` /
  `tg_register_invoice_no`) pada tabel transaksi. Duplikat nomor dalam (org, fiscal_year) yang
  sama **ditolak database** meski tabel transaksi tetap dipartisi (partition pruning tetap utuh).
- **fiscal_year** diturunkan dari **tahun kalender** `posting_date`/`invoice_date` (proxy; jika
  fiscal year organisasi ≠ tahun kalender, sesuaikan derivasi di revisi berikutnya).
- **Payment/Receipt Number** (`trs.payment.payment_no`) dan **Tax Number** (`tax.tax_transaction.efaktur_no`)
  **sudah dijaga database** via `UNIQUE` (tabel tidak dipartisi) → tidak perlu registry.
- **Regression:** `db/tests/document_number_uniqueness.sql` → duplikat same-year diblok,
  cross-year diizinkan, registry terisi otomatis. **PASS.**

### N1 — Indeks FK yang belum ada — RESOLVED
- Ditambahkan: `idx_stock_balance_warehouse`, `idx_stock_movement_warehouse`,
  `idx_stock_reservation_warehouse` (inventory_db); `idx_elimination_entry_rule` (consolidation_db);
  `idx_bank_stmt_line_payment` (treasury_db).

### N2 — Konsistensi penamaan — RESOLVED
- `coa_db`: `fk_finorg_*`/`idx_finorg_*` → `fk_financial_organization_*`/`idx_financial_organization_*`.
- `dimension_db`: `fk_cca_*`/`fk_car_*`/`fk_pca_*` dan `idx_*` disingkat → nama penuh
  `cost_center_assignment`/`cost_allocation_rule`/`profit_center_assignment`.

### N3 — GIN pada JSONB yang di-query — RESOLVED
- `idx_customer_segment_criteria` (crm_db), `idx_asset_specification` (asset_db).

### N4 — Indeks redundan — RESOLVED (32 di-drop)
- Di-drop 32 indeks single-kolom yang duplikat dengan **leading column** composite UNIQUE
  (mis. `idx_journal_line_entry` tercakup `uq_journal_line (journal_entry_id, line_no)`).
  Mengurangi write-amplification tanpa kehilangan kemampuan lookup/FK-check.
- Indeks parsial `*_active (organization_id) WHERE deleted_at IS NULL` **DIPERTAHANKAN**:
  analisis menunjukkan bermanfaat untuk query "list aktif per org" (berbeda dari UNIQUE
  `(organization_id, code)`), jadi **bukan** redundan.

### Regression tests & benchmark (baru)
- `db/tests/rls_regression.sql`, `db/tests/optimistic_lock_regression.sql`,
  `db/tests/document_number_uniqueness.sql` — self-asserting (RAISE EXCEPTION on fail).
- `db/benchmark/explain_critical_queries.sql` — EXPLAIN ANALYZE GL/AP/AR.
  **Temuan:** query dengan predikat kunci partisi (`posting_date`/`invoice_date`) → **pruning ke 1
  partisi**; query tanpa kunci partisi (mis. hanya `fiscal_period_id`/`status`) → scan seluruh
  partisi (Append). **Rekomendasi:** sertakan predikat tanggal pada query pelaporan kritikal.

### Verifikasi
- UP 30/30 PASS (114 file migrasi). DOWN 30/30 PASS **dengan data** (fully reversible).
- Ketiga regression test PASS pada DB bersih; down+re-up remediation OK (gl/coa/sales spot-check).

### Catatan Expand–Contract untuk migrasi berikutnya (O4)
- Tambah kolom `NULLABLE` → backfill → set `NOT NULL`/constraint di migrasi terpisah.
- Buat indeks besar dengan `CREATE INDEX CONCURRENTLY` (file terpisah, tanpa statement lain,
  karena tidak transaksional). Hindari DROP/rename destruktif pada fase expand.
