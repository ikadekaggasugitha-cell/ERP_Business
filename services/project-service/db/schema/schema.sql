CREATE SCHEMA IF NOT EXISTS prj;

CREATE TABLE prj.project (
    id                UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID         NOT NULL,
    code              VARCHAR(50)  NOT NULL,
    name              VARCHAR(200) NOT NULL,
    project_type      VARCHAR(50)  NOT NULL,
    organization_ref  UUID         NOT NULL,
    currency_code     VARCHAR(3)   NOT NULL,
    start_date        DATE,
    end_date          DATE,
    status            VARCHAR(20)  NOT NULL DEFAULT 'Draft',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by        UUID         NOT NULL,
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by        UUID,
    deleted_at        TIMESTAMPTZ,
    row_version       INTEGER      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE (organization_id, code)
);

CREATE TABLE prj.wbs_element (
    id              UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id UUID         NOT NULL,
    project_id      UUID         NOT NULL REFERENCES prj.project (id) ON DELETE CASCADE,
    parent_id       UUID         REFERENCES prj.wbs_element (id) ON DELETE RESTRICT,
    code            VARCHAR(50)  NOT NULL,
    name            VARCHAR(200) NOT NULL,
    level_no        INTEGER      NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by      UUID         NOT NULL,
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by      UUID,
    deleted_at      TIMESTAMPTZ,
    row_version     INTEGER      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE (project_id, code)
);

CREATE TABLE prj.project_budget (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    project_id       UUID           NOT NULL REFERENCES prj.project (id) ON DELETE CASCADE,
    wbs_element_id   UUID           NOT NULL REFERENCES prj.wbs_element (id) ON DELETE CASCADE,
    version          INTEGER        NOT NULL DEFAULT 1,
    budget_amount    NUMERIC(20,4)  NOT NULL DEFAULT 0,
    status           VARCHAR(20)    NOT NULL DEFAULT 'Draft',
    is_baseline      BOOLEAN        NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE prj.project_cost (
    id                   UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id      UUID           NOT NULL,
    project_id           UUID           NOT NULL REFERENCES prj.project (id) ON DELETE CASCADE,
    wbs_element_id       UUID           NOT NULL REFERENCES prj.wbs_element (id) ON DELETE RESTRICT,
    cost_type            VARCHAR(20)    NOT NULL,
    amount               NUMERIC(20,4)  NOT NULL DEFAULT 0,
    source_document_ref  VARCHAR(100),
    posting_date         DATE           NOT NULL,
    created_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by           UUID           NOT NULL,
    updated_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by           UUID,
    deleted_at           TIMESTAMPTZ,
    row_version          INTEGER        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE prj.project_progress (
    id                    UUID          NOT NULL DEFAULT gen_random_uuid(),
    organization_id       UUID          NOT NULL,
    wbs_element_id        UUID          NOT NULL REFERENCES prj.wbs_element (id) ON DELETE CASCADE,
    period_date           DATE          NOT NULL,
    physical_progress_pct NUMERIC(5,2)  NOT NULL DEFAULT 0,
    recorded_at           TIMESTAMPTZ   NOT NULL DEFAULT now(),
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT now(),
    created_by            UUID          NOT NULL,
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_by            UUID,
    deleted_at            TIMESTAMPTZ,
    row_version           INTEGER       NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE (wbs_element_id, period_date)
);

CREATE TABLE prj.project_billing (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    project_id       UUID           NOT NULL REFERENCES prj.project (id) ON DELETE CASCADE,
    billing_type     VARCHAR(20)    NOT NULL,
    amount           NUMERIC(20,4)  NOT NULL DEFAULT 0,
    ar_invoice_ref   VARCHAR(100),
    status           VARCHAR(20)    NOT NULL DEFAULT 'Draft',
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE prj.change_order (
    id                     UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id        UUID           NOT NULL,
    project_id             UUID           NOT NULL REFERENCES prj.project (id) ON DELETE CASCADE,
    description            TEXT           NOT NULL,
    budget_impact_amount   NUMERIC(20,4)  NOT NULL DEFAULT 0,
    schedule_impact_days   INTEGER        NOT NULL DEFAULT 0,
    status                 VARCHAR(20)    NOT NULL DEFAULT 'Draft',
    created_at             TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by             UUID           NOT NULL,
    updated_at             TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by             UUID,
    deleted_at             TIMESTAMPTZ,
    row_version            INTEGER        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE prj.outbox (
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

CREATE TABLE prj.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
