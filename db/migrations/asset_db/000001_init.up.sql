-- =============================================================================
-- asset_db — Asset Management (operational) (SVC-026)
-- DBDD BAB 9.8 | Standards DBS-001..016 | BRD 9.11 (BR-736..745)
-- Operational counterpart of fixedasset_db (fixed_asset_ref = logical ref).
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

CREATE SCHEMA IF NOT EXISTS ast;

-- =============================================================================
-- ast.asset (BR-736/742)
-- =============================================================================
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
    CONSTRAINT pk_asset PRIMARY KEY (id),
    CONSTRAINT uq_asset_code UNIQUE (organization_id, asset_code),
    CONSTRAINT ck_asset_status CHECK (status IN ('Active','UnderMaintenance','Idle','Damaged','Disposed'))
);

-- =============================================================================
-- ast.asset_location_history (BR-737)
-- =============================================================================
CREATE TABLE ast.asset_location_history (
    id                UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID         NOT NULL,
    asset_id          UUID         NOT NULL,
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
    CONSTRAINT pk_asset_location_history PRIMARY KEY (id),
    CONSTRAINT fk_asset_location_history_asset
        FOREIGN KEY (asset_id) REFERENCES ast.asset (id) ON DELETE CASCADE
);

-- =============================================================================
-- ast.asset_movement (BR-741)
-- =============================================================================
CREATE TABLE ast.asset_movement (
    id                       UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id          UUID         NOT NULL,
    asset_id                 UUID         NOT NULL,
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
    CONSTRAINT pk_asset_movement PRIMARY KEY (id),
    CONSTRAINT fk_asset_movement_asset
        FOREIGN KEY (asset_id) REFERENCES ast.asset (id) ON DELETE CASCADE,
    CONSTRAINT ck_asset_movement_status CHECK (status IN ('Draft','Approved','Completed'))
);

-- =============================================================================
-- ast.asset_verification (BR-745)
-- =============================================================================
CREATE TABLE ast.asset_verification (
    id                 UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id    UUID         NOT NULL,
    asset_id           UUID         NOT NULL,
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
    CONSTRAINT pk_asset_verification PRIMARY KEY (id),
    CONSTRAINT fk_asset_verification_asset
        FOREIGN KEY (asset_id) REFERENCES ast.asset (id) ON DELETE CASCADE
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_asset_status        ON ast.asset (organization_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_asset_orgref        ON ast.asset (organization_ref);
CREATE INDEX idx_asset_fixed_ref     ON ast.asset (fixed_asset_ref);
CREATE INDEX idx_asset_loc_hist_asset ON ast.asset_location_history (asset_id);
CREATE INDEX idx_asset_movement_asset ON ast.asset_movement (asset_id);
CREATE INDEX idx_asset_verification_asset ON ast.asset_verification (asset_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE ast.asset                  ENABLE ROW LEVEL SECURITY;
ALTER TABLE ast.asset_location_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE ast.asset_movement         ENABLE ROW LEVEL SECURITY;
ALTER TABLE ast.asset_verification     ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON ast.asset                  USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON ast.asset_location_history USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON ast.asset_movement         USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON ast.asset_verification     USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON ast.asset                  FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON ast.asset_location_history FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON ast.asset_movement         FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON ast.asset_verification     FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
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
    CONSTRAINT pk_outbox PRIMARY KEY (id)
);
CREATE INDEX idx_outbox_unpublished ON ast.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON ast.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON ast.outbox (correlation_id);

CREATE TABLE ast.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
