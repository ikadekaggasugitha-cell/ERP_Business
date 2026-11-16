-- payroll_db — Phase 1 remediation (pii_and_partitions)

ALTER TABLE pay.payroll_result
    ADD COLUMN IF NOT EXISTS encrypted_gross TEXT,
    ADD COLUMN IF NOT EXISTS encrypted_net TEXT,
    ADD COLUMN IF NOT EXISTS encrypted_tax TEXT,
    ADD COLUMN IF NOT EXISTS pii_key_version VARCHAR(32);

DO $$
DECLARE m date; has_rls boolean;
BEGIN
  FOR m IN SELECT generate_series('2025-01-01'::date, '2028-12-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('pay.outbox'::regclass, m);
  END LOOP;
  IF to_regclass('pay.outbox_default') IS NULL THEN
    EXECUTE 'CREATE TABLE pay.outbox_default PARTITION OF pay.outbox DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'pay' AND c.relname = 'outbox';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE pay.outbox_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON pay.outbox_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
END $$;
