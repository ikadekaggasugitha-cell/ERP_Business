-- =============================================================================
-- sales_db — Sales (SVC-015)
-- DBDD BAB 6.4 | Standards DBS-001..016 | BRD 9.14 | INT-001
-- Order -> delivery (-> inventory.goods.issued -> ar.invoice.issued).
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

CREATE SCHEMA IF NOT EXISTS so;

-- =============================================================================
-- so.sales_order (+ so_line)
-- customer_id is a logical ref (masterdata_db).
-- =============================================================================
CREATE TABLE so.sales_order (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    so_no            VARCHAR(50)    NOT NULL,
    customer_id      UUID           NOT NULL,
    order_date       DATE           NOT NULL,
    currency_code    VARCHAR(3)     NOT NULL,
    total_amount     NUMERIC(20,4)  NOT NULL DEFAULT 0,
    status           VARCHAR(20)    NOT NULL DEFAULT 'Draft',
    correlation_id   UUID           NOT NULL,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_sales_order PRIMARY KEY (id),
    CONSTRAINT uq_sales_order_no UNIQUE (organization_id, so_no),
    CONSTRAINT ck_sales_order_status
        CHECK (status IN ('Draft','Confirmed','PartiallyDelivered','Delivered','Invoiced','Cancelled'))
);

CREATE TABLE so.so_line (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    sales_order_id   UUID           NOT NULL,
    line_no          INTEGER        NOT NULL,
    item_id          UUID           NOT NULL,   -- logical ref masterdata_db
    order_qty        NUMERIC(20,4)  NOT NULL DEFAULT 0,
    unit_price       NUMERIC(20,4)  NOT NULL DEFAULT 0,
    delivered_qty    NUMERIC(20,4)  NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_so_line PRIMARY KEY (id),
    CONSTRAINT uq_so_line UNIQUE (sales_order_id, line_no),
    CONSTRAINT fk_so_line_order
        FOREIGN KEY (sales_order_id) REFERENCES so.sales_order (id) ON DELETE CASCADE
);

-- =============================================================================
-- so.delivery (+ delivery_line) — warehouse_id logical ref (inventory_db)
-- =============================================================================
CREATE TABLE so.delivery (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    delivery_no      VARCHAR(50)    NOT NULL,
    sales_order_id   UUID           NOT NULL,
    warehouse_id     UUID           NOT NULL,   -- logical ref inventory_db
    delivery_date    DATE           NOT NULL,
    status           VARCHAR(20)    NOT NULL DEFAULT 'Draft',
    correlation_id   UUID           NOT NULL,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_delivery PRIMARY KEY (id),
    CONSTRAINT uq_delivery_no UNIQUE (organization_id, delivery_no),
    CONSTRAINT fk_delivery_order
        FOREIGN KEY (sales_order_id) REFERENCES so.sales_order (id) ON DELETE RESTRICT,
    CONSTRAINT ck_delivery_status
        CHECK (status IN ('Draft','Shipped','Delivered','Cancelled'))
);

CREATE TABLE so.delivery_line (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    delivery_id      UUID           NOT NULL,
    so_line_id       UUID           NOT NULL,   -- intra-DB FK
    line_no          INTEGER        NOT NULL,
    item_id          UUID           NOT NULL,   -- logical ref masterdata_db
    delivered_qty    NUMERIC(20,4)  NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_delivery_line PRIMARY KEY (id),
    CONSTRAINT uq_delivery_line UNIQUE (delivery_id, line_no),
    CONSTRAINT fk_delivery_line_delivery
        FOREIGN KEY (delivery_id) REFERENCES so.delivery (id) ON DELETE CASCADE,
    CONSTRAINT fk_delivery_line_so_line
        FOREIGN KEY (so_line_id) REFERENCES so.so_line (id) ON DELETE RESTRICT
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_sales_order_customer ON so.sales_order (customer_id);
CREATE INDEX idx_sales_order_status   ON so.sales_order (organization_id, status);
CREATE INDEX idx_sales_order_corr     ON so.sales_order (correlation_id);
CREATE INDEX idx_so_line_order        ON so.so_line (sales_order_id);
CREATE INDEX idx_so_line_item         ON so.so_line (item_id);
CREATE INDEX idx_delivery_order       ON so.delivery (sales_order_id);
CREATE INDEX idx_delivery_corr        ON so.delivery (correlation_id);
CREATE INDEX idx_delivery_line_delivery ON so.delivery_line (delivery_id);
CREATE INDEX idx_delivery_line_soline   ON so.delivery_line (so_line_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE so.sales_order   ENABLE ROW LEVEL SECURITY;
ALTER TABLE so.so_line       ENABLE ROW LEVEL SECURITY;
ALTER TABLE so.delivery      ENABLE ROW LEVEL SECURITY;
ALTER TABLE so.delivery_line ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON so.sales_order   USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON so.so_line       USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON so.delivery      USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON so.delivery_line USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON so.sales_order   FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON so.so_line       FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON so.delivery      FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON so.delivery_line FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE so.outbox (
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
CREATE INDEX idx_outbox_unpublished ON so.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON so.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON so.outbox (correlation_id);

CREATE TABLE so.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
