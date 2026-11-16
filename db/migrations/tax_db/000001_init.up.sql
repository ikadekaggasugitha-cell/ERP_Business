-- =============================================================================
-- tax_db — Tax Management (SVC-010)
-- DBDD BAB 5.5 | Standards DBS-001..016 | BRD 9.5.13 | INT-006 (e-Faktur)
-- golang-migrate: 000001_init.up.sql
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS btree_gin;

CREATE OR REPLACE FUNCTION public.tg_touch_row()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at  := now();
  NEW.row_version := COALESCE(OLD.row_version, 0) + 1;
  RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION public.current_org_scope()
RETURNS uuid[] LANGUAGE sql STABLE AS $$
  SELECT NULLIF(current_setting('app.org_scope', true), '')::uuid[];
$$;

CREATE SCHEMA IF NOT EXISTS tax;

-- =============================================================================
-- tax.tax_code
-- =============================================================================
CREATE TABLE tax.tax_code (
    id               UUID          NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID          NOT NULL,
    code             VARCHAR(30)   NOT NULL,   -- PPN/PPh/...
    rate             NUMERIC(9,4)  NOT NULL,   -- percent
    tax_type         VARCHAR(30)   NOT NULL,   -- VAT/WHT/...
    gl_account_id    UUID,                      -- logical ref coa_db
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    created_by       UUID          NOT NULL,
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER       NOT NULL DEFAULT 1,
    CONSTRAINT pk_tax_code PRIMARY KEY (id),
    CONSTRAINT uq_tax_code_org_code UNIQUE (organization_id, code),
    CONSTRAINT ck_tax_code_rate CHECK (rate >= 0)
);

-- =============================================================================
-- tax.tax_transaction
-- source_document_ref -> AR/AP invoice (logical ref).
-- =============================================================================
CREATE TABLE tax.tax_transaction (
    id                   UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id      UUID           NOT NULL,
    source_document_ref  VARCHAR(100)   NOT NULL,
    tax_code_id          UUID           NOT NULL,
    base_amount          NUMERIC(20,4)  NOT NULL DEFAULT 0,
    tax_amount           NUMERIC(20,4)  NOT NULL DEFAULT 0,
    efaktur_no           VARCHAR(50),
    efaktur_status       VARCHAR(20),
    created_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by           UUID           NOT NULL,
    updated_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by           UUID,
    deleted_at           TIMESTAMPTZ,
    row_version          INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_tax_transaction PRIMARY KEY (id),
    CONSTRAINT uq_tax_transaction_efaktur UNIQUE (organization_id, efaktur_no),
    CONSTRAINT fk_tax_transaction_code
        FOREIGN KEY (tax_code_id) REFERENCES tax.tax_code (id) ON DELETE RESTRICT,
    CONSTRAINT ck_tax_transaction_efaktur_status
        CHECK (efaktur_status IS NULL OR efaktur_status IN ('Draft','Approved','Reported'))
);
CREATE INDEX idx_tax_transaction_source ON tax.tax_transaction (source_document_ref);
CREATE INDEX idx_tax_transaction_code   ON tax.tax_transaction (tax_code_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE tax.tax_code        ENABLE ROW LEVEL SECURITY;
ALTER TABLE tax.tax_transaction ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON tax.tax_code
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON tax.tax_transaction
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON tax.tax_code
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON tax.tax_transaction
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE tax.outbox (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    aggregate_type   VARCHAR(100) NOT NULL,
    aggregate_id     UUID         NOT NULL,
    event_type       VARCHAR(150) NOT NULL,
    payload          JSONB        NOT NULL,
    organization_id  UUID         NOT NULL,
    correlation_id   UUID         NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    published_at     TIMESTAMPTZ,
    CONSTRAINT pk_outbox PRIMARY KEY (id)
);
CREATE INDEX idx_outbox_unpublished ON tax.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON tax.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON tax.outbox (correlation_id);

CREATE TABLE tax.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
