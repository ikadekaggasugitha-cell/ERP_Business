-- =============================================================================
-- security_db — Security Management (SVC-003)
-- DBDD BAB 3.4 | Standards DBS-001..016 | BRD 9.4 (BR-031..040) — RBAC + Scope
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

CREATE SCHEMA IF NOT EXISTS sec;

-- =============================================================================
-- sec.role
-- =============================================================================
CREATE TABLE sec.role (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(50)  NOT NULL,
    name             VARCHAR(150) NOT NULL,
    is_system        BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_role PRIMARY KEY (id),
    CONSTRAINT uq_role_org_code UNIQUE (organization_id, code)
);

-- =============================================================================
-- sec.permission  (catalogue; global entries may be seeded under a system org)
-- =============================================================================
CREATE TABLE sec.permission (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(100) NOT NULL,   -- e.g. gl.journal.post
    module           VARCHAR(50)  NOT NULL,
    action           VARCHAR(50)  NOT NULL,   -- create/read/update/delete/approve
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_permission PRIMARY KEY (id),
    CONSTRAINT uq_permission_org_code UNIQUE (organization_id, code)
);

-- =============================================================================
-- sec.role_permission  (junction)
-- =============================================================================
CREATE TABLE sec.role_permission (
    organization_id  UUID        NOT NULL,
    role_id          UUID        NOT NULL,
    permission_id    UUID        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by       UUID        NOT NULL,
    CONSTRAINT pk_role_permission PRIMARY KEY (role_id, permission_id),
    CONSTRAINT fk_role_permission_role
        FOREIGN KEY (role_id)       REFERENCES sec.role (id)       ON DELETE CASCADE,
    CONSTRAINT fk_role_permission_perm
        FOREIGN KEY (permission_id) REFERENCES sec.permission (id) ON DELETE CASCADE
);

-- =============================================================================
-- sec.role_scope  (Scope-Based Access — BR-031)
-- organization_ref_id is a logical ref to org_db.
-- =============================================================================
CREATE TABLE sec.role_scope (
    id                   UUID        NOT NULL DEFAULT gen_random_uuid(),
    organization_id      UUID        NOT NULL,
    role_id              UUID        NOT NULL,
    scope_type           VARCHAR(30) NOT NULL,   -- organization/branch/self/all
    organization_ref_id  UUID,                   -- specific scope (nullable)
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by           UUID        NOT NULL,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by           UUID,
    deleted_at           TIMESTAMPTZ,
    row_version          INTEGER     NOT NULL DEFAULT 1,
    CONSTRAINT pk_role_scope PRIMARY KEY (id),
    CONSTRAINT fk_role_scope_role
        FOREIGN KEY (role_id) REFERENCES sec.role (id) ON DELETE CASCADE,
    CONSTRAINT ck_role_scope_type
        CHECK (scope_type IN ('organization','branch','self','all'))
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_role_active        ON sec.role (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_permission_module  ON sec.permission (module);
CREATE INDEX idx_role_permission_perm ON sec.role_permission (permission_id);
CREATE INDEX idx_role_scope_role    ON sec.role_scope (role_id);
CREATE INDEX idx_role_scope_orgref  ON sec.role_scope (organization_ref_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE sec.role             ENABLE ROW LEVEL SECURITY;
ALTER TABLE sec.permission       ENABLE ROW LEVEL SECURITY;
ALTER TABLE sec.role_permission  ENABLE ROW LEVEL SECURITY;
ALTER TABLE sec.role_scope       ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON sec.role
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON sec.permission
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON sec.role_permission
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON sec.role_scope
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON sec.role
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON sec.permission
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON sec.role_scope
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE sec.outbox (
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
CREATE INDEX idx_outbox_unpublished ON sec.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON sec.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON sec.outbox (correlation_id);

CREATE TABLE sec.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
