-- sqlc catalog for consolidation-service (mirrors db/migrations/consolidation_db).
CREATE SCHEMA IF NOT EXISTS cons;

CREATE TABLE cons.consolidation_group (
    id                           uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id              uuid         NOT NULL,
    code                         varchar(50)  NOT NULL,
    name                         varchar(150) NOT NULL,
    parent_group_id              uuid,
    presentation_currency_code   varchar(3)   NOT NULL,
    created_at                   timestamptz  NOT NULL DEFAULT now(),
    created_by                   uuid         NOT NULL,
    updated_at                   timestamptz  NOT NULL DEFAULT now(),
    updated_by                   uuid,
    deleted_at                   timestamptz,
    row_version                  integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE cons.consolidation_member (
    id                       uuid           NOT NULL DEFAULT gen_random_uuid(),
    organization_id          uuid           NOT NULL,
    consolidation_group_id   uuid           NOT NULL,
    company_ref              uuid           NOT NULL,
    ownership_percentage     numeric(9,4)   NOT NULL,
    consolidation_method     varchar(20)    NOT NULL,
    created_at               timestamptz    NOT NULL DEFAULT now(),
    created_by               uuid           NOT NULL,
    updated_at               timestamptz    NOT NULL DEFAULT now(),
    updated_by               uuid,
    deleted_at               timestamptz,
    row_version              integer        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE cons.elimination_rule (
    id                       uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id          uuid         NOT NULL,
    consolidation_group_id   uuid         NOT NULL,
    rule_type                varchar(40)  NOT NULL,
    config                   jsonb        NOT NULL,
    is_active                boolean      NOT NULL DEFAULT TRUE,
    created_at               timestamptz  NOT NULL DEFAULT now(),
    created_by               uuid         NOT NULL,
    updated_at               timestamptz  NOT NULL DEFAULT now(),
    updated_by               uuid,
    deleted_at               timestamptz,
    row_version              integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE cons.consolidation_run (
    id                       uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id          uuid         NOT NULL,
    consolidation_group_id   uuid         NOT NULL,
    fiscal_period_ref        uuid         NOT NULL,
    status                   varchar(20)  NOT NULL DEFAULT 'Draft',
    version                  integer      NOT NULL DEFAULT 1,
    created_at               timestamptz  NOT NULL DEFAULT now(),
    created_by               uuid         NOT NULL,
    updated_at               timestamptz  NOT NULL DEFAULT now(),
    updated_by               uuid,
    deleted_at               timestamptz,
    row_version              integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE cons.elimination_entry (
    id                     uuid           NOT NULL DEFAULT gen_random_uuid(),
    organization_id        uuid           NOT NULL,
    consolidation_run_id   uuid           NOT NULL,
    elimination_rule_id    uuid,
    account_ref            uuid           NOT NULL,
    debit_amount           numeric(20,4)  NOT NULL DEFAULT 0,
    credit_amount          numeric(20,4)  NOT NULL DEFAULT 0,
    description            text,
    created_at             timestamptz    NOT NULL DEFAULT now(),
    created_by             uuid           NOT NULL,
    updated_at             timestamptz    NOT NULL DEFAULT now(),
    updated_by             uuid,
    deleted_at             timestamptz,
    row_version            integer        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE cons.consolidation_adjustment (
    id                     uuid           NOT NULL DEFAULT gen_random_uuid(),
    organization_id        uuid           NOT NULL,
    consolidation_run_id   uuid           NOT NULL,
    account_ref            uuid           NOT NULL,
    debit_amount           numeric(20,4)  NOT NULL DEFAULT 0,
    credit_amount          numeric(20,4)  NOT NULL DEFAULT 0,
    adjustment_type        varchar(20)    NOT NULL,
    notes                  text,
    created_at             timestamptz    NOT NULL DEFAULT now(),
    created_by             uuid           NOT NULL,
    updated_at             timestamptz    NOT NULL DEFAULT now(),
    updated_by             uuid,
    deleted_at             timestamptz,
    row_version            integer        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE cons.outbox (
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

CREATE TABLE cons.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
