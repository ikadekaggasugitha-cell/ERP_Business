-- sqlc catalog for procurement-service (plain DDL for type inference ONLY).
-- Authoritative schema (RLS via app.org_scope, tg_touch_row trigger, RANGE-
-- partitioned outbox, doc-number uniqueness) lives in db/migrations/procurement_db.
-- Schemas: pr (requisition + outbox/cache), po (purchase order), gr (goods receipt).
-- Keep this catalog in sync (sqlc diff gate in CI).
CREATE SCHEMA IF NOT EXISTS pr;
CREATE SCHEMA IF NOT EXISTS po;
CREATE SCHEMA IF NOT EXISTS gr;

CREATE TABLE pr.purchase_requisition (
    id              uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id uuid          NOT NULL,
    pr_no           varchar(50)   NOT NULL,
    requester_id    uuid          NOT NULL,
    status          varchar(20)   NOT NULL DEFAULT 'Draft',
    workflow_ref    varchar(100),
    needed_date     date,
    created_at      timestamptz   NOT NULL DEFAULT now(),
    created_by      uuid          NOT NULL,
    updated_at      timestamptz   NOT NULL DEFAULT now(),
    updated_by      uuid,
    deleted_at      timestamptz,
    row_version     integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE pr.pr_line (
    id                      uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id         uuid          NOT NULL,
    purchase_requisition_id uuid          NOT NULL,
    line_no                 integer       NOT NULL,
    item_id                 uuid          NOT NULL,
    request_qty             numeric(20,4) NOT NULL DEFAULT 0,
    needed_date             date,
    created_at              timestamptz   NOT NULL DEFAULT now(),
    created_by              uuid          NOT NULL,
    updated_at              timestamptz   NOT NULL DEFAULT now(),
    updated_by              uuid,
    deleted_at              timestamptz,
    row_version             integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE po.purchase_order (
    id                      uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id         uuid          NOT NULL,
    po_no                   varchar(50)   NOT NULL,
    vendor_id               uuid          NOT NULL,
    purchase_requisition_id uuid,
    order_date              date          NOT NULL,
    currency_code           varchar(3)    NOT NULL,
    total_amount            numeric(20,4) NOT NULL DEFAULT 0,
    status                  varchar(20)   NOT NULL DEFAULT 'Draft',
    created_at              timestamptz   NOT NULL DEFAULT now(),
    created_by              uuid          NOT NULL,
    updated_at              timestamptz   NOT NULL DEFAULT now(),
    updated_by              uuid,
    deleted_at              timestamptz,
    row_version             integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE po.po_line (
    id                uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id   uuid          NOT NULL,
    purchase_order_id uuid          NOT NULL,
    line_no           integer       NOT NULL,
    item_id           uuid          NOT NULL,
    order_qty         numeric(20,4) NOT NULL DEFAULT 0,
    received_qty      numeric(20,4) NOT NULL DEFAULT 0,
    unit_price        numeric(20,4) NOT NULL DEFAULT 0,
    created_at        timestamptz   NOT NULL DEFAULT now(),
    created_by        uuid          NOT NULL,
    updated_at        timestamptz   NOT NULL DEFAULT now(),
    updated_by        uuid,
    deleted_at        timestamptz,
    row_version       integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE gr.goods_receipt (
    id                uuid        NOT NULL DEFAULT gen_random_uuid(),
    organization_id   uuid        NOT NULL,
    gr_no             varchar(50) NOT NULL,
    purchase_order_id uuid        NOT NULL,
    warehouse_id      uuid        NOT NULL,
    receipt_date      date        NOT NULL,
    status            varchar(20) NOT NULL DEFAULT 'Draft',
    correlation_id    uuid        NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    created_by        uuid        NOT NULL,
    updated_at        timestamptz NOT NULL DEFAULT now(),
    updated_by        uuid,
    deleted_at        timestamptz,
    row_version       integer     NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE gr.gr_line (
    id               uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid          NOT NULL,
    goods_receipt_id uuid          NOT NULL,
    po_line_id       uuid          NOT NULL,
    line_no          integer       NOT NULL,
    received_qty     numeric(20,4) NOT NULL DEFAULT 0,
    created_at       timestamptz   NOT NULL DEFAULT now(),
    created_by       uuid          NOT NULL,
    updated_at       timestamptz   NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE pr.outbox (
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

CREATE TABLE pr.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);

CREATE TABLE pr.masterdata_cache (
    organization_id uuid         NOT NULL,
    party_id        uuid         NOT NULL,
    code            varchar(50)  NOT NULL DEFAULT '',
    name            varchar(200) NOT NULL DEFAULT '',
    party_type      varchar(20)  NOT NULL DEFAULT '',
    updated_at      timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, party_id)
);
