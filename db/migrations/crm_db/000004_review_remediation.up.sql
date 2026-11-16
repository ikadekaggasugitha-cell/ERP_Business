-- crm_db — Phase 1 remediation (review_remediation)

CREATE INDEX IF NOT EXISTS idx_crm_activity_task_related ON crm.activity_task (related_type, related_ref);

CREATE INDEX IF NOT EXISTS idx_crm_interaction_occurred ON crm.interaction (organization_id, occurred_at);

DO $$
DECLARE m date; has_rls boolean;
BEGIN
  FOR m IN SELECT generate_series('2025-01-01'::date, '2028-12-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('crm.outbox'::regclass, m);
  END LOOP;
  IF to_regclass('crm.outbox_default') IS NULL THEN
    EXECUTE 'CREATE TABLE crm.outbox_default PARTITION OF crm.outbox DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'crm' AND c.relname = 'outbox';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE crm.outbox_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON crm.outbox_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
END $$;
