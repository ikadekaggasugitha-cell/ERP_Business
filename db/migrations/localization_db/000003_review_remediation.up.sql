-- localization_db — Phase 1 remediation (review_remediation)

ALTER TABLE loc.language ADD CONSTRAINT ck_language_code_nonempty CHECK (length(trim(code)) >= 2);

ALTER TABLE loc.locale_format ADD CONSTRAINT ck_locale_format_nonempty CHECK (length(trim(locale_code)) >= 2);

CREATE INDEX IF NOT EXISTS idx_org_lang_code ON loc.organization_language (organization_id, default_language_code);

DO $$
DECLARE m date; has_rls boolean;
BEGIN
  FOR m IN SELECT generate_series('2025-01-01'::date, '2028-12-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('loc.outbox'::regclass, m);
  END LOOP;
  IF to_regclass('loc.outbox_default') IS NULL THEN
    EXECUTE 'CREATE TABLE loc.outbox_default PARTITION OF loc.outbox DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'loc' AND c.relname = 'outbox';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE loc.outbox_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON loc.outbox_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
END $$;
