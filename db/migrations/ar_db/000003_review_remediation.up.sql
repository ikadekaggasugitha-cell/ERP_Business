-- =============================================================================
-- ar_db — Review remediation (M1 partition RLS, M2 doc-number registry, N4 index)
-- Ref: db/REVIEW.md | golang-migrate: 000003
-- =============================================================================

-- M1: RLS on partitions
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
  FOR part IN SELECT inhrelid::regclass FROM pg_inherits WHERE inhparent = 'ar.ar_invoice'::regclass LOOP
    PERFORM public.enable_partition_rls(part);
  END LOOP;
END $$;

-- M2: Document Number Registry (AR invoice_no) — DB-enforced uniqueness
CREATE TABLE ar.document_number_registry (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    document_type    VARCHAR(50)  NOT NULL,
    document_number  VARCHAR(50)  NOT NULL,
    fiscal_year      INTEGER      NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'Issued',
    source_ref       VARCHAR(100),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_document_number_registry PRIMARY KEY (id),
    CONSTRAINT uq_document_number_registry UNIQUE (organization_id, document_type, fiscal_year, document_number),
    CONSTRAINT ck_document_number_registry_status CHECK (status IN ('Reserved','Issued','Cancelled'))
);
CREATE INDEX idx_document_number_registry_lookup ON ar.document_number_registry (organization_id, document_type, fiscal_year);

ALTER TABLE ar.document_number_registry ENABLE ROW LEVEL SECURITY;
CREATE POLICY scope_isolation ON ar.document_number_registry
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE TRIGGER touch_row BEFORE UPDATE ON ar.document_number_registry
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

CREATE OR REPLACE FUNCTION ar.tg_register_invoice_no()
RETURNS trigger LANGUAGE plpgsql AS $fn$
BEGIN
  INSERT INTO ar.document_number_registry
      (organization_id, document_type, document_number, fiscal_year, status, source_ref, created_by)
  VALUES
      (NEW.organization_id, 'ARInvoice', NEW.invoice_no,
       EXTRACT(YEAR FROM NEW.invoice_date)::int, 'Issued', NEW.id::text, NEW.created_by);
  RETURN NEW;
END;
$fn$;
CREATE TRIGGER register_document_number BEFORE INSERT ON ar.ar_invoice
    FOR EACH ROW EXECUTE FUNCTION ar.tg_register_invoice_no();

-- N4: drop redundant single-column indexes
DROP INDEX IF EXISTS ar.idx_ar_invoice_line_invoice;
DROP INDEX IF EXISTS ar.idx_ar_alloc_invoice;
