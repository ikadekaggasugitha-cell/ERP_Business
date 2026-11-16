-- =============================================================================
-- Regression test: Row-Level Security (M1 — parent + partition isolation)
-- Target DB: gl_db (after 000001..000003). Run: psql -d gl_db -f this
-- Self-asserting: RAISES EXCEPTION on any leak; prints PASS otherwise.
-- =============================================================================
\set ON_ERROR_STOP on

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='app_role') THEN CREATE ROLE app_role NOLOGIN; END IF;
END $$;
GRANT USAGE ON SCHEMA gl TO app_role;
GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA gl TO app_role;

-- Seed two organizations, same month (same partition) — as owner (RLS bypassed).
INSERT INTO gl.journal_entry (organization_id,journal_no,financial_organization_id,ledger_id,fiscal_period_id,posting_date,journal_source,currency_code,correlation_id,created_by)
VALUES ('11111111-1111-1111-1111-111111111111','JE-R1','f0000000-0000-0000-0000-000000000001','1ed00000-0000-0000-0000-000000000001','f9000000-0000-0000-0000-000000000001','2026-08-10','Manual','IDR','c0000000-0000-0000-0000-000000000001','99999999-9999-9999-9999-999999999999'),
       ('22222222-2222-2222-2222-222222222222','JE-R2','f0000000-0000-0000-0000-000000000001','1ed00000-0000-0000-0000-000000000001','f9000000-0000-0000-0000-000000000001','2026-08-11','Manual','IDR','c0000000-0000-0000-0000-000000000001','99999999-9999-9999-9999-999999999999');

-- Read as scoped app_role (scope = org1) via parent AND directly via partition.
SET ROLE app_role;
SELECT set_config('app.org_scope','{11111111-1111-1111-1111-111111111111}', false);
CREATE TEMP TABLE _rls_res AS SELECT
  (SELECT count(*) FROM gl.journal_entry) AS parent_all,
  (SELECT count(*) FROM gl.journal_entry WHERE organization_id='22222222-2222-2222-2222-222222222222') AS parent_other,
  (SELECT count(*) FROM gl.journal_entry_y2026m08) AS partition_all,
  (SELECT count(*) FROM gl.journal_entry_y2026m08 WHERE organization_id='22222222-2222-2222-2222-222222222222') AS partition_other;
RESET ROLE;

DO $$
DECLARE r record;
BEGIN
  SELECT * INTO r FROM _rls_res;
  IF r.parent_all    <> 1 THEN RAISE EXCEPTION 'RLS FAIL: parent visible=% (want 1)', r.parent_all; END IF;
  IF r.parent_other  <> 0 THEN RAISE EXCEPTION 'RLS FAIL: cross-tenant leak via parent=%', r.parent_other; END IF;
  IF r.partition_all <> 1 THEN RAISE EXCEPTION 'RLS FAIL: direct-partition visible=% (want 1; RLS not applied to partition!)', r.partition_all; END IF;
  IF r.partition_other <> 0 THEN RAISE EXCEPTION 'RLS FAIL: cross-tenant leak via direct partition=%', r.partition_other; END IF;
  RAISE NOTICE 'PASS rls_regression: parent 1/0, direct-partition 1/0 (visible/cross-tenant)';
END $$;
DROP TABLE _rls_res;
