-- inventory_db — rollback 000003_review_remediation
DROP INDEX IF EXISTS inv.idx_stock_reservation_warehouse;
DROP INDEX IF EXISTS inv.idx_stock_movement_warehouse;
DROP INDEX IF EXISTS inv.idx_stock_balance_warehouse;

DO $$
DECLARE part regclass;
BEGIN
  FOR part IN SELECT inhrelid::regclass FROM pg_inherits WHERE inhparent = 'inv.stock_movement'::regclass LOOP
    EXECUTE format('DROP POLICY IF EXISTS scope_isolation ON %s', part);
    EXECUTE format('ALTER TABLE %s DISABLE ROW LEVEL SECURITY', part);
  END LOOP;
END $$;

DROP FUNCTION IF EXISTS public.enable_partition_rls(regclass);
