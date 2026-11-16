-- =============================================================================
-- inventory_db — Inventory (SVC-013)
-- DBDD BAB 6.3 | Standards DBS-001..016 | BRD 9.8 | INT-001/002
-- stock_movement RANGE-partitioned by movement_date (DBS-013).
-- Reservation is Exactly-Once (ADR-005): balance updates use optimistic
-- locking (row_version) / row locks in the application layer.
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

CREATE SCHEMA IF NOT EXISTS inv;

-- =============================================================================
-- inv.warehouse
-- =============================================================================
CREATE TABLE inv.warehouse (
    id                    UUID          NOT NULL DEFAULT gen_random_uuid(),
    organization_id       UUID          NOT NULL,
    code                  VARCHAR(50)   NOT NULL,
    name                  VARCHAR(150)  NOT NULL,
    organization_ref_id   UUID          NOT NULL,   -- logical ref org_db
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT now(),
    created_by            UUID          NOT NULL,
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_by            UUID,
    deleted_at            TIMESTAMPTZ,
    row_version           INTEGER       NOT NULL DEFAULT 1,
    CONSTRAINT pk_warehouse PRIMARY KEY (id),
    CONSTRAINT uq_warehouse_org_code UNIQUE (organization_id, code)
);

-- =============================================================================
-- inv.stock_balance  (available_qty derived = on_hand - reserved)
-- =============================================================================
CREATE TABLE inv.stock_balance (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    warehouse_id     UUID           NOT NULL,
    item_id          UUID           NOT NULL,   -- logical ref masterdata_db
    on_hand_qty      NUMERIC(20,4)  NOT NULL DEFAULT 0,
    reserved_qty     NUMERIC(20,4)  NOT NULL DEFAULT 0,
    available_qty    NUMERIC(20,4)  GENERATED ALWAYS AS (on_hand_qty - reserved_qty) STORED,
    avg_cost         NUMERIC(20,4)  NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_stock_balance PRIMARY KEY (id),
    CONSTRAINT uq_stock_balance UNIQUE (organization_id, warehouse_id, item_id),
    CONSTRAINT fk_stock_balance_warehouse
        FOREIGN KEY (warehouse_id) REFERENCES inv.warehouse (id) ON DELETE RESTRICT
);

-- =============================================================================
-- inv.stock_movement  (PARTITION BY RANGE (movement_date))
-- =============================================================================
CREATE TABLE inv.stock_movement (
    id                   UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id      UUID           NOT NULL,
    movement_date        DATE           NOT NULL,   -- partition key
    warehouse_id         UUID           NOT NULL,
    item_id              UUID           NOT NULL,   -- logical ref masterdata_db
    movement_type        VARCHAR(20)    NOT NULL,   -- Receipt/Issue/Transfer/Adjustment
    quantity             NUMERIC(20,4)  NOT NULL,   -- +/-
    source_document_ref  VARCHAR(100),               -- GR/SO/WO (logical ref)
    correlation_id       UUID           NOT NULL,
    created_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by           UUID           NOT NULL,
    updated_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by           UUID,
    deleted_at           TIMESTAMPTZ,
    row_version          INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_stock_movement PRIMARY KEY (id, movement_date),
    CONSTRAINT fk_stock_movement_warehouse
        FOREIGN KEY (warehouse_id) REFERENCES inv.warehouse (id) ON DELETE RESTRICT,
    CONSTRAINT ck_stock_movement_type
        CHECK (movement_type IN ('Receipt','Issue','Transfer','Adjustment'))
) PARTITION BY RANGE (movement_date);

CREATE INDEX idx_stock_movement_wh_item   ON inv.stock_movement (organization_id, warehouse_id, item_id);
CREATE INDEX idx_stock_movement_source    ON inv.stock_movement (source_document_ref);
CREATE INDEX idx_stock_movement_correlation ON inv.stock_movement (correlation_id);
CREATE INDEX idx_stock_movement_date_brin ON inv.stock_movement USING brin (movement_date);

-- =============================================================================
-- inv.stock_reservation
-- =============================================================================
CREATE TABLE inv.stock_reservation (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    item_id          UUID           NOT NULL,   -- logical ref masterdata_db
    warehouse_id     UUID           NOT NULL,
    reserved_qty     NUMERIC(20,4)  NOT NULL DEFAULT 0,
    sales_order_ref  VARCHAR(100)   NOT NULL,   -- SO trigger (INT-001)
    status           VARCHAR(20)    NOT NULL DEFAULT 'Reserved',
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_stock_reservation PRIMARY KEY (id),
    CONSTRAINT fk_stock_reservation_warehouse
        FOREIGN KEY (warehouse_id) REFERENCES inv.warehouse (id) ON DELETE RESTRICT,
    CONSTRAINT ck_stock_reservation_status
        CHECK (status IN ('Reserved','Released','Consumed'))
);
CREATE INDEX idx_stock_reservation_item ON inv.stock_reservation (organization_id, item_id);
CREATE INDEX idx_stock_reservation_so   ON inv.stock_reservation (sales_order_ref);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE inv.warehouse         ENABLE ROW LEVEL SECURITY;
ALTER TABLE inv.stock_balance     ENABLE ROW LEVEL SECURITY;
ALTER TABLE inv.stock_movement    ENABLE ROW LEVEL SECURITY;
ALTER TABLE inv.stock_reservation ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON inv.warehouse         USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON inv.stock_balance     USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON inv.stock_movement    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON inv.stock_reservation USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON inv.warehouse         FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON inv.stock_balance     FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON inv.stock_movement    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON inv.stock_reservation FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE inv.outbox (
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
CREATE INDEX idx_outbox_unpublished ON inv.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON inv.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON inv.outbox (correlation_id);

CREATE TABLE inv.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
