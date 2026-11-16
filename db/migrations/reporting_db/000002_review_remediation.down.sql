-- reporting_db — rollback 000002_review_remediation

DROP FUNCTION IF EXISTS public.enable_partition_rls(regclass);
DROP INDEX IF EXISTS rpt.idx_inventory_position_wh;
DROP INDEX IF EXISTS rpt.idx_inventory_position_item;
DROP INDEX IF EXISTS rpt.idx_gl_balance_snapshot_account;
ALTER TABLE rpt.sales_summary DROP CONSTRAINT IF EXISTS uq_sales_summary;
ALTER TABLE rpt.sales_summary ADD CONSTRAINT uq_sales_summary UNIQUE (organization_id, period_year, period_month, customer_id);
