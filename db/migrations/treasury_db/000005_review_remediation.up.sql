-- treasury_db — Phase 1 remediation (review_remediation)

CREATE INDEX IF NOT EXISTS idx_bank_stmt_line_payment ON trs.bank_statement_line (matched_payment_id);

CREATE INDEX IF NOT EXISTS idx_bank_account_gl ON trs.bank_account (gl_account_id);

DO $$
DECLARE m date; has_rls boolean;
BEGIN
  FOR m IN SELECT generate_series('2025-01-01'::date, '2028-12-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('trs.outbox'::regclass, m);
  END LOOP;
  IF to_regclass('trs.outbox_default') IS NULL THEN
    EXECUTE 'CREATE TABLE trs.outbox_default PARTITION OF trs.outbox DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'trs' AND c.relname = 'outbox';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE trs.outbox_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON trs.outbox_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
END $$;
