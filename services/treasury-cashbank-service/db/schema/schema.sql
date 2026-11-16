-- sqlc catalog for treasury-cashbank-service (plain DDL for type inference ONLY).
-- Authoritative schema (RLS via app.org_scope, tg_touch_row trigger, RANGE-
-- partitioned outbox) lives in db/migrations/treasury_db. Keep this catalog in
-- sync (sqlc diff gate in CI). Principal aggregate is trs.payment (+ allocation
-- child); payment_no is DB-unique (no document_number_registry, not partitioned).
CREATE SCHEMA IF NOT EXISTS trs;

CREATE TABLE trs.bank_account (
    id               uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid          NOT NULL,
    code             varchar(50)   NOT NULL,
    bank_name        varchar(150)  NOT NULL,
    account_no       varchar(50)   NOT NULL,
    currency_code    varchar(3)    NOT NULL,
    gl_account_id    uuid          NOT NULL,
    created_at       timestamptz   NOT NULL DEFAULT now(),
    created_by       uuid          NOT NULL,
    updated_at       timestamptz   NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE trs.payment (
    id               uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid          NOT NULL,
    payment_no       varchar(50)   NOT NULL,
    payment_type     varchar(20)   NOT NULL,
    bank_account_id  uuid          NOT NULL,
    party_id         uuid,
    amount           numeric(20,4) NOT NULL DEFAULT 0,
    payment_date     date          NOT NULL,
    status           varchar(20)   NOT NULL DEFAULT 'Draft',
    external_ref     varchar(100),
    correlation_id   uuid          NOT NULL,
    created_at       timestamptz   NOT NULL DEFAULT now(),
    created_by       uuid          NOT NULL,
    updated_at       timestamptz   NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE trs.payment_allocation (
    id                  uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id     uuid          NOT NULL,
    payment_id          uuid          NOT NULL,
    target_document_ref varchar(100)  NOT NULL,
    allocated_amount    numeric(20,4) NOT NULL DEFAULT 0,
    allocated_at        timestamptz   NOT NULL DEFAULT now(),
    created_at          timestamptz   NOT NULL DEFAULT now(),
    created_by          uuid          NOT NULL,
    updated_at          timestamptz   NOT NULL DEFAULT now(),
    updated_by          uuid,
    deleted_at          timestamptz,
    row_version         integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE trs.bank_statement_line (
    id                  uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id     uuid         NOT NULL,
    bank_account_id     uuid         NOT NULL REFERENCES trs.bank_account(id) ON DELETE RESTRICT,
    statement_date      date         NOT NULL,
    description         text         NOT NULL,
    amount              numeric(20,4) NOT NULL,
    external_ref        varchar(100),
    matched_payment_id  uuid         REFERENCES trs.payment(id) ON DELETE SET NULL,
    is_reconciled       boolean      NOT NULL DEFAULT false,
    created_at          timestamptz  NOT NULL DEFAULT now(),
    created_by          uuid         NOT NULL,
    updated_at          timestamptz  NOT NULL DEFAULT now(),
    updated_by          uuid,
    deleted_at          timestamptz,
    row_version         integer      NOT NULL DEFAULT 1,
    CONSTRAINT pk_bank_statement_line PRIMARY KEY (id)
);

CREATE TABLE trs.outbox (
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

CREATE TABLE trs.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
