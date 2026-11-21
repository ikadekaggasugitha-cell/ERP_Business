-- sqlc catalog for reporting-service (plain DDL for type inference ONLY).
-- Authoritative schema (RLS via app.org_scope) lives in db/migrations/reporting_db.
-- reporting_db is a CQRS READ MODEL (SVC-019, ADR-009): derived, denormalized,
-- fully rebuildable projections. Read models omit the full Base Entity (no
-- soft-delete / optimistic-lock / created_by) — they carry only a refreshed_at
-- watermark and are upserted idempotently by the projector. Consumer only: there
-- is NO outbox; idempotency is tracked via processed_event.
-- NOTE: the UNIQUE key on gl_balance_snapshot is retained (not just the PK) because
-- the upsert-by-event query targets it in ON CONFLICT. Keep this catalog in sync
-- (sqlc diff gate in CI).
CREATE SCHEMA IF NOT EXISTS rpt;

CREATE TABLE rpt.gl_balance_snapshot (
    id                uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id   uuid          NOT NULL,
    fiscal_period_id  uuid          NOT NULL,
    account_code      varchar(50)   NOT NULL,
    closing_balance   numeric(20,4) NOT NULL DEFAULT 0,
    refreshed_at      timestamptz   NOT NULL DEFAULT now(),
    PRIMARY KEY (id),
    UNIQUE (organization_id, fiscal_period_id, account_code)
);

CREATE TABLE rpt.sales_summary (
    id                 uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id    uuid          NOT NULL,
    period_year        integer       NOT NULL,
    period_month       integer       NOT NULL,
    customer_id        uuid,
    total_sales_amount numeric(20,4) NOT NULL DEFAULT 0,
    order_count        integer       NOT NULL DEFAULT 0,
    refreshed_at       timestamptz   NOT NULL DEFAULT now(),
    PRIMARY KEY (id),
    UNIQUE (organization_id, period_year, period_month, customer_id)
);

CREATE TABLE rpt.inventory_position (
    id               uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid          NOT NULL,
    warehouse_code   varchar(50)   NOT NULL,
    item_code        varchar(50)   NOT NULL,
    on_hand_qty      numeric(20,4) NOT NULL DEFAULT 0,
    available_qty    numeric(20,4) NOT NULL DEFAULT 0,
    refreshed_at     timestamptz   NOT NULL DEFAULT now(),
    PRIMARY KEY (id),
    UNIQUE (organization_id, warehouse_code, item_code)
);

CREATE TABLE rpt.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);

CREATE TABLE rpt.coa_cache (
    organization_id  uuid          NOT NULL,
    account_id       uuid          NOT NULL,
    account_code     varchar(50)   NOT NULL,
    account_name     varchar(200)  NOT NULL DEFAULT '',
    account_type     varchar(50)   NOT NULL DEFAULT '',
    refreshed_at     timestamptz   NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, account_id)
);
