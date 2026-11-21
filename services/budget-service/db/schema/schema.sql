-- sqlc catalog for budget-service (mirrors db/migrations/budget_db).
CREATE SCHEMA IF NOT EXISTS bud;

CREATE TABLE bud.budget (
    id                            uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id               uuid         NOT NULL,
    code                          varchar(50)  NOT NULL,
    name                          varchar(150) NOT NULL,
    financial_organization_ref    uuid         NOT NULL,
    fiscal_year_ref               uuid         NOT NULL,
    version                       integer      NOT NULL DEFAULT 1,
    status                        varchar(20)  NOT NULL DEFAULT 'Draft',
    is_active                     boolean      NOT NULL DEFAULT FALSE,
    created_at                    timestamptz  NOT NULL DEFAULT now(),
    created_by                    uuid         NOT NULL,
    updated_at                    timestamptz  NOT NULL DEFAULT now(),
    updated_by                    uuid,
    deleted_at                    timestamptz,
    row_version                   integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE bud.budget_line (
    id               uuid           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid           NOT NULL,
    budget_id        uuid           NOT NULL,
    account_ref      uuid           NOT NULL,
    cost_center_ref  uuid,
    project_ref      uuid,
    period_no        integer        NOT NULL,
    budget_amount    numeric(20,4)  NOT NULL DEFAULT 0,
    created_at       timestamptz    NOT NULL DEFAULT now(),
    created_by       uuid           NOT NULL,
    updated_at       timestamptz    NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE bud.budget_control_policy (
    id                uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id   uuid         NOT NULL,
    account_ref       uuid,
    transaction_type  varchar(50),
    control_type      varchar(20)  NOT NULL,
    created_at        timestamptz  NOT NULL DEFAULT now(),
    created_by        uuid         NOT NULL,
    updated_at        timestamptz  NOT NULL DEFAULT now(),
    updated_by        uuid,
    deleted_at        timestamptz,
    row_version       integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE bud.budget_commitment (
    id                     uuid           NOT NULL DEFAULT gen_random_uuid(),
    organization_id        uuid           NOT NULL,
    budget_id              uuid           NOT NULL,
    source_document_ref    varchar(100)   NOT NULL,
    committed_amount       numeric(20,4)  NOT NULL DEFAULT 0,
    status                 varchar(20)    NOT NULL DEFAULT 'Open',
    created_at             timestamptz    NOT NULL DEFAULT now(),
    created_by             uuid           NOT NULL,
    updated_at             timestamptz    NOT NULL DEFAULT now(),
    updated_by             uuid,
    deleted_at             timestamptz,
    row_version            integer        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE bud.outbox (
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

CREATE TABLE bud.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
