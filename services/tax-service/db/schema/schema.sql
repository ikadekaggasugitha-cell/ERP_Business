-- sqlc catalog for tax-service (plain DDL for type inference ONLY).
-- Authoritative schema (RLS via app.org_scope, tg_touch_row trigger, RANGE-
-- partitioned outbox) lives in db/migrations/tax_db. Keep this catalog in sync
-- (sqlc diff gate in CI). Neither tax_code nor tax_transaction has a base-entity
-- `status` column (tax_transaction.efaktur_status is a nullable doc flag), so
-- there is no optimistic-locked transition query.
CREATE SCHEMA IF NOT EXISTS tax;

CREATE TABLE tax.tax_code (
    id               uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid          NOT NULL,
    code             varchar(30)   NOT NULL,
    rate             numeric(9,4)  NOT NULL,
    tax_type         varchar(30)   NOT NULL,
    gl_account_id    uuid,
    created_at       timestamptz   NOT NULL DEFAULT now(),
    created_by       uuid          NOT NULL,
    updated_at       timestamptz   NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE tax.tax_transaction (
    id                  uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id     uuid          NOT NULL,
    source_document_ref varchar(100)  NOT NULL,
    tax_code_id         uuid          NOT NULL,
    base_amount         numeric(20,4) NOT NULL DEFAULT 0,
    tax_amount          numeric(20,4) NOT NULL DEFAULT 0,
    efaktur_no          varchar(50),
    efaktur_status      varchar(20),
    created_at          timestamptz   NOT NULL DEFAULT now(),
    created_by          uuid          NOT NULL,
    updated_at          timestamptz   NOT NULL DEFAULT now(),
    updated_by          uuid,
    deleted_at          timestamptz,
    row_version         integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE tax.outbox (
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

CREATE TABLE tax.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
