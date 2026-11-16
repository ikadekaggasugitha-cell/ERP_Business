-- audit_db — Phase 1 remediation (review_remediation)

CREATE OR REPLACE FUNCTION public.enable_partition_rls(part regclass)
RETURNS void LANGUAGE plpgsql AS $fn$
BEGIN
  EXECUTE format('ALTER TABLE %s ENABLE ROW LEVEL SECURITY', part);
  EXECUTE format('DROP POLICY IF EXISTS scope_isolation ON %s', part);
  EXECUTE format('CREATE POLICY scope_isolation ON %s USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))', part);
END;
$fn$;

DO $$
DECLARE part regclass;
BEGIN
  FOR part IN SELECT inhrelid::regclass FROM pg_inherits WHERE inhparent = 'audit.audit_log'::regclass LOOP
    PERFORM public.enable_partition_rls(part);
  END LOOP;
END $$;

DO $$
DECLARE m date; has_rls boolean;
BEGIN
  FOR m IN SELECT generate_series('2025-01-01'::date, '2028-12-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('audit.audit_log'::regclass, m);
  END LOOP;
  IF to_regclass('audit.audit_log_default') IS NULL THEN
    EXECUTE 'CREATE TABLE audit.audit_log_default PARTITION OF audit.audit_log DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'audit' AND c.relname = 'audit_log';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE audit.audit_log_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON audit.audit_log_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
END $$;
