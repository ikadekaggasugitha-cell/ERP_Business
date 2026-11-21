-- sqlc catalog for financialclose-service (mirrors db/migrations/financialclose_db).
CREATE SCHEMA IF NOT EXISTS fcl;

CREATE TABLE fcl.closing_checklist_template (
    id                            uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id               uuid         NOT NULL,
    code                          varchar(50)  NOT NULL,
    name                          varchar(150) NOT NULL,
    financial_organization_ref    uuid         NOT NULL,
    created_at                    timestamptz  NOT NULL DEFAULT now(),
    created_by                    uuid         NOT NULL,
    updated_at                    timestamptz  NOT NULL DEFAULT now(),
    updated_by                    uuid,
    deleted_at                    timestamptz,
    row_version                   integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE fcl.closing_task_template (
    id              uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id uuid         NOT NULL,
    template_id     uuid         NOT NULL,
    task_order      integer      NOT NULL,
    task_name       varchar(150) NOT NULL,
    is_mandatory    boolean      NOT NULL DEFAULT TRUE,
    created_at      timestamptz  NOT NULL DEFAULT now(),
    created_by      uuid         NOT NULL,
    updated_at      timestamptz  NOT NULL DEFAULT now(),
    updated_by      uuid,
    deleted_at      timestamptz,
    row_version     integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE fcl.closing_run (
    id                            uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id               uuid         NOT NULL,
    financial_organization_ref    uuid         NOT NULL,
    fiscal_period_ref             uuid         NOT NULL,
    run_type                      varchar(20)  NOT NULL DEFAULT 'PeriodEnd',
    status                        varchar(20)  NOT NULL DEFAULT 'Open',
    closed_at                     timestamptz,
    closed_by                     uuid,
    created_at                    timestamptz  NOT NULL DEFAULT now(),
    created_by                    uuid         NOT NULL,
    updated_at                    timestamptz  NOT NULL DEFAULT now(),
    updated_by                    uuid,
    deleted_at                    timestamptz,
    row_version                   integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE fcl.closing_task (
    id              uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id uuid         NOT NULL,
    closing_run_id  uuid         NOT NULL,
    task_name       varchar(150) NOT NULL,
    is_mandatory    boolean      NOT NULL DEFAULT TRUE,
    status          varchar(20)  NOT NULL DEFAULT 'Pending',
    assignee_ref    uuid,
    completed_at    timestamptz,
    created_at      timestamptz  NOT NULL DEFAULT now(),
    created_by      uuid         NOT NULL,
    updated_at      timestamptz  NOT NULL DEFAULT now(),
    updated_by      uuid,
    deleted_at      timestamptz,
    row_version     integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE fcl.period_status (
    id                            uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id               uuid         NOT NULL,
    financial_organization_ref    uuid         NOT NULL,
    fiscal_period_ref             uuid         NOT NULL,
    module                        varchar(30)  NOT NULL,
    status                        varchar(20)  NOT NULL DEFAULT 'Open',
    created_at                    timestamptz  NOT NULL DEFAULT now(),
    created_by                    uuid         NOT NULL,
    updated_at                    timestamptz  NOT NULL DEFAULT now(),
    updated_by                    uuid,
    deleted_at                    timestamptz,
    row_version                   integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    CONSTRAINT uq_period_status UNIQUE (organization_id, financial_organization_ref, fiscal_period_ref, module)
);

CREATE TABLE fcl.period_reopening (
    id                            uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id               uuid         NOT NULL,
    financial_organization_ref    uuid         NOT NULL,
    fiscal_period_ref             uuid         NOT NULL,
    requested_by                  uuid         NOT NULL,
    approved_by                   uuid,
    reason                        text         NOT NULL,
    reopened_at                   timestamptz,
    created_at                    timestamptz  NOT NULL DEFAULT now(),
    created_by                    uuid         NOT NULL,
    updated_at                    timestamptz  NOT NULL DEFAULT now(),
    updated_by                    uuid,
    deleted_at                    timestamptz,
    row_version                   integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE fcl.outbox (
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

CREATE TABLE fcl.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
