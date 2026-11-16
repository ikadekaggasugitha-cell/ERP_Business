-- =============================================================================
-- workflow_db — Workflow Management (SVC-004)
-- DBDD BAB 3.5 | Standards DBS-001..016 | BRD 9.3, INT-004
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

CREATE SCHEMA IF NOT EXISTS wf;

-- =============================================================================
-- wf.workflow_definition
-- =============================================================================
CREATE TABLE wf.workflow_definition (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(50)  NOT NULL,   -- e.g. PR_APPROVAL
    document_type    VARCHAR(50)  NOT NULL,
    version          INTEGER      NOT NULL DEFAULT 1,
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_workflow_definition PRIMARY KEY (id),
    CONSTRAINT uq_workflow_definition UNIQUE (organization_id, code, version)
);

-- =============================================================================
-- wf.workflow_step
-- =============================================================================
CREATE TABLE wf.workflow_step (
    id               UUID        NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID        NOT NULL,
    definition_id    UUID        NOT NULL,
    step_order       INTEGER     NOT NULL,
    approver_rule    JSONB       NOT NULL,    -- role/scope/condition
    escalation_rule  JSONB,                   -- SLA/escalation (optional)
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by       UUID        NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER     NOT NULL DEFAULT 1,
    CONSTRAINT pk_workflow_step PRIMARY KEY (id),
    CONSTRAINT uq_workflow_step UNIQUE (definition_id, step_order),
    CONSTRAINT fk_workflow_step_definition
        FOREIGN KEY (definition_id) REFERENCES wf.workflow_definition (id) ON DELETE CASCADE
);

-- =============================================================================
-- wf.workflow_instance
-- =============================================================================
CREATE TABLE wf.workflow_instance (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    definition_id    UUID         NOT NULL,
    document_ref     VARCHAR(100) NOT NULL,   -- source document id (PR/PO/...)
    status           VARCHAR(20)  NOT NULL DEFAULT 'Pending',
    current_step     INTEGER,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_workflow_instance PRIMARY KEY (id),
    CONSTRAINT fk_workflow_instance_definition
        FOREIGN KEY (definition_id) REFERENCES wf.workflow_definition (id) ON DELETE RESTRICT,
    CONSTRAINT ck_workflow_instance_status
        CHECK (status IN ('Pending','Approved','Rejected','Cancelled'))
);

-- =============================================================================
-- wf.workflow_task
-- assignee_id is a logical ref to user_db.
-- =============================================================================
CREATE TABLE wf.workflow_task (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    instance_id      UUID         NOT NULL,
    assignee_id      UUID         NOT NULL,
    decision         VARCHAR(20),             -- Approve/Reject/Delegate
    decided_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_workflow_task PRIMARY KEY (id),
    CONSTRAINT fk_workflow_task_instance
        FOREIGN KEY (instance_id) REFERENCES wf.workflow_instance (id) ON DELETE CASCADE,
    CONSTRAINT ck_workflow_task_decision
        CHECK (decision IS NULL OR decision IN ('Approve','Reject','Delegate'))
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_workflow_definition_active ON wf.workflow_definition (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_workflow_step_definition   ON wf.workflow_step (definition_id);
CREATE INDEX idx_workflow_step_rule         ON wf.workflow_step USING gin (approver_rule);
CREATE INDEX idx_workflow_instance_doc      ON wf.workflow_instance (document_ref);
CREATE INDEX idx_workflow_instance_defn     ON wf.workflow_instance (definition_id);
CREATE INDEX idx_workflow_task_instance     ON wf.workflow_task (instance_id);
CREATE INDEX idx_workflow_task_assignee     ON wf.workflow_task (assignee_id) WHERE decision IS NULL;

-- --- RLS --------------------------------------------------------------------
ALTER TABLE wf.workflow_definition ENABLE ROW LEVEL SECURITY;
ALTER TABLE wf.workflow_step       ENABLE ROW LEVEL SECURITY;
ALTER TABLE wf.workflow_instance   ENABLE ROW LEVEL SECURITY;
ALTER TABLE wf.workflow_task       ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON wf.workflow_definition
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON wf.workflow_step
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON wf.workflow_instance
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON wf.workflow_task
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON wf.workflow_definition
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON wf.workflow_step
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON wf.workflow_instance
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON wf.workflow_task
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE wf.outbox (
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
CREATE INDEX idx_outbox_unpublished ON wf.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON wf.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON wf.outbox (correlation_id);

CREATE TABLE wf.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
