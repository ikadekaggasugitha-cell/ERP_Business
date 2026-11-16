-- sqlc catalog for sales-service (plain DDL for type inference ONLY).
-- Authoritative schema (RLS via app.org_scope, tg_touch_row trigger, RANGE-
-- partitioned outbox) lives in db/migrations/sales_db.
-- Keep this catalog in sync (sqlc diff gate in CI).
CREATE SCHEMA IF NOT EXISTS so;

CREATE TABLE so.sales_order (
    id              uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id uuid          NOT NULL,
    so_no           varchar(50)   NOT NULL,
    customer_id     uuid          NOT NULL,
    order_date      date          NOT NULL,
    currency_code   varchar(3)    NOT NULL,
    total_amount    numeric(20,4) NOT NULL DEFAULT 0,
    status          varchar(20)   NOT NULL DEFAULT 'Draft',
    correlation_id  uuid          NOT NULL,
    created_at      timestamptz   NOT NULL DEFAULT now(),
    created_by      uuid          NOT NULL,
    updated_at      timestamptz   NOT NULL DEFAULT now(),
    updated_by      uuid,
    deleted_at      timestamptz,
    row_version     integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE so.so_line (
    id              uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id uuid          NOT NULL,
    sales_order_id  uuid          NOT NULL,
    line_no         integer       NOT NULL,
    item_id         uuid          NOT NULL,
    order_qty       numeric(20,4) NOT NULL DEFAULT 0,
    unit_price      numeric(20,4) NOT NULL DEFAULT 0,
    delivered_qty   numeric(20,4) NOT NULL DEFAULT 0,
    created_at      timestamptz   NOT NULL DEFAULT now(),
    created_by      uuid          NOT NULL,
    updated_at      timestamptz   NOT NULL DEFAULT now(),
    updated_by      uuid,
    deleted_at      timestamptz,
    row_version     integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE so.delivery (
    id              uuid        NOT NULL DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL,
    delivery_no     varchar(50) NOT NULL,
    sales_order_id  uuid        NOT NULL,
    warehouse_id    uuid        NOT NULL,
    delivery_date   date        NOT NULL,
    status          varchar(20) NOT NULL DEFAULT 'Draft',
    correlation_id  uuid        NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    created_by      uuid        NOT NULL,
    updated_at      timestamptz NOT NULL DEFAULT now(),
    updated_by      uuid,
    deleted_at      timestamptz,
    row_version     integer     NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE so.delivery_line (
    id              uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id uuid          NOT NULL,
    delivery_id     uuid          NOT NULL,
    so_line_id      uuid          NOT NULL,
    line_no         integer       NOT NULL,
    item_id         uuid          NOT NULL,
    delivered_qty   numeric(20,4) NOT NULL DEFAULT 0,
    created_at      timestamptz   NOT NULL DEFAULT now(),
    created_by      uuid          NOT NULL,
    updated_at      timestamptz   NOT NULL DEFAULT now(),
    updated_by      uuid,
    deleted_at      timestamptz,
    row_version     integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE so.outbox (
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

CREATE TABLE so.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);

CREATE TABLE so.masterdata_cache (
    organization_id uuid         NOT NULL,
    party_id        uuid         NOT NULL,
    code            varchar(50)  NOT NULL DEFAULT '',
    name            varchar(200) NOT NULL DEFAULT '',
    party_type      varchar(20)  NOT NULL DEFAULT '',
    updated_at      timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, party_id)
);
