-- gl_db — rollback 000003_review_remediation
CREATE INDEX IF NOT EXISTS idx_journal_line_entry ON gl.journal_line (journal_entry_id);

DROP TRIGGER IF EXISTS register_document_number ON gl.journal_entry;
DROP FUNCTION IF EXISTS gl.tg_register_journal_no();
DROP TABLE IF EXISTS gl.document_number_registry;

DO $$
DECLARE part regclass;
BEGIN
  FOR part IN SELECT inhrelid::regclass FROM pg_inherits WHERE inhparent = 'gl.journal_entry'::regclass LOOP
    EXECUTE format('DROP POLICY IF EXISTS scope_isolation ON %s', part);
    EXECUTE format('ALTER TABLE %s DISABLE ROW LEVEL SECURITY', part);
  END LOOP;
END $$;

DROP FUNCTION IF EXISTS public.enable_partition_rls(regclass);
