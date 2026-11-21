CREATE SCHEMA IF NOT EXISTS mnt;

CREATE TABLE mnt.maintenance_schedule (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    asset_ref        UUID         NOT NULL,
    schedule_type    VARCHAR(20)  NOT NULL,
    interval_value   INTEGER      NOT NULL,
    interval_unit    VARCHAR(20)  NOT NULL,
    next_due_date    DATE,
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE mnt.work_order (
    id                UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID         NOT NULL,
    wo_no             VARCHAR(50)  NOT NULL,
    asset_ref         UUID         NOT NULL,
    maintenance_type  VARCHAR(20)  NOT NULL,
    status            VARCHAR(20)  NOT NULL DEFAULT 'Open',
    assignee_ref      UUID,
    vendor_ref        UUID,
    scheduled_date    DATE,
    downtime_start    TIMESTAMPTZ,
    downtime_end      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by        UUID         NOT NULL,
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by        UUID,
    deleted_at        TIMESTAMPTZ,
    row_version       INTEGER      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE (organization_id, wo_no)
);

CREATE TABLE mnt.work_order_part (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    work_order_id    UUID           NOT NULL REFERENCES mnt.work_order (id) ON DELETE CASCADE,
    item_ref         UUID           NOT NULL,
    quantity         NUMERIC(20,4)  NOT NULL DEFAULT 0,
    unit_cost        NUMERIC(20,4)  NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE mnt.work_order_checklist (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    work_order_id    UUID         NOT NULL REFERENCES mnt.work_order (id) ON DELETE CASCADE,
    step_no          INTEGER      NOT NULL,
    description      VARCHAR(200) NOT NULL,
    is_done          BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE (work_order_id, step_no)
);

CREATE TABLE mnt.maintenance_cost (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    work_order_id    UUID           NOT NULL REFERENCES mnt.work_order (id) ON DELETE CASCADE,
    cost_type        VARCHAR(20)    NOT NULL,
    amount           NUMERIC(20,4)  NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE mnt.outbox (
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

CREATE TABLE mnt.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
