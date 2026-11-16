-- =============================================================================
-- Per-database least-privilege grants (run INSIDE each service DB, as owner).
--   psql -U postgres -d gl_db -f db/ops/apply_grants.sql   (repeat per DB)
--
-- Strategy (REVIEW M1/O1): grant on PARENT/regular tables only — NOT on child
-- partitions — so the app accesses partitioned tables only via the parent.
-- Combined with per-partition RLS (000003), direct partition access is doubly
-- protected (no privilege + policy). PUBLIC privileges are revoked.
-- =============================================================================

DO $$
DECLARE s text; t text; r text;
BEGIN
  -- schemas
  FOR s IN SELECT nspname FROM pg_namespace
           WHERE nspname NOT LIKE 'pg\_%' AND nspname NOT IN ('information_schema','public') LOOP
    EXECUTE format('REVOKE ALL ON SCHEMA %I FROM PUBLIC', s);
    EXECUTE format('GRANT USAGE ON SCHEMA %I TO erp_app_rw, erp_app_ro', s);
    -- future tables created by either current_user (dev/superuser) or erp_migrator (prod) inherit grants
    FOR r IN SELECT DISTINCT rolname FROM pg_roles WHERE rolname IN (current_user, 'erp_migrator') LOOP
      IF s = 'audit' THEN
        EXECUTE format('ALTER DEFAULT PRIVILEGES FOR ROLE %I IN SCHEMA %I GRANT SELECT, INSERT ON TABLES TO erp_app_rw', r, s);
      ELSE
        EXECUTE format('ALTER DEFAULT PRIVILEGES FOR ROLE %I IN SCHEMA %I GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO erp_app_rw', r, s);
      END IF;
      EXECUTE format('ALTER DEFAULT PRIVILEGES FOR ROLE %I IN SCHEMA %I GRANT SELECT ON TABLES TO erp_app_ro', r, s);
    END LOOP;
  END LOOP;

  -- tables: parents ('p') + regular tables ('r'); EXCLUDE child partitions
  FOR t IN SELECT format('%I.%I', n.nspname, c.relname)
           FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
           WHERE c.relkind IN ('r','p') AND NOT c.relispartition
             AND n.nspname NOT LIKE 'pg\_%' AND n.nspname NOT IN ('information_schema','public') LOOP
    EXECUTE format('REVOKE ALL ON %s FROM PUBLIC', t);
    IF t = 'audit.audit_log' THEN
      -- audit.audit_log is strictly append-only (BR-826..835): no UPDATE/DELETE/TRUNCATE for app roles
      EXECUTE format('REVOKE UPDATE, DELETE, TRUNCATE ON %s FROM erp_app_rw, erp_app_ro', t);
      EXECUTE format('GRANT SELECT, INSERT ON %s TO erp_app_rw', t);
    ELSE
      EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %s TO erp_app_rw', t);
    END IF;
    EXECUTE format('GRANT SELECT ON %s TO erp_app_ro', t);
  END LOOP;

  -- RLS helper must be callable by app roles
  EXECUTE 'GRANT EXECUTE ON FUNCTION public.current_org_scope() TO erp_app_rw, erp_app_ro';
END $$;
