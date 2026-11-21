-- sqlc catalog for gl_db (plain DDL only — the real DDL, incl. RANGE partitioning,
-- RLS, triggers and the document-number registry, lives in db/migrations/gl_db).
-- Columns here MUST match the migrated schema.
CREATE SCHEMA IF NOT EXISTS gl;

CREATE TABLE gl.journal_entry (
    id                          uuid           NOT NULL DEFAULT gen_random_uuid(),
    organization_id             uuid           NOT NULL,
    journal_no                  varchar(50)    NOT NULL,
    financial_organization_id   uuid           NOT NULL,
    ledger_id                   uuid           NOT NULL,
    fiscal_period_id            uuid           NOT NULL,
    posting_date                date           NOT NULL,
    journal_source              varchar(30)    NOT NULL,
    source_module               varchar(50),
    source_document_ref         varchar(100),
    currency_code               varchar(3)     NOT NULL,
    exchange_rate               numeric(20,8)  NOT NULL DEFAULT 1,
    status                      varchar(20)    NOT NULL DEFAULT 'Draft',
    reversed_entry_id           uuid,
    correlation_id              uuid           NOT NULL,
    posted_at                   timestamptz,
    created_at                  timestamptz    NOT NULL DEFAULT now(),
    created_by                  uuid           NOT NULL,
    updated_at                  timestamptz    NOT NULL DEFAULT now(),
    updated_by                  uuid,
    deleted_at                  timestamptz,
    row_version                 integer        NOT NULL DEFAULT 1,
    PRIMARY KEY (id, posting_date)
);

CREATE TABLE gl.journal_line (
    id                       uuid           NOT NULL DEFAULT gen_random_uuid(),
    organization_id          uuid           NOT NULL,
    journal_entry_id         uuid           NOT NULL,
    posting_date             date           NOT NULL,
    line_no                  integer        NOT NULL,
    account_id               uuid           NOT NULL,
    account_combination_id   uuid,
    debit_amount             numeric(20,4)  NOT NULL DEFAULT 0,
    credit_amount            numeric(20,4)  NOT NULL DEFAULT 0,
    functional_amount        numeric(20,4)  NOT NULL DEFAULT 0,
    cost_center_id           uuid,
    project_id               uuid,
    description              text,
    created_at               timestamptz    NOT NULL DEFAULT now(),
    created_by               uuid           NOT NULL,
    updated_at               timestamptz    NOT NULL DEFAULT now(),
    updated_by               uuid,
    deleted_at               timestamptz,
    row_version              integer        NOT NULL DEFAULT 1,
    PRIMARY KEY (id, posting_date)
);

CREATE TABLE gl.account_balance (
    id                uuid           NOT NULL DEFAULT gen_random_uuid(),
    organization_id   uuid           NOT NULL,
    account_id        uuid           NOT NULL,
    fiscal_period_id  uuid           NOT NULL,
    ledger_id         uuid           NOT NULL,
    opening_balance   numeric(20,4)  NOT NULL DEFAULT 0,
    period_debit      numeric(20,4)  NOT NULL DEFAULT 0,
    period_credit     numeric(20,4)  NOT NULL DEFAULT 0,
    closing_balance   numeric(20,4)  NOT NULL DEFAULT 0,
    created_at        timestamptz    NOT NULL DEFAULT now(),
    created_by        uuid           NOT NULL,
    updated_at        timestamptz    NOT NULL DEFAULT now(),
    updated_by        uuid,
    deleted_at        timestamptz,
    row_version       integer        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE gl.outbox (
    id               uuid          NOT NULL DEFAULT gen_random_uuid(),
    aggregate_type   varchar(100)  NOT NULL,
    aggregate_id     uuid          NOT NULL,
    event_type       varchar(150)  NOT NULL,
    payload          jsonb         NOT NULL,
    organization_id  uuid          NOT NULL,
    correlation_id   uuid          NOT NULL,
    created_at       timestamptz   NOT NULL DEFAULT now(),
    published_at     timestamptz,
    PRIMARY KEY (id, created_at)
);

-- Consumer idempotency ledger (technical table, no RLS) — GL also consumes
-- subledger events (ar.invoice.issued / ap.invoice.recorded) and auto-posts a
-- journal; each event_id is recorded once so re-delivery is a no-op (SDD-030).
CREATE TABLE gl.processed_event (
    event_id      uuid          NOT NULL,
    event_type    varchar(150)  NOT NULL,
    processed_at  timestamptz   NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);

CREATE TABLE gl.fiscal_period_status (
    organization_id uuid        NOT NULL,
    period_id       uuid        NOT NULL,
    status          varchar(20) NOT NULL DEFAULT 'Open',
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, period_id)
);

CREATE TABLE gl.recurring_journal_template (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    code             varchar(50)  NOT NULL,
    frequency        varchar(20)  NOT NULL,
    next_run_date    date,
    template_lines   jsonb        NOT NULL,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid         NOT NULL,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    deleted_at       timestamptz,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    CONSTRAINT uq_recurring_journal_template UNIQUE (organization_id, code)
);

CREATE TABLE gl.document_number_registry (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  uuid         NOT NULL,
    document_type    varchar(50)  NOT NULL,
    document_number  varchar(50)  NOT NULL,
    fiscal_year      integer      NOT NULL,
    status           varchar(20)  NOT NULL DEFAULT 'Issued',
    source_ref       varchar(100),
    created_at       timestamptz  NOT NULL DEFAULT now(),
    created_by       uuid,
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    updated_by       uuid,
    row_version      integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    CONSTRAINT uq_document_number_registry UNIQUE (organization_id, document_type, fiscal_year, document_number)
);
