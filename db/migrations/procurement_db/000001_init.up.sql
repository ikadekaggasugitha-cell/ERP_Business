-- =============================================================================
-- procurement_db — Procurement (SVC-012)
-- DBDD BAB 6.2 | Standards DBS-001..016 | BRD 9.7 | INT-002/INT-004
-- Schemas: pr (requisition), po (purchase order), gr (goods receipt).
-- Technical tables (outbox/processed_event) live in schema pr.
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

CREATE SCHEMA IF NOT EXISTS pr;
CREATE SCHEMA IF NOT EXISTS po;
CREATE SCHEMA IF NOT EXISTS gr;

-- =============================================================================
-- pr.purchase_requisition (+ pr_line)
-- requester_id / workflow_ref are logical refs (user_db / workflow_db).
-- =============================================================================
CREATE TABLE pr.purchase_requisition (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    pr_no            VARCHAR(50)  NOT NULL,
    requester_id     UUID         NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'Draft',
    workflow_ref     VARCHAR(100),
    needed_date      DATE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_purchase_requisition PRIMARY KEY (id),
    CONSTRAINT uq_purchase_requisition_no UNIQUE (organization_id, pr_no),
    CONSTRAINT ck_purchase_requisition_status
        CHECK (status IN ('Draft','Submitted','Approved','Rejected','Closed'))
);

CREATE TABLE pr.pr_line (
    id                        UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id           UUID           NOT NULL,
    purchase_requisition_id   UUID           NOT NULL,
    line_no                   INTEGER        NOT NULL,
    item_id                   UUID           NOT NULL,   -- logical ref masterdata_db
    request_qty               NUMERIC(20,4)  NOT NULL DEFAULT 0,
    needed_date               DATE,
    created_at                TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by                UUID           NOT NULL,
    updated_at                TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by                UUID,
    deleted_at                TIMESTAMPTZ,
    row_version               INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_pr_line PRIMARY KEY (id),
    CONSTRAINT uq_pr_line UNIQUE (purchase_requisition_id, line_no),
    CONSTRAINT fk_pr_line_pr
        FOREIGN KEY (purchase_requisition_id) REFERENCES pr.purchase_requisition (id) ON DELETE CASCADE
);

-- =============================================================================
-- po.purchase_order (+ po_line)
-- vendor_id / item_id are logical refs; purchase_requisition_id intra-DB FK.
-- =============================================================================
CREATE TABLE po.purchase_order (
    id                        UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id           UUID           NOT NULL,
    po_no                     VARCHAR(50)    NOT NULL,
    vendor_id                 UUID           NOT NULL,   -- logical ref masterdata_db
    purchase_requisition_id   UUID,                       -- sourced-from PR (intra-DB)
    order_date                DATE           NOT NULL,
    currency_code             VARCHAR(3)     NOT NULL,
    total_amount              NUMERIC(20,4)  NOT NULL DEFAULT 0,
    status                    VARCHAR(20)    NOT NULL DEFAULT 'Draft',
    created_at                TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by                UUID           NOT NULL,
    updated_at                TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by                UUID,
    deleted_at                TIMESTAMPTZ,
    row_version               INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_purchase_order PRIMARY KEY (id),
    CONSTRAINT uq_purchase_order_no UNIQUE (organization_id, po_no),
    CONSTRAINT fk_purchase_order_pr
        FOREIGN KEY (purchase_requisition_id) REFERENCES pr.purchase_requisition (id) ON DELETE RESTRICT,
    CONSTRAINT ck_purchase_order_status
        CHECK (status IN ('Draft','Approved','PartiallyReceived','Received','Closed'))
);

CREATE TABLE po.po_line (
    id                  UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id     UUID           NOT NULL,
    purchase_order_id   UUID           NOT NULL,
    line_no             INTEGER        NOT NULL,
    item_id             UUID           NOT NULL,   -- logical ref masterdata_db
    order_qty           NUMERIC(20,4)  NOT NULL DEFAULT 0,
    received_qty        NUMERIC(20,4)  NOT NULL DEFAULT 0,
    unit_price          NUMERIC(20,4)  NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by          UUID           NOT NULL,
    updated_at          TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by          UUID,
    deleted_at          TIMESTAMPTZ,
    row_version         INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_po_line PRIMARY KEY (id),
    CONSTRAINT uq_po_line UNIQUE (purchase_order_id, line_no),
    CONSTRAINT fk_po_line_po
        FOREIGN KEY (purchase_order_id) REFERENCES po.purchase_order (id) ON DELETE CASCADE
);

-- =============================================================================
-- gr.goods_receipt (+ gr_line) — emits procurement.goods.received (INT-002)
-- warehouse_id is a logical ref (inventory_db).
-- =============================================================================
CREATE TABLE gr.goods_receipt (
    id                  UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id     UUID           NOT NULL,
    gr_no               VARCHAR(50)    NOT NULL,
    purchase_order_id   UUID           NOT NULL,   -- intra-DB FK
    warehouse_id        UUID           NOT NULL,   -- logical ref inventory_db
    receipt_date        DATE           NOT NULL,
    status              VARCHAR(20)    NOT NULL DEFAULT 'Draft',
    correlation_id      UUID           NOT NULL,
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by          UUID           NOT NULL,
    updated_at          TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by          UUID,
    deleted_at          TIMESTAMPTZ,
    row_version         INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_goods_receipt PRIMARY KEY (id),
    CONSTRAINT uq_goods_receipt_no UNIQUE (organization_id, gr_no),
    CONSTRAINT fk_goods_receipt_po
        FOREIGN KEY (purchase_order_id) REFERENCES po.purchase_order (id) ON DELETE RESTRICT
);

CREATE TABLE gr.gr_line (
    id                 UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id    UUID           NOT NULL,
    goods_receipt_id   UUID           NOT NULL,
    po_line_id         UUID           NOT NULL,   -- intra-DB FK
    line_no            INTEGER        NOT NULL,
    received_qty       NUMERIC(20,4)  NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by         UUID           NOT NULL,
    updated_at         TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by         UUID,
    deleted_at         TIMESTAMPTZ,
    row_version        INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_gr_line PRIMARY KEY (id),
    CONSTRAINT uq_gr_line UNIQUE (goods_receipt_id, line_no),
    CONSTRAINT fk_gr_line_gr
        FOREIGN KEY (goods_receipt_id) REFERENCES gr.goods_receipt (id) ON DELETE CASCADE,
    CONSTRAINT fk_gr_line_po_line
        FOREIGN KEY (po_line_id) REFERENCES po.po_line (id) ON DELETE RESTRICT
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_pr_requester    ON pr.purchase_requisition (requester_id);
CREATE INDEX idx_pr_status       ON pr.purchase_requisition (organization_id, status);
CREATE INDEX idx_pr_line_pr      ON pr.pr_line (purchase_requisition_id);
CREATE INDEX idx_po_vendor       ON po.purchase_order (vendor_id);
CREATE INDEX idx_po_status       ON po.purchase_order (organization_id, status);
CREATE INDEX idx_po_pr           ON po.purchase_order (purchase_requisition_id);
CREATE INDEX idx_po_line_po      ON po.po_line (purchase_order_id);
CREATE INDEX idx_po_line_item    ON po.po_line (item_id);
CREATE INDEX idx_gr_po           ON gr.goods_receipt (purchase_order_id);
CREATE INDEX idx_gr_correlation  ON gr.goods_receipt (correlation_id);
CREATE INDEX idx_gr_line_gr      ON gr.gr_line (goods_receipt_id);
CREATE INDEX idx_gr_line_po_line ON gr.gr_line (po_line_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE pr.purchase_requisition ENABLE ROW LEVEL SECURITY;
ALTER TABLE pr.pr_line              ENABLE ROW LEVEL SECURITY;
ALTER TABLE po.purchase_order       ENABLE ROW LEVEL SECURITY;
ALTER TABLE po.po_line              ENABLE ROW LEVEL SECURITY;
ALTER TABLE gr.goods_receipt        ENABLE ROW LEVEL SECURITY;
ALTER TABLE gr.gr_line              ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON pr.purchase_requisition USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON pr.pr_line              USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON po.purchase_order       USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON po.po_line              USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON gr.goods_receipt        USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON gr.gr_line              USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON pr.purchase_requisition FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON pr.pr_line              FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON po.purchase_order       FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON po.po_line              FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON gr.goods_receipt        FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON gr.gr_line              FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE pr.outbox (
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
CREATE INDEX idx_outbox_unpublished ON pr.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON pr.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON pr.outbox (correlation_id);

CREATE TABLE pr.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
