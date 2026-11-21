-- reporting_db — Add local COA account code cache for CQRS GL snapshot projection.
CREATE TABLE IF NOT EXISTS rpt.coa_cache (
    organization_id  UUID          NOT NULL,
    account_id       UUID          NOT NULL,
    account_code     VARCHAR(50)   NOT NULL,
    account_name     VARCHAR(200)  NOT NULL DEFAULT '',
    account_type     VARCHAR(50)   NOT NULL DEFAULT '',
    refreshed_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT pk_coa_cache PRIMARY KEY (organization_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_coa_cache_code ON rpt.coa_cache (organization_id, account_code);

ALTER TABLE rpt.coa_cache ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS scope_isolation ON rpt.coa_cache;
CREATE POLICY scope_isolation ON rpt.coa_cache
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
