-- procurement_db — Phase 1 remediation (masterdata_and_remediation)

CREATE TABLE IF NOT EXISTS pr.masterdata_cache (
    organization_id uuid         NOT NULL,
    party_id        uuid         NOT NULL,
    code            varchar(50)  NOT NULL DEFAULT '',
    name            varchar(200) NOT NULL DEFAULT '',
    party_type      varchar(20)  NOT NULL DEFAULT '',
    updated_at      timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT pk_pr_masterdata_cache PRIMARY KEY (organization_id, party_id)
);
CREATE INDEX IF NOT EXISTS idx_pr_masterdata_cache_type ON pr.masterdata_cache (organization_id, party_type);
ALTER TABLE pr.masterdata_cache ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS scope_isolation ON pr.masterdata_cache;
CREATE POLICY scope_isolation ON pr.masterdata_cache
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));

ALTER TABLE gr.goods_receipt ADD CONSTRAINT ck_goods_receipt_status CHECK (status IN ('Draft','Received','Inspected','Accepted','Cancelled'));

ALTER TABLE pr.pr_line ADD CONSTRAINT ck_pr_line_qty CHECK (request_qty >= 0);

ALTER TABLE po.po_line ADD CONSTRAINT ck_po_line_qty CHECK (order_qty >= 0 AND received_qty >= 0 AND unit_price >= 0);

DO $$
DECLARE m date; has_rls boolean;
BEGIN
  FOR m IN SELECT generate_series('2025-01-01'::date, '2028-12-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('pr.outbox'::regclass, m);
  END LOOP;
  IF to_regclass('pr.outbox_default') IS NULL THEN
    EXECUTE 'CREATE TABLE pr.outbox_default PARTITION OF pr.outbox DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'pr' AND c.relname = 'outbox';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE pr.outbox_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON pr.outbox_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
END $$;
