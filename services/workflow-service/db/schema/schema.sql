-- sqlc catalog for workflow-service (plain DDL for type inference ONLY).
-- Authoritative schema (RLS via app.org_scope, tg_touch_row trigger, RANGE-
-- partitioned outbox by created_at) lives in db/migrations/workflow_db. Keep
-- this catalog in sync (sqlc diff gate in CI).
CREATE SCHEMA IF NOT EXISTS wf;

CREATE TABLE wf.workflow_definition (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    code             varchar(50)  NOT NULL,
    document_type    varchar(50)  NOT NULL,
    version          integer      NOT NULL DEFAULT 1,
    is_active        boolean      NOT NULL DEFAULT true,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE wf.workflow_step (
    id               uuid        NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid        NOT NULL,
    definition_id    uuid        NOT NULL REFERENCES wf.workflow_definition(id) ON DELETE CASCADE,
    step_order       integer     NOT NULL,
    approver_rule    jsonb       NOT NULL,
    escalation_rule  jsonb,
    created_at       timestamptz NOT NULL DEFAULT now(),
    created_by       uuid        NOT NULL,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer     NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE wf.workflow_instance (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    definition_id    uuid         NOT NULL REFERENCES wf.workflow_definition(id) ON DELETE RESTRICT,
    document_ref     varchar(100) NOT NULL,
    status           varchar(20)  NOT NULL DEFAULT 'Pending',
    current_step     integer,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE wf.workflow_task (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    instance_id      uuid         NOT NULL REFERENCES wf.workflow_instance(id) ON DELETE CASCADE,
    assignee_id      uuid         NOT NULL,
    decision         varchar(20),
    decided_at       timestamptz,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE wf.outbox (
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

CREATE TABLE wf.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
