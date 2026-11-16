-- =============================================================================
-- reporting_db — Reporting/Dashboard read model (SVC-019, CQRS / ADR-009)
-- DBDD BAB 8.2 | Standards DBS-001..016 (adapted) | NFR-003
-- DERIVED, denormalized projections built from cross-domain events — NOT a
-- system of record and fully REBUILDABLE. Read models therefore omit the full
-- Base Entity (no soft-delete/optimistic-lock/created_by); they carry only a
-- refreshed_at watermark and are upserted idempotently by the projector.
-- Consumer only -> processed_event (no outbox).
-- golang-migrate: 000001_init.up.sql
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS btree_gin;

CREATE OR REPLACE FUNCTION public.current_org_scope()
RETURNS uuid[] LANGUAGE sql STABLE AS $$
  SELECT NULLIF(current_setting('app.org_scope', true), '')::uuid[];
$$;

CREATE SCHEMA IF NOT EXISTS rpt;

-- =============================================================================
-- rpt.gl_balance_snapshot  (balance aggregation for dashboards)
-- =============================================================================
CREATE TABLE rpt.gl_balance_snapshot (
    id                UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID           NOT NULL,
    fiscal_period_id  UUID           NOT NULL,
    account_code      VARCHAR(50)    NOT NULL,   -- denormalized
    closing_balance   NUMERIC(20,4)  NOT NULL DEFAULT 0,
    refreshed_at      TIMESTAMPTZ    NOT NULL DEFAULT now(),   -- eventual watermark
    CONSTRAINT pk_gl_balance_snapshot PRIMARY KEY (id),
    CONSTRAINT uq_gl_balance_snapshot UNIQUE (organization_id, fiscal_period_id, account_code)
);
CREATE INDEX idx_gl_balance_snapshot_scope ON rpt.gl_balance_snapshot (organization_id, fiscal_period_id);
CREATE INDEX idx_gl_balance_snapshot_acct  ON rpt.gl_balance_snapshot (account_code);

-- =============================================================================
-- rpt.sales_summary  (denormalized sales aggregation)
-- =============================================================================
CREATE TABLE rpt.sales_summary (
    id                  UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id     UUID           NOT NULL,
    period_year         INTEGER        NOT NULL,
    period_month        INTEGER        NOT NULL,
    customer_id         UUID,                       -- denormalized (NULL = all)
    total_sales_amount  NUMERIC(20,4)  NOT NULL DEFAULT 0,
    order_count         INTEGER        NOT NULL DEFAULT 0,
    refreshed_at        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    CONSTRAINT pk_sales_summary PRIMARY KEY (id),
    CONSTRAINT uq_sales_summary UNIQUE (organization_id, period_year, period_month, customer_id),
    CONSTRAINT ck_sales_summary_month CHECK (period_month BETWEEN 1 AND 12)
);
CREATE INDEX idx_sales_summary_scope ON rpt.sales_summary (organization_id, period_year, period_month);

-- =============================================================================
-- rpt.inventory_position  (denormalized stock position)
-- =============================================================================
CREATE TABLE rpt.inventory_position (
    id                UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID           NOT NULL,
    warehouse_code    VARCHAR(50)    NOT NULL,   -- denormalized
    item_code         VARCHAR(50)    NOT NULL,   -- denormalized
    on_hand_qty       NUMERIC(20,4)  NOT NULL DEFAULT 0,
    available_qty     NUMERIC(20,4)  NOT NULL DEFAULT 0,
    refreshed_at      TIMESTAMPTZ    NOT NULL DEFAULT now(),
    CONSTRAINT pk_inventory_position PRIMARY KEY (id),
    CONSTRAINT uq_inventory_position UNIQUE (organization_id, warehouse_code, item_code)
);
CREATE INDEX idx_inventory_position_scope ON rpt.inventory_position (organization_id, warehouse_code);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE rpt.gl_balance_snapshot ENABLE ROW LEVEL SECURITY;
ALTER TABLE rpt.sales_summary       ENABLE ROW LEVEL SECURITY;
ALTER TABLE rpt.inventory_position  ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON rpt.gl_balance_snapshot USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON rpt.sales_summary       USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON rpt.inventory_position  USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- =============================================================================
-- Idempotency (DBS-014) — reporting is a pure event consumer (no outbox)
-- =============================================================================
CREATE TABLE rpt.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
