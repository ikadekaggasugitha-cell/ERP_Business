-- sqlc catalog for inventory-service (plain DDL for type inference ONLY).
-- Authoritative schema (RANGE-partitioned stock_movement by movement_date, RLS
-- via app.org_scope, tg_touch_row trigger, generated available_qty column,
-- RANGE-partitioned outbox) lives in db/migrations/inventory_db.
-- Keep this catalog in sync (sqlc diff gate in CI).
CREATE SCHEMA IF NOT EXISTS inv;

CREATE TABLE inv.warehouse (
    id                  uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id     uuid         NOT NULL,
    code                varchar(50)  NOT NULL,
    name                varchar(150) NOT NULL,
    organization_ref_id uuid         NOT NULL,
    created_at          timestamptz  NOT NULL DEFAULT now(),
    created_by          uuid         NOT NULL,
    updated_at          timestamptz  NOT NULL DEFAULT now(),
    updated_by          uuid,
    deleted_at          timestamptz,
    row_version         integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE inv.stock_balance (
    id              uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id uuid          NOT NULL,
    warehouse_id    uuid          NOT NULL,
    item_id         uuid          NOT NULL,
    on_hand_qty     numeric(20,4) NOT NULL DEFAULT 0,
    reserved_qty    numeric(20,4) NOT NULL DEFAULT 0,
    available_qty   numeric(20,4) NOT NULL DEFAULT 0,
    avg_cost        numeric(20,4) NOT NULL DEFAULT 0,
    created_at      timestamptz   NOT NULL DEFAULT now(),
    created_by      uuid          NOT NULL,
    updated_at      timestamptz   NOT NULL DEFAULT now(),
    updated_by      uuid,
    deleted_at      timestamptz,
    row_version     integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    CONSTRAINT uq_stock_balance UNIQUE (organization_id, warehouse_id, item_id)
);

CREATE TABLE inv.stock_movement (
    id                  uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id     uuid          NOT NULL,
    movement_date       date          NOT NULL,
    warehouse_id        uuid          NOT NULL,
    item_id             uuid          NOT NULL,
    movement_type       varchar(20)   NOT NULL,
    quantity            numeric(20,4) NOT NULL,
    source_document_ref varchar(100),
    correlation_id      uuid          NOT NULL,
    created_at          timestamptz   NOT NULL DEFAULT now(),
    created_by          uuid          NOT NULL,
    updated_at          timestamptz   NOT NULL DEFAULT now(),
    updated_by          uuid,
    deleted_at          timestamptz,
    row_version         integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id, movement_date)
);

CREATE TABLE inv.stock_reservation (
    id              uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id uuid          NOT NULL,
    item_id         uuid          NOT NULL,
    warehouse_id    uuid          NOT NULL,
    reserved_qty    numeric(20,4) NOT NULL DEFAULT 0,
    sales_order_ref varchar(100)  NOT NULL,
    status          varchar(20)   NOT NULL DEFAULT 'Reserved',
    created_at      timestamptz   NOT NULL DEFAULT now(),
    created_by      uuid          NOT NULL,
    updated_at      timestamptz   NOT NULL DEFAULT now(),
    updated_by      uuid,
    deleted_at      timestamptz,
    row_version     integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE inv.outbox (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    aggregate_type   varchar(100) NOT NULL,
    aggregate_id     uuid         NOT NULL,
    event_type       varchar(150) NOT NULL,
    payload          jsonb        NOT NULL,
    organization_id  uuid         NOT NULL,
    correlation_id   uuid         NOT NULL,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    published_at     timestamptz,
    PRIMARY KEY (id, created_at)
);

CREATE TABLE inv.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);

CREATE TABLE inv.masterdata_cache (
    organization_id uuid         NOT NULL,
    party_id        uuid         NOT NULL,
    code            varchar(50)  NOT NULL DEFAULT '',
    name            varchar(200) NOT NULL DEFAULT '',
    party_type      varchar(20)  NOT NULL DEFAULT '',
    updated_at      timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, party_id)
);
