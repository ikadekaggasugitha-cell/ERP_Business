-- sqlc catalog for coa-service (plain DDL for type inference ONLY).
-- Mirrors db/migrations/coa_db (fin + coa schemas).
CREATE SCHEMA IF NOT EXISTS fin;
CREATE SCHEMA IF NOT EXISTS coa;

CREATE TABLE coa.chart_of_accounts (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    code             varchar(50)  NOT NULL,
    name             varchar(150) NOT NULL,
    structure        jsonb        NOT NULL,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE fin.fiscal_calendar (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    code             varchar(50)  NOT NULL,
    name             varchar(150) NOT NULL,
    period_type      varchar(20)  NOT NULL,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE fin.fiscal_year (
    id                  uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id     uuid         NOT NULL,
    fiscal_calendar_id  uuid         NOT NULL,
    year_no             integer      NOT NULL,
    name                varchar(50)  NOT NULL,
    start_date          date         NOT NULL,
    end_date            date         NOT NULL,
    status              varchar(20)  NOT NULL DEFAULT 'Open',
    created_at          timestamptz  NOT NULL DEFAULT now(),
    created_by          uuid         NOT NULL,
    updated_at          timestamptz  NOT NULL DEFAULT now(),
    updated_by          uuid,
    deleted_at          timestamptz,
    row_version         integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE fin.fiscal_period (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    fiscal_year_id   uuid         NOT NULL,
    period_no        integer      NOT NULL,
    start_date       date         NOT NULL,
    end_date         date         NOT NULL,
    status           varchar(20)  NOT NULL DEFAULT 'Open',
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE fin.financial_organization (
    id                        uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id           uuid          NOT NULL,
    code                      varchar(50)   NOT NULL,
    name                      varchar(200)  NOT NULL,
    organization_ref_id       uuid          NOT NULL,
    functional_currency_code  varchar(3)    NOT NULL,
    chart_of_accounts_id      uuid          NOT NULL,
    fiscal_calendar_id        uuid          NOT NULL,
    created_at                timestamptz   NOT NULL DEFAULT now(),
    created_by                uuid          NOT NULL,
    updated_at                timestamptz   NOT NULL DEFAULT now(),
    updated_by                uuid,
    deleted_at                timestamptz,
    row_version               integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE fin.ledger (
    id                          uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id             uuid         NOT NULL,
    financial_organization_id   uuid         NOT NULL,
    code                        varchar(50)  NOT NULL,
    ledger_type                 varchar(30)  NOT NULL,
    currency_code               varchar(3)   NOT NULL,
    created_at                  timestamptz  NOT NULL DEFAULT now(),
    created_by                  uuid         NOT NULL,
    updated_at                  timestamptz  NOT NULL DEFAULT now(),
    updated_by                  uuid,
    deleted_at                  timestamptz,
    row_version                 integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE coa.account (
    id                   uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id      uuid          NOT NULL,
    chart_of_accounts_id uuid          NOT NULL,
    code                 varchar(50)   NOT NULL,
    name                 varchar(200)  NOT NULL,
    account_type         varchar(30)   NOT NULL,
    account_class        varchar(30),
    is_control           boolean       NOT NULL DEFAULT FALSE,
    is_monetary          boolean       NOT NULL DEFAULT FALSE,
    allow_posting        boolean       NOT NULL DEFAULT TRUE,
    parent_id            uuid,
    created_at           timestamptz   NOT NULL DEFAULT now(),
    created_by           uuid          NOT NULL,
    updated_at           timestamptz   NOT NULL DEFAULT now(),
    updated_by           uuid,
    deleted_at           timestamptz,
    row_version          integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE coa.posting_rule (
    id                   uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id      uuid          NOT NULL,
    chart_of_accounts_id uuid          NOT NULL,
    source_module        varchar(50)   NOT NULL,
    event_key            varchar(100)  NOT NULL,
    debit_account_rule   jsonb         NOT NULL,
    credit_account_rule  jsonb         NOT NULL,
    created_at           timestamptz   NOT NULL DEFAULT now(),
    created_by           uuid          NOT NULL,
    updated_at           timestamptz   NOT NULL DEFAULT now(),
    updated_by           uuid,
    deleted_at           timestamptz,
    row_version          integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE coa.outbox (
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

CREATE TABLE coa.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
