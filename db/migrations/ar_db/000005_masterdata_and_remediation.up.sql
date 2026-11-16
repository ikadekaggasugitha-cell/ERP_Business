-- ar_db — Phase 1 remediation (masterdata_and_remediation)

CREATE TABLE IF NOT EXISTS ar.masterdata_cache (
    organization_id uuid         NOT NULL,
    party_id        uuid         NOT NULL,
    code            varchar(50)  NOT NULL DEFAULT '',
    name            varchar(200) NOT NULL DEFAULT '',
    party_type      varchar(20)  NOT NULL DEFAULT '',
    updated_at      timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT pk_ar_masterdata_cache PRIMARY KEY (organization_id, party_id)
);
CREATE INDEX IF NOT EXISTS idx_ar_masterdata_cache_type ON ar.masterdata_cache (organization_id, party_type);
ALTER TABLE ar.masterdata_cache ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS scope_isolation ON ar.masterdata_cache;
CREATE POLICY scope_isolation ON ar.masterdata_cache
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));

DO $$
DECLARE m date; has_rls boolean;
BEGIN
  FOR m IN SELECT generate_series('2025-01-01'::date, '2028-12-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('ar.ar_invoice'::regclass, m);
    PERFORM public.create_monthly_partition('ar.outbox'::regclass, m);
  END LOOP;
  IF to_regclass('ar.ar_invoice_default') IS NULL THEN
    EXECUTE 'CREATE TABLE ar.ar_invoice_default PARTITION OF ar.ar_invoice DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'ar' AND c.relname = 'ar_invoice';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE ar.ar_invoice_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON ar.ar_invoice_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
  IF to_regclass('ar.outbox_default') IS NULL THEN
    EXECUTE 'CREATE TABLE ar.outbox_default PARTITION OF ar.outbox DEFAULT';
    SELECT c.relrowsecurity INTO has_rls FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'ar' AND c.relname = 'outbox';
    IF has_rls THEN
      EXECUTE 'ALTER TABLE ar.outbox_default ENABLE ROW LEVEL SECURITY';
      EXECUTE 'CREATE POLICY scope_isolation ON ar.outbox_default USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))';
    END IF;
  END IF;
END $$;
