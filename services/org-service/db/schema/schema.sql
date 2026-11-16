-- sqlc catalog for org-service (plain DDL for type inference ONLY).
-- Authoritative schema (RLS via app.org_scope, tg_touch_row trigger, org tree
-- FKs/closure table, RANGE-partitioned outbox) lives in db/migrations/org_db.
-- Keep this catalog in sync (sqlc diff gate in CI).
CREATE SCHEMA IF NOT EXISTS org;

CREATE TABLE org.organization_type (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    code             varchar(50)  NOT NULL,
    name             varchar(150) NOT NULL,
    level_hint       integer,
    is_active        boolean      NOT NULL DEFAULT true,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE org.organization (
    id                    uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id       uuid         NOT NULL,
    code                  varchar(50)  NOT NULL,
    name                  varchar(200) NOT NULL,
    organization_type_id  uuid         NOT NULL REFERENCES org.organization_type(id) ON DELETE RESTRICT,
    parent_id             uuid         REFERENCES org.organization(id) ON DELETE RESTRICT,
    path                  text         NOT NULL,
    status                varchar(20)  NOT NULL DEFAULT 'Draft',
    effective_from        date         NOT NULL,
    effective_to          date,
    created_at            timestamptz  NOT NULL DEFAULT now(),
    created_by            uuid         NOT NULL,
    updated_at            timestamptz  NOT NULL DEFAULT now(),
    updated_by            uuid,
    deleted_at            timestamptz,
    row_version           integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE org.organization_closure (
    organization_id  uuid        NOT NULL,
    ancestor_id      uuid        NOT NULL REFERENCES org.organization(id) ON DELETE CASCADE,
    descendant_id    uuid        NOT NULL REFERENCES org.organization(id) ON DELETE CASCADE,
    depth            integer     NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    created_by       uuid        NOT NULL,
    PRIMARY KEY (ancestor_id, descendant_id)
);

CREATE TABLE org.organization_attribute (
    id                   uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id      uuid         NOT NULL,
    organization_ref_id  uuid         NOT NULL,
    attr_key             varchar(100) NOT NULL,
    attr_value           jsonb        NOT NULL,
    created_at           timestamptz  NOT NULL DEFAULT now(),
    created_by           uuid         NOT NULL,
    updated_at           timestamptz  NOT NULL DEFAULT now(),
    updated_by           uuid,
    deleted_at           timestamptz,
    row_version          integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE org.outbox (
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

CREATE TABLE org.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
