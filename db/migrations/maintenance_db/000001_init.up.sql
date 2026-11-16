-- =============================================================================
-- maintenance_db — Maintenance Management (SVC-027)
-- DBDD BAB 9.9 | Standards DBS-001..016 | BRD 9.16 (BR-786..795)
-- asset_ref / item_ref / vendor_ref / assignee_ref are logical refs.
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

CREATE SCHEMA IF NOT EXISTS mnt;

-- =============================================================================
-- mnt.maintenance_schedule (BR-786)
-- =============================================================================
CREATE TABLE mnt.maintenance_schedule (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    asset_ref        UUID         NOT NULL,
    schedule_type    VARCHAR(20)  NOT NULL,
    interval_value   INTEGER      NOT NULL,
    interval_unit    VARCHAR(20)  NOT NULL,
    next_due_date    DATE,
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_maintenance_schedule PRIMARY KEY (id),
    CONSTRAINT ck_maintenance_schedule_type CHECK (schedule_type IN ('TimeBased','UsageBased')),
    CONSTRAINT ck_maintenance_schedule_unit CHECK (interval_unit IN ('Day','Month','Hour','Cycle'))
);

-- =============================================================================
-- mnt.work_order (BR-787/788/791)
-- =============================================================================
CREATE TABLE mnt.work_order (
    id                UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID         NOT NULL,
    wo_no             VARCHAR(50)  NOT NULL,
    asset_ref         UUID         NOT NULL,
    maintenance_type  VARCHAR(20)  NOT NULL,
    status            VARCHAR(20)  NOT NULL DEFAULT 'Open',
    assignee_ref      UUID,
    vendor_ref        UUID,
    scheduled_date    DATE,
    downtime_start    TIMESTAMPTZ,
    downtime_end      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by        UUID         NOT NULL,
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by        UUID,
    deleted_at        TIMESTAMPTZ,
    row_version       INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_work_order PRIMARY KEY (id),
    CONSTRAINT uq_work_order_no UNIQUE (organization_id, wo_no),
    CONSTRAINT ck_work_order_type CHECK (maintenance_type IN ('Preventive','Corrective')),
    CONSTRAINT ck_work_order_status CHECK (status IN ('Open','Assigned','InProgress','Completed','Closed'))
);

-- =============================================================================
-- mnt.work_order_part (BR-789) / work_order_checklist (BR-793) / maintenance_cost (BR-790)
-- =============================================================================
CREATE TABLE mnt.work_order_part (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    work_order_id    UUID           NOT NULL,
    item_ref         UUID           NOT NULL,
    quantity         NUMERIC(20,4)  NOT NULL DEFAULT 0,
    unit_cost        NUMERIC(20,4)  NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_work_order_part PRIMARY KEY (id),
    CONSTRAINT fk_work_order_part_wo
        FOREIGN KEY (work_order_id) REFERENCES mnt.work_order (id) ON DELETE CASCADE
);

CREATE TABLE mnt.work_order_checklist (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    work_order_id    UUID         NOT NULL,
    step_no          INTEGER      NOT NULL,
    description      VARCHAR(200) NOT NULL,
    is_done          BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_work_order_checklist PRIMARY KEY (id),
    CONSTRAINT uq_work_order_checklist UNIQUE (work_order_id, step_no),
    CONSTRAINT fk_work_order_checklist_wo
        FOREIGN KEY (work_order_id) REFERENCES mnt.work_order (id) ON DELETE CASCADE
);

CREATE TABLE mnt.maintenance_cost (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    work_order_id    UUID           NOT NULL,
    cost_type        VARCHAR(20)    NOT NULL,
    amount           NUMERIC(20,4)  NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_maintenance_cost PRIMARY KEY (id),
    CONSTRAINT fk_maintenance_cost_wo
        FOREIGN KEY (work_order_id) REFERENCES mnt.work_order (id) ON DELETE CASCADE,
    CONSTRAINT ck_maintenance_cost_type CHECK (cost_type IN ('Labor','Part','Vendor'))
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_maintenance_schedule_asset ON mnt.maintenance_schedule (asset_ref);
CREATE INDEX idx_maintenance_schedule_due   ON mnt.maintenance_schedule (next_due_date) WHERE is_active;
CREATE INDEX idx_work_order_asset           ON mnt.work_order (asset_ref);
CREATE INDEX idx_work_order_status          ON mnt.work_order (organization_id, status);
CREATE INDEX idx_work_order_part_wo         ON mnt.work_order_part (work_order_id);
CREATE INDEX idx_work_order_checklist_wo    ON mnt.work_order_checklist (work_order_id);
CREATE INDEX idx_maintenance_cost_wo        ON mnt.maintenance_cost (work_order_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE mnt.maintenance_schedule ENABLE ROW LEVEL SECURITY;
ALTER TABLE mnt.work_order           ENABLE ROW LEVEL SECURITY;
ALTER TABLE mnt.work_order_part      ENABLE ROW LEVEL SECURITY;
ALTER TABLE mnt.work_order_checklist ENABLE ROW LEVEL SECURITY;
ALTER TABLE mnt.maintenance_cost     ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON mnt.maintenance_schedule USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON mnt.work_order           USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON mnt.work_order_part      USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON mnt.work_order_checklist USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON mnt.maintenance_cost     USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON mnt.maintenance_schedule FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON mnt.work_order           FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON mnt.work_order_part      FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON mnt.work_order_checklist FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON mnt.maintenance_cost     FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE mnt.outbox (
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
CREATE INDEX idx_outbox_unpublished ON mnt.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON mnt.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON mnt.outbox (correlation_id);

CREATE TABLE mnt.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
