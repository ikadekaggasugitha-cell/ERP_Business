-- maintenance_db — Phase 1 remediation (partitions_extended)

DO $$
DECLARE m date; has_rls boolean;
BEGIN
  FOR m IN SELECT generate_series('2025-01-01'::date, '2028-12-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('mnt.outbox'::regclass, m);
  END LOOP;
  IF to_regclass('mnt.outbox_default') IS NULL THEN
    EXECUTE 'CREATE TABLE mnt.outbox_default PARTITION OF mnt.outbox DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'mnt' AND c.relname = 'outbox';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE mnt.outbox_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON mnt.outbox_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
END $$;
