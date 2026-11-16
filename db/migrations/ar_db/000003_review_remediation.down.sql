-- ar_db — rollback 000003_review_remediation
CREATE INDEX IF NOT EXISTS idx_ar_alloc_invoice ON ar.ar_receipt_allocation (ar_invoice_id);
CREATE INDEX IF NOT EXISTS idx_ar_invoice_line_invoice ON ar.ar_invoice_line (ar_invoice_id);

DROP TRIGGER IF EXISTS register_document_number ON ar.ar_invoice;
DROP FUNCTION IF EXISTS ar.tg_register_invoice_no();
DROP TABLE IF EXISTS ar.document_number_registry;

DO $$
DECLARE part regclass;
BEGIN
  FOR part IN SELECT inhrelid::regclass FROM pg_inherits WHERE inhparent = 'ar.ar_invoice'::regclass LOOP
    EXECUTE format('DROP POLICY IF EXISTS scope_isolation ON %s', part);
    EXECUTE format('ALTER TABLE %s DISABLE ROW LEVEL SECURITY', part);
  END LOOP;
END $$;

DROP FUNCTION IF EXISTS public.enable_partition_rls(regclass);
