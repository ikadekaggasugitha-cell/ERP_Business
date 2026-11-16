-- =============================================================================
-- inventory_db — Review remediation (M1 partition RLS, N1 FK indexes)
-- Ref: db/REVIEW.md (M1, N1) | golang-migrate: 000003
-- =============================================================================

-- M1: RLS on stock_movement partitions
CREATE OR REPLACE FUNCTION public.enable_partition_rls(part regclass)
RETURNS void LANGUAGE plpgsql AS $fn$
BEGIN
  EXECUTE format('ALTER TABLE %s ENABLE ROW LEVEL SECURITY', part);
  EXECUTE format('DROP POLICY IF EXISTS scope_isolation ON %s', part);
  EXECUTE format('CREATE POLICY scope_isolation ON %s USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))', part);
END;
$fn$;

DO $$
DECLARE part regclass;
BEGIN
  FOR part IN SELECT inhrelid::regclass FROM pg_inherits WHERE inhparent = 'inv.stock_movement'::regclass LOOP
    PERFORM public.enable_partition_rls(part);
  END LOOP;
END $$;

-- N1: supporting indexes for warehouse_id FKs (leading col not covered by any
-- existing index; composite indexes lead with organization_id).
CREATE INDEX idx_stock_balance_warehouse     ON inv.stock_balance (warehouse_id);
CREATE INDEX idx_stock_movement_warehouse    ON inv.stock_movement (warehouse_id);
CREATE INDEX idx_stock_reservation_warehouse ON inv.stock_reservation (warehouse_id);
