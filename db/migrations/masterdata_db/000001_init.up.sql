-- =============================================================================
-- masterdata_db — Master Data Management (SVC-005)
-- DBDD BAB 3.6 | Standards DBS-001..016 | BRD 9.18, BO-019 (shared/isolated)
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

CREATE SCHEMA IF NOT EXISTS md;

-- =============================================================================
-- md.party  (Customer/Vendor super-type)
-- =============================================================================
CREATE TABLE md.party (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(50)  NOT NULL,
    name             VARCHAR(200) NOT NULL,
    party_type       VARCHAR(20)  NOT NULL,   -- customer/vendor/both
    tax_id           VARCHAR(50),             -- NPWP / tax identity
    is_shared        BOOLEAN      NOT NULL DEFAULT FALSE,   -- BO-019
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_party PRIMARY KEY (id),
    CONSTRAINT uq_party_org_code UNIQUE (organization_id, code),
    CONSTRAINT ck_party_type CHECK (party_type IN ('customer','vendor','both'))
);

-- =============================================================================
-- md.party_address  (standard pattern — domain columns + common columns)
-- =============================================================================
CREATE TABLE md.party_address (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    party_id         UUID         NOT NULL,
    address_type     VARCHAR(30)  NOT NULL DEFAULT 'billing',  -- billing/shipping/...
    line1            VARCHAR(200) NOT NULL,
    line2            VARCHAR(200),
    city             VARCHAR(100),
    state_province   VARCHAR(100),
    postal_code      VARCHAR(20),
    country_code     VARCHAR(2),              -- ISO 3166-1 alpha-2
    is_primary       BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_party_address PRIMARY KEY (id),
    CONSTRAINT fk_party_address_party
        FOREIGN KEY (party_id) REFERENCES md.party (id) ON DELETE CASCADE
);

-- =============================================================================
-- md.item
-- =============================================================================
CREATE TABLE md.item (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(50)  NOT NULL,
    name             VARCHAR(200) NOT NULL,
    item_type        VARCHAR(30)  NOT NULL,   -- stock/service/asset
    base_uom         VARCHAR(20)  NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_item PRIMARY KEY (id),
    CONSTRAINT uq_item_org_code UNIQUE (organization_id, code),
    CONSTRAINT ck_item_type CHECK (item_type IN ('stock','service','asset'))
);

-- =============================================================================
-- md.item_uom  (unit conversions — standard pattern)
-- =============================================================================
CREATE TABLE md.item_uom (
    id                UUID          NOT NULL DEFAULT gen_random_uuid(),
    organization_id   UUID          NOT NULL,
    item_id           UUID          NOT NULL,
    uom_code          VARCHAR(20)   NOT NULL,
    conversion_to_base NUMERIC(20,4) NOT NULL DEFAULT 1,   -- factor to base_uom
    is_base           BOOLEAN       NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT now(),
    created_by        UUID          NOT NULL,
    updated_at        TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_by        UUID,
    deleted_at        TIMESTAMPTZ,
    row_version       INTEGER       NOT NULL DEFAULT 1,
    CONSTRAINT pk_item_uom PRIMARY KEY (id),
    CONSTRAINT uq_item_uom UNIQUE (item_id, uom_code),
    CONSTRAINT fk_item_uom_item
        FOREIGN KEY (item_id) REFERENCES md.item (id) ON DELETE CASCADE,
    CONSTRAINT ck_item_uom_conv CHECK (conversion_to_base > 0)
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_party_active        ON md.party (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_party_type          ON md.party (organization_id, party_type);
CREATE INDEX idx_party_address_party ON md.party_address (party_id);
CREATE INDEX idx_item_active         ON md.item (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_item_uom_item       ON md.item_uom (item_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE md.party          ENABLE ROW LEVEL SECURITY;
ALTER TABLE md.party_address  ENABLE ROW LEVEL SECURITY;
ALTER TABLE md.item           ENABLE ROW LEVEL SECURITY;
ALTER TABLE md.item_uom       ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON md.party
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON md.party_address
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON md.item
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON md.item_uom
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON md.party
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON md.party_address
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON md.item
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON md.item_uom
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014) — Published Language producer (SAD 6.7)
-- =============================================================================
CREATE TABLE md.outbox (
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
CREATE INDEX idx_outbox_unpublished ON md.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON md.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON md.outbox (correlation_id);

CREATE TABLE md.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
