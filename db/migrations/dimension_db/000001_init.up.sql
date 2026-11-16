-- =============================================================================
-- dimension_db — Cost & Profit Center (SVC-021)
-- DBDD BAB 9.3 | Standards DBS-001..016 | BRD 9.5.10/9.5.11 (BR-616..635)
-- Schemas: cc (Cost Center), pc (Profit Center).
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

CREATE SCHEMA IF NOT EXISTS cc;
CREATE SCHEMA IF NOT EXISTS pc;

-- =============================================================================
-- cc.cost_center (BR-616/617/621/622)
-- =============================================================================
CREATE TABLE cc.cost_center (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(50)  NOT NULL,
    name             VARCHAR(150) NOT NULL,
    description      TEXT,
    parent_id        UUID,
    owner_ref        UUID,
    status           VARCHAR(20)  NOT NULL DEFAULT 'Draft',
    effective_from   DATE         NOT NULL,
    effective_to     DATE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_cost_center PRIMARY KEY (id),
    CONSTRAINT uq_cost_center_org_code UNIQUE (organization_id, code),
    CONSTRAINT fk_cost_center_parent
        FOREIGN KEY (parent_id) REFERENCES cc.cost_center (id) ON DELETE RESTRICT,
    CONSTRAINT ck_cost_center_status CHECK (status IN ('Draft','Active','Inactive','Closed')),
    CONSTRAINT ck_cost_center_effective CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE TABLE cc.cost_center_assignment (
    id               UUID        NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID        NOT NULL,
    cost_center_id   UUID        NOT NULL,
    organization_ref UUID        NOT NULL,   -- logical ref org_db
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by       UUID        NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER     NOT NULL DEFAULT 1,
    CONSTRAINT pk_cost_center_assignment PRIMARY KEY (id),
    CONSTRAINT uq_cost_center_assignment UNIQUE (cost_center_id, organization_ref),
    CONSTRAINT fk_cca_cost_center
        FOREIGN KEY (cost_center_id) REFERENCES cc.cost_center (id) ON DELETE CASCADE
);

CREATE TABLE cc.cost_allocation_rule (
    id                     UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id        UUID         NOT NULL,
    source_cost_center_id  UUID         NOT NULL,
    name                   VARCHAR(150) NOT NULL,
    allocation_basis       VARCHAR(50)  NOT NULL,
    targets                JSONB        NOT NULL,   -- [{cost_center_id, percentage}]
    is_active              BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by             UUID         NOT NULL,
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by             UUID,
    deleted_at             TIMESTAMPTZ,
    row_version            INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_cost_allocation_rule PRIMARY KEY (id),
    CONSTRAINT fk_car_cost_center
        FOREIGN KEY (source_cost_center_id) REFERENCES cc.cost_center (id) ON DELETE CASCADE
);

-- =============================================================================
-- pc.profit_center (BR-626/627/635)
-- =============================================================================
CREATE TABLE pc.profit_center (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(50)  NOT NULL,
    name             VARCHAR(150) NOT NULL,
    parent_id        UUID,
    status           VARCHAR(20)  NOT NULL DEFAULT 'Draft',
    effective_from   DATE         NOT NULL,
    effective_to     DATE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_profit_center PRIMARY KEY (id),
    CONSTRAINT uq_profit_center_org_code UNIQUE (organization_id, code),
    CONSTRAINT fk_profit_center_parent
        FOREIGN KEY (parent_id) REFERENCES pc.profit_center (id) ON DELETE RESTRICT,
    CONSTRAINT ck_profit_center_status CHECK (status IN ('Draft','Active','Inactive','Closed')),
    CONSTRAINT ck_profit_center_effective CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE TABLE pc.profit_center_assignment (
    id                 UUID        NOT NULL DEFAULT gen_random_uuid(),
    organization_id    UUID        NOT NULL,
    profit_center_id   UUID        NOT NULL,
    organization_ref   UUID        NOT NULL,   -- logical ref org_db
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by         UUID        NOT NULL,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by         UUID,
    deleted_at         TIMESTAMPTZ,
    row_version        INTEGER     NOT NULL DEFAULT 1,
    CONSTRAINT pk_profit_center_assignment PRIMARY KEY (id),
    CONSTRAINT uq_profit_center_assignment UNIQUE (profit_center_id, organization_ref),
    CONSTRAINT fk_pca_profit_center
        FOREIGN KEY (profit_center_id) REFERENCES pc.profit_center (id) ON DELETE CASCADE
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_cost_center_parent   ON cc.cost_center (parent_id);
CREATE INDEX idx_cost_center_status   ON cc.cost_center (organization_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_cca_cost_center      ON cc.cost_center_assignment (cost_center_id);
CREATE INDEX idx_cca_orgref           ON cc.cost_center_assignment (organization_ref);
CREATE INDEX idx_car_source           ON cc.cost_allocation_rule (source_cost_center_id);
CREATE INDEX idx_car_targets          ON cc.cost_allocation_rule USING gin (targets);
CREATE INDEX idx_profit_center_parent ON pc.profit_center (parent_id);
CREATE INDEX idx_profit_center_status ON pc.profit_center (organization_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_pca_profit_center    ON pc.profit_center_assignment (profit_center_id);
CREATE INDEX idx_pca_orgref           ON pc.profit_center_assignment (organization_ref);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE cc.cost_center            ENABLE ROW LEVEL SECURITY;
ALTER TABLE cc.cost_center_assignment ENABLE ROW LEVEL SECURITY;
ALTER TABLE cc.cost_allocation_rule   ENABLE ROW LEVEL SECURITY;
ALTER TABLE pc.profit_center          ENABLE ROW LEVEL SECURITY;
ALTER TABLE pc.profit_center_assignment ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON cc.cost_center             USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON cc.cost_center_assignment  USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON cc.cost_allocation_rule    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON pc.profit_center           USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON pc.profit_center_assignment USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON cc.cost_center             FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON cc.cost_center_assignment  FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON cc.cost_allocation_rule    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON pc.profit_center           FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON pc.profit_center_assignment FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE cc.outbox (
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
CREATE INDEX idx_outbox_unpublished ON cc.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON cc.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON cc.outbox (correlation_id);

CREATE TABLE cc.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
