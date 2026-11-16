-- =============================================================================
-- financialclose_db — Financial Close (SVC-023)
-- DBDD BAB 9.5 | Standards DBS-001..016 | BRD 9.5.14 (BR-656..665)
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

CREATE SCHEMA IF NOT EXISTS fcl;

-- =============================================================================
-- fcl.closing_checklist_template (+ closing_task_template) — BR-656
-- =============================================================================
CREATE TABLE fcl.closing_checklist_template (
    id                            UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id               UUID         NOT NULL,
    code                          VARCHAR(50)  NOT NULL,
    name                          VARCHAR(150) NOT NULL,
    financial_organization_ref    UUID         NOT NULL,
    created_at                    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by                    UUID         NOT NULL,
    updated_at                    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by                    UUID,
    deleted_at                    TIMESTAMPTZ,
    row_version                   INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_closing_checklist_template PRIMARY KEY (id),
    CONSTRAINT uq_closing_checklist_template UNIQUE (organization_id, code)
);

CREATE TABLE fcl.closing_task_template (
    id           UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id UUID      NOT NULL,
    template_id  UUID         NOT NULL,
    task_order   INTEGER      NOT NULL,
    task_name    VARCHAR(150) NOT NULL,
    is_mandatory BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by   UUID         NOT NULL,
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by   UUID,
    deleted_at   TIMESTAMPTZ,
    row_version  INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_closing_task_template PRIMARY KEY (id),
    CONSTRAINT uq_closing_task_template UNIQUE (template_id, task_order),
    CONSTRAINT fk_closing_task_template_tpl
        FOREIGN KEY (template_id) REFERENCES fcl.closing_checklist_template (id) ON DELETE CASCADE
);

-- =============================================================================
-- fcl.closing_run (+ closing_task) — BR-659/661/663
-- =============================================================================
CREATE TABLE fcl.closing_run (
    id                            UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id               UUID         NOT NULL,
    financial_organization_ref    UUID         NOT NULL,
    fiscal_period_ref             UUID         NOT NULL,
    run_type                      VARCHAR(20)  NOT NULL DEFAULT 'PeriodEnd',
    status                        VARCHAR(20)  NOT NULL DEFAULT 'Open',
    closed_at                     TIMESTAMPTZ,
    closed_by                     UUID,
    created_at                    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by                    UUID         NOT NULL,
    updated_at                    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by                    UUID,
    deleted_at                    TIMESTAMPTZ,
    row_version                   INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_closing_run PRIMARY KEY (id),
    CONSTRAINT uq_closing_run UNIQUE (organization_id, financial_organization_ref, fiscal_period_ref, run_type),
    CONSTRAINT ck_closing_run_type CHECK (run_type IN ('PeriodEnd','YearEnd')),
    CONSTRAINT ck_closing_run_status CHECK (status IN ('Open','PendingClose','Closed','Locked'))
);

CREATE TABLE fcl.closing_task (
    id              UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id UUID         NOT NULL,
    closing_run_id  UUID         NOT NULL,
    task_name       VARCHAR(150) NOT NULL,
    is_mandatory    BOOLEAN      NOT NULL DEFAULT TRUE,
    status          VARCHAR(20)  NOT NULL DEFAULT 'Pending',
    assignee_ref    UUID,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by      UUID         NOT NULL,
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by      UUID,
    deleted_at      TIMESTAMPTZ,
    row_version     INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_closing_task PRIMARY KEY (id),
    CONSTRAINT fk_closing_task_run
        FOREIGN KEY (closing_run_id) REFERENCES fcl.closing_run (id) ON DELETE CASCADE,
    CONSTRAINT ck_closing_task_status CHECK (status IN ('Pending','InProgress','Completed'))
);

-- =============================================================================
-- fcl.period_status (BR-659 — period lock per scope/module)
-- =============================================================================
CREATE TABLE fcl.period_status (
    id                            UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id               UUID         NOT NULL,
    financial_organization_ref    UUID         NOT NULL,
    fiscal_period_ref             UUID         NOT NULL,
    module                        VARCHAR(30)  NOT NULL,
    status                        VARCHAR(20)  NOT NULL DEFAULT 'Open',
    created_at                    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by                    UUID         NOT NULL,
    updated_at                    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by                    UUID,
    deleted_at                    TIMESTAMPTZ,
    row_version                   INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_period_status PRIMARY KEY (id),
    CONSTRAINT uq_period_status UNIQUE (organization_id, financial_organization_ref, fiscal_period_ref, module),
    CONSTRAINT ck_period_status CHECK (status IN ('Open','PendingClose','Closed','Locked'))
);

-- =============================================================================
-- fcl.period_reopening (BR-664 — exception log)
-- =============================================================================
CREATE TABLE fcl.period_reopening (
    id                            UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id               UUID         NOT NULL,
    financial_organization_ref    UUID         NOT NULL,
    fiscal_period_ref             UUID         NOT NULL,
    requested_by                  UUID         NOT NULL,
    approved_by                   UUID,
    reason                        TEXT         NOT NULL,
    reopened_at                   TIMESTAMPTZ,
    created_at                    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by                    UUID         NOT NULL,
    updated_at                    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by                    UUID,
    deleted_at                    TIMESTAMPTZ,
    row_version                   INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_period_reopening PRIMARY KEY (id)
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_closing_task_template_tpl ON fcl.closing_task_template (template_id);
CREATE INDEX idx_closing_run_scope         ON fcl.closing_run (financial_organization_ref, fiscal_period_ref);
CREATE INDEX idx_closing_run_status        ON fcl.closing_run (organization_id, status);
CREATE INDEX idx_closing_task_run          ON fcl.closing_task (closing_run_id);
CREATE INDEX idx_period_status_scope       ON fcl.period_status (financial_organization_ref, fiscal_period_ref);
CREATE INDEX idx_period_reopening_scope    ON fcl.period_reopening (financial_organization_ref, fiscal_period_ref);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE fcl.closing_checklist_template ENABLE ROW LEVEL SECURITY;
ALTER TABLE fcl.closing_task_template      ENABLE ROW LEVEL SECURITY;
ALTER TABLE fcl.closing_run                ENABLE ROW LEVEL SECURITY;
ALTER TABLE fcl.closing_task               ENABLE ROW LEVEL SECURITY;
ALTER TABLE fcl.period_status              ENABLE ROW LEVEL SECURITY;
ALTER TABLE fcl.period_reopening           ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON fcl.closing_checklist_template USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON fcl.closing_task_template      USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON fcl.closing_run                USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON fcl.closing_task               USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON fcl.period_status              USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON fcl.period_reopening           USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON fcl.closing_checklist_template FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON fcl.closing_task_template      FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON fcl.closing_run                FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON fcl.closing_task               FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON fcl.period_status              FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON fcl.period_reopening           FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE fcl.outbox (
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
CREATE INDEX idx_outbox_unpublished ON fcl.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON fcl.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON fcl.outbox (correlation_id);

CREATE TABLE fcl.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
