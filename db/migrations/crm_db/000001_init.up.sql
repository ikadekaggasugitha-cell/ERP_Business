-- =============================================================================
-- crm_db — Customer Relationship Management (SVC-028)
-- DBDD BAB 9.10 | Standards DBS-001..016 | BRD 9.15 (BR-776..785)
-- customer_ref / assignee_ref are logical refs (masterdata_db / user_db).
-- Note: the "Case" entity (BR-780) is realized as `support_case` to avoid the
-- reserved SQL keyword CASE.
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

CREATE SCHEMA IF NOT EXISTS crm;

-- =============================================================================
-- crm.lead (BR-777)
-- =============================================================================
CREATE TABLE crm.lead (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    code             VARCHAR(50)    NOT NULL,
    name             VARCHAR(200)   NOT NULL,
    source           VARCHAR(50),
    status           VARCHAR(20)    NOT NULL DEFAULT 'New',
    estimated_value  NUMERIC(20,4),
    customer_ref     UUID,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_lead PRIMARY KEY (id),
    CONSTRAINT uq_lead_org_code UNIQUE (organization_id, code),
    CONSTRAINT ck_lead_status CHECK (status IN ('New','Qualified','Converted','Lost'))
);

-- =============================================================================
-- crm.opportunity (BR-781)
-- =============================================================================
CREATE TABLE crm.opportunity (
    id                    UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id       UUID           NOT NULL,
    code                  VARCHAR(50)    NOT NULL,
    lead_id               UUID,
    customer_ref          UUID,
    stage                 VARCHAR(50)    NOT NULL,
    probability_pct       NUMERIC(5,2),
    estimated_value       NUMERIC(20,4),
    expected_close_date   DATE,
    status                VARCHAR(20)    NOT NULL DEFAULT 'Open',
    created_at            TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by            UUID           NOT NULL,
    updated_at            TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by            UUID,
    deleted_at            TIMESTAMPTZ,
    row_version           INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_opportunity PRIMARY KEY (id),
    CONSTRAINT uq_opportunity_org_code UNIQUE (organization_id, code),
    CONSTRAINT fk_opportunity_lead
        FOREIGN KEY (lead_id) REFERENCES crm.lead (id) ON DELETE SET NULL,
    CONSTRAINT ck_opportunity_status CHECK (status IN ('Open','Won','Lost')),
    CONSTRAINT ck_opportunity_prob CHECK (probability_pct IS NULL OR probability_pct BETWEEN 0 AND 100)
);

-- =============================================================================
-- crm.customer_segment (BR-778)
-- =============================================================================
CREATE TABLE crm.customer_segment (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(50)  NOT NULL,
    name             VARCHAR(150) NOT NULL,
    criteria         JSONB,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_customer_segment PRIMARY KEY (id),
    CONSTRAINT uq_customer_segment_org_code UNIQUE (organization_id, code)
);

-- =============================================================================
-- crm.interaction (BR-779)
-- =============================================================================
CREATE TABLE crm.interaction (
    id                UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID         NOT NULL,
    customer_ref      UUID,
    lead_id           UUID,
    interaction_type  VARCHAR(20)  NOT NULL,
    occurred_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    notes             TEXT,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by        UUID         NOT NULL,
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by        UUID,
    deleted_at        TIMESTAMPTZ,
    row_version       INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_interaction PRIMARY KEY (id),
    CONSTRAINT fk_interaction_lead
        FOREIGN KEY (lead_id) REFERENCES crm.lead (id) ON DELETE SET NULL,
    CONSTRAINT ck_interaction_type CHECK (interaction_type IN ('Call','Meeting','Email','Note'))
);

-- =============================================================================
-- crm.support_case (BR-780 — "Case")
-- =============================================================================
CREATE TABLE crm.support_case (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    case_no          VARCHAR(50)  NOT NULL,
    customer_ref     UUID,
    subject          VARCHAR(200) NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'Open',
    resolution       TEXT,
    opened_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    resolved_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_support_case PRIMARY KEY (id),
    CONSTRAINT uq_support_case_no UNIQUE (organization_id, case_no),
    CONSTRAINT ck_support_case_status CHECK (status IN ('Open','Escalated','Resolved','Closed'))
);

-- =============================================================================
-- crm.activity_task (BR-782)
-- =============================================================================
CREATE TABLE crm.activity_task (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    assignee_ref     UUID         NOT NULL,
    related_type     VARCHAR(30),
    related_ref      UUID,
    due_date         DATE,
    status           VARCHAR(20)  NOT NULL DEFAULT 'Pending',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_activity_task PRIMARY KEY (id),
    CONSTRAINT ck_activity_task_status CHECK (status IN ('Pending','Done'))
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_lead_status         ON crm.lead (organization_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_lead_customer       ON crm.lead (customer_ref);
CREATE INDEX idx_opportunity_lead    ON crm.opportunity (lead_id);
CREATE INDEX idx_opportunity_customer ON crm.opportunity (customer_ref);
CREATE INDEX idx_opportunity_status  ON crm.opportunity (organization_id, status);
CREATE INDEX idx_interaction_customer ON crm.interaction (customer_ref);
CREATE INDEX idx_interaction_lead    ON crm.interaction (lead_id);
CREATE INDEX idx_support_case_customer ON crm.support_case (customer_ref);
CREATE INDEX idx_support_case_status ON crm.support_case (organization_id, status);
CREATE INDEX idx_activity_task_assignee ON crm.activity_task (assignee_ref) WHERE status = 'Pending';

-- --- RLS --------------------------------------------------------------------
ALTER TABLE crm.lead             ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.opportunity      ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.customer_segment ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.interaction      ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.support_case     ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.activity_task    ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON crm.lead             USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON crm.opportunity      USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON crm.customer_segment USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON crm.interaction      USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON crm.support_case     USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON crm.activity_task    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON crm.lead             FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON crm.opportunity      FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON crm.customer_segment FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON crm.interaction      FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON crm.support_case     FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON crm.activity_task    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE crm.outbox (
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
CREATE INDEX idx_outbox_unpublished ON crm.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON crm.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON crm.outbox (correlation_id);

CREATE TABLE crm.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
