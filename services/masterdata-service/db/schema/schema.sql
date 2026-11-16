-- sqlc catalog for masterdata-service (plain DDL for type inference ONLY).
-- Authoritative schema (RLS via app.org_scope, tg_touch_row trigger, RANGE-
-- partitioned outbox by created_at) lives in db/migrations/masterdata_db. Keep
-- this catalog in sync (sqlc diff gate in CI).
CREATE SCHEMA IF NOT EXISTS md;

CREATE TABLE md.party (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    code             varchar(50)  NOT NULL,
    name             varchar(200) NOT NULL,
    party_type       varchar(20)  NOT NULL,
    tax_id           varchar(50),
    is_shared        boolean      NOT NULL DEFAULT false,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE md.party_address (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    party_id         uuid         NOT NULL REFERENCES md.party(id) ON DELETE CASCADE,
    address_type     varchar(30)  NOT NULL DEFAULT 'billing',
    line1            varchar(200) NOT NULL,
    line2            varchar(200),
    city             varchar(100),
    state_province   varchar(100),
    postal_code      varchar(20),
    country_code     varchar(2),
    is_primary       boolean      NOT NULL DEFAULT false,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE md.item (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    code             varchar(50)  NOT NULL,
    name             varchar(200) NOT NULL,
    item_type        varchar(30)  NOT NULL,
    base_uom         varchar(20)  NOT NULL,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE md.item_uom (
    id                 uuid          NOT NULL DEFAULT gen_random_uuid(),
    organization_id    uuid          NOT NULL,
    item_id            uuid          NOT NULL REFERENCES md.item(id) ON DELETE CASCADE,
    uom_code           varchar(20)   NOT NULL,
    conversion_to_base numeric(20,4) NOT NULL DEFAULT 1,
    is_base            boolean       NOT NULL DEFAULT false,
    created_at         timestamptz   NOT NULL DEFAULT now(),
    created_by         uuid          NOT NULL,
    updated_at         timestamptz   NOT NULL DEFAULT now(),
    updated_by         uuid,
    deleted_at         timestamptz,
    row_version        integer       NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE md.outbox (
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

CREATE TABLE md.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
