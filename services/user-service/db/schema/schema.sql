-- sqlc catalog for user-service (plain DDL for type inference ONLY).
-- Authoritative schema (RLS via app.org_scope, tg_touch_row trigger, IdP
-- federation, RANGE-partitioned outbox) lives in db/migrations/user_db.
-- No passwords stored — authentication is delegated to the IdP (SAD BAB 7.3).
-- Keep this catalog in sync (sqlc diff gate in CI).
CREATE SCHEMA IF NOT EXISTS usr;

CREATE TABLE usr.app_user (
    id                       uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id          uuid         NOT NULL,
    username                 varchar(100) NOT NULL,
    email                    varchar(255) NOT NULL,
    full_name                varchar(200) NOT NULL,
    status                   varchar(20)  NOT NULL DEFAULT 'Active',
    default_organization_id  uuid,
    created_at               timestamptz  NOT NULL DEFAULT now(),
    created_by               uuid         NOT NULL,
    updated_at               timestamptz  NOT NULL DEFAULT now(),
    updated_by               uuid,
    deleted_at               timestamptz,
    row_version              integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE usr.user_identity (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    user_id          uuid         NOT NULL REFERENCES usr.app_user(id) ON DELETE CASCADE,
    provider         varchar(50)  NOT NULL,
    subject          varchar(255) NOT NULL,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE usr.user_org_assignment (
    id                   uuid        NOT NULL DEFAULT gen_random_uuid(),
    organization_id      uuid        NOT NULL,
    user_id              uuid        NOT NULL REFERENCES usr.app_user(id) ON DELETE CASCADE,
    organization_ref_id  uuid        NOT NULL,
    role_id              uuid        NOT NULL,
    created_at           timestamptz NOT NULL DEFAULT now(),
    created_by           uuid        NOT NULL,
    updated_at           timestamptz NOT NULL DEFAULT now(),
    updated_by           uuid,
    deleted_at           timestamptz,
    row_version          integer     NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    CONSTRAINT uq_user_org_assignment UNIQUE (user_id, organization_ref_id, role_id)
);

CREATE TABLE usr.outbox (
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

CREATE TABLE usr.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
