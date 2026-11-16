-- =============================================================================
-- gl_db — General Ledger (SVC-007)
-- DBDD BAB 5.2 | Standards DBS-001..016 | BRD 9.5.4 (BR-556..562)
-- journal_entry RANGE-partitioned by posting_date (DBS-013). Exactly-Once
-- posting (ADR-005) via correlation_id + outbox.
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

CREATE SCHEMA IF NOT EXISTS gl;

-- =============================================================================
-- gl.journal_entry  (PARTITION BY RANGE (posting_date))
-- PK/UNIQUE include partition key (PostgreSQL requirement).
-- reversed_entry_id is a logical self-ref (no composite self-FK).
-- =============================================================================
CREATE TABLE gl.journal_entry (
    id                          UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id             UUID           NOT NULL,
    journal_no                  VARCHAR(50)    NOT NULL,
    financial_organization_id   UUID           NOT NULL,   -- logical ref coa_db
    ledger_id                   UUID           NOT NULL,   -- logical ref coa_db
    fiscal_period_id            UUID           NOT NULL,   -- logical ref coa_db (BR-562)
    posting_date                DATE           NOT NULL,   -- partition key
    journal_source              VARCHAR(30)    NOT NULL,
    source_module               VARCHAR(50),
    source_document_ref         VARCHAR(100),
    currency_code               VARCHAR(3)     NOT NULL,
    exchange_rate               NUMERIC(20,8)  NOT NULL DEFAULT 1,
    status                      VARCHAR(20)    NOT NULL DEFAULT 'Draft',
    reversed_entry_id           UUID,                       -- logical ref (reversal)
    correlation_id              UUID           NOT NULL,
    posted_at                   TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by                  UUID           NOT NULL,
    updated_at                  TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by                  UUID,
    deleted_at                  TIMESTAMPTZ,
    row_version                 INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_journal_entry PRIMARY KEY (id, posting_date),
    CONSTRAINT uq_journal_entry_no UNIQUE (organization_id, journal_no, posting_date),
    CONSTRAINT ck_journal_entry_status CHECK (status IN ('Draft','Posted','Reversed')),
    CONSTRAINT ck_journal_entry_source
        CHECK (journal_source IN ('Manual','SubLedger','Recurring','Revaluation','Allocation'))
) PARTITION BY RANGE (posting_date);

CREATE INDEX idx_journal_entry_org_period ON gl.journal_entry (organization_id, fiscal_period_id);
CREATE INDEX idx_journal_entry_finorg     ON gl.journal_entry (financial_organization_id);
CREATE INDEX idx_journal_entry_ledger     ON gl.journal_entry (ledger_id);
CREATE INDEX idx_journal_entry_status     ON gl.journal_entry (status);
CREATE INDEX idx_journal_entry_source_doc ON gl.journal_entry (source_document_ref);
CREATE INDEX idx_journal_entry_correlation ON gl.journal_entry (correlation_id);
CREATE INDEX idx_journal_entry_posting_brin ON gl.journal_entry USING brin (posting_date);

-- =============================================================================
-- gl.journal_line  (child; posting_date denormalized for composite FK + pruning)
-- =============================================================================
CREATE TABLE gl.journal_line (
    id                       UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id          UUID           NOT NULL,
    journal_entry_id         UUID           NOT NULL,
    posting_date             DATE           NOT NULL,   -- denormalized from parent
    line_no                  INTEGER        NOT NULL,
    account_id               UUID           NOT NULL,   -- logical ref coa_db
    account_combination_id   UUID,
    debit_amount             NUMERIC(20,4)  NOT NULL DEFAULT 0,
    credit_amount            NUMERIC(20,4)  NOT NULL DEFAULT 0,
    functional_amount        NUMERIC(20,4)  NOT NULL DEFAULT 0,
    cost_center_id           UUID,
    project_id               UUID,
    description              TEXT,
    created_at               TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by               UUID           NOT NULL,
    updated_at               TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by               UUID,
    deleted_at               TIMESTAMPTZ,
    row_version              INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_journal_line PRIMARY KEY (id),
    CONSTRAINT uq_journal_line UNIQUE (journal_entry_id, line_no),
    CONSTRAINT fk_journal_line_entry
        FOREIGN KEY (journal_entry_id, posting_date)
        REFERENCES gl.journal_entry (id, posting_date) ON DELETE CASCADE,
    CONSTRAINT ck_journal_line_amounts
        CHECK (debit_amount >= 0 AND credit_amount >= 0)
);
CREATE INDEX idx_journal_line_entry   ON gl.journal_line (journal_entry_id);
CREATE INDEX idx_journal_line_account ON gl.journal_line (account_id);

-- =============================================================================
-- gl.recurring_journal_template (BR-559)
-- =============================================================================
CREATE TABLE gl.recurring_journal_template (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(50)  NOT NULL,
    frequency        VARCHAR(20)  NOT NULL,   -- Monthly/Quarterly/...
    next_run_date    DATE,
    template_lines   JSONB        NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_recurring_journal_template PRIMARY KEY (id),
    CONSTRAINT uq_recurring_journal_template UNIQUE (organization_id, code)
);
CREATE INDEX idx_recurring_template_lines ON gl.recurring_journal_template USING gin (template_lines);
CREATE INDEX idx_recurring_template_due   ON gl.recurring_journal_template (next_run_date) WHERE deleted_at IS NULL;

-- =============================================================================
-- gl.account_balance  (aggregated balances per account/period/ledger)
-- =============================================================================
CREATE TABLE gl.account_balance (
    id                UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID           NOT NULL,
    account_id        UUID           NOT NULL,   -- logical ref coa_db
    fiscal_period_id  UUID           NOT NULL,
    ledger_id         UUID           NOT NULL,
    opening_balance   NUMERIC(20,4)  NOT NULL DEFAULT 0,
    period_debit      NUMERIC(20,4)  NOT NULL DEFAULT 0,
    period_credit     NUMERIC(20,4)  NOT NULL DEFAULT 0,
    closing_balance   NUMERIC(20,4)  NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by        UUID           NOT NULL,
    updated_at        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by        UUID,
    deleted_at        TIMESTAMPTZ,
    row_version       INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_account_balance PRIMARY KEY (id),
    CONSTRAINT uq_account_balance UNIQUE (organization_id, ledger_id, account_id, fiscal_period_id)
);
CREATE INDEX idx_account_balance_account ON gl.account_balance (account_id, fiscal_period_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE gl.journal_entry               ENABLE ROW LEVEL SECURITY;
ALTER TABLE gl.journal_line                ENABLE ROW LEVEL SECURITY;
ALTER TABLE gl.recurring_journal_template  ENABLE ROW LEVEL SECURITY;
ALTER TABLE gl.account_balance             ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON gl.journal_entry
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON gl.journal_line
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON gl.recurring_journal_template
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON gl.account_balance
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON gl.journal_entry
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON gl.journal_line
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON gl.recurring_journal_template
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON gl.account_balance
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE gl.outbox (
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
CREATE INDEX idx_outbox_unpublished ON gl.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON gl.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON gl.outbox (correlation_id);

CREATE TABLE gl.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
