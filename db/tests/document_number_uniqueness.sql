-- =============================================================================
-- Regression test: Document Number Uniqueness (M2 registry + auto trigger)
-- Target DB: gl_db (after 000003). Run: psql -d gl_db -f this
-- Proves: duplicate journal_no in same (org, fiscal_year) is REJECTED by the DB
-- across partitions; the same number in a different fiscal_year is allowed.
-- =============================================================================
\set ON_ERROR_STOP on

DO $$
DECLARE dup_blocked boolean := false;
BEGIN
  -- first issue (Aug 2026)
  INSERT INTO gl.journal_entry (organization_id,journal_no,financial_organization_id,ledger_id,fiscal_period_id,posting_date,journal_source,currency_code,correlation_id,created_by)
  VALUES ('11111111-1111-1111-1111-111111111111','DUP-1','f0000000-0000-0000-0000-000000000001','1ed00000-0000-0000-0000-000000000001','f9000000-0000-0000-0000-000000000001','2026-08-10','Manual','IDR','c0000000-0000-0000-0000-000000000001','99999999-9999-9999-9999-999999999999');

  -- duplicate: same org + same journal_no + same fiscal_year (different partition)
  BEGIN
    INSERT INTO gl.journal_entry (organization_id,journal_no,financial_organization_id,ledger_id,fiscal_period_id,posting_date,journal_source,currency_code,correlation_id,created_by)
    VALUES ('11111111-1111-1111-1111-111111111111','DUP-1','f0000000-0000-0000-0000-000000000001','1ed00000-0000-0000-0000-000000000001','f9000000-0000-0000-0000-000000000001','2026-09-10','Manual','IDR','c0000000-0000-0000-0000-000000000001','99999999-9999-9999-9999-999999999999');
    RAISE EXCEPTION 'DOCNO FAIL: duplicate journal_no accepted in same fiscal_year (registry did not enforce)';
  EXCEPTION WHEN unique_violation THEN
    dup_blocked := true;
  END;
  IF NOT dup_blocked THEN RAISE EXCEPTION 'DOCNO FAIL: expected unique_violation on duplicate'; END IF;

  -- same journal_no but different fiscal_year (2027) must be allowed
  INSERT INTO gl.journal_entry (organization_id,journal_no,financial_organization_id,ledger_id,fiscal_period_id,posting_date,journal_source,currency_code,correlation_id,created_by)
  VALUES ('11111111-1111-1111-1111-111111111111','DUP-1','f0000000-0000-0000-0000-000000000001','1ed00000-0000-0000-0000-000000000001','f9000000-0000-0000-0000-000000000001','2027-01-10','Manual','IDR','c0000000-0000-0000-0000-000000000001','99999999-9999-9999-9999-999999999999');

  RAISE NOTICE 'PASS document_number_uniqueness: same-year duplicate blocked by DB; cross-year allowed';
END $$;

-- registry reflects both issued numbers (2026 + 2027)
DO $$
DECLARE c int;
BEGIN
  SELECT count(*) INTO c FROM gl.document_number_registry
   WHERE organization_id='11111111-1111-1111-1111-111111111111' AND document_type='JournalEntry' AND document_number='DUP-1';
  IF c <> 2 THEN RAISE EXCEPTION 'DOCNO FAIL: registry rows for DUP-1 = % (want 2)', c; END IF;
  RAISE NOTICE 'PASS registry populated automatically: DUP-1 rows=% (2026,2027)', c;
END $$;
