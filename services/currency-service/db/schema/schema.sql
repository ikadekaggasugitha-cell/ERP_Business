-- sqlc catalog for currency-service (plain DDL for type inference ONLY).
-- Mirrors db/migrations/currency_db (cur schema).
CREATE SCHEMA IF NOT EXISTS cur;

CREATE TABLE cur.currency (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    code             varchar(3)   NOT NULL,   -- ISO 4217 alpha
    numeric_code     varchar(3),              -- ISO 4217 numeric
    name             varchar(100) NOT NULL,
    precision        integer      NOT NULL DEFAULT 2,
    is_active        boolean      NOT NULL DEFAULT TRUE,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    CONSTRAINT uq_currency_org_code UNIQUE (organization_id, code),
    CONSTRAINT ck_currency_precision CHECK (precision >= 0)
);

CREATE TABLE cur.exchange_rate_type (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    code             varchar(30)  NOT NULL,
    name             varchar(100) NOT NULL,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    CONSTRAINT uq_exchange_rate_type_org_code UNIQUE (organization_id, code)
);

CREATE TABLE cur.exchange_rate (
    id               uuid           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid           NOT NULL,
    from_currency    varchar(3)     NOT NULL,
    to_currency      varchar(3)     NOT NULL,
    rate_type_id     uuid           NOT NULL,
    rate             numeric(20,8)  NOT NULL,
    valid_from       date           NOT NULL,
    valid_to         date,
    created_at       timestamptz    NOT NULL DEFAULT now(),
    created_by       uuid           NOT NULL,
    updated_at       timestamptz    NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer        NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    CONSTRAINT uq_exchange_rate UNIQUE (organization_id, from_currency, to_currency, rate_type_id, valid_from),
    CONSTRAINT fk_exchange_rate_type
        FOREIGN KEY (rate_type_id) REFERENCES cur.exchange_rate_type (id) ON DELETE RESTRICT,
    CONSTRAINT ck_exchange_rate_positive CHECK (rate > 0),
    CONSTRAINT ck_exchange_rate_range CHECK (valid_to IS NULL OR valid_to >= valid_from)
);

CREATE TABLE cur.outbox (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    aggregate_type   varchar(100) NOT NULL,
    aggregate_id     uuid         NOT NULL,
    event_type       varchar(150) NOT NULL,
    payload          jsonb        NOT NULL,
    organization_id  uuid         NOT NULL,
    correlation_id   uuid         NOT NULL,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    published_at     timestamptz,
    PRIMARY KEY (id)
);

CREATE TABLE cur.processed_event (
    event_id      uuid         NOT NULL,
    event_type    varchar(150) NOT NULL,
    processed_at  timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);

