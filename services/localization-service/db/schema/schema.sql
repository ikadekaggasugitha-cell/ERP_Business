-- sqlc catalog for localization-service (plain DDL for type inference ONLY).
-- Mirrors db/migrations/localization_db (loc schema).
CREATE SCHEMA IF NOT EXISTS loc;

CREATE TABLE loc.language (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    code             varchar(10)  NOT NULL,
    name             varchar(100) NOT NULL,
    is_active        boolean      NOT NULL DEFAULT TRUE,
    is_fallback      boolean      NOT NULL DEFAULT FALSE,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE loc.translation (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    language_code    varchar(10)  NOT NULL,
    resource_key     varchar(200) NOT NULL,
    resource_value   text         NOT NULL,
    module           varchar(50),
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE loc.locale_format (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    locale_code      varchar(20)  NOT NULL,
    date_format      varchar(50)  NOT NULL,
    number_format    varchar(50)  NOT NULL,
    currency_format  varchar(50)  NOT NULL,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE loc.outbox (
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

CREATE TABLE loc.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);

CREATE TABLE loc.user_language_preference (
    id               uuid        NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid        NOT NULL,
    user_ref         uuid        NOT NULL,
    language_code    varchar(10) NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    created_by       uuid        NOT NULL,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer     NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    CONSTRAINT uq_user_language_preference UNIQUE (organization_id, user_ref)
);

CREATE TABLE loc.organization_language (
    id                     uuid        NOT NULL DEFAULT gen_random_uuid(),
    organization_id        uuid        NOT NULL,
    organization_ref       uuid        NOT NULL,
    default_language_code  varchar(10) NOT NULL,
    created_at             timestamptz NOT NULL DEFAULT now(),
    created_by             uuid        NOT NULL,
    updated_at             timestamptz NOT NULL DEFAULT now(),
    updated_by             uuid,
    deleted_at             timestamptz,
    row_version            integer     NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    CONSTRAINT uq_organization_language UNIQUE (organization_id, organization_ref)
);

