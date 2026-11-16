-- =============================================================================
-- project_db — Project Management (SVC-025)
-- DBDD BAB 9.7 | Standards DBS-001..016 | BRD 9.10 (BR-726..735)
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

CREATE SCHEMA IF NOT EXISTS prj;

-- =============================================================================
-- prj.project (BR-726) — organization_ref logical ref (org_db)
-- =============================================================================
CREATE TABLE prj.project (
    id                UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID         NOT NULL,
    code              VARCHAR(50)  NOT NULL,
    name              VARCHAR(200) NOT NULL,
    project_type      VARCHAR(50)  NOT NULL,
    organization_ref  UUID         NOT NULL,
    currency_code     VARCHAR(3)   NOT NULL,
    start_date        DATE,
    end_date          DATE,
    status            VARCHAR(20)  NOT NULL DEFAULT 'Draft',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by        UUID         NOT NULL,
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by        UUID,
    deleted_at        TIMESTAMPTZ,
    row_version       INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_project PRIMARY KEY (id),
    CONSTRAINT uq_project_org_code UNIQUE (organization_id, code),
    CONSTRAINT ck_project_status CHECK (status IN ('Draft','Active','OnHold','Closed')),
    CONSTRAINT ck_project_dates CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date)
);

-- =============================================================================
-- prj.wbs_element (BR-727 — hierarchy)
-- =============================================================================
CREATE TABLE prj.wbs_element (
    id           UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id UUID      NOT NULL,
    project_id   UUID         NOT NULL,
    parent_id    UUID,
    code         VARCHAR(50)  NOT NULL,
    name         VARCHAR(200) NOT NULL,
    level_no     INTEGER      NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by   UUID         NOT NULL,
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by   UUID,
    deleted_at   TIMESTAMPTZ,
    row_version  INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_wbs_element PRIMARY KEY (id),
    CONSTRAINT uq_wbs_element UNIQUE (project_id, code),
    CONSTRAINT fk_wbs_element_project
        FOREIGN KEY (project_id) REFERENCES prj.project (id) ON DELETE CASCADE,
    CONSTRAINT fk_wbs_element_parent
        FOREIGN KEY (parent_id) REFERENCES prj.wbs_element (id) ON DELETE RESTRICT
);

-- =============================================================================
-- prj.project_budget (BR-728) / project_cost (BR-729) / project_progress (BR-730)
-- =============================================================================
CREATE TABLE prj.project_budget (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    project_id       UUID           NOT NULL,
    wbs_element_id   UUID           NOT NULL,
    version          INTEGER        NOT NULL DEFAULT 1,
    budget_amount    NUMERIC(20,4)  NOT NULL DEFAULT 0,
    status           VARCHAR(20)    NOT NULL DEFAULT 'Draft',
    is_baseline      BOOLEAN        NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_project_budget PRIMARY KEY (id),
    CONSTRAINT fk_project_budget_project
        FOREIGN KEY (project_id) REFERENCES prj.project (id) ON DELETE CASCADE,
    CONSTRAINT fk_project_budget_wbs
        FOREIGN KEY (wbs_element_id) REFERENCES prj.wbs_element (id) ON DELETE CASCADE,
    CONSTRAINT ck_project_budget_status CHECK (status IN ('Draft','Approved'))
);

CREATE TABLE prj.project_cost (
    id                   UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id      UUID           NOT NULL,
    project_id           UUID           NOT NULL,
    wbs_element_id       UUID           NOT NULL,
    cost_type            VARCHAR(20)    NOT NULL,
    amount               NUMERIC(20,4)  NOT NULL DEFAULT 0,
    source_document_ref  VARCHAR(100),
    posting_date         DATE           NOT NULL,
    created_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by           UUID           NOT NULL,
    updated_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by           UUID,
    deleted_at           TIMESTAMPTZ,
    row_version          INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_project_cost PRIMARY KEY (id),
    CONSTRAINT fk_project_cost_project
        FOREIGN KEY (project_id) REFERENCES prj.project (id) ON DELETE CASCADE,
    CONSTRAINT fk_project_cost_wbs
        FOREIGN KEY (wbs_element_id) REFERENCES prj.wbs_element (id) ON DELETE RESTRICT,
    CONSTRAINT ck_project_cost_type CHECK (cost_type IN ('Labor','Material','Subcontractor','Overhead'))
);

CREATE TABLE prj.project_progress (
    id                    UUID          NOT NULL DEFAULT gen_random_uuid(),
    organization_id       UUID          NOT NULL,
    wbs_element_id        UUID          NOT NULL,
    period_date           DATE          NOT NULL,
    physical_progress_pct NUMERIC(5,2)  NOT NULL DEFAULT 0,
    recorded_at           TIMESTAMPTZ   NOT NULL DEFAULT now(),
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT now(),
    created_by            UUID          NOT NULL,
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_by            UUID,
    deleted_at            TIMESTAMPTZ,
    row_version           INTEGER       NOT NULL DEFAULT 1,
    CONSTRAINT pk_project_progress PRIMARY KEY (id),
    CONSTRAINT uq_project_progress UNIQUE (wbs_element_id, period_date),
    CONSTRAINT fk_project_progress_wbs
        FOREIGN KEY (wbs_element_id) REFERENCES prj.wbs_element (id) ON DELETE CASCADE,
    CONSTRAINT ck_project_progress_pct CHECK (physical_progress_pct BETWEEN 0 AND 100)
);

-- =============================================================================
-- prj.project_billing (BR-731) / prj.change_order (BR-733)
-- =============================================================================
CREATE TABLE prj.project_billing (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    project_id       UUID           NOT NULL,
    billing_type     VARCHAR(20)    NOT NULL,
    amount           NUMERIC(20,4)  NOT NULL DEFAULT 0,
    ar_invoice_ref   VARCHAR(100),
    status           VARCHAR(20)    NOT NULL DEFAULT 'Draft',
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_project_billing PRIMARY KEY (id),
    CONSTRAINT fk_project_billing_project
        FOREIGN KEY (project_id) REFERENCES prj.project (id) ON DELETE CASCADE,
    CONSTRAINT ck_project_billing_type CHECK (billing_type IN ('Milestone','Progress','TimeMaterial')),
    CONSTRAINT ck_project_billing_status CHECK (status IN ('Draft','Invoiced'))
);

CREATE TABLE prj.change_order (
    id                     UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id        UUID           NOT NULL,
    project_id             UUID           NOT NULL,
    description            TEXT           NOT NULL,
    budget_impact_amount   NUMERIC(20,4)  NOT NULL DEFAULT 0,
    schedule_impact_days   INTEGER        NOT NULL DEFAULT 0,
    status                 VARCHAR(20)    NOT NULL DEFAULT 'Draft',
    created_at             TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by             UUID           NOT NULL,
    updated_at             TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by             UUID,
    deleted_at             TIMESTAMPTZ,
    row_version            INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_change_order PRIMARY KEY (id),
    CONSTRAINT fk_change_order_project
        FOREIGN KEY (project_id) REFERENCES prj.project (id) ON DELETE CASCADE,
    CONSTRAINT ck_change_order_status CHECK (status IN ('Draft','Approved','Rejected'))
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_project_orgref       ON prj.project (organization_ref);
CREATE INDEX idx_project_status       ON prj.project (organization_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_wbs_element_project  ON prj.wbs_element (project_id);
CREATE INDEX idx_wbs_element_parent   ON prj.wbs_element (parent_id);
CREATE INDEX idx_project_budget_project ON prj.project_budget (project_id);
CREATE INDEX idx_project_budget_wbs     ON prj.project_budget (wbs_element_id);
CREATE INDEX idx_project_cost_project   ON prj.project_cost (project_id);
CREATE INDEX idx_project_cost_wbs       ON prj.project_cost (wbs_element_id);
CREATE INDEX idx_project_progress_wbs   ON prj.project_progress (wbs_element_id);
CREATE INDEX idx_project_billing_project ON prj.project_billing (project_id);
CREATE INDEX idx_change_order_project    ON prj.change_order (project_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE prj.project          ENABLE ROW LEVEL SECURITY;
ALTER TABLE prj.wbs_element      ENABLE ROW LEVEL SECURITY;
ALTER TABLE prj.project_budget   ENABLE ROW LEVEL SECURITY;
ALTER TABLE prj.project_cost     ENABLE ROW LEVEL SECURITY;
ALTER TABLE prj.project_progress ENABLE ROW LEVEL SECURITY;
ALTER TABLE prj.project_billing  ENABLE ROW LEVEL SECURITY;
ALTER TABLE prj.change_order     ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON prj.project          USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON prj.wbs_element      USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON prj.project_budget   USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON prj.project_cost     USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON prj.project_progress USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON prj.project_billing  USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON prj.change_order     USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON prj.project          FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON prj.wbs_element      FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON prj.project_budget   FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON prj.project_cost     FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON prj.project_progress FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON prj.project_billing  FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON prj.change_order     FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE prj.outbox (
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
CREATE INDEX idx_outbox_unpublished ON prj.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON prj.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON prj.outbox (correlation_id);

CREATE TABLE prj.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
