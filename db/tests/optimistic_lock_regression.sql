-- =============================================================================
-- Regression test: Optimistic Locking (row_version via tg_touch_row)
-- Target DB: org_db (freshly migrated). Run: psql -d org_db -f this
-- Proves: (1) row_version auto-increments on UPDATE; (2) updated_at advances
-- across transactions; (3) a stale-version UPDATE affects 0 rows.
-- INSERT and UPDATE are separate (autocommit) statements so now() differs
-- between them (now() is constant within a single transaction).
-- =============================================================================
\set ON_ERROR_STOP on

-- txn 1: seed (created_at = now() of this txn)
INSERT INTO org.organization_type (id, organization_id, code, name, created_by)
VALUES ('0a0000aa-0000-0000-0000-0000000000aa','11111111-1111-1111-1111-111111111111','T-OL','Type OL','99999999-9999-9999-9999-999999999999');
INSERT INTO org.organization (id, organization_id, code, name, organization_type_id, path, effective_from, created_by)
VALUES ('0a000001-0000-0000-0000-000000000001','11111111-1111-1111-1111-111111111111','OL-1','Org OL',
        '0a0000aa-0000-0000-0000-0000000000aa','/ol','2026-01-01','99999999-9999-9999-9999-999999999999');

-- txn 2: update (updated_at = now() of a later txn -> strictly greater)
UPDATE org.organization SET name='Org OL v2' WHERE id='0a000001-0000-0000-0000-000000000001';

-- txn 3: assertions
DO $$
DECLARE v int; t0 timestamptz; t1 timestamptz; n int;
BEGIN
  SELECT row_version, created_at, updated_at INTO v, t0, t1
    FROM org.organization WHERE id='0a000001-0000-0000-0000-000000000001';
  IF v  <> 2   THEN RAISE EXCEPTION 'OPTLOCK FAIL: row_version=% (want 2)', v; END IF;
  IF t1 <= t0  THEN RAISE EXCEPTION 'OPTLOCK FAIL: updated_at(%) not > created_at(%)', t1, t0; END IF;

  -- stale update with the OLD version must match nothing
  UPDATE org.organization SET name='Org OL stale'
   WHERE id='0a000001-0000-0000-0000-000000000001' AND row_version = 1;
  GET DIAGNOSTICS n = ROW_COUNT;
  IF n <> 0 THEN RAISE EXCEPTION 'OPTLOCK FAIL: stale update affected % rows (want 0)', n; END IF;

  RAISE NOTICE 'PASS optimistic_lock_regression: row_version=2, updated_at>created_at, stale_rows=%', n;
END $$;
