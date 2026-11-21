CREATE SCHEMA IF NOT EXISTS ast;

CREATE TABLE ast.asset (
    id                UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID         NOT NULL,
    asset_code        VARCHAR(50)  NOT NULL,
    name              VARCHAR(200) NOT NULL,
    category          VARCHAR(50)  NOT NULL,
    specification     JSONB,
    acquisition_date  DATE,
    organization_ref  UUID         NOT NULL,
    fixed_asset_ref   UUID,
    barcode           VARCHAR(100),
    status            VARCHAR(20)  NOT NULL DEFAULT 'Active',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by        UUID         NOT NULL,
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by        UUID,
    deleted_at        TIMESTAMPTZ,
    row_version       INTEGER      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE (organization_id, asset_code)
);

CREATE TABLE ast.asset_location_history (
    id                UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID         NOT NULL,
    asset_id          UUID         NOT NULL REFERENCES ast.asset (id) ON DELETE CASCADE,
    organization_ref  UUID         NOT NULL,
    warehouse_ref     UUID,
    location_note     VARCHAR(200),
    effective_from    DATE         NOT NULL,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by        UUID         NOT NULL,
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by        UUID,
    deleted_at        TIMESTAMPTZ,
    row_version       INTEGER      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE ast.asset_movement (
    id                       UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id          UUID         NOT NULL,
    asset_id                 UUID         NOT NULL REFERENCES ast.asset (id) ON DELETE CASCADE,
    from_organization_ref    UUID,
    to_organization_ref      UUID         NOT NULL,
    status                   VARCHAR(20)  NOT NULL DEFAULT 'Draft',
    approved_by              UUID,
    moved_at                 TIMESTAMPTZ,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by               UUID         NOT NULL,
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by               UUID,
    deleted_at               TIMESTAMPTZ,
    row_version              INTEGER      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE ast.asset_verification (
    id                 UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id    UUID         NOT NULL,
    asset_id           UUID         NOT NULL REFERENCES ast.asset (id) ON DELETE CASCADE,
    verification_date  DATE         NOT NULL,
    system_status      VARCHAR(20),
    physical_status    VARCHAR(20),
    discrepancy_note   TEXT,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by         UUID         NOT NULL,
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by         UUID,
    deleted_at         TIMESTAMPTZ,
    row_version        INTEGER      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE ast.outbox (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    aggregate_type   VARCHAR(100) NOT NULL,
    aggregate_id     UUID         NOT NULL,
    event_type       VARCHAR(150) NOT NULL,
    payload          JSONB        NOT NULL,
    organization_id  UUID         NOT NULL,
    correlation_id   UUID         NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    published_at     TIMESTAMPTZ,
    PRIMARY KEY (id, created_at)
);

CREATE TABLE ast.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
