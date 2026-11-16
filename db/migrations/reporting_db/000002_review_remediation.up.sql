-- reporting_db — Phase 1 remediation (review_remediation)

-- Fix nullable customer_id uniqueness (PostgreSQL 15+ NULLS NOT DISTINCT)
ALTER TABLE rpt.sales_summary DROP CONSTRAINT uq_sales_summary;
ALTER TABLE rpt.sales_summary
    ADD CONSTRAINT uq_sales_summary UNIQUE NULLS NOT DISTINCT (organization_id, period_year, period_month, customer_id);

CREATE INDEX IF NOT EXISTS idx_gl_balance_snapshot_account ON rpt.gl_balance_snapshot (organization_id, account_code);
CREATE INDEX IF NOT EXISTS idx_inventory_position_item ON rpt.inventory_position (organization_id, item_code);
CREATE INDEX IF NOT EXISTS idx_inventory_position_wh ON rpt.inventory_position (organization_id, warehouse_code);

CREATE OR REPLACE FUNCTION public.enable_partition_rls(part regclass)
RETURNS void LANGUAGE plpgsql AS $fn$
BEGIN
  EXECUTE format('ALTER TABLE %s ENABLE ROW LEVEL SECURITY', part);
  EXECUTE format('DROP POLICY IF EXISTS scope_isolation ON %s', part);
  EXECUTE format('CREATE POLICY scope_isolation ON %s USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))', part);
END;
$fn$;
