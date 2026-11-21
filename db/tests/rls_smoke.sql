-- =============================================================================
-- Generic RLS smoke test — runs against ANY ERP database (db/tests/rls_smoke.sql)
-- Target: any <name>_db after its migrations. Run:
--   psql "$PGURL/<db>?sslmode=disable" -f db/tests/rls_smoke.sql
--
-- Asserts, for every app-schema table that has an organization_id column and is
-- not a partition child (the parent is the enforcement point), excluding
-- infrastructure tables — `outbox` is drained cross-org by design (AGENTS.md
-- transactional-outbox guardrail) and `processed_event` is global bookkeeping:
--   (A) RLS enabled AND at least one policy exists.
--   (B) FAIL-CLOSED: as a non-owner role with app.org_scope = '{}', zero rows
--       are visible. System-org rows are tolerated, in case a policy hardcodes
--       a system-org bypass (current policies do not).
--   (C) CROSS-TENANT: per data-bearing table, scope to the org owning the most
--       rows; zero rows of any other organization may be visible.
--   (D) POSITIVE VISIBILITY: scoped to that same org, the number of visible
--       own-org rows equals the owner-side count — a deny-all (USING (false))
--       or over-restrictive policy regresses here, not just leaky ones.
--
-- Self-asserting: RAISES EXCEPTION on any violation; prints PASS otherwise
-- (ON_ERROR_STOP makes the exception fail the psql invocation).
-- State-independent: reads the live catalog and existing data only; safe on
-- fresh or smoke-contaminated databases; re-runnable. A dataless DB passes
-- checks B/C vacuously — the NOTICE line reports how many tables carried data.
-- CI note: db-regression applies db/seeds/<db>/seed.sql when present plus the
-- synthetic tenant fixtures in db/tests/rls_smoke_tenants.sql (two orgs, one
-- row each), so seeded databases exercise B/C/D behaviorally; databases
-- without tenant rows stay vacuous by design and pass with a NOTICE.
-- =============================================================================
\set ON_ERROR_STOP on

DO $smoke$
DECLARE
  sys_org  CONSTANT uuid := '00000000-0000-0000-0000-000000000000';
  r        record;
  v_tmpsch text;
  v_scope  uuid;
  v_orgs   bigint;
  v_own    bigint;
  cnt      bigint;
  n_data   int := 0;
  n_multi  int := 0;
BEGIN
  -- Non-owner execution context so RLS is actually enforced (checks B/C).
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_role') THEN
    CREATE ROLE app_role NOLOGIN;
  END IF;
  FOR r IN
    SELECT nspname FROM pg_namespace
    WHERE nspname NOT IN ('pg_catalog', 'information_schema', 'public')
  LOOP
    EXECUTE format('GRANT USAGE ON SCHEMA %I TO app_role', r.nspname);
    EXECUTE format('GRANT SELECT ON ALL TABLES IN SCHEMA %I TO app_role', r.nspname);
  END LOOP;

  CREATE TEMP TABLE _rls_candidate AS
    SELECT n.nspname AS schemaname, c.relname AS tablename, c.oid AS reloid,
           NULL::uuid AS scope_org, 0::bigint AS org_count, 0::bigint AS own_rows
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relkind IN ('r', 'p')
      AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'public')
      AND c.relname NOT IN ('outbox', 'processed_event')
      AND EXISTS (SELECT 1 FROM information_schema.columns col
                  WHERE col.table_schema = n.nspname
                    AND col.table_name   = c.relname
                    AND col.column_name  = 'organization_id')
      AND NOT EXISTS (SELECT 1 FROM pg_inherits i
                      JOIN pg_class p ON p.oid = i.inhparent
                      WHERE i.inhrelid = c.oid AND p.relkind = 'p');

  -- Checks B/C read this temp table as app_role: grant the temp schema first.
  SELECT nspname INTO v_tmpsch FROM pg_namespace WHERE oid = pg_my_temp_schema();
  EXECUTE format('GRANT USAGE ON SCHEMA %I TO app_role', v_tmpsch);
  EXECUTE format('GRANT SELECT ON %I._rls_candidate TO app_role', v_tmpsch);

  -- (A) Catalog invariants: every tenanted parent table must be RLS-enforced.
  FOR r IN SELECT * FROM _rls_candidate ORDER BY 1, 2 LOOP
    IF NOT (SELECT relrowsecurity FROM pg_class WHERE oid = r.reloid) THEN
      RAISE EXCEPTION 'RLS SMOKE FAIL (A): %.% has organization_id but RLS is not enabled', r.schemaname, r.tablename;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policy WHERE polrelid = r.reloid) THEN
      RAISE EXCEPTION 'RLS SMOKE FAIL (A): %.% is RLS-enabled but has no policy', r.schemaname, r.tablename;
    END IF;
  END LOOP;
  RAISE NOTICE 'PASS (A): RLS + policy present on all % tenanted table(s)', (SELECT count(*) FROM _rls_candidate);

  -- Data calibration (as owner: sees everything) for checks B/C.
  FOR r IN SELECT * FROM _rls_candidate ORDER BY 1, 2 LOOP
    EXECUTE format(
      'SELECT organization_id FROM %I.%I
       WHERE organization_id IS NOT NULL AND organization_id <> %L
       GROUP BY 1 ORDER BY count(*) DESC LIMIT 1',
      r.schemaname, r.tablename, sys_org) INTO v_scope;
    EXECUTE format(
      'SELECT count(DISTINCT organization_id) FROM %I.%I
       WHERE organization_id IS NOT NULL AND organization_id <> %L',
      r.schemaname, r.tablename, sys_org) INTO v_orgs;
    v_own := 0;
    IF v_scope IS NOT NULL THEN
      EXECUTE format(
        'SELECT count(*) FROM %I.%I WHERE organization_id = %L',
        r.schemaname, r.tablename, v_scope) INTO v_own;
    END IF;
    UPDATE _rls_candidate
       SET scope_org = v_scope, org_count = v_orgs, own_rows = v_own
     WHERE reloid = r.reloid;
  END LOOP;

  -- From here on everything runs as app_role so RLS policies bite.
  PERFORM set_config('role', 'app_role', true);
  PERFORM set_config('app.org_scope', '{}', true);

  -- (B) Fail-closed: an empty scope must hide every non-system-org row.
  FOR r IN SELECT * FROM _rls_candidate ORDER BY 1, 2 LOOP
    EXECUTE format(
      'SELECT count(*) FROM %I.%I WHERE organization_id IS DISTINCT FROM %L',
      r.schemaname, r.tablename, sys_org) INTO cnt;
    IF cnt <> 0 THEN
      RAISE EXCEPTION 'RLS SMOKE FAIL (B): % row(s) visible in %.% under empty org scope', cnt, r.schemaname, r.tablename;
    END IF;
  END LOOP;
  RAISE NOTICE 'PASS (B): fail-closed under empty org scope on all % table(s)', (SELECT count(*) FROM _rls_candidate);

  -- (C) Cross-tenant: scope to each data-bearing table's dominant org.
  FOR r IN SELECT * FROM _rls_candidate WHERE scope_org IS NOT NULL ORDER BY 1, 2 LOOP
    PERFORM set_config('app.org_scope', format('{%s}', r.scope_org), true);
    EXECUTE format(
      'SELECT count(*) FROM %I.%I
       WHERE organization_id IS DISTINCT FROM %L AND organization_id IS DISTINCT FROM %L',
      r.schemaname, r.tablename, r.scope_org, sys_org) INTO cnt;
    IF cnt <> 0 THEN
      RAISE EXCEPTION 'RLS SMOKE FAIL (C): % foreign-org row(s) visible in %.% scoped to %s', cnt, r.schemaname, r.tablename, r.scope_org;
    END IF;
    -- (D) Positive visibility: under the correct scope the dominant org must
    -- see EXACTLY its owner-side row count (deny-all / over-restrict catch).
    EXECUTE format(
      'SELECT count(*) FROM %I.%I WHERE organization_id = %L',
      r.schemaname, r.tablename, r.scope_org) INTO cnt;
    IF cnt <> r.own_rows THEN
      RAISE EXCEPTION 'RLS SMOKE FAIL (D): %.% shows % own-org row(s) under correct scope, expected %', r.schemaname, r.tablename, cnt, r.own_rows;
    END IF;
    n_data := n_data + 1;
    IF r.org_count > 1 THEN n_multi := n_multi + 1; END IF;
  END LOOP;

  IF n_data = 0 THEN
    RAISE NOTICE 'PASS (C/D): skipped — database holds no tenant rows (B/C/D vacuous)';
  ELSE
    RAISE NOTICE 'PASS (C): cross-tenant isolation on % data-bearing table(s) (% with rows from >= 2 orgs)', n_data, n_multi;
    RAISE NOTICE 'PASS (D): own-org visibility exact on % data-bearing table(s)', n_data;
  END IF;

  RAISE NOTICE 'RLS SMOKE PASS';
END
$smoke$;
