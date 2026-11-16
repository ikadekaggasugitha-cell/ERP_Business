-- localization_db — rollback 000003_review_remediation

DO $$
DECLARE r record;
BEGIN
  FOR r IN SELECT i.inhparent::regclass AS parent, c.oid::regclass AS part, c.relname
           FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
           WHERE i.inhparent IN ('loc.outbox'::regclass)
             AND (
               c.relname LIKE '%_default'
               OR (c.relname !~ '_y2026m(0[1-9]|1[0-2])$' AND c.relname !~ '_y2027m01$')
             ) LOOP
    EXECUTE format('ALTER TABLE %s DETACH PARTITION %s', r.parent, r.part);
    EXECUTE format('DROP TABLE IF EXISTS %s', r.part);
  END LOOP;
END $$;

DROP INDEX IF EXISTS loc.idx_org_lang_code;

ALTER TABLE loc.locale_format DROP CONSTRAINT IF EXISTS ck_locale_format_nonempty;

ALTER TABLE loc.language DROP CONSTRAINT IF EXISTS ck_language_code_nonempty;
