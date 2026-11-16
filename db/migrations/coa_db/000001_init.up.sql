-- =============================================================================
-- coa_db — Financial Organization, Fiscal Calendar & Chart of Accounts (SVC-006)
-- DBDD BAB 4.2/4.3/4.5 | Standards DBS-001..016 | BRD 9.5.1/9.5.2/9.6
-- Schemas: fin (financial org + fiscal calendar), coa (chart of accounts).
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

CREATE SCHEMA IF NOT EXISTS fin;
CREATE SCHEMA IF NOT EXISTS coa;

-- =============================================================================
-- coa.chart_of_accounts  (created first — referenced by fin.financial_organization)
-- =============================================================================
CREATE TABLE coa.chart_of_accounts (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(50)  NOT NULL,
    name             VARCHAR(150) NOT NULL,
    structure        JSONB        NOT NULL,   -- segment definition (order/length)
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_chart_of_accounts PRIMARY KEY (id),
    CONSTRAINT uq_chart_of_accounts_org_code UNIQUE (organization_id, code)
);

-- =============================================================================
-- fin.fiscal_calendar / fiscal_year / fiscal_period
-- =============================================================================
CREATE TABLE fin.fiscal_calendar (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(50)  NOT NULL,
    name             VARCHAR(150) NOT NULL,
    period_type      VARCHAR(20)  NOT NULL,   -- Monthly/Quarterly
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_fiscal_calendar PRIMARY KEY (id),
    CONSTRAINT uq_fiscal_calendar_org_code UNIQUE (organization_id, code),
    CONSTRAINT ck_fiscal_calendar_period_type CHECK (period_type IN ('Monthly','Quarterly'))
);

CREATE TABLE fin.fiscal_year (
    id                  UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id     UUID         NOT NULL,
    fiscal_calendar_id  UUID         NOT NULL,
    year_no             INTEGER      NOT NULL,
    name                VARCHAR(50)  NOT NULL,
    start_date          DATE         NOT NULL,
    end_date            DATE         NOT NULL,
    status              VARCHAR(20)  NOT NULL DEFAULT 'Open',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by          UUID         NOT NULL,
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by          UUID,
    deleted_at          TIMESTAMPTZ,
    row_version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_fiscal_year PRIMARY KEY (id),
    CONSTRAINT uq_fiscal_year UNIQUE (fiscal_calendar_id, year_no),
    CONSTRAINT fk_fiscal_year_calendar
        FOREIGN KEY (fiscal_calendar_id) REFERENCES fin.fiscal_calendar (id) ON DELETE RESTRICT,
    CONSTRAINT ck_fiscal_year_status CHECK (status IN ('Open','Closed','Locked')),
    CONSTRAINT ck_fiscal_year_range CHECK (end_date >= start_date)
);

CREATE TABLE fin.fiscal_period (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    fiscal_year_id   UUID         NOT NULL,
    period_no        INTEGER      NOT NULL,   -- 1..12/13
    start_date       DATE         NOT NULL,
    end_date         DATE         NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'Open',   -- BR-562
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_fiscal_period PRIMARY KEY (id),
    CONSTRAINT uq_fiscal_period UNIQUE (fiscal_year_id, period_no),
    CONSTRAINT fk_fiscal_period_year
        FOREIGN KEY (fiscal_year_id) REFERENCES fin.fiscal_year (id) ON DELETE RESTRICT,
    CONSTRAINT ck_fiscal_period_status CHECK (status IN ('Open','Closed','Locked')),
    CONSTRAINT ck_fiscal_period_range CHECK (end_date >= start_date)
);

-- =============================================================================
-- fin.financial_organization / ledger
-- functional_currency_code / currency_code -> logical refs to currency_db.
-- =============================================================================
CREATE TABLE fin.financial_organization (
    id                        UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id           UUID         NOT NULL,
    code                      VARCHAR(50)  NOT NULL,   -- company code
    name                      VARCHAR(200) NOT NULL,
    organization_ref_id       UUID         NOT NULL,   -- logical ref -> org_db
    functional_currency_code  VARCHAR(3)   NOT NULL,
    chart_of_accounts_id      UUID         NOT NULL,
    fiscal_calendar_id        UUID         NOT NULL,
    created_at                TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by                UUID         NOT NULL,
    updated_at                TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by                UUID,
    deleted_at                TIMESTAMPTZ,
    row_version               INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_financial_organization PRIMARY KEY (id),
    CONSTRAINT uq_financial_organization_org_code UNIQUE (organization_id, code),
    CONSTRAINT fk_finorg_coa
        FOREIGN KEY (chart_of_accounts_id) REFERENCES coa.chart_of_accounts (id) ON DELETE RESTRICT,
    CONSTRAINT fk_finorg_calendar
        FOREIGN KEY (fiscal_calendar_id) REFERENCES fin.fiscal_calendar (id) ON DELETE RESTRICT
);

CREATE TABLE fin.ledger (
    id                          UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id             UUID         NOT NULL,
    financial_organization_id   UUID         NOT NULL,
    code                        VARCHAR(50)  NOT NULL,
    ledger_type                 VARCHAR(30)  NOT NULL,   -- Primary/Statutory/Management/Consolidation
    currency_code               VARCHAR(3)   NOT NULL,
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by                  UUID         NOT NULL,
    updated_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by                  UUID,
    deleted_at                  TIMESTAMPTZ,
    row_version                 INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_ledger PRIMARY KEY (id),
    CONSTRAINT uq_ledger UNIQUE (financial_organization_id, code),
    CONSTRAINT fk_ledger_finorg
        FOREIGN KEY (financial_organization_id) REFERENCES fin.financial_organization (id) ON DELETE RESTRICT,
    CONSTRAINT ck_ledger_type
        CHECK (ledger_type IN ('Primary','Statutory','Management','Consolidation'))
);

-- =============================================================================
-- coa.account (+ hierarchy) / segments / combination / posting_rule
-- =============================================================================
CREATE TABLE coa.account (
    id                    UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id       UUID         NOT NULL,
    chart_of_accounts_id  UUID         NOT NULL,
    code                  VARCHAR(50)  NOT NULL,
    name                  VARCHAR(200) NOT NULL,
    account_type          VARCHAR(30)  NOT NULL,   -- Asset/Liability/Equity/Revenue/Expense
    account_class         VARCHAR(30),
    is_control            BOOLEAN      NOT NULL DEFAULT FALSE,
    is_monetary           BOOLEAN      NOT NULL DEFAULT FALSE,   -- BR-561 revaluation
    allow_posting         BOOLEAN      NOT NULL DEFAULT TRUE,    -- leaf vs header
    parent_id             UUID,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by            UUID         NOT NULL,
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by            UUID,
    deleted_at            TIMESTAMPTZ,
    row_version           INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_account PRIMARY KEY (id),
    CONSTRAINT uq_account UNIQUE (chart_of_accounts_id, code),
    CONSTRAINT fk_account_coa
        FOREIGN KEY (chart_of_accounts_id) REFERENCES coa.chart_of_accounts (id) ON DELETE RESTRICT,
    CONSTRAINT fk_account_parent
        FOREIGN KEY (parent_id) REFERENCES coa.account (id) ON DELETE RESTRICT,
    CONSTRAINT ck_account_type
        CHECK (account_type IN ('Asset','Liability','Equity','Revenue','Expense'))
);

CREATE TABLE coa.account_segment (
    id                    UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id       UUID         NOT NULL,
    chart_of_accounts_id  UUID         NOT NULL,
    segment_no            INTEGER      NOT NULL,
    name                  VARCHAR(100) NOT NULL,
    length                INTEGER      NOT NULL,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by            UUID         NOT NULL,
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by            UUID,
    deleted_at            TIMESTAMPTZ,
    row_version           INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_account_segment PRIMARY KEY (id),
    CONSTRAINT uq_account_segment UNIQUE (chart_of_accounts_id, segment_no),
    CONSTRAINT fk_account_segment_coa
        FOREIGN KEY (chart_of_accounts_id) REFERENCES coa.chart_of_accounts (id) ON DELETE CASCADE,
    CONSTRAINT ck_account_segment_length CHECK (length > 0)
);

CREATE TABLE coa.segment_value (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    segment_id       UUID         NOT NULL,
    value            VARCHAR(50)  NOT NULL,
    description      VARCHAR(200),
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_segment_value PRIMARY KEY (id),
    CONSTRAINT uq_segment_value UNIQUE (segment_id, value),
    CONSTRAINT fk_segment_value_segment
        FOREIGN KEY (segment_id) REFERENCES coa.account_segment (id) ON DELETE CASCADE
);

CREATE TABLE coa.account_combination (
    id                    UUID          NOT NULL DEFAULT gen_random_uuid(),
    organization_id       UUID          NOT NULL,
    chart_of_accounts_id  UUID          NOT NULL,
    combination_code      VARCHAR(100)  NOT NULL,   -- concatenated segments
    is_enabled            BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT now(),
    created_by            UUID          NOT NULL,
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_by            UUID,
    deleted_at            TIMESTAMPTZ,
    row_version           INTEGER       NOT NULL DEFAULT 1,
    CONSTRAINT pk_account_combination PRIMARY KEY (id),
    CONSTRAINT uq_account_combination UNIQUE (chart_of_accounts_id, combination_code),
    CONSTRAINT fk_account_combination_coa
        FOREIGN KEY (chart_of_accounts_id) REFERENCES coa.chart_of_accounts (id) ON DELETE CASCADE
);

CREATE TABLE coa.posting_rule (
    id                    UUID          NOT NULL DEFAULT gen_random_uuid(),
    organization_id       UUID          NOT NULL,
    chart_of_accounts_id  UUID          NOT NULL,
    source_module         VARCHAR(50)   NOT NULL,   -- AP/AR/Inventory/Payroll/...
    event_key             VARCHAR(100)  NOT NULL,   -- goods_issue, invoice, ...
    debit_account_rule    JSONB         NOT NULL,   -- derivation
    credit_account_rule   JSONB         NOT NULL,
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT now(),
    created_by            UUID          NOT NULL,
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_by            UUID,
    deleted_at            TIMESTAMPTZ,
    row_version           INTEGER       NOT NULL DEFAULT 1,
    CONSTRAINT pk_posting_rule PRIMARY KEY (id),
    CONSTRAINT uq_posting_rule UNIQUE (chart_of_accounts_id, source_module, event_key),
    CONSTRAINT fk_posting_rule_coa
        FOREIGN KEY (chart_of_accounts_id) REFERENCES coa.chart_of_accounts (id) ON DELETE CASCADE
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_coa_active            ON coa.chart_of_accounts (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_coa_structure         ON coa.chart_of_accounts USING gin (structure);
CREATE INDEX idx_fiscal_year_calendar  ON fin.fiscal_year (fiscal_calendar_id);
CREATE INDEX idx_fiscal_period_year    ON fin.fiscal_period (fiscal_year_id);
CREATE INDEX idx_finorg_orgref         ON fin.financial_organization (organization_ref_id);
CREATE INDEX idx_finorg_coa            ON fin.financial_organization (chart_of_accounts_id);
CREATE INDEX idx_finorg_calendar       ON fin.financial_organization (fiscal_calendar_id);
CREATE INDEX idx_ledger_finorg         ON fin.ledger (financial_organization_id);
CREATE INDEX idx_account_coa           ON coa.account (chart_of_accounts_id);
CREATE INDEX idx_account_parent        ON coa.account (parent_id);
CREATE INDEX idx_account_type          ON coa.account (organization_id, account_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_account_segment_coa   ON coa.account_segment (chart_of_accounts_id);
CREATE INDEX idx_segment_value_segment ON coa.segment_value (segment_id);
CREATE INDEX idx_account_combination_coa ON coa.account_combination (chart_of_accounts_id);
CREATE INDEX idx_posting_rule_source   ON coa.posting_rule (organization_id, source_module);
CREATE INDEX idx_posting_rule_debit    ON coa.posting_rule USING gin (debit_account_rule);
CREATE INDEX idx_posting_rule_credit   ON coa.posting_rule USING gin (credit_account_rule);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE coa.chart_of_accounts     ENABLE ROW LEVEL SECURITY;
ALTER TABLE fin.fiscal_calendar       ENABLE ROW LEVEL SECURITY;
ALTER TABLE fin.fiscal_year           ENABLE ROW LEVEL SECURITY;
ALTER TABLE fin.fiscal_period         ENABLE ROW LEVEL SECURITY;
ALTER TABLE fin.financial_organization ENABLE ROW LEVEL SECURITY;
ALTER TABLE fin.ledger                ENABLE ROW LEVEL SECURITY;
ALTER TABLE coa.account               ENABLE ROW LEVEL SECURITY;
ALTER TABLE coa.account_segment       ENABLE ROW LEVEL SECURITY;
ALTER TABLE coa.segment_value         ENABLE ROW LEVEL SECURITY;
ALTER TABLE coa.account_combination   ENABLE ROW LEVEL SECURITY;
ALTER TABLE coa.posting_rule          ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON coa.chart_of_accounts      USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON fin.fiscal_calendar        USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON fin.fiscal_year            USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON fin.fiscal_period          USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON fin.financial_organization USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON fin.ledger                 USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON coa.account                USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON coa.account_segment        USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON coa.segment_value          USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON coa.account_combination    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON coa.posting_rule           USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON coa.chart_of_accounts      FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON fin.fiscal_calendar        FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON fin.fiscal_year            FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON fin.fiscal_period          FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON fin.financial_organization FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON fin.ledger                 FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON coa.account                FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON coa.account_segment        FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON coa.segment_value          FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON coa.account_combination    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON coa.posting_rule           FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE coa.outbox (
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
CREATE INDEX idx_outbox_unpublished ON coa.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON coa.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON coa.outbox (correlation_id);

CREATE TABLE coa.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
