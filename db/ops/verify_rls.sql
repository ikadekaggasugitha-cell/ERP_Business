-- =============================================================================
-- RLS enforcement check (REVIEW O1-O3). Read-only; raises an exception listing
-- every violation so a deploy pipeline or operator can gate on the exit code.
--   psql -v ON_ERROR_STOP=1 -d <db> -f db/ops/verify_rls.sql
-- Run per database (see deploy/verify-rls.sh for all 30).
-- =============================================================================
DO $$
DECLARE
  violations text[] := '{}';
  app_oids oid[];
  r record;
BEGIN
  -- Explicit membership closure. pg_has_role() is unusable here: it reports every
  -- superuser as a member of every role, which would flag the migration owner.
  WITH RECURSIVE m(oid) AS (
    SELECT oid FROM pg_roles WHERE rolname IN ('erp_app_rw', 'erp_app_ro')
    UNION
    SELECT a.member FROM pg_auth_members a JOIN m ON a.roleid = m.oid
  )
  SELECT coalesce(array_agg(oid), '{}') INTO app_oids FROM m;

  -- 1. Group roles and every member must be unable to bypass RLS.
  FOR r IN
    SELECT rolname FROM pg_roles
    WHERE (rolsuper OR rolbypassrls)
      AND (rolname = 'erp_migrator' OR oid = ANY (app_oids))
  LOOP
    violations := violations || format('role %s has SUPERUSER or BYPASSRLS', r.rolname);
  END LOOP;

  -- 2. The application must not own tables (owners bypass RLS unless FORCEd).
  FOR r IN
    SELECT n.nspname, c.relname, pg_get_userbyid(c.relowner) AS owner
    FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relkind IN ('r', 'p')
      AND n.nspname NOT IN ('pg_catalog', 'information_schema')
      AND n.nspname !~ '^pg_toast'
      AND c.relowner = ANY (app_oids)
  LOOP
    violations := violations || format('%I.%I is owned by app role %s', r.nspname, r.relname, r.owner);
  END LOOP;

  -- 3. Every tenant table (has organization_id) has RLS on. This includes
  --    partitions created later by pg_partman / maintain-partitions.sh (O3).
  --    Outbox tables are intentionally RLS-free (relay reads across tenants).
  FOR r IN
    SELECT n.nspname, c.relname
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN pg_attribute a ON a.attrelid = c.oid AND a.attname = 'organization_id' AND NOT a.attisdropped
    WHERE c.relkind IN ('r', 'p')
      AND n.nspname NOT IN ('pg_catalog', 'information_schema')
      AND c.relname !~ '^outbox'
      AND NOT c.relrowsecurity
  LOOP
    violations := violations || format('%I.%I has organization_id but RLS is disabled', r.nspname, r.relname);
  END LOOP;

  -- 4. RLS enabled without any policy silently denies everything.
  FOR r IN
    SELECT n.nspname, c.relname
    FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relkind IN ('r', 'p')
      AND c.relrowsecurity
      AND n.nspname NOT IN ('pg_catalog', 'information_schema')
      AND NOT EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid = c.oid)
  LOOP
    violations := violations || format('%I.%I has RLS enabled but no policy', r.nspname, r.relname);
  END LOOP;

  IF array_length(violations, 1) IS NOT NULL THEN
    RAISE EXCEPTION E'RLS verification failed (% issue(s)):\n  - %',
      array_length(violations, 1), array_to_string(violations, E'\n  - ');
  END IF;
  RAISE NOTICE 'RLS verification passed for database %', current_database();
END $$;
