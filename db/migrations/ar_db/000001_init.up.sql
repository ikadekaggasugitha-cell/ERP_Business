-- =============================================================================
-- ar_db — Accounts Receivable (SVC-009)
-- DBDD BAB 5.4 | Standards DBS-001..016 | BRD 9.5.6 | INT-001
-- ar_invoice RANGE-partitioned by invoice_date (DBS-013).
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

CREATE SCHEMA IF NOT EXISTS ar;

-- =============================================================================
-- ar.ar_invoice  (PARTITION BY RANGE (invoice_date))
-- customer_id / sales_order_ref / gl_journal_ref are logical refs.
-- =============================================================================
CREATE TABLE ar.ar_invoice (
    id                   UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id      UUID           NOT NULL,
    invoice_no           VARCHAR(50)    NOT NULL,
    customer_id          UUID           NOT NULL,   -- logical ref masterdata_db
    invoice_date         DATE           NOT NULL,   -- partition key
    due_date             DATE           NOT NULL,
    currency_code        VARCHAR(3)     NOT NULL,
    total_amount         NUMERIC(20,4)  NOT NULL DEFAULT 0,
    outstanding_amount   NUMERIC(20,4)  NOT NULL DEFAULT 0,
    status               VARCHAR(20)    NOT NULL DEFAULT 'Issued',
    sales_order_ref      VARCHAR(100),               -- logical ref sales_db
    gl_journal_ref       VARCHAR(50),                -- logical ref gl_db
    created_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by           UUID           NOT NULL,
    updated_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by           UUID,
    deleted_at           TIMESTAMPTZ,
    row_version          INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_ar_invoice PRIMARY KEY (id, invoice_date),
    CONSTRAINT uq_ar_invoice_no UNIQUE (organization_id, invoice_no, invoice_date),
    CONSTRAINT ck_ar_invoice_status
        CHECK (status IN ('Issued','PartiallyPaid','Paid','Overdue','Cancelled'))
) PARTITION BY RANGE (invoice_date);

CREATE INDEX idx_ar_invoice_customer  ON ar.ar_invoice (customer_id);
CREATE INDEX idx_ar_invoice_due       ON ar.ar_invoice (due_date);
CREATE INDEX idx_ar_invoice_status    ON ar.ar_invoice (organization_id, status);
CREATE INDEX idx_ar_invoice_so_ref    ON ar.ar_invoice (sales_order_ref);
CREATE INDEX idx_ar_invoice_date_brin ON ar.ar_invoice USING brin (invoice_date);

-- =============================================================================
-- ar.ar_invoice_line
-- =============================================================================
CREATE TABLE ar.ar_invoice_line (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    ar_invoice_id    UUID           NOT NULL,
    invoice_date     DATE           NOT NULL,   -- denormalized for composite FK
    line_no          INTEGER        NOT NULL,
    account_id       UUID           NOT NULL,   -- logical ref coa_db
    item_id          UUID,                       -- logical ref masterdata_db
    quantity         NUMERIC(20,4),
    unit_price       NUMERIC(20,4),
    amount           NUMERIC(20,4)  NOT NULL DEFAULT 0,
    description      TEXT,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_ar_invoice_line PRIMARY KEY (id),
    CONSTRAINT uq_ar_invoice_line UNIQUE (ar_invoice_id, line_no),
    CONSTRAINT fk_ar_invoice_line_invoice
        FOREIGN KEY (ar_invoice_id, invoice_date)
        REFERENCES ar.ar_invoice (id, invoice_date) ON DELETE CASCADE
);
CREATE INDEX idx_ar_invoice_line_invoice ON ar.ar_invoice_line (ar_invoice_id);
CREATE INDEX idx_ar_invoice_line_account ON ar.ar_invoice_line (account_id);

-- =============================================================================
-- ar.ar_receipt_allocation  (receipt_id logical ref treasury_db)
-- =============================================================================
CREATE TABLE ar.ar_receipt_allocation (
    id                 UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id    UUID           NOT NULL,
    ar_invoice_id      UUID           NOT NULL,
    invoice_date       DATE           NOT NULL,   -- denormalized for composite FK
    receipt_id         UUID           NOT NULL,   -- logical ref treasury_db (payment)
    allocated_amount   NUMERIC(20,4)  NOT NULL DEFAULT 0,
    allocated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_at         TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by         UUID           NOT NULL,
    updated_at         TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by         UUID,
    deleted_at         TIMESTAMPTZ,
    row_version        INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_ar_receipt_allocation PRIMARY KEY (id),
    CONSTRAINT uq_ar_receipt_allocation UNIQUE (ar_invoice_id, receipt_id),
    CONSTRAINT fk_ar_alloc_invoice
        FOREIGN KEY (ar_invoice_id, invoice_date)
        REFERENCES ar.ar_invoice (id, invoice_date) ON DELETE CASCADE
);
CREATE INDEX idx_ar_alloc_invoice ON ar.ar_receipt_allocation (ar_invoice_id);
CREATE INDEX idx_ar_alloc_receipt ON ar.ar_receipt_allocation (receipt_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE ar.ar_invoice            ENABLE ROW LEVEL SECURITY;
ALTER TABLE ar.ar_invoice_line       ENABLE ROW LEVEL SECURITY;
ALTER TABLE ar.ar_receipt_allocation ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON ar.ar_invoice
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON ar.ar_invoice_line
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON ar.ar_receipt_allocation
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON ar.ar_invoice
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON ar.ar_invoice_line
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON ar.ar_receipt_allocation
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE ar.outbox (
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
CREATE INDEX idx_outbox_unpublished ON ar.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON ar.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON ar.outbox (correlation_id);

CREATE TABLE ar.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
