-- sqlc catalog for payroll-service (plain DDL for type inference ONLY).
-- Authoritative schema (RLS via app.org_scope, tg_touch_row trigger, RANGE-
-- partitioned outbox, sensitive PII amounts per BAB 7.4) lives in
-- db/migrations/payroll_db. Keep this catalog in sync (sqlc diff gate in CI).
CREATE SCHEMA IF NOT EXISTS pay;

CREATE TABLE pay.payroll_run (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    run_no           varchar(50)  NOT NULL,
    period_month     integer      NOT NULL,
    period_year      integer      NOT NULL,
    status           varchar(20)  NOT NULL DEFAULT 'Draft',
    gl_journal_ref   varchar(50),
    correlation_id   uuid         NOT NULL,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE pay.payroll_result (
    id                  uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id     uuid          NOT NULL,
    payroll_run_id      uuid          NOT NULL,
    employee_id         uuid          NOT NULL,
    gross_amount        numeric(20,4) NOT NULL DEFAULT 0,
    deduction_amount    numeric(20,4) NOT NULL DEFAULT 0,
    tax_amount          numeric(20,4) NOT NULL DEFAULT 0,
    net_amount          numeric(20,4) NOT NULL DEFAULT 0,
    encrypted_gross     text,
    encrypted_deduction text,
    encrypted_net       text,
    encrypted_tax       text,
    pii_key_version     varchar(32),
    created_at          timestamptz   NOT NULL DEFAULT now(),
    created_by          uuid          NOT NULL,
    updated_at          timestamptz   NOT NULL DEFAULT now(),
    updated_by          uuid,
    deleted_at          timestamptz,
    row_version         integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE pay.payroll_result_line (
    id                 uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id    uuid          NOT NULL,
    payroll_result_id  uuid          NOT NULL,
    component_code     varchar(30)   NOT NULL,
    amount             numeric(20,4) NOT NULL DEFAULT 0,
    encrypted_amount   text,
    created_at         timestamptz   NOT NULL DEFAULT now(),
    created_by         uuid          NOT NULL,
    updated_at         timestamptz   NOT NULL DEFAULT now(),
    updated_by         uuid,
    deleted_at         timestamptz,
    row_version        integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE pay.statutory_contribution (
    id                        uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id           uuid          NOT NULL,
    payroll_result_id         uuid          NOT NULL,
    contribution_type         varchar(30)   NOT NULL,
    employee_amount           numeric(20,4) NOT NULL DEFAULT 0,
    employer_amount           numeric(20,4) NOT NULL DEFAULT 0,
    encrypted_employee_amount text,
    encrypted_employer_amount text,
    created_at                timestamptz   NOT NULL DEFAULT now(),
    created_by                uuid          NOT NULL,
    updated_at                timestamptz   NOT NULL DEFAULT now(),
    updated_by                uuid,
    deleted_at                timestamptz,
    row_version               integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE pay.employee_cache (
    organization_id uuid        NOT NULL,
    employee_id     uuid        NOT NULL,
    employee_no     varchar(50) NOT NULL DEFAULT '',
    status          varchar(20) NOT NULL DEFAULT 'Active',
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, employee_id)
);

CREATE TABLE pay.outbox (
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

CREATE TABLE pay.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
