-- sqlc catalog for manufacturing-service (plain DDL for type inference ONLY).
-- Authoritative schema (RLS via app.org_scope, tg_touch_row trigger, RANGE-
-- partitioned outbox) lives in db/migrations/manufacturing_db.
-- Principal aggregate = mfg.work_order; BOM aggregate = mfg.bill_of_material
-- (+ mfg.bom_line) provides the child/line pattern.
-- Keep this catalog in sync (sqlc diff gate in CI).
CREATE SCHEMA IF NOT EXISTS mfg;

CREATE TABLE mfg.bill_of_material (
    id              uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id uuid         NOT NULL,
    code            varchar(50)  NOT NULL,
    product_item_id uuid         NOT NULL,
    version         integer      NOT NULL DEFAULT 1,
    is_active       boolean      NOT NULL DEFAULT TRUE,
    created_at      timestamptz  NOT NULL DEFAULT now(),
    created_by      uuid         NOT NULL,
    updated_at      timestamptz  NOT NULL DEFAULT now(),
    updated_by      uuid,
    deleted_at      timestamptz,
    row_version     integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE mfg.bom_line (
    id                uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id   uuid          NOT NULL,
    bom_id            uuid          NOT NULL,
    line_no           integer       NOT NULL,
    component_item_id uuid          NOT NULL,
    quantity          numeric(20,4) NOT NULL DEFAULT 0,
    uom               varchar(20)   NOT NULL,
    created_at        timestamptz   NOT NULL DEFAULT now(),
    created_by        uuid          NOT NULL,
    updated_at        timestamptz   NOT NULL DEFAULT now(),
    updated_by        uuid,
    deleted_at        timestamptz,
    row_version       integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE mfg.work_order (
    id              uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id uuid          NOT NULL,
    wo_no           varchar(50)   NOT NULL,
    product_item_id uuid          NOT NULL,
    bom_id          uuid          NOT NULL,
    planned_qty     numeric(20,4) NOT NULL DEFAULT 0,
    produced_qty    numeric(20,4) NOT NULL DEFAULT 0,
    status          varchar(20)   NOT NULL DEFAULT 'Planned',
    sales_order_ref varchar(100),
    correlation_id  uuid          NOT NULL,
    created_at      timestamptz   NOT NULL DEFAULT now(),
    created_by      uuid          NOT NULL,
    updated_at      timestamptz   NOT NULL DEFAULT now(),
    updated_by      uuid,
    deleted_at      timestamptz,
    row_version     integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE mfg.outbox (
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

CREATE TABLE mfg.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
