-- =============================================================================
-- currency_db — Currency Management (currency-service)
-- DBDD BAB 4.4 | Standards DBS-001..016 | BRD 9.5.3
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

CREATE SCHEMA IF NOT EXISTS cur;

-- =============================================================================
-- cur.currency  (ISO 4217)
-- =============================================================================
CREATE TABLE cur.currency (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(3)   NOT NULL,   -- ISO 4217 alpha
    numeric_code     VARCHAR(3),              -- ISO 4217 numeric
    name             VARCHAR(100) NOT NULL,
    precision        INTEGER      NOT NULL DEFAULT 2,
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_currency PRIMARY KEY (id),
    CONSTRAINT uq_currency_org_code UNIQUE (organization_id, code),
    CONSTRAINT ck_currency_precision CHECK (precision >= 0)
);

-- =============================================================================
-- cur.exchange_rate_type  (Spot/Historical/Budget/Average/Closing)
-- =============================================================================
CREATE TABLE cur.exchange_rate_type (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(30)  NOT NULL,
    name             VARCHAR(100) NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_exchange_rate_type PRIMARY KEY (id),
    CONSTRAINT uq_exchange_rate_type_org_code UNIQUE (organization_id, code)
);

-- =============================================================================
-- cur.exchange_rate  (dated rates — DBS-016 effectivity)
-- from_currency / to_currency reference cur.currency by code (logical, no FK
-- so historical rates survive currency deactivation).
-- =============================================================================
CREATE TABLE cur.exchange_rate (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    from_currency    VARCHAR(3)     NOT NULL,
    to_currency      VARCHAR(3)     NOT NULL,
    rate_type_id     UUID           NOT NULL,
    rate             NUMERIC(20,8)  NOT NULL,
    valid_from       DATE           NOT NULL,
    valid_to         DATE,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_exchange_rate PRIMARY KEY (id),
    CONSTRAINT uq_exchange_rate UNIQUE (organization_id, from_currency, to_currency, rate_type_id, valid_from),
    CONSTRAINT fk_exchange_rate_type
        FOREIGN KEY (rate_type_id) REFERENCES cur.exchange_rate_type (id) ON DELETE RESTRICT,
    CONSTRAINT ck_exchange_rate_positive CHECK (rate > 0),
    CONSTRAINT ck_exchange_rate_range CHECK (valid_to IS NULL OR valid_to >= valid_from)
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_currency_active     ON cur.currency (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_exchange_rate_pair  ON cur.exchange_rate (organization_id, from_currency, to_currency, valid_from);
CREATE INDEX idx_exchange_rate_type  ON cur.exchange_rate (rate_type_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE cur.currency            ENABLE ROW LEVEL SECURITY;
ALTER TABLE cur.exchange_rate_type  ENABLE ROW LEVEL SECURITY;
ALTER TABLE cur.exchange_rate       ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON cur.currency
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON cur.exchange_rate_type
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON cur.exchange_rate
    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON cur.currency
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON cur.exchange_rate_type
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON cur.exchange_rate
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE cur.outbox (
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
CREATE INDEX idx_outbox_unpublished ON cur.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON cur.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON cur.outbox (correlation_id);

CREATE TABLE cur.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
