CREATE SCHEMA IF NOT EXISTS crm;

CREATE TABLE crm.lead (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    code             VARCHAR(50)    NOT NULL,
    name             VARCHAR(200)   NOT NULL,
    source           VARCHAR(50),
    status           VARCHAR(20)    NOT NULL DEFAULT 'New',
    estimated_value  NUMERIC(20,4),
    customer_ref     UUID,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE (organization_id, code)
);

CREATE TABLE crm.opportunity (
    id                    UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id       UUID           NOT NULL,
    code                  VARCHAR(50)    NOT NULL,
    lead_id               UUID           REFERENCES crm.lead (id) ON DELETE SET NULL,
    customer_ref          UUID,
    stage                 VARCHAR(50)    NOT NULL,
    probability_pct       NUMERIC(5,2),
    estimated_value       NUMERIC(20,4),
    expected_close_date   DATE,
    status                VARCHAR(20)    NOT NULL DEFAULT 'Open',
    created_at            TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by            UUID           NOT NULL,
    updated_at            TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by            UUID,
    deleted_at            TIMESTAMPTZ,
    row_version           INTEGER        NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE (organization_id, code)
);

CREATE TABLE crm.customer_segment (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(50)  NOT NULL,
    name             VARCHAR(150) NOT NULL,
    criteria         JSONB,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE (organization_id, code)
);

CREATE TABLE crm.interaction (
    id                UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID         NOT NULL,
    customer_ref      UUID,
    lead_id           UUID         REFERENCES crm.lead (id) ON DELETE SET NULL,
    interaction_type  VARCHAR(20)  NOT NULL,
    occurred_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    notes             TEXT,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by        UUID         NOT NULL,
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by        UUID,
    deleted_at        TIMESTAMPTZ,
    row_version       INTEGER      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE crm.support_case (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    case_no          VARCHAR(50)  NOT NULL,
    customer_ref     UUID,
    subject          VARCHAR(200) NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'Open',
    resolution       TEXT,
    opened_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    resolved_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE (organization_id, case_no)
);

CREATE TABLE crm.activity_task (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    assignee_ref     UUID         NOT NULL,
    related_type     VARCHAR(30),
    related_ref      UUID,
    due_date         DATE,
    status           VARCHAR(20)  NOT NULL DEFAULT 'Pending',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE crm.outbox (
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

CREATE TABLE crm.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
