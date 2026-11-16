-- =============================================================================
-- org_db — Organization Management (SVC-001)
-- DBDD BAB 3.2 | Standards DBS-001..016 | BRD 9.1 (BR-001..006)
-- golang-migrate: 000001_init.up.sql
-- =============================================================================

-- --- Extensions ------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- composite/GIN indexing

-- --- Shared helper functions (per-database) --------------------------------
-- DBS-008/010: maintain updated_at + optimistic-lock row_version on UPDATE.
CREATE OR REPLACE FUNCTION public.tg_touch_row()
RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at  := now();
  NEW.row_version := COALESCE(OLD.row_version, 0) + 1;
  RETURN NEW;
END;
$$;

-- DBS-011/012: resolve the caller's org scope (uuid[]) from session GUC.
-- Unset/empty -> NULL -> policy denies all rows (deny-by-default).
CREATE OR REPLACE FUNCTION public.current_org_scope()
RETURNS uuid[]
LANGUAGE sql STABLE AS $$
  SELECT NULLIF(current_setting('app.org_scope', true), '')::uuid[];
$$;

-- --- Schema -----------------------------------------------------------------
CREATE SCHEMA IF NOT EXISTS org;

-- =============================================================================
-- org.organization_type  (BR-002 configurable org types)
-- =============================================================================
CREATE TABLE org.organization_type (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(50)  NOT NULL,   -- Holding, Company, Branch, ...
    name             VARCHAR(150) NOT NULL,
    level_hint       INTEGER,                 -- indicative order (not a limiter)
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_organization_type PRIMARY KEY (id),
    CONSTRAINT uq_organization_type_org_code UNIQUE (organization_id, code)
);

-- =============================================================================
-- org.organization  (BR-001 unbounded tree, BR-003 parent-child, BR-004 lifecycle)
-- =============================================================================
CREATE TABLE org.organization (
    id                    UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id       UUID         NOT NULL,  -- owning tenant/holding
    code                  VARCHAR(50)  NOT NULL,  -- BR-005 unique per scope
    name                  VARCHAR(200) NOT NULL,  -- BR-006
    organization_type_id  UUID         NOT NULL,
    parent_id             UUID,                    -- NULL = root (BR-003)
    path                  TEXT         NOT NULL,   -- materialized path /root/sub/...
    status                VARCHAR(20)  NOT NULL DEFAULT 'Draft',
    effective_from        DATE         NOT NULL,   -- DBS-016
    effective_to          DATE,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by            UUID         NOT NULL,
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by            UUID,
    deleted_at            TIMESTAMPTZ,
    row_version           INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_organization PRIMARY KEY (id),
    CONSTRAINT uq_organization_org_code UNIQUE (organization_id, code),
    CONSTRAINT fk_organization_type
        FOREIGN KEY (organization_type_id) REFERENCES org.organization_type (id) ON DELETE RESTRICT,
    CONSTRAINT fk_organization_parent
        FOREIGN KEY (parent_id) REFERENCES org.organization (id) ON DELETE RESTRICT,
    CONSTRAINT ck_organization_status
        CHECK (status IN ('Draft','Active','Inactive','Merge','Split','Archived')),
    CONSTRAINT ck_organization_effective_range
        CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

-- =============================================================================
-- org.organization_closure  (fast ancestor/descendant traversal)
-- organization_id denormalized for RLS scope (DBS-011).
-- =============================================================================
CREATE TABLE org.organization_closure (
    organization_id  UUID    NOT NULL,
    ancestor_id      UUID    NOT NULL,
    descendant_id    UUID    NOT NULL,
    depth            INTEGER NOT NULL,          -- 0 = self
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by       UUID    NOT NULL,
    CONSTRAINT pk_organization_closure PRIMARY KEY (ancestor_id, descendant_id),
    CONSTRAINT fk_org_closure_ancestor
        FOREIGN KEY (ancestor_id)   REFERENCES org.organization (id) ON DELETE CASCADE,
    CONSTRAINT fk_org_closure_descendant
        FOREIGN KEY (descendant_id) REFERENCES org.organization (id) ON DELETE CASCADE,
    CONSTRAINT ck_org_closure_depth CHECK (depth >= 0)
);

-- =============================================================================
-- org.organization_attribute  (flexible per-type attributes)
-- =============================================================================
CREATE TABLE org.organization_attribute (
    id                    UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id       UUID         NOT NULL,
    organization_ref_id   UUID         NOT NULL,
    attr_key              VARCHAR(100) NOT NULL,
    attr_value            JSONB        NOT NULL,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by            UUID         NOT NULL,
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by            UUID,
    deleted_at            TIMESTAMPTZ,
    row_version           INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_organization_attribute PRIMARY KEY (id),
    CONSTRAINT uq_organization_attribute UNIQUE (organization_ref_id, attr_key),
    CONSTRAINT fk_org_attribute_org
        FOREIGN KEY (organization_ref_id) REFERENCES org.organization (id) ON DELETE CASCADE
);

-- --- Indexes (DBS-015) ------------------------------------------------------
CREATE INDEX idx_organization_type_active   ON org.organization_type (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_organization_parent        ON org.organization (parent_id);
CREATE INDEX idx_organization_type_ref      ON org.organization (organization_type_id);
CREATE INDEX idx_organization_path          ON org.organization USING btree (path text_pattern_ops);
CREATE INDEX idx_organization_active        ON org.organization (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_org_closure_descendant     ON org.organization_closure (descendant_id);
CREATE INDEX idx_org_attribute_value        ON org.organization_attribute USING gin (attr_value);

-- --- Row-Level Security (DBS-011/012) --------------------------------------
ALTER TABLE org.organization_type       ENABLE ROW LEVEL SECURITY;
ALTER TABLE org.organization            ENABLE ROW LEVEL SECURITY;
ALTER TABLE org.organization_closure    ENABLE ROW LEVEL SECURITY;
ALTER TABLE org.organization_attribute  ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON org.organization_type
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON org.organization
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON org.organization_closure
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON org.organization_attribute
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Row maintenance triggers (DBS-008/010) --------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON org.organization_type
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON org.organization
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON org.organization_attribute
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Transactional Outbox & Idempotency (DBS-014)
-- Technical infra tables: no RLS (accessed by relay/consumer workers).
-- =============================================================================
CREATE TABLE org.outbox (
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
CREATE INDEX idx_outbox_unpublished ON org.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON org.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON org.outbox (correlation_id);

CREATE TABLE org.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
