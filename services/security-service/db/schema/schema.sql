-- sqlc catalog for security-service (plain DDL for type inference ONLY).
-- Authoritative schema (RLS via app.org_scope, tg_touch_row trigger, RBAC +
-- scope-based access, RANGE-partitioned outbox) lives in db/migrations/security_db.
-- Keep this catalog in sync (sqlc diff gate in CI).
CREATE SCHEMA IF NOT EXISTS sec;

CREATE TABLE sec.role (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    code             varchar(50)  NOT NULL,
    name             varchar(150) NOT NULL,
    is_system        boolean      NOT NULL DEFAULT false,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE sec.permission (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    code             varchar(100) NOT NULL,
    module           varchar(50)  NOT NULL,
    action           varchar(50)  NOT NULL,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE sec.role_permission (
    organization_id  uuid        NOT NULL,
    role_id          uuid        NOT NULL REFERENCES sec.role(id) ON DELETE CASCADE,
    permission_id    uuid        NOT NULL REFERENCES sec.permission(id) ON DELETE CASCADE,
    created_at       timestamptz NOT NULL DEFAULT now(),
    created_by       uuid        NOT NULL,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE sec.role_scope (
    id                   uuid        NOT NULL DEFAULT gen_random_uuid(),
    organization_id      uuid        NOT NULL,
    role_id              uuid        NOT NULL REFERENCES sec.role(id) ON DELETE CASCADE,
    scope_type           varchar(30) NOT NULL,
    organization_ref_id  uuid,
    created_at           timestamptz NOT NULL DEFAULT now(),
    created_by           uuid        NOT NULL,
    updated_at           timestamptz NOT NULL DEFAULT now(),
    updated_by           uuid,
    deleted_at           timestamptz,
    row_version          integer     NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE sec.outbox (
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

CREATE TABLE sec.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
