-- =============================================================================
-- At-scale performance benchmark (~100k+ rows) — proves index usage + partition
-- pruning on the hot GL/AP/AR paths. Run on a disposable server:
--   psql -U postgres -f db/benchmark/benchmark_at_scale.sql
-- Seeds throwaway data; safe only on a scratch database.
-- =============================================================================
\set ON_ERROR_STOP on
\timing on
\set N 100000

-- ============================ GENERAL LEDGER ================================
\c gl_db
\echo '### Seeding GL: :N journal entries + 2x lines ...'
INSERT INTO gl.journal_entry (organization_id,journal_no,financial_organization_id,ledger_id,fiscal_period_id,posting_date,journal_source,currency_code,correlation_id,created_by,source_document_ref)
SELECT '11111111-1111-1111-1111-111111111111','BM-'||g,
       'f0000000-0000-0000-0000-000000000001','1ed00000-0000-0000-0000-000000000001',
       ('f9000000-0000-0000-0000-0000000000' || lpad(((g%12)+1)::text,2,'0'))::uuid,
       DATE '2026-01-01' + (g % 360), 'Manual','IDR','c0000000-0000-0000-0000-000000000001',
       '99999999-9999-9999-9999-999999999999', 'SRC-'||g
FROM generate_series(1, :N) g;

INSERT INTO gl.journal_line (organization_id,journal_entry_id,posting_date,line_no,account_id,debit_amount,credit_amount,functional_amount,created_by)
SELECT je.organization_id, je.id, je.posting_date, ln,
       ('acc00000-0000-0000-0000-0000000000' || lpad(((abs(hashtext(je.id::text)) % 20) + 1)::text,2,'0'))::uuid,
       CASE WHEN ln=1 THEN 100 ELSE 0 END, CASE WHEN ln=1 THEN 0 ELSE 100 END, 100,
       '99999999-9999-9999-9999-999999999999'
FROM gl.journal_entry je CROSS JOIN generate_series(1,2) ln;
ANALYZE gl.journal_entry; ANALYZE gl.journal_line;
\echo '### GL row counts:'
SELECT (SELECT count(*) FROM gl.journal_entry) AS journal_entries, (SELECT count(*) FROM gl.journal_line) AS journal_lines;

\echo '=== GL-A: selective drill-down by source_document_ref (expect Index Scan idx_journal_entry_source_doc) ==='
EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT id, journal_no FROM gl.journal_entry
WHERE organization_id='11111111-1111-1111-1111-111111111111' AND source_document_ref='SRC-50000';

\echo '=== GL-B: trial-balance aggregation over one month (expect prune to journal_line_y2026m08) ==='
EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT account_id, SUM(debit_amount), SUM(credit_amount)
FROM gl.journal_line
WHERE organization_id='11111111-1111-1111-1111-111111111111'
  AND posting_date BETWEEN '2026-08-01' AND '2026-08-31'
GROUP BY account_id;

-- ============================ ACCOUNTS PAYABLE ==============================
\c ap_db
\echo '### Seeding AP ...'
INSERT INTO ap.ap_invoice (organization_id,invoice_no,vendor_id,invoice_date,due_date,currency_code,total_amount,outstanding_amount,status,created_by)
SELECT '11111111-1111-1111-1111-111111111111','BM-AP-'||g,
       ('ced00000-0000-0000-0000-0000000000'||lpad(((g%20)+1)::text,2,'0'))::uuid,
       DATE '2026-01-01' + (g % 360), DATE '2026-02-01' + (g % 360),'IDR',1000,
       CASE WHEN g%3=0 THEN 0 ELSE 1000 END,
       (ARRAY['Recorded','PartiallyPaid','Paid']::varchar[])[1+(g%3)],
       '99999999-9999-9999-9999-999999999999'
FROM generate_series(1, :N/2) g;
ANALYZE ap.ap_invoice;

\echo '=== AP-A: open items by vendor (expect Index Scan idx_ap_invoice_vendor) ==='
EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT count(*), sum(outstanding_amount) FROM ap.ap_invoice
WHERE vendor_id='ced00000-0000-0000-0000-000000000005' AND outstanding_amount > 0;

\echo '=== AP-B: due within month (expect prune to ap_invoice_y2026m08) ==='
EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT count(*) FROM ap.ap_invoice
WHERE organization_id='11111111-1111-1111-1111-111111111111'
  AND invoice_date BETWEEN '2026-08-01' AND '2026-08-31';

-- ============================ ACCOUNTS RECEIVABLE ===========================
\c ar_db
\echo '### Seeding AR ...'
INSERT INTO ar.ar_invoice (organization_id,invoice_no,customer_id,invoice_date,due_date,currency_code,total_amount,outstanding_amount,status,created_by)
SELECT '11111111-1111-1111-1111-111111111111','BM-AR-'||g,
       ('c05a0000-0000-0000-0000-0000000000'||lpad(((g%20)+1)::text,2,'0'))::uuid,
       DATE '2026-01-01' + (g % 360), DATE '2026-02-01' + (g % 360),'IDR',1000,
       CASE WHEN g%3=0 THEN 0 ELSE 1000 END,
       (ARRAY['Issued','PartiallyPaid','Overdue']::varchar[])[1+(g%3)],
       '99999999-9999-9999-9999-999999999999'
FROM generate_series(1, :N/2) g;
ANALYZE ar.ar_invoice;

\echo '=== AR-A: aging by customer + overdue (expect Index Scan idx_ar_invoice_customer) ==='
EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT count(*), sum(outstanding_amount) FROM ar.ar_invoice
WHERE customer_id='c05a0000-0000-0000-0000-000000000005' AND status='Overdue';

-- ============================ OUTBOX RELAY ==================================
\c gl_db
\echo '### Seeding outbox (mostly published, few pending) ...'
INSERT INTO gl.outbox (aggregate_type,aggregate_id,event_type,payload,organization_id,correlation_id,created_at,published_at)
SELECT 'journal_entry', gen_random_uuid(), 'erp.gl.journal.posted.v1','{}','11111111-1111-1111-1111-111111111111',
       gen_random_uuid(), DATE '2026-08-01' + (g % 20),
       CASE WHEN g % 500 = 0 THEN NULL ELSE now() END      -- ~0.2% pending
FROM generate_series(1, :N/2) g;
ANALYZE gl.outbox;

\echo '=== OUTBOX: relay poll (expect Index Scan on partial idx_outbox_unpublished) ==='
EXPLAIN (ANALYZE, BUFFERS, TIMING OFF)
SELECT id, event_type FROM gl.outbox WHERE published_at IS NULL ORDER BY created_at LIMIT 100;
