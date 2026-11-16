-- =============================================================================
-- consolidation_db — Financial Consolidation (SVC-024)
-- DBDD BAB 9.6 | Standards DBS-001..016 | BRD 9.5.15 (BR-666..675)
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

CREATE SCHEMA IF NOT EXISTS cons;

-- =============================================================================
-- cons.consolidation_group  (multi-tier via parent_group_id — BR-666/675)
-- =============================================================================
CREATE TABLE cons.consolidation_group (
    id                           UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id              UUID         NOT NULL,
    code                         VARCHAR(50)  NOT NULL,
    name                         VARCHAR(150) NOT NULL,
    parent_group_id              UUID,
    presentation_currency_code   VARCHAR(3)   NOT NULL,
    created_at                   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by                   UUID         NOT NULL,
    updated_at                   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by                   UUID,
    deleted_at                   TIMESTAMPTZ,
    row_version                  INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_consolidation_group PRIMARY KEY (id),
    CONSTRAINT uq_consolidation_group UNIQUE (organization_id, code),
    CONSTRAINT fk_consolidation_group_parent
        FOREIGN KEY (parent_group_id) REFERENCES cons.consolidation_group (id) ON DELETE RESTRICT
);

-- =============================================================================
-- cons.consolidation_member (BR-667/671) — company_ref logical ref (org_db)
-- =============================================================================
CREATE TABLE cons.consolidation_member (
    id                       UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id          UUID           NOT NULL,
    consolidation_group_id   UUID           NOT NULL,
    company_ref              UUID           NOT NULL,
    ownership_percentage     NUMERIC(9,4)   NOT NULL,
    consolidation_method     VARCHAR(20)    NOT NULL,
    created_at               TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by               UUID           NOT NULL,
    updated_at               TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by               UUID,
    deleted_at               TIMESTAMPTZ,
    row_version              INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_consolidation_member PRIMARY KEY (id),
    CONSTRAINT uq_consolidation_member UNIQUE (consolidation_group_id, company_ref),
    CONSTRAINT fk_consolidation_member_group
        FOREIGN KEY (consolidation_group_id) REFERENCES cons.consolidation_group (id) ON DELETE CASCADE,
    CONSTRAINT ck_consolidation_method CHECK (consolidation_method IN ('Full','Proportional','Equity')),
    CONSTRAINT ck_ownership_pct CHECK (ownership_percentage >= 0 AND ownership_percentage <= 100)
);

-- =============================================================================
-- cons.elimination_rule (BR-669)
-- =============================================================================
CREATE TABLE cons.elimination_rule (
    id                       UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id          UUID         NOT NULL,
    consolidation_group_id   UUID         NOT NULL,
    rule_type                VARCHAR(40)  NOT NULL,
    config                   JSONB        NOT NULL,
    is_active                BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by               UUID         NOT NULL,
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by               UUID,
    deleted_at               TIMESTAMPTZ,
    row_version              INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_elimination_rule PRIMARY KEY (id),
    CONSTRAINT fk_elimination_rule_group
        FOREIGN KEY (consolidation_group_id) REFERENCES cons.consolidation_group (id) ON DELETE CASCADE,
    CONSTRAINT ck_elimination_rule_type
        CHECK (rule_type IN ('ICReceivablePayable','ICSalesPurchase','UnrealizedProfit','InvestmentEquity'))
);

-- =============================================================================
-- cons.consolidation_run (BR-673/674)
-- =============================================================================
CREATE TABLE cons.consolidation_run (
    id                       UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id          UUID         NOT NULL,
    consolidation_group_id   UUID         NOT NULL,
    fiscal_period_ref        UUID         NOT NULL,
    status                   VARCHAR(20)  NOT NULL DEFAULT 'Draft',
    version                  INTEGER      NOT NULL DEFAULT 1,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by               UUID         NOT NULL,
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by               UUID,
    deleted_at               TIMESTAMPTZ,
    row_version              INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_consolidation_run PRIMARY KEY (id),
    CONSTRAINT fk_consolidation_run_group
        FOREIGN KEY (consolidation_group_id) REFERENCES cons.consolidation_group (id) ON DELETE RESTRICT,
    CONSTRAINT ck_consolidation_run_status CHECK (status IN ('Draft','Eliminated','Adjusted','Finalized'))
);

-- =============================================================================
-- cons.elimination_entry (BR-669) / cons.consolidation_adjustment (BR-672)
-- account_ref = logical ref coa_db.
-- =============================================================================
CREATE TABLE cons.elimination_entry (
    id                     UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id        UUID           NOT NULL,
    consolidation_run_id   UUID           NOT NULL,
    elimination_rule_id    UUID,
    account_ref            UUID           NOT NULL,
    debit_amount           NUMERIC(20,4)  NOT NULL DEFAULT 0,
    credit_amount          NUMERIC(20,4)  NOT NULL DEFAULT 0,
    description            TEXT,
    created_at             TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by             UUID           NOT NULL,
    updated_at             TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by             UUID,
    deleted_at             TIMESTAMPTZ,
    row_version            INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_elimination_entry PRIMARY KEY (id),
    CONSTRAINT fk_elimination_entry_run
        FOREIGN KEY (consolidation_run_id) REFERENCES cons.consolidation_run (id) ON DELETE CASCADE,
    CONSTRAINT fk_elimination_entry_rule
        FOREIGN KEY (elimination_rule_id) REFERENCES cons.elimination_rule (id) ON DELETE SET NULL
);

CREATE TABLE cons.consolidation_adjustment (
    id                     UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id        UUID           NOT NULL,
    consolidation_run_id   UUID           NOT NULL,
    account_ref            UUID           NOT NULL,
    debit_amount           NUMERIC(20,4)  NOT NULL DEFAULT 0,
    credit_amount          NUMERIC(20,4)  NOT NULL DEFAULT 0,
    adjustment_type        VARCHAR(20)    NOT NULL,
    notes                  TEXT,
    created_at             TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by             UUID           NOT NULL,
    updated_at             TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by             UUID,
    deleted_at             TIMESTAMPTZ,
    row_version            INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_consolidation_adjustment PRIMARY KEY (id),
    CONSTRAINT fk_consolidation_adjustment_run
        FOREIGN KEY (consolidation_run_id) REFERENCES cons.consolidation_run (id) ON DELETE CASCADE,
    CONSTRAINT ck_consolidation_adjustment_type CHECK (adjustment_type IN ('TopSide','Reclass','Policy'))
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_consolidation_group_parent ON cons.consolidation_group (parent_group_id);
CREATE INDEX idx_consolidation_member_group ON cons.consolidation_member (consolidation_group_id);
CREATE INDEX idx_consolidation_member_company ON cons.consolidation_member (company_ref);
CREATE INDEX idx_elimination_rule_group     ON cons.elimination_rule (consolidation_group_id);
CREATE INDEX idx_elimination_rule_config    ON cons.elimination_rule USING gin (config);
CREATE INDEX idx_consolidation_run_group    ON cons.consolidation_run (consolidation_group_id, fiscal_period_ref);
CREATE INDEX idx_elimination_entry_run      ON cons.elimination_entry (consolidation_run_id);
CREATE INDEX idx_consolidation_adj_run      ON cons.consolidation_adjustment (consolidation_run_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE cons.consolidation_group      ENABLE ROW LEVEL SECURITY;
ALTER TABLE cons.consolidation_member     ENABLE ROW LEVEL SECURITY;
ALTER TABLE cons.elimination_rule         ENABLE ROW LEVEL SECURITY;
ALTER TABLE cons.consolidation_run        ENABLE ROW LEVEL SECURITY;
ALTER TABLE cons.elimination_entry        ENABLE ROW LEVEL SECURITY;
ALTER TABLE cons.consolidation_adjustment ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON cons.consolidation_group      USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON cons.consolidation_member     USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON cons.elimination_rule         USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON cons.consolidation_run        USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON cons.elimination_entry        USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON cons.consolidation_adjustment USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON cons.consolidation_group      FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON cons.consolidation_member     FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON cons.elimination_rule         FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON cons.consolidation_run        FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON cons.elimination_entry        FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON cons.consolidation_adjustment FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE cons.outbox (
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
CREATE INDEX idx_outbox_unpublished ON cons.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON cons.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON cons.outbox (correlation_id);

CREATE TABLE cons.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
