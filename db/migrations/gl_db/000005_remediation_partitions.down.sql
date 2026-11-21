-- gl_db — rollback 000005_remediation_partitions

ALTER TABLE gl.journal_line DROP CONSTRAINT IF EXISTS fk_journal_line_entry;

DO $$
DECLARE r record;
BEGIN
  FOR r IN SELECT i.inhparent::regclass AS parent, c.oid::regclass AS part, c.relname
           FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
           WHERE i.inhparent IN ('gl.journal_line'::regclass, 'gl.journal_entry'::regclass, 'gl.outbox'::regclass)
             AND (
               c.relname LIKE '%_default'
               OR (c.relname !~ '_y2026m(0[1-9]|1[0-2])$' AND c.relname !~ '_y2027m01$')
             )
           ORDER BY CASE i.inhparent
             WHEN 'gl.journal_line'::regclass  THEN 1
             WHEN 'gl.journal_entry'::regclass THEN 2
             ELSE 3
           END, c.relname LOOP
    EXECUTE format('ALTER TABLE %s DETACH PARTITION %s', r.parent, r.part);
    EXECUTE format('DROP TABLE IF EXISTS %s', r.part);
  END LOOP;
END $$;

ALTER TABLE gl.journal_line
    ADD CONSTRAINT fk_journal_line_entry FOREIGN KEY (journal_entry_id, posting_date)
    REFERENCES gl.journal_entry (id, posting_date) ON DELETE CASCADE;
