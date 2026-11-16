-- audit_db — rollback 000005_review_remediation

DO $$
DECLARE r record;
BEGIN
  FOR r IN SELECT i.inhparent::regclass AS parent, c.oid::regclass AS part, c.relname
           FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
           WHERE i.inhparent IN ('audit.audit_log'::regclass)
             AND (
               c.relname LIKE '%_default'
               OR (c.relname !~ '_y2026m(0[1-9]|1[0-2])$' AND c.relname !~ '_y2027m01$')
             ) LOOP
    EXECUTE format('ALTER TABLE %s DETACH PARTITION %s', r.parent, r.part);
    EXECUTE format('DROP TABLE IF EXISTS %s', r.part);
  END LOOP;
END $$;

DO $$
DECLARE part regclass;
BEGIN
  FOR part IN SELECT inhrelid::regclass FROM pg_inherits WHERE inhparent = 'audit.audit_log'::regclass LOOP
    EXECUTE format('DROP POLICY IF EXISTS scope_isolation ON %s', part);
    EXECUTE format('ALTER TABLE %s DISABLE ROW LEVEL SECURITY', part);
  END LOOP;
END $$;
DROP FUNCTION IF EXISTS public.enable_partition_rls(regclass);
