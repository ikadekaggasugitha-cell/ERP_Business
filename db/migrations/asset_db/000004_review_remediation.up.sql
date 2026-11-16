-- asset_db — Phase 1 remediation (review_remediation)

CREATE INDEX IF NOT EXISTS idx_asset_barcode ON ast.asset (organization_id, barcode) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_asset_movement_status ON ast.asset_movement (organization_id, status);

DO $$
DECLARE m date; has_rls boolean;
BEGIN
  FOR m IN SELECT generate_series('2025-01-01'::date, '2028-12-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('ast.outbox'::regclass, m);
  END LOOP;
  IF to_regclass('ast.outbox_default') IS NULL THEN
    EXECUTE 'CREATE TABLE ast.outbox_default PARTITION OF ast.outbox DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'ast' AND c.relname = 'outbox';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE ast.outbox_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON ast.outbox_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
END $$;
