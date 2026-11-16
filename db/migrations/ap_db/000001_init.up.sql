-- =============================================================================
-- ap_db — Accounts Payable (SVC-008)
-- DBDD BAB 5.3 | Standards DBS-001..016 | BRD 9.5.5 | INT-002
-- ap_invoice RANGE-partitioned by invoice_date (DBS-013).
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

CREATE SCHEMA IF NOT EXISTS ap;

-- =============================================================================
-- ap.ap_invoice  (PARTITION BY RANGE (invoice_date))
-- vendor_id / goods_receipt_ref / gl_journal_ref are logical refs.
-- =============================================================================
CREATE TABLE ap.ap_invoice (
    id                   UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id      UUID           NOT NULL,
    invoice_no           VARCHAR(50)    NOT NULL,
    vendor_id            UUID           NOT NULL,   -- logical ref masterdata_db
    invoice_date         DATE           NOT NULL,   -- partition key
    due_date             DATE           NOT NULL,
    currency_code        VARCHAR(3)     NOT NULL,
    total_amount         NUMERIC(20,4)  NOT NULL DEFAULT 0,
    outstanding_amount   NUMERIC(20,4)  NOT NULL DEFAULT 0,
    status               VARCHAR(20)    NOT NULL DEFAULT 'Draft',
    goods_receipt_ref    VARCHAR(100),               -- logical ref procurement_db
    gl_journal_ref       VARCHAR(50),                -- logical ref gl_db
    created_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by           UUID           NOT NULL,
    updated_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by           UUID,
    deleted_at           TIMESTAMPTZ,
    row_version          INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_ap_invoice PRIMARY KEY (id, invoice_date),
    CONSTRAINT uq_ap_invoice_no UNIQUE (organization_id, invoice_no, invoice_date),
    CONSTRAINT ck_ap_invoice_status
        CHECK (status IN ('Draft','Recorded','PartiallyPaid','Paid','Cancelled'))
) PARTITION BY RANGE (invoice_date);

CREATE INDEX idx_ap_invoice_vendor    ON ap.ap_invoice (vendor_id);
CREATE INDEX idx_ap_invoice_due       ON ap.ap_invoice (due_date);
CREATE INDEX idx_ap_invoice_status    ON ap.ap_invoice (organization_id, status);
CREATE INDEX idx_ap_invoice_gr_ref    ON ap.ap_invoice (goods_receipt_ref);
CREATE INDEX idx_ap_invoice_date_brin ON ap.ap_invoice USING brin (invoice_date);

-- =============================================================================
-- ap.ap_invoice_line
-- =============================================================================
CREATE TABLE ap.ap_invoice_line (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    ap_invoice_id    UUID           NOT NULL,
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
    CONSTRAINT pk_ap_invoice_line PRIMARY KEY (id),
    CONSTRAINT uq_ap_invoice_line UNIQUE (ap_invoice_id, line_no),
    CONSTRAINT fk_ap_invoice_line_invoice
        FOREIGN KEY (ap_invoice_id, invoice_date)
        REFERENCES ap.ap_invoice (id, invoice_date) ON DELETE CASCADE
);
CREATE INDEX idx_ap_invoice_line_invoice ON ap.ap_invoice_line (ap_invoice_id);
CREATE INDEX idx_ap_invoice_line_account ON ap.ap_invoice_line (account_id);

-- =============================================================================
-- ap.ap_payment_allocation  (settlement links; payment_id logical ref treasury_db)
-- =============================================================================
CREATE TABLE ap.ap_payment_allocation (
    id                 UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id    UUID           NOT NULL,
    ap_invoice_id      UUID           NOT NULL,
    invoice_date       DATE           NOT NULL,   -- denormalized for composite FK
    payment_id         UUID           NOT NULL,   -- logical ref treasury_db
    allocated_amount   NUMERIC(20,4)  NOT NULL DEFAULT 0,
    allocated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_at         TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by         UUID           NOT NULL,
    updated_at         TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by         UUID,
    deleted_at         TIMESTAMPTZ,
    row_version        INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_ap_payment_allocation PRIMARY KEY (id),
    CONSTRAINT uq_ap_payment_allocation UNIQUE (ap_invoice_id, payment_id),
    CONSTRAINT fk_ap_alloc_invoice
        FOREIGN KEY (ap_invoice_id, invoice_date)
        REFERENCES ap.ap_invoice (id, invoice_date) ON DELETE CASCADE
);
CREATE INDEX idx_ap_alloc_invoice ON ap.ap_payment_allocation (ap_invoice_id);
CREATE INDEX idx_ap_alloc_payment ON ap.ap_payment_allocation (payment_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE ap.ap_invoice            ENABLE ROW LEVEL SECURITY;
ALTER TABLE ap.ap_invoice_line       ENABLE ROW LEVEL SECURITY;
ALTER TABLE ap.ap_payment_allocation ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON ap.ap_invoice
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON ap.ap_invoice_line
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON ap.ap_payment_allocation
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON ap.ap_invoice
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON ap.ap_invoice_line
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON ap.ap_payment_allocation
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE ap.outbox (
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
CREATE INDEX idx_outbox_unpublished ON ap.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON ap.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON ap.outbox (correlation_id);

CREATE TABLE ap.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
