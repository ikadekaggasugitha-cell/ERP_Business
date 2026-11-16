-- =============================================================================
-- gl_db — Review remediation (M1 partition RLS, M2 doc-number registry, N4 index)
-- Ref: db/REVIEW.md (M1, M2, N4) | golang-migrate: 000003
-- =============================================================================

-- -----------------------------------------------------------------------------
-- M1: Row-Level Security on partitions.
-- RLS policies on a partitioned parent are NOT applied when a partition is
-- accessed directly. Enable RLS + the same scope_isolation policy on every
-- partition (defense-in-depth). Reusable helper is called by the partition
-- provisioning job for future partitions too.
-- -----------------------------------------------------------------------------
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
  FOR part IN SELECT inhrelid::regclass FROM pg_inherits WHERE inhparent = 'gl.journal_entry'::regclass LOOP
    PERFORM public.enable_partition_rls(part);
  END LOOP;
END $$;

-- -----------------------------------------------------------------------------
-- M2: Document Number Registry (journal_no) — DB-enforced uniqueness.
-- gl.journal_entry is partitioned by posting_date, so its UNIQUE must include
-- posting_date and cannot guarantee journal_no uniqueness across partitions.
-- The (non-partitioned) registry enforces it at the database via a UNIQUE and
-- an auto-populate trigger, independent of the application.
-- -----------------------------------------------------------------------------
CREATE TABLE gl.document_number_registry (
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
CREATE INDEX idx_document_number_registry_lookup ON gl.document_number_registry (organization_id, document_type, fiscal_year);

ALTER TABLE gl.document_number_registry ENABLE ROW LEVEL SECURITY;
CREATE POLICY scope_isolation ON gl.document_number_registry
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE TRIGGER touch_row BEFORE UPDATE ON gl.document_number_registry
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- Auto-populate on journal creation; duplicate journal_no -> unique violation.
-- fiscal_year is derived from posting_date's calendar year (see CHANGELOG).
CREATE OR REPLACE FUNCTION gl.tg_register_journal_no()
RETURNS trigger LANGUAGE plpgsql AS $fn$
BEGIN
  INSERT INTO gl.document_number_registry
      (organization_id, document_type, document_number, fiscal_year, status, source_ref, created_by)
  VALUES
      (NEW.organization_id, 'JournalEntry', NEW.journal_no,
       EXTRACT(YEAR FROM NEW.posting_date)::int, 'Issued', NEW.id::text, NEW.created_by);
  RETURN NEW;
END;
$fn$;
CREATE TRIGGER register_document_number BEFORE INSERT ON gl.journal_entry
    FOR EACH ROW EXECUTE FUNCTION gl.tg_register_journal_no();

-- -----------------------------------------------------------------------------
-- N4: drop redundant single-column index (covered by uq_journal_line leading col)
-- -----------------------------------------------------------------------------
DROP INDEX IF EXISTS gl.idx_journal_line_entry;
