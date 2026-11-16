-- =============================================================================
-- budget_db — Budget Management (SVC-022)
-- DBDD BAB 9.4 | Standards DBS-001..016 | BRD 9.5.12 (BR-636..645)
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

CREATE SCHEMA IF NOT EXISTS bud;

-- =============================================================================
-- bud.budget (BR-636/638/639)
-- =============================================================================
CREATE TABLE bud.budget (
    id                            UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id               UUID         NOT NULL,
    code                          VARCHAR(50)  NOT NULL,
    name                          VARCHAR(150) NOT NULL,
    financial_organization_ref    UUID         NOT NULL,
    fiscal_year_ref               UUID         NOT NULL,
    version                       INTEGER      NOT NULL DEFAULT 1,
    status                        VARCHAR(20)  NOT NULL DEFAULT 'Draft',
    is_active                     BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at                    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by                    UUID         NOT NULL,
    updated_at                    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by                    UUID,
    deleted_at                    TIMESTAMPTZ,
    row_version                   INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_budget PRIMARY KEY (id),
    CONSTRAINT uq_budget UNIQUE (organization_id, code, version),
    CONSTRAINT ck_budget_status CHECK (status IN ('Draft','Submitted','Approved','Revised','Closed'))
);

-- =============================================================================
-- bud.budget_line (BR-637/640)
-- account_ref / cost_center_ref / project_ref are logical refs.
-- =============================================================================
CREATE TABLE bud.budget_line (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    budget_id        UUID           NOT NULL,
    account_ref      UUID           NOT NULL,
    cost_center_ref  UUID,
    project_ref      UUID,
    period_no        INTEGER        NOT NULL,
    budget_amount    NUMERIC(20,4)  NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_budget_line PRIMARY KEY (id),
    CONSTRAINT fk_budget_line_budget
        FOREIGN KEY (budget_id) REFERENCES bud.budget (id) ON DELETE CASCADE,
    CONSTRAINT ck_budget_line_period CHECK (period_no BETWEEN 1 AND 12)
);

-- =============================================================================
-- bud.budget_control_policy (BR-642)
-- =============================================================================
CREATE TABLE bud.budget_control_policy (
    id                UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID         NOT NULL,
    account_ref       UUID,                        -- NULL = all accounts
    transaction_type  VARCHAR(50),                 -- NULL = all transaction types
    control_type      VARCHAR(20)  NOT NULL,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by        UUID         NOT NULL,
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by        UUID,
    deleted_at        TIMESTAMPTZ,
    row_version       INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_budget_control_policy PRIMARY KEY (id),
    CONSTRAINT ck_budget_control_type CHECK (control_type IN ('HardStop','SoftWarning','ApprovalOverride'))
);

-- =============================================================================
-- bud.budget_commitment (BR-644 — encumbrance)
-- =============================================================================
CREATE TABLE bud.budget_commitment (
    id                     UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id        UUID           NOT NULL,
    budget_id              UUID           NOT NULL,
    source_document_ref    VARCHAR(100)   NOT NULL,
    committed_amount       NUMERIC(20,4)  NOT NULL DEFAULT 0,
    status                 VARCHAR(20)    NOT NULL DEFAULT 'Open',
    created_at             TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by             UUID           NOT NULL,
    updated_at             TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by             UUID,
    deleted_at             TIMESTAMPTZ,
    row_version            INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_budget_commitment PRIMARY KEY (id),
    CONSTRAINT fk_budget_commitment_budget
        FOREIGN KEY (budget_id) REFERENCES bud.budget (id) ON DELETE RESTRICT,
    CONSTRAINT ck_budget_commitment_status CHECK (status IN ('Open','Released','Consumed'))
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_budget_active        ON bud.budget (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_budget_fiscal_year   ON bud.budget (fiscal_year_ref);
CREATE INDEX idx_budget_line_budget   ON bud.budget_line (budget_id);
CREATE INDEX idx_budget_line_account  ON bud.budget_line (account_ref);
CREATE INDEX idx_budget_line_cc       ON bud.budget_line (cost_center_ref);
CREATE INDEX idx_budget_commitment_budget ON bud.budget_commitment (budget_id);
CREATE INDEX idx_budget_commitment_src    ON bud.budget_commitment (source_document_ref);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE bud.budget                 ENABLE ROW LEVEL SECURITY;
ALTER TABLE bud.budget_line            ENABLE ROW LEVEL SECURITY;
ALTER TABLE bud.budget_control_policy  ENABLE ROW LEVEL SECURITY;
ALTER TABLE bud.budget_commitment      ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON bud.budget                USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON bud.budget_line           USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON bud.budget_control_policy USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON bud.budget_commitment     USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON bud.budget                FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON bud.budget_line           FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON bud.budget_control_policy FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON bud.budget_commitment     FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE bud.outbox (
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
CREATE INDEX idx_outbox_unpublished ON bud.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON bud.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON bud.outbox (correlation_id);

CREATE TABLE bud.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
