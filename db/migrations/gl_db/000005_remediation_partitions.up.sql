-- gl_db — Phase 1 remediation (remediation_partitions)

DO $$
DECLARE m date; has_rls boolean;
BEGIN
  FOR m IN SELECT generate_series('2025-01-01'::date, '2028-12-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('gl.journal_entry'::regclass, m);
    PERFORM public.create_monthly_partition('gl.journal_line'::regclass, m);
    PERFORM public.create_monthly_partition('gl.outbox'::regclass, m);
  END LOOP;
  IF to_regclass('gl.journal_entry_default') IS NULL THEN
    EXECUTE 'CREATE TABLE gl.journal_entry_default PARTITION OF gl.journal_entry DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'gl' AND c.relname = 'journal_entry';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE gl.journal_entry_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON gl.journal_entry_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
  IF to_regclass('gl.journal_line_default') IS NULL THEN
    EXECUTE 'CREATE TABLE gl.journal_line_default PARTITION OF gl.journal_line DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'gl' AND c.relname = 'journal_line';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE gl.journal_line_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON gl.journal_line_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
  IF to_regclass('gl.outbox_default') IS NULL THEN
    EXECUTE 'CREATE TABLE gl.outbox_default PARTITION OF gl.outbox DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'gl' AND c.relname = 'outbox';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE gl.outbox_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON gl.outbox_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
END $$;
