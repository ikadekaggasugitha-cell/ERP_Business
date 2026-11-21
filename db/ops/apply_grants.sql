-- =============================================================================
-- Per-database least-privilege grants (run INSIDE each service DB, as owner).
--   psql -U postgres -d gl_db -f db/ops/apply_grants.sql   (repeat per DB)
--
-- Strategy (REVIEW M1/O1): grant on PARENT/regular tables only — NOT on child
-- partitions — so the app accesses partitioned tables only via the parent.
-- Combined with per-partition RLS (000003), direct partition access is doubly
-- protected (no privilege + policy). PUBLIC privileges are revoked.
-- =============================================================================

CREATE TABLE IF NOT EXISTS public.idempotency_key (
    scope_key  text        NOT NULL,
    idem_key   text        NOT NULL,
    req_hash   text        NOT NULL DEFAULT '',
    done       boolean     NOT NULL DEFAULT FALSE,
    status     integer     NOT NULL DEFAULT 0,
    etag       text        NOT NULL DEFAULT '',
    body       bytea,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (scope_key, idem_key)
);

DO $$
DECLARE s text; t text; r text;
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'erp_migrator') THEN CREATE ROLE erp_migrator NOLOGIN NOSUPERUSER NOBYPASSRLS; END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'erp_app_rw')   THEN CREATE ROLE erp_app_rw   NOLOGIN NOSUPERUSER NOBYPASSRLS; END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'erp_app_ro')   THEN CREATE ROLE erp_app_ro   NOLOGIN NOSUPERUSER NOBYPASSRLS; END IF;

  REVOKE ALL ON TABLE public.idempotency_key FROM PUBLIC;
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE public.idempotency_key TO erp_app_rw;
  GRANT SELECT ON TABLE public.idempotency_key TO erp_app_ro;

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

  -- child partitions: explicitly revoke direct privileges so app roles must query via parent
  FOR t IN SELECT format('%I.%I', n.nspname, c.relname)
           FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
           WHERE c.relispartition
             AND n.nspname NOT LIKE 'pg\_%' AND n.nspname NOT IN ('information_schema','public') LOOP
    EXECUTE format('REVOKE ALL ON %s FROM PUBLIC, erp_app_rw, erp_app_ro', t);
  END LOOP;

  -- If public.create_monthly_partition exists, harden it so newly created child partitions also revoke direct grants
  IF to_regprocedure('public.create_monthly_partition(regclass,date)') IS NOT NULL THEN
    CREATE OR REPLACE FUNCTION public.create_monthly_partition(parent regclass, month date)
    RETURNS void LANGUAGE plpgsql AS $fn$
    DECLARE nsp text; base text; part text; s date; e date; has_rls boolean;
    BEGIN
      SELECT n.nspname, c.relname, c.relrowsecurity INTO nsp, base, has_rls
      FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE c.oid = parent;
      s := date_trunc('month', month)::date;
      e := (s + interval '1 month')::date;
      part := format('%s_y%sm%s', base, to_char(s,'YYYY'), to_char(s,'MM'));
      IF to_regclass(format('%I.%I', nsp, part)) IS NULL THEN
        EXECUTE format('CREATE TABLE %I.%I PARTITION OF %s FOR VALUES FROM (%L) TO (%L)', nsp, part, parent, s, e);
        IF has_rls THEN
          EXECUTE format('ALTER TABLE %I.%I ENABLE ROW LEVEL SECURITY', nsp, part);
          EXECUTE format('CREATE POLICY scope_isolation ON %I.%I USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))', nsp, part);
        END IF;
      END IF;
      IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'erp_app_rw') THEN
        EXECUTE format('REVOKE ALL ON %I.%I FROM PUBLIC, erp_app_rw, erp_app_ro', nsp, part);
      ELSE
        EXECUTE format('REVOKE ALL ON %I.%I FROM PUBLIC', nsp, part);
      END IF;
    END;
    $fn$;
  END IF;

  -- RLS helper must be callable by app roles
  IF to_regprocedure('public.current_org_scope()') IS NOT NULL THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.current_org_scope() TO erp_app_rw, erp_app_ro';
  END IF;
END $$;
