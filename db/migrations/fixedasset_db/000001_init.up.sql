-- =============================================================================
-- fixedasset_db — Fixed Asset (SVC-020)
-- DBDD BAB 9.2 | Standards DBS-001..016 | BRD 9.5.7 (BR-586..595)
-- golang-migrate: 000001_init.up.sql
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS btree_gin;

CREATE OR REPLACE FUNCTION public.tg_touch_row()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at  := now();
  NEW.row_version := COALESCE(OLD.row_version, 0) + 1;
  RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION public.current_org_scope()
RETURNS uuid[] LANGUAGE sql STABLE AS $$
  SELECT NULLIF(current_setting('app.org_scope', true), '')::uuid[];
$$;

CREATE SCHEMA IF NOT EXISTS fa;

-- =============================================================================
-- fa.asset_category
-- =============================================================================
CREATE TABLE fa.asset_category (
    id                           UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id              UUID         NOT NULL,
    code                         VARCHAR(50)  NOT NULL,
    name                         VARCHAR(150) NOT NULL,
    default_depreciation_method  VARCHAR(30)  NOT NULL DEFAULT 'StraightLine',
    default_useful_life_months   INTEGER,
    is_active                    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at                   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by                   UUID         NOT NULL,
    updated_at                   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by                   UUID,
    deleted_at                   TIMESTAMPTZ,
    row_version                  INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_asset_category PRIMARY KEY (id),
    CONSTRAINT uq_asset_category_org_code UNIQUE (organization_id, code),
    CONSTRAINT ck_asset_category_method
        CHECK (default_depreciation_method IN ('StraightLine','DecliningBalance','UnitsOfProduction'))
);

-- =============================================================================
-- fa.fixed_asset  (parent_asset_id = component accounting, BR-593)
-- financial_organization_ref / *_account_id / cost_center_ref are logical refs.
-- =============================================================================
CREATE TABLE fa.fixed_asset (
    id                                UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id                   UUID           NOT NULL,
    asset_no                          VARCHAR(50)    NOT NULL,
    name                              VARCHAR(200)   NOT NULL,
    asset_category_id                 UUID           NOT NULL,
    financial_organization_ref        UUID           NOT NULL,
    acquisition_date                  DATE           NOT NULL,
    acquisition_cost                  NUMERIC(20,4)  NOT NULL DEFAULT 0,
    salvage_value                     NUMERIC(20,4)  NOT NULL DEFAULT 0,
    useful_life_months                INTEGER        NOT NULL,
    depreciation_method               VARCHAR(30)    NOT NULL DEFAULT 'StraightLine',
    depreciation_start_date           DATE,
    asset_account_id                  UUID           NOT NULL,
    accum_depreciation_account_id     UUID           NOT NULL,
    depreciation_expense_account_id   UUID           NOT NULL,
    cost_center_ref                   UUID,
    net_book_value                    NUMERIC(20,4)  NOT NULL DEFAULT 0,
    parent_asset_id                   UUID,
    status                            VARCHAR(20)    NOT NULL DEFAULT 'Draft',
    created_at                        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by                        UUID           NOT NULL,
    updated_at                        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by                        UUID,
    deleted_at                        TIMESTAMPTZ,
    row_version                       INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_fixed_asset PRIMARY KEY (id),
    CONSTRAINT uq_fixed_asset_no UNIQUE (organization_id, asset_no),
    CONSTRAINT fk_fixed_asset_category
        FOREIGN KEY (asset_category_id) REFERENCES fa.asset_category (id) ON DELETE RESTRICT,
    CONSTRAINT fk_fixed_asset_parent
        FOREIGN KEY (parent_asset_id) REFERENCES fa.fixed_asset (id) ON DELETE RESTRICT,
    CONSTRAINT ck_fixed_asset_method
        CHECK (depreciation_method IN ('StraightLine','DecliningBalance','UnitsOfProduction')),
    CONSTRAINT ck_fixed_asset_status
        CHECK (status IN ('Draft','Active','UnderRevaluation','Disposed')),
    CONSTRAINT ck_fixed_asset_life CHECK (useful_life_months > 0)
);

-- =============================================================================
-- fa.depreciation_entry (BR-589)
-- =============================================================================
CREATE TABLE fa.depreciation_entry (
    id                        UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id           UUID           NOT NULL,
    fixed_asset_id            UUID           NOT NULL,
    fiscal_period_ref         UUID           NOT NULL,
    depreciation_amount       NUMERIC(20,4)  NOT NULL DEFAULT 0,
    accumulated_depreciation  NUMERIC(20,4)  NOT NULL DEFAULT 0,
    net_book_value            NUMERIC(20,4)  NOT NULL DEFAULT 0,
    is_posted                 BOOLEAN        NOT NULL DEFAULT FALSE,
    gl_journal_ref            VARCHAR(50),
    run_date                  DATE           NOT NULL,
    created_at                TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by                UUID           NOT NULL,
    updated_at                TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by                UUID,
    deleted_at                TIMESTAMPTZ,
    row_version               INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_depreciation_entry PRIMARY KEY (id),
    CONSTRAINT uq_depreciation_entry UNIQUE (fixed_asset_id, fiscal_period_ref),
    CONSTRAINT fk_depreciation_entry_asset
        FOREIGN KEY (fixed_asset_id) REFERENCES fa.fixed_asset (id) ON DELETE CASCADE
);

-- =============================================================================
-- fa.asset_transaction (capitalization/revaluation/transfer/disposal/impairment)
-- =============================================================================
CREATE TABLE fa.asset_transaction (
    id                                UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id                   UUID           NOT NULL,
    fixed_asset_id                    UUID           NOT NULL,
    transaction_type                  VARCHAR(30)    NOT NULL,
    transaction_date                  DATE           NOT NULL,
    amount                            NUMERIC(20,4)  NOT NULL DEFAULT 0,
    gain_loss_amount                  NUMERIC(20,4),
    from_financial_organization_ref   UUID,
    to_financial_organization_ref     UUID,
    gl_journal_ref                    VARCHAR(50),
    notes                             TEXT,
    created_at                        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by                        UUID           NOT NULL,
    updated_at                        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by                        UUID,
    deleted_at                        TIMESTAMPTZ,
    row_version                       INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_asset_transaction PRIMARY KEY (id),
    CONSTRAINT fk_asset_transaction_asset
        FOREIGN KEY (fixed_asset_id) REFERENCES fa.fixed_asset (id) ON DELETE CASCADE,
    CONSTRAINT ck_asset_transaction_type
        CHECK (transaction_type IN ('Capitalization','Revaluation','Transfer','Disposal','Impairment'))
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_asset_category_active ON fa.asset_category (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_fixed_asset_category  ON fa.fixed_asset (asset_category_id);
CREATE INDEX idx_fixed_asset_finorg    ON fa.fixed_asset (financial_organization_ref);
CREATE INDEX idx_fixed_asset_parent    ON fa.fixed_asset (parent_asset_id);
CREATE INDEX idx_fixed_asset_status    ON fa.fixed_asset (organization_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_depreciation_entry_asset  ON fa.depreciation_entry (fixed_asset_id);
CREATE INDEX idx_depreciation_entry_period ON fa.depreciation_entry (fiscal_period_ref);
CREATE INDEX idx_asset_transaction_asset   ON fa.asset_transaction (fixed_asset_id);
CREATE INDEX idx_asset_transaction_type    ON fa.asset_transaction (organization_id, transaction_type);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE fa.asset_category      ENABLE ROW LEVEL SECURITY;
ALTER TABLE fa.fixed_asset         ENABLE ROW LEVEL SECURITY;
ALTER TABLE fa.depreciation_entry  ENABLE ROW LEVEL SECURITY;
ALTER TABLE fa.asset_transaction   ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON fa.asset_category     USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON fa.fixed_asset        USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON fa.depreciation_entry USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON fa.asset_transaction  USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON fa.asset_category     FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON fa.fixed_asset        FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON fa.depreciation_entry FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON fa.asset_transaction  FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE fa.outbox (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    aggregate_type   VARCHAR(100) NOT NULL,
    aggregate_id     UUID         NOT NULL,
    event_type       VARCHAR(150) NOT NULL,
    payload          JSONB        NOT NULL,
    organization_id  UUID         NOT NULL,
    correlation_id   UUID         NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    published_at     TIMESTAMPTZ,
    CONSTRAINT pk_outbox PRIMARY KEY (id)
);
CREATE INDEX idx_outbox_unpublished ON fa.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON fa.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON fa.outbox (correlation_id);

CREATE TABLE fa.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
