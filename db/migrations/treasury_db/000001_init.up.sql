-- =============================================================================
-- treasury_db — Treasury, Cash & Bank (SVC-011)
-- DBDD BAB 5.6 | Standards DBS-001..016 | BRD 9.5.8/9.5.9 | INT-005
-- Payment is Exactly-Once (ADR-005) via correlation_id + outbox.
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

CREATE SCHEMA IF NOT EXISTS trs;

-- =============================================================================
-- trs.bank_account
-- gl_account_id is a logical ref to coa_db.
-- =============================================================================
CREATE TABLE trs.bank_account (
    id               UUID          NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID          NOT NULL,
    code             VARCHAR(50)   NOT NULL,
    bank_name        VARCHAR(150)  NOT NULL,
    account_no       VARCHAR(50)   NOT NULL,
    currency_code    VARCHAR(3)    NOT NULL,
    gl_account_id    UUID          NOT NULL,   -- logical ref coa_db
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    created_by       UUID          NOT NULL,
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER       NOT NULL DEFAULT 1,
    CONSTRAINT pk_bank_account PRIMARY KEY (id),
    CONSTRAINT uq_bank_account_org_code UNIQUE (organization_id, code)
);

-- =============================================================================
-- trs.payment  (Outgoing/Incoming; party_id logical ref masterdata_db)
-- =============================================================================
CREATE TABLE trs.payment (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    payment_no       VARCHAR(50)    NOT NULL,
    payment_type     VARCHAR(20)    NOT NULL,   -- Outgoing (AP)/Incoming (AR)
    bank_account_id  UUID           NOT NULL,
    party_id         UUID,                       -- vendor/customer (logical ref)
    amount           NUMERIC(20,4)  NOT NULL DEFAULT 0,
    payment_date     DATE           NOT NULL,
    status           VARCHAR(20)    NOT NULL DEFAULT 'Draft',
    external_ref     VARCHAR(100),               -- bank/gateway ref (INT-005)
    correlation_id   UUID           NOT NULL,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_payment PRIMARY KEY (id),
    CONSTRAINT uq_payment_no UNIQUE (organization_id, payment_no),
    CONSTRAINT fk_payment_bank_account
        FOREIGN KEY (bank_account_id) REFERENCES trs.bank_account (id) ON DELETE RESTRICT,
    CONSTRAINT ck_payment_type CHECK (payment_type IN ('Outgoing','Incoming')),
    CONSTRAINT ck_payment_status CHECK (status IN ('Draft','Settled','Reconciled','Failed'))
);
CREATE INDEX idx_payment_bank        ON trs.payment (bank_account_id);
CREATE INDEX idx_payment_party       ON trs.payment (party_id);
CREATE INDEX idx_payment_date        ON trs.payment (payment_date);
CREATE INDEX idx_payment_status      ON trs.payment (organization_id, status);
CREATE INDEX idx_payment_correlation ON trs.payment (correlation_id);

-- =============================================================================
-- trs.payment_allocation  (settlement against AP/AR invoice — logical ref)
-- =============================================================================
CREATE TABLE trs.payment_allocation (
    id                   UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id      UUID           NOT NULL,
    payment_id           UUID           NOT NULL,
    target_document_ref  VARCHAR(100)   NOT NULL,   -- AP/AR invoice (logical ref)
    allocated_amount     NUMERIC(20,4)  NOT NULL DEFAULT 0,
    allocated_at         TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by           UUID           NOT NULL,
    updated_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by           UUID,
    deleted_at           TIMESTAMPTZ,
    row_version          INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_payment_allocation PRIMARY KEY (id),
    CONSTRAINT fk_payment_allocation_payment
        FOREIGN KEY (payment_id) REFERENCES trs.payment (id) ON DELETE CASCADE
);
CREATE INDEX idx_payment_alloc_payment ON trs.payment_allocation (payment_id);
CREATE INDEX idx_payment_alloc_target  ON trs.payment_allocation (target_document_ref);

-- =============================================================================
-- trs.bank_statement_line  (reconciliation source)
-- =============================================================================
CREATE TABLE trs.bank_statement_line (
    id                  UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id     UUID           NOT NULL,
    bank_account_id     UUID           NOT NULL,
    statement_date      DATE           NOT NULL,
    description         TEXT,
    amount              NUMERIC(20,4)  NOT NULL DEFAULT 0,   -- +/- movement
    external_ref        VARCHAR(100),
    matched_payment_id  UUID,                                 -- reconciled payment
    is_reconciled       BOOLEAN        NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by          UUID           NOT NULL,
    updated_at          TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by          UUID,
    deleted_at          TIMESTAMPTZ,
    row_version         INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_bank_statement_line PRIMARY KEY (id),
    CONSTRAINT fk_bank_statement_line_account
        FOREIGN KEY (bank_account_id) REFERENCES trs.bank_account (id) ON DELETE RESTRICT,
    CONSTRAINT fk_bank_statement_line_payment
        FOREIGN KEY (matched_payment_id) REFERENCES trs.payment (id) ON DELETE SET NULL
);
CREATE INDEX idx_bank_stmt_line_account ON trs.bank_statement_line (bank_account_id, statement_date);
CREATE INDEX idx_bank_stmt_line_unrecon ON trs.bank_statement_line (bank_account_id) WHERE is_reconciled = FALSE;

-- --- RLS --------------------------------------------------------------------
ALTER TABLE trs.bank_account        ENABLE ROW LEVEL SECURITY;
ALTER TABLE trs.payment             ENABLE ROW LEVEL SECURITY;
ALTER TABLE trs.payment_allocation  ENABLE ROW LEVEL SECURITY;
ALTER TABLE trs.bank_statement_line ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON trs.bank_account
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON trs.payment
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON trs.payment_allocation
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON trs.bank_statement_line
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON trs.bank_account
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON trs.payment
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON trs.payment_allocation
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON trs.bank_statement_line
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE trs.outbox (
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
CREATE INDEX idx_outbox_unpublished ON trs.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON trs.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON trs.outbox (correlation_id);

CREATE TABLE trs.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
