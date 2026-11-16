-- =============================================================================
-- Simple performance benchmark — EXPLAIN ANALYZE on critical GL/AP/AR queries.
-- Verifies partition pruning and index usage. Run against the same server:
--   psql -U postgres -f db/benchmark/explain_critical_queries.sql
-- Seeds throwaway data (BM-* numbers). Intended for a disposable environment.
-- =============================================================================
\set ON_ERROR_STOP on
\timing on

-- ======================= GENERAL LEDGER (gl_db) =============================
\c gl_db
INSERT INTO gl.journal_entry (organization_id,journal_no,financial_organization_id,ledger_id,fiscal_period_id,posting_date,journal_source,currency_code,correlation_id,created_by)
SELECT '11111111-1111-1111-1111-111111111111','BM-'||g,
       'f0000000-0000-0000-0000-000000000001','1ed00000-0000-0000-0000-000000000001',
       (ARRAY['f9000000-0000-0000-0000-000000000001','f9000000-0000-0000-0000-000000000002',
              'f9000000-0000-0000-0000-000000000003','f9000000-0000-0000-0000-000000000004']::uuid[])[1 + (g % 4)],
       DATE '2026-01-01' + (g % 360), 'Manual','IDR','c0000000-0000-0000-0000-000000000001',
       '99999999-9999-9999-9999-999999999999'
FROM generate_series(1,3000) g;
ANALYZE gl.journal_entry;

\echo '--- GL Q1: partition pruning by posting_date range (expect only *_y2026m08 scanned) ---'
EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT count(*) FROM gl.journal_entry
WHERE organization_id='11111111-1111-1111-1111-111111111111'
  AND posting_date BETWEEN '2026-08-01' AND '2026-08-31';

\echo '--- GL Q2: by (organization_id, fiscal_period_id) — expect idx_journal_entry_org_period ---'
EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT count(*) FROM gl.journal_entry
WHERE organization_id='11111111-1111-1111-1111-111111111111'
  AND fiscal_period_id='f9000000-0000-0000-0000-000000000002';

-- ======================= ACCOUNTS PAYABLE (ap_db) ==========================
\c ap_db
INSERT INTO ap.ap_invoice (organization_id,invoice_no,vendor_id,invoice_date,due_date,currency_code,total_amount,outstanding_amount,status,created_by)
SELECT '11111111-1111-1111-1111-111111111111','BM-AP-'||g,
       (ARRAY['ced00000-0000-0000-0000-000000000001','ced00000-0000-0000-0000-000000000002']::uuid[])[1+(g%2)],
       DATE '2026-01-01' + (g % 360), DATE '2026-02-01' + (g % 360),'IDR',1000,
       CASE WHEN g%3=0 THEN 0 ELSE 1000 END,
       (ARRAY['Recorded','PartiallyPaid','Paid']::varchar[])[1+(g%3)],
       '99999999-9999-9999-9999-999999999999'
FROM generate_series(1,2000) g;
ANALYZE ap.ap_invoice;

\echo '--- AP Q1: outstanding by vendor (expect idx_ap_invoice_vendor) ---'
EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT count(*) FROM ap.ap_invoice
WHERE vendor_id='ced00000-0000-0000-0000-000000000001' AND outstanding_amount > 0;

\echo '--- AP Q2: partition pruning by invoice_date range ---'
EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT count(*) FROM ap.ap_invoice
WHERE organization_id='11111111-1111-1111-1111-111111111111'
  AND invoice_date BETWEEN '2026-08-01' AND '2026-08-31';

-- ======================= ACCOUNTS RECEIVABLE (ar_db) =======================
\c ar_db
INSERT INTO ar.ar_invoice (organization_id,invoice_no,customer_id,invoice_date,due_date,currency_code,total_amount,outstanding_amount,status,created_by)
SELECT '11111111-1111-1111-1111-111111111111','BM-AR-'||g,
       (ARRAY['c05a0000-0000-0000-0000-000000000001','c05a0000-0000-0000-0000-000000000002']::uuid[])[1+(g%2)],
       DATE '2026-01-01' + (g % 360), DATE '2026-02-01' + (g % 360),'IDR',1000,
       CASE WHEN g%3=0 THEN 0 ELSE 1000 END,
       (ARRAY['Issued','PartiallyPaid','Overdue']::varchar[])[1+(g%3)],
       '99999999-9999-9999-9999-999999999999'
FROM generate_series(1,2000) g;
ANALYZE ar.ar_invoice;

\echo '--- AR Q1: outstanding by customer + status (expect idx_ar_invoice_status/customer) ---'
EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT count(*) FROM ar.ar_invoice
WHERE organization_id='11111111-1111-1111-1111-111111111111' AND status='Overdue';

\echo '--- AR Q2: partition pruning by invoice_date range ---'
EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT count(*) FROM ar.ar_invoice
WHERE customer_id='c05a0000-0000-0000-0000-000000000001'
  AND invoice_date BETWEEN '2026-08-01' AND '2026-08-31';
