-- sqlc catalog for dimension-service (plain DDL for type inference ONLY).
-- Mirrors db/migrations/dimension_db (cc, pc schemas).
CREATE SCHEMA IF NOT EXISTS cc;
CREATE SCHEMA IF NOT EXISTS pc;

CREATE TABLE cc.cost_center (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    code             varchar(50)  NOT NULL,
    name             varchar(150) NOT NULL,
    description      text,
    parent_id        uuid,
    owner_ref        uuid,
    status           varchar(20)  NOT NULL DEFAULT 'Draft',
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

CREATE TABLE cc.cost_allocation_rule (
    id                     uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id        uuid         NOT NULL,
    source_cost_center_id  uuid         NOT NULL,
    name                   varchar(150) NOT NULL,
    allocation_basis       varchar(50)  NOT NULL,
    targets                jsonb        NOT NULL,
    is_active              boolean      NOT NULL DEFAULT TRUE,
    created_at             timestamptz  NOT NULL DEFAULT now(),
    created_by             uuid         NOT NULL,
    updated_at             timestamptz  NOT NULL DEFAULT now(),
    updated_by             uuid,
    deleted_at             timestamptz,
    row_version            integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE pc.profit_center (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    code             varchar(50)  NOT NULL,
    name             varchar(150) NOT NULL,
    parent_id        uuid,
    status           varchar(20)  NOT NULL DEFAULT 'Draft',
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

CREATE TABLE cc.outbox (
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

CREATE TABLE cc.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);

CREATE TABLE cc.cost_center_assignment (
    id               uuid        NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid        NOT NULL,
    cost_center_id   uuid        NOT NULL,
    organization_ref uuid        NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    created_by       uuid        NOT NULL,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer     NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    CONSTRAINT uq_cost_center_assignment UNIQUE (cost_center_id, organization_ref)
);

CREATE TABLE pc.profit_center_assignment (
    id                 uuid        NOT NULL DEFAULT gen_random_uuid(),
    organization_id    uuid        NOT NULL,
    profit_center_id   uuid        NOT NULL,
    organization_ref   uuid        NOT NULL,
    created_at         timestamptz NOT NULL DEFAULT now(),
    created_by         uuid        NOT NULL,
    updated_at         timestamptz NOT NULL DEFAULT now(),
    updated_by         uuid,
    deleted_at         timestamptz,
    row_version        integer     NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    CONSTRAINT uq_profit_center_assignment UNIQUE (profit_center_id, organization_ref)
);

