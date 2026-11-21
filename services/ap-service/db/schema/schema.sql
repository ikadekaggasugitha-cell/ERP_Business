-- sqlc catalog for ap-service (plain DDL for type inference ONLY).
-- Authoritative schema (RANGE partitioning by invoice_date/created_at, RLS via
-- app.org_scope, tg_touch_row trigger, invoice_no registry trigger) lives in
-- db/migrations/ap_db. Keep this catalog in sync (sqlc diff gate in CI).
CREATE SCHEMA IF NOT EXISTS ap;

CREATE TABLE ap.ap_invoice (
    id                 uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id    uuid          NOT NULL,
    invoice_no         varchar(50)   NOT NULL,
    vendor_id          uuid          NOT NULL,
    invoice_date       date          NOT NULL,
    due_date           date          NOT NULL,
    currency_code      varchar(3)    NOT NULL,
    total_amount       numeric(20,4) NOT NULL DEFAULT 0,
    outstanding_amount numeric(20,4) NOT NULL DEFAULT 0,
    status             varchar(20)   NOT NULL DEFAULT 'Draft',
    goods_receipt_ref  varchar(100),
    gl_journal_ref     varchar(50),
    created_at         timestamptz   NOT NULL DEFAULT now(),
    created_by         uuid          NOT NULL,
    updated_at         timestamptz   NOT NULL DEFAULT now(),
    updated_by         uuid,
    deleted_at         timestamptz,
    row_version        integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id, invoice_date)
);

CREATE TABLE ap.ap_invoice_line (
    id               uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid          NOT NULL,
    ap_invoice_id    uuid          NOT NULL,
    invoice_date     date          NOT NULL,
    line_no          integer       NOT NULL,
    account_id       uuid          NOT NULL,
    item_id          uuid,
    quantity         numeric(20,4),
    unit_price       numeric(20,4),
    amount           numeric(20,4) NOT NULL DEFAULT 0,
    description      text,
    created_at       timestamptz   NOT NULL DEFAULT now(),
    created_by       uuid          NOT NULL,
    updated_at       timestamptz   NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE ap.ap_payment_allocation (
    id               uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid          NOT NULL,
    ap_invoice_id    uuid          NOT NULL,
    invoice_date     date          NOT NULL,
    payment_id       uuid          NOT NULL,
    allocated_amount numeric(20,4) NOT NULL DEFAULT 0,
    allocated_at     timestamptz   NOT NULL DEFAULT now(),
    created_at       timestamptz   NOT NULL DEFAULT now(),
    created_by       uuid          NOT NULL,
    updated_at       timestamptz   NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE ap.outbox (
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

CREATE TABLE ap.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);

CREATE TABLE ap.masterdata_cache (
    organization_id uuid         NOT NULL,
    party_id        uuid         NOT NULL,
    code            varchar(50)  NOT NULL DEFAULT '',
    name            varchar(200) NOT NULL DEFAULT '',
    party_type      varchar(20)  NOT NULL DEFAULT '',
    updated_at      timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, party_id)
);

CREATE TABLE ap.document_number_registry (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    document_type    varchar(50)  NOT NULL,
    document_number  varchar(50)  NOT NULL,
    fiscal_year      integer      NOT NULL,
    status           varchar(20)  NOT NULL DEFAULT 'Issued',
    source_ref       varchar(100),
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    CONSTRAINT uq_document_number_registry UNIQUE (organization_id, document_type, fiscal_year, document_number)
);
