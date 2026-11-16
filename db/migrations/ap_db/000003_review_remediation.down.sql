-- ap_db — rollback 000003_review_remediation
CREATE INDEX IF NOT EXISTS idx_ap_alloc_invoice ON ap.ap_payment_allocation (ap_invoice_id);
CREATE INDEX IF NOT EXISTS idx_ap_invoice_line_invoice ON ap.ap_invoice_line (ap_invoice_id);

DROP TRIGGER IF EXISTS register_document_number ON ap.ap_invoice;
DROP FUNCTION IF EXISTS ap.tg_register_invoice_no();
DROP TABLE IF EXISTS ap.document_number_registry;

DO $$
DECLARE part regclass;
BEGIN
  FOR part IN SELECT inhrelid::regclass FROM pg_inherits WHERE inhparent = 'ap.ap_invoice'::regclass LOOP
    EXECUTE format('DROP POLICY IF EXISTS scope_isolation ON %s', part);
    EXECUTE format('ALTER TABLE %s DISABLE ROW LEVEL SECURITY', part);
  END LOOP;
END $$;

DROP FUNCTION IF EXISTS public.enable_partition_rls(regclass);
