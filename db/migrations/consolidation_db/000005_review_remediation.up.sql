-- consolidation_db — Phase 1 remediation (review_remediation)

CREATE INDEX IF NOT EXISTS idx_elimination_entry_rule ON cons.elimination_entry (elimination_rule_id);

CREATE INDEX IF NOT EXISTS idx_consolidation_run_status ON cons.consolidation_run (organization_id, status);

DO $$
DECLARE m date; has_rls boolean;
BEGIN
  FOR m IN SELECT generate_series('2025-01-01'::date, '2028-12-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('cons.outbox'::regclass, m);
  END LOOP;
  IF to_regclass('cons.outbox_default') IS NULL THEN
    EXECUTE 'CREATE TABLE cons.outbox_default PARTITION OF cons.outbox DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'cons' AND c.relname = 'outbox';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE cons.outbox_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON cons.outbox_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
END $$;
