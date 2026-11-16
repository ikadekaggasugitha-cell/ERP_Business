-- =============================================================================
-- localization_db — Localization & Multi-Language (SVC-029)
-- DBDD BAB 9.11 | Standards DBS-001..016 | BRD 9.19 (BR-816..825)
-- Configuration-driven: adding languages/translations requires no code change.
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

CREATE SCHEMA IF NOT EXISTS loc;

-- =============================================================================
-- loc.language (BR-816/823/824)
-- =============================================================================
CREATE TABLE loc.language (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    code             VARCHAR(10)  NOT NULL,   -- e.g. en, id
    name             VARCHAR(100) NOT NULL,
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    is_fallback      BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_language PRIMARY KEY (id),
    CONSTRAINT uq_language_org_code UNIQUE (organization_id, code)
);

-- =============================================================================
-- loc.translation (BR-817)
-- =============================================================================
CREATE TABLE loc.translation (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    language_code    VARCHAR(10)  NOT NULL,
    resource_key     VARCHAR(200) NOT NULL,
    resource_value   TEXT         NOT NULL,
    module           VARCHAR(50),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_translation PRIMARY KEY (id),
    CONSTRAINT uq_translation UNIQUE (organization_id, language_code, resource_key)
);

-- =============================================================================
-- loc.locale_format (BR-820)
-- =============================================================================
CREATE TABLE loc.locale_format (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    locale_code      VARCHAR(20)  NOT NULL,   -- e.g. id-ID, en-US
    date_format      VARCHAR(50)  NOT NULL,
    number_format    VARCHAR(50)  NOT NULL,
    currency_format  VARCHAR(50)  NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_locale_format PRIMARY KEY (id),
    CONSTRAINT uq_locale_format UNIQUE (organization_id, locale_code)
);

-- =============================================================================
-- loc.user_language_preference (BR-818) — user_ref logical ref (user_db)
-- =============================================================================
CREATE TABLE loc.user_language_preference (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    user_ref         UUID         NOT NULL,
    language_code    VARCHAR(10)  NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_user_language_preference PRIMARY KEY (id),
    CONSTRAINT uq_user_language_preference UNIQUE (organization_id, user_ref)
);

-- =============================================================================
-- loc.organization_language (BR-819)
-- =============================================================================
CREATE TABLE loc.organization_language (
    id                     UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id        UUID         NOT NULL,
    organization_ref       UUID         NOT NULL,
    default_language_code  VARCHAR(10)  NOT NULL,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by             UUID         NOT NULL,
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by             UUID,
    deleted_at             TIMESTAMPTZ,
    row_version            INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_organization_language PRIMARY KEY (id),
    CONSTRAINT uq_organization_language UNIQUE (organization_id, organization_ref)
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_language_active   ON loc.language (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_translation_lang  ON loc.translation (organization_id, language_code);
CREATE INDEX idx_translation_key   ON loc.translation (resource_key);
CREATE INDEX idx_user_lang_pref_user ON loc.user_language_preference (user_ref);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE loc.language                 ENABLE ROW LEVEL SECURITY;
ALTER TABLE loc.translation              ENABLE ROW LEVEL SECURITY;
ALTER TABLE loc.locale_format            ENABLE ROW LEVEL SECURITY;
ALTER TABLE loc.user_language_preference ENABLE ROW LEVEL SECURITY;
ALTER TABLE loc.organization_language    ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON loc.language                 USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON loc.translation              USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON loc.locale_format            USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON loc.user_language_preference USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON loc.organization_language    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON loc.language                 FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON loc.translation              FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON loc.locale_format            FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON loc.user_language_preference FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON loc.organization_language    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014)
-- =============================================================================
CREATE TABLE loc.outbox (
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
CREATE INDEX idx_outbox_unpublished ON loc.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON loc.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON loc.outbox (correlation_id);

CREATE TABLE loc.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
