-- sqlc catalog for fixedasset-service (mirrors db/migrations/fixedasset_db).
CREATE SCHEMA IF NOT EXISTS fa;

CREATE TABLE fa.asset_category (
    id                           uuid         NOT NULL DEFAULT gen_random_uuid(),
    organization_id              uuid         NOT NULL,
    code                         varchar(50)  NOT NULL,
    name                         varchar(150) NOT NULL,
    default_depreciation_method  varchar(30)  NOT NULL DEFAULT 'StraightLine',
    default_useful_life_months   integer,
    is_active                    boolean      NOT NULL DEFAULT TRUE,
    created_at                   timestamptz  NOT NULL DEFAULT now(),
    created_by                   uuid         NOT NULL,
    updated_at                   timestamptz  NOT NULL DEFAULT now(),
    updated_by                   uuid,
    deleted_at                   timestamptz,
    row_version                  integer      NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE fa.fixed_asset (
    id                                uuid           NOT NULL DEFAULT gen_random_uuid(),
    organization_id                   uuid           NOT NULL,
    asset_no                          varchar(50)    NOT NULL,
    name                              varchar(200)   NOT NULL,
    asset_category_id                 uuid           NOT NULL,
    financial_organization_ref        uuid           NOT NULL,
    acquisition_date                  date           NOT NULL,
    acquisition_cost                  numeric(20,4)  NOT NULL DEFAULT 0,
    salvage_value                     numeric(20,4)  NOT NULL DEFAULT 0,
    useful_life_months                integer        NOT NULL,
    depreciation_method               varchar(30)    NOT NULL DEFAULT 'StraightLine',
    depreciation_start_date           date,
    asset_account_id                  uuid           NOT NULL,
    accum_depreciation_account_id     uuid           NOT NULL,
    depreciation_expense_account_id   uuid           NOT NULL,
    cost_center_ref                   uuid,
    net_book_value                    numeric(20,4)  NOT NULL DEFAULT 0,
    parent_asset_id                   uuid,
    status                            varchar(20)    NOT NULL DEFAULT 'Draft',
    created_at                        timestamptz    NOT NULL DEFAULT now(),
    created_by                        uuid           NOT NULL,
    updated_at                        timestamptz    NOT NULL DEFAULT now(),
    updated_by                        uuid,
    deleted_at                        timestamptz,
    row_version                       integer        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE fa.depreciation_entry (
    id                        uuid           NOT NULL DEFAULT gen_random_uuid(),
    organization_id           uuid           NOT NULL,
    fixed_asset_id            uuid           NOT NULL,
    fiscal_period_ref         uuid           NOT NULL,
    depreciation_amount       numeric(20,4)  NOT NULL DEFAULT 0,
    accumulated_depreciation  numeric(20,4)  NOT NULL DEFAULT 0,
    net_book_value            numeric(20,4)  NOT NULL DEFAULT 0,
    is_posted                 boolean        NOT NULL DEFAULT FALSE,
    gl_journal_ref            varchar(50),
    run_date                  date           NOT NULL,
    created_at                timestamptz    NOT NULL DEFAULT now(),
    created_by                uuid           NOT NULL,
    updated_at                timestamptz    NOT NULL DEFAULT now(),
    updated_by                uuid,
    deleted_at                timestamptz,
    row_version               integer        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE fa.asset_transaction (
    id                                uuid           NOT NULL DEFAULT gen_random_uuid(),
    organization_id                   uuid           NOT NULL,
    fixed_asset_id                    uuid           NOT NULL,
    transaction_type                  varchar(30)    NOT NULL,
    transaction_date                  date           NOT NULL,
    amount                            numeric(20,4)  NOT NULL DEFAULT 0,
    gain_loss_amount                  numeric(20,4),
    from_financial_organization_ref   uuid,
    to_financial_organization_ref     uuid,
    gl_journal_ref                    varchar(50),
    notes                             text,
    created_at                        timestamptz    NOT NULL DEFAULT now(),
    created_by                        uuid           NOT NULL,
    updated_at                        timestamptz    NOT NULL DEFAULT now(),
    updated_by                        uuid,
    deleted_at                        timestamptz,
    row_version                       integer        NOT NULL DEFAULT 1,
    PRIMARY KEY (id)
);

CREATE TABLE fa.outbox (
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

CREATE TABLE fa.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
