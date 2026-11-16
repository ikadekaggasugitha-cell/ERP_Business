-- hr_db — Phase 1 remediation (pii_and_partitions)

ALTER TABLE hr.employee
    ALTER COLUMN full_name TYPE VARCHAR(512),
    ALTER COLUMN national_id TYPE VARCHAR(255),
    ALTER COLUMN email TYPE VARCHAR(512),
    ADD COLUMN IF NOT EXISTS encrypted_full_name TEXT,
    ADD COLUMN IF NOT EXISTS encrypted_national_id TEXT,
    ADD COLUMN IF NOT EXISTS encrypted_dob TEXT,
    ADD COLUMN IF NOT EXISTS encrypted_email TEXT,
    ADD COLUMN IF NOT EXISTS national_id_hash VARCHAR(64);
CREATE INDEX IF NOT EXISTS idx_employee_national_id_hash ON hr.employee (organization_id, national_id_hash) WHERE national_id_hash IS NOT NULL;

DO $$
DECLARE m date; has_rls boolean;
BEGIN
  FOR m IN SELECT generate_series('2025-01-01'::date, '2028-12-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('hr.outbox'::regclass, m);
  END LOOP;
  IF to_regclass('hr.outbox_default') IS NULL THEN
    EXECUTE 'CREATE TABLE hr.outbox_default PARTITION OF hr.outbox DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'hr' AND c.relname = 'outbox';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE hr.outbox_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON hr.outbox_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
END $$;
