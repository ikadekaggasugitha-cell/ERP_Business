-- =============================================================================
-- user_db — User Management (SVC-002)
-- DBDD BAB 3.3 | Standards DBS-001..016 | BRD 9.2, INT-012
-- No passwords stored — authentication delegated to IdP (SAD BAB 7.3).
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

CREATE SCHEMA IF NOT EXISTS usr;

-- =============================================================================
-- usr.app_user
-- =============================================================================
CREATE TABLE usr.app_user (
    id                       UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id          UUID         NOT NULL,
    username                 VARCHAR(100) NOT NULL,
    email                    VARCHAR(255) NOT NULL,
    full_name                VARCHAR(200) NOT NULL,
    status                   VARCHAR(20)  NOT NULL DEFAULT 'Active',
    default_organization_id  UUID,        -- logical ref -> org_db (no physical FK)
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by               UUID         NOT NULL,
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by               UUID,
    deleted_at               TIMESTAMPTZ,
    row_version              INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_app_user PRIMARY KEY (id),
    CONSTRAINT uq_app_user_username UNIQUE (username),
    CONSTRAINT uq_app_user_email    UNIQUE (email),
    CONSTRAINT ck_app_user_status   CHECK (status IN ('Active','Suspended','Inactive'))
);

-- =============================================================================
-- usr.user_identity  (IdP federation / SSO — INT-012)
-- =============================================================================
CREATE TABLE usr.user_identity (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    user_id          UUID         NOT NULL,
    provider         VARCHAR(50)  NOT NULL,   -- azure-ad / google / ldap
    subject          VARCHAR(255) NOT NULL,   -- IdP subject/claim
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_user_identity PRIMARY KEY (id),
    CONSTRAINT uq_user_identity_subject UNIQUE (provider, subject),
    CONSTRAINT fk_user_identity_user
        FOREIGN KEY (user_id) REFERENCES usr.app_user (id) ON DELETE CASCADE
);

-- =============================================================================
-- usr.user_org_assignment  (scope-based access)
-- role_id / organization_ref_id are logical refs (security_db / org_db).
-- =============================================================================
CREATE TABLE usr.user_org_assignment (
    id                   UUID        NOT NULL DEFAULT gen_random_uuid(),
    organization_id      UUID        NOT NULL,
    user_id              UUID        NOT NULL,
    organization_ref_id  UUID        NOT NULL,
    role_id              UUID        NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by           UUID        NOT NULL,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by           UUID,
    deleted_at           TIMESTAMPTZ,
    row_version          INTEGER     NOT NULL DEFAULT 1,
    CONSTRAINT pk_user_org_assignment PRIMARY KEY (id),
    CONSTRAINT uq_user_org_assignment UNIQUE (user_id, organization_ref_id, role_id),
    CONSTRAINT fk_user_org_assignment_user
        FOREIGN KEY (user_id) REFERENCES usr.app_user (id) ON DELETE CASCADE
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_app_user_active         ON usr.app_user (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_user_identity_user      ON usr.user_identity (user_id);
CREATE INDEX idx_user_org_assign_user    ON usr.user_org_assignment (user_id);
CREATE INDEX idx_user_org_assign_orgref  ON usr.user_org_assignment (organization_ref_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE usr.app_user             ENABLE ROW LEVEL SECURITY;
ALTER TABLE usr.user_identity        ENABLE ROW LEVEL SECURITY;
ALTER TABLE usr.user_org_assignment  ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON usr.app_user
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON usr.user_identity
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON usr.user_org_assignment
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON usr.app_user
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON usr.user_identity
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON usr.user_org_assignment
    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE usr.outbox (
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
CREATE INDEX idx_outbox_unpublished ON usr.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON usr.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON usr.outbox (correlation_id);

CREATE TABLE usr.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
