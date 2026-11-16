-- inventory_db — Phase 1 remediation (masterdata_and_remediation)

CREATE TABLE IF NOT EXISTS inv.masterdata_cache (
    organization_id uuid         NOT NULL,
    party_id        uuid         NOT NULL,
    code            varchar(50)  NOT NULL DEFAULT '',
    name            varchar(200) NOT NULL DEFAULT '',
    party_type      varchar(20)  NOT NULL DEFAULT '',
    updated_at      timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT pk_inv_masterdata_cache PRIMARY KEY (organization_id, party_id)
);
CREATE INDEX IF NOT EXISTS idx_inv_masterdata_cache_type ON inv.masterdata_cache (organization_id, party_type);
ALTER TABLE inv.masterdata_cache ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS scope_isolation ON inv.masterdata_cache;
CREATE POLICY scope_isolation ON inv.masterdata_cache
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));

ALTER TABLE inv.stock_balance ADD CONSTRAINT ck_stock_balance_nonneg CHECK (on_hand_qty >= 0 AND reserved_qty >= 0);

DO $$
DECLARE m date; has_rls boolean;
BEGIN
  FOR m IN SELECT generate_series('2025-01-01'::date, '2028-12-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('inv.stock_movement'::regclass, m);
    PERFORM public.create_monthly_partition('inv.outbox'::regclass, m);
  END LOOP;
  IF to_regclass('inv.stock_movement_default') IS NULL THEN
    EXECUTE 'CREATE TABLE inv.stock_movement_default PARTITION OF inv.stock_movement DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'inv' AND c.relname = 'stock_movement';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE inv.stock_movement_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON inv.stock_movement_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
  IF to_regclass('inv.outbox_default') IS NULL THEN
    EXECUTE 'CREATE TABLE inv.outbox_default PARTITION OF inv.outbox DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'inv' AND c.relname = 'outbox';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE inv.outbox_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON inv.outbox_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
END $$;
