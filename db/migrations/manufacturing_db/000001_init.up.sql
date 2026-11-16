-- =============================================================================
-- manufacturing_db — Manufacturing (SVC-014)
-- DBDD BAB 6.5 | Standards DBS-001..016 | BRD 9.9
-- Emits manufacturing.workorder.completed.
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

CREATE SCHEMA IF NOT EXISTS mfg;

-- =============================================================================
-- mfg.bill_of_material (+ bom_line) — item refs are logical (masterdata_db)
-- =============================================================================
CREATE TABLE mfg.bill_of_material (
    id                UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID         NOT NULL,
    code              VARCHAR(50)  NOT NULL,
    product_item_id   UUID         NOT NULL,   -- logical ref masterdata_db
    version           INTEGER      NOT NULL DEFAULT 1,
    is_active         BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by        UUID         NOT NULL,
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by        UUID,
    deleted_at        TIMESTAMPTZ,
    row_version       INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_bill_of_material PRIMARY KEY (id),
    CONSTRAINT uq_bill_of_material UNIQUE (organization_id, code, version)
);

CREATE TABLE mfg.bom_line (
    id                   UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id      UUID           NOT NULL,
    bom_id               UUID           NOT NULL,
    line_no              INTEGER        NOT NULL,
    component_item_id    UUID           NOT NULL,   -- logical ref masterdata_db
    quantity             NUMERIC(20,4)  NOT NULL DEFAULT 0,
    uom                  VARCHAR(20)    NOT NULL,
    created_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by           UUID           NOT NULL,
    updated_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by           UUID,
    deleted_at           TIMESTAMPTZ,
    row_version          INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_bom_line PRIMARY KEY (id),
    CONSTRAINT uq_bom_line UNIQUE (bom_id, line_no),
    CONSTRAINT fk_bom_line_bom
        FOREIGN KEY (bom_id) REFERENCES mfg.bill_of_material (id) ON DELETE CASCADE
);

-- =============================================================================
-- mfg.work_order  — sales_order_ref logical ref (make-to-order)
-- =============================================================================
CREATE TABLE mfg.work_order (
    id                UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID           NOT NULL,
    wo_no             VARCHAR(50)    NOT NULL,
    product_item_id   UUID           NOT NULL,   -- logical ref masterdata_db
    bom_id            UUID           NOT NULL,
    planned_qty       NUMERIC(20,4)  NOT NULL DEFAULT 0,
    produced_qty      NUMERIC(20,4)  NOT NULL DEFAULT 0,
    status            VARCHAR(20)    NOT NULL DEFAULT 'Planned',
    sales_order_ref   VARCHAR(100),
    correlation_id    UUID           NOT NULL,
    created_at        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by        UUID           NOT NULL,
    updated_at        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by        UUID,
    deleted_at        TIMESTAMPTZ,
    row_version       INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_work_order PRIMARY KEY (id),
    CONSTRAINT uq_work_order_no UNIQUE (organization_id, wo_no),
    CONSTRAINT fk_work_order_bom
        FOREIGN KEY (bom_id) REFERENCES mfg.bill_of_material (id) ON DELETE RESTRICT,
    CONSTRAINT ck_work_order_status
        CHECK (status IN ('Planned','Released','InProgress','Completed','Closed'))
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_bom_active         ON mfg.bill_of_material (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_bom_product        ON mfg.bill_of_material (product_item_id);
CREATE INDEX idx_bom_line_bom       ON mfg.bom_line (bom_id);
CREATE INDEX idx_work_order_product ON mfg.work_order (product_item_id);
CREATE INDEX idx_work_order_status  ON mfg.work_order (organization_id, status);
CREATE INDEX idx_work_order_so_ref  ON mfg.work_order (sales_order_ref);
CREATE INDEX idx_work_order_bom     ON mfg.work_order (bom_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE mfg.bill_of_material ENABLE ROW LEVEL SECURITY;
ALTER TABLE mfg.bom_line         ENABLE ROW LEVEL SECURITY;
ALTER TABLE mfg.work_order       ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON mfg.bill_of_material USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON mfg.bom_line         USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON mfg.work_order       USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON mfg.bill_of_material FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON mfg.bom_line         FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON mfg.work_order       FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE mfg.outbox (
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
CREATE INDEX idx_outbox_unpublished ON mfg.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON mfg.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON mfg.outbox (correlation_id);

CREATE TABLE mfg.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
