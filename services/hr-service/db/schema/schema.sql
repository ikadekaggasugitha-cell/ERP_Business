-- sqlc catalog for hr-service (plain DDL for type inference ONLY).
-- Authoritative schema (RLS via app.org_scope, tg_touch_row trigger, RANGE-
-- partitioned outbox, PII columns per BAB 7.4) lives in db/migrations/hr_db.
-- Keep this catalog in sync (sqlc diff gate in CI).
CREATE SCHEMA IF NOT EXISTS hr;

CREATE TABLE hr.employee (
    id                    uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id       uuid         NOT NULL,
    employee_no           varchar(50)  NOT NULL,
    full_name             varchar(512) NOT NULL,
    national_id           varchar(255),
    date_of_birth         date,
    email                 varchar(512),
    hire_date             date         NOT NULL,
    status                varchar(20)  NOT NULL DEFAULT 'Active',
    user_ref_id           uuid,
    encrypted_full_name   text,
    encrypted_national_id text,
    encrypted_dob         text,
    encrypted_email       text,
    national_id_hash      varchar(64),
    created_at            timestamptz  NOT NULL DEFAULT now(),
    created_by            uuid         NOT NULL,
    updated_at            timestamptz  NOT NULL DEFAULT now(),
    updated_by            uuid,
    deleted_at            timestamptz,
    row_version           integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE hr.position (
    id                   uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id      uuid         NOT NULL,
    code                 varchar(50)  NOT NULL,
    title                varchar(150) NOT NULL,
    organization_ref_id  uuid         NOT NULL,
    parent_position_id   uuid,
    created_at           timestamptz  NOT NULL DEFAULT now(),
    created_by           uuid         NOT NULL,
    updated_at           timestamptz  NOT NULL DEFAULT now(),
    updated_by           uuid,
    deleted_at           timestamptz,
    row_version          integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE hr.employment (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    employee_id      uuid         NOT NULL,
    position_id      uuid         NOT NULL,
    employment_type  varchar(20)  NOT NULL,
    effective_from   date         NOT NULL,
    effective_to     date,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE hr.employee_compensation (
    id               uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid          NOT NULL,
    employee_id      uuid          NOT NULL,
    component_code   varchar(30)   NOT NULL,
    amount           numeric(20,4) NOT NULL DEFAULT 0,
    effective_from   date          NOT NULL,
    created_at       timestamptz   NOT NULL DEFAULT now(),
    created_by       uuid          NOT NULL,
    updated_at       timestamptz   NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE hr.outbox (
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

CREATE TABLE hr.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
