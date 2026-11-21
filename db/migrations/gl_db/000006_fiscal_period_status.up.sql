-- gl_db — Local projection of fiscal period status for Closed/Locked posting enforcement.
CREATE TABLE IF NOT EXISTS gl.fiscal_period_status (
    organization_id UUID        NOT NULL,
    period_id       UUID        NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'Open',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_fiscal_period_status PRIMARY KEY (organization_id, period_id)
);

ALTER TABLE gl.fiscal_period_status ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS scope_isolation ON gl.fiscal_period_status;
CREATE POLICY scope_isolation ON gl.fiscal_period_status
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
