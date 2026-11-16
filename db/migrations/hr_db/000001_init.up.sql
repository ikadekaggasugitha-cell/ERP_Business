-- =============================================================================
-- hr_db — Human Resource Management (SVC-016)
-- DBDD BAB 7.2/7.4 | Standards DBS-001..016 | BRD 9.12 | INT-003
-- PII columns (full_name, national_id, date_of_birth, email, amount) are
-- classified per BAB 7.4. Field-level encryption is applied at the APPLICATION
-- layer (pgcrypto / app crypto) — column types remain as documented so the
-- ciphertext round-trips through the API contract. RLS restricts cross-scope
-- access; PII access is audited (audit_db).
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

CREATE SCHEMA IF NOT EXISTS hr;

-- =============================================================================
-- hr.employee  — user_ref_id logical ref (user_db)
-- =============================================================================
CREATE TABLE hr.employee (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    employee_no      VARCHAR(50)  NOT NULL,
    full_name        VARCHAR(200) NOT NULL,   -- PII
    national_id      VARCHAR(50),             -- PII (encrypted at app layer)
    date_of_birth    DATE,                    -- PII
    email            VARCHAR(255),            -- PII
    hire_date        DATE         NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'Active',
    user_ref_id      UUID,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_employee PRIMARY KEY (id),
    CONSTRAINT uq_employee_no UNIQUE (organization_id, employee_no),
    CONSTRAINT uq_employee_national_id UNIQUE (organization_id, national_id),
    CONSTRAINT ck_employee_status CHECK (status IN ('Active','OnLeave','Terminated'))
);

-- =============================================================================
-- hr.position  — organization_ref_id logical ref (org_db); parent self-FK
-- =============================================================================
CREATE TABLE hr.position (
    id                    UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id       UUID         NOT NULL,
    code                  VARCHAR(50)  NOT NULL,
    title                 VARCHAR(150) NOT NULL,
    organization_ref_id   UUID         NOT NULL,
    parent_position_id    UUID,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by            UUID         NOT NULL,
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by            UUID,
    deleted_at            TIMESTAMPTZ,
    row_version           INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_position PRIMARY KEY (id),
    CONSTRAINT uq_position_code UNIQUE (organization_id, code),
    CONSTRAINT fk_position_parent
        FOREIGN KEY (parent_position_id) REFERENCES hr.position (id) ON DELETE RESTRICT
);

-- =============================================================================
-- hr.employment  (effective placement — DBS-016 temporal)
-- =============================================================================
CREATE TABLE hr.employment (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    employee_id      UUID         NOT NULL,
    position_id      UUID         NOT NULL,
    employment_type  VARCHAR(20)  NOT NULL,   -- Permanent/Contract/Outsource
    effective_from   DATE         NOT NULL,
    effective_to     DATE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_employment PRIMARY KEY (id),
    CONSTRAINT fk_employment_employee
        FOREIGN KEY (employee_id) REFERENCES hr.employee (id) ON DELETE CASCADE,
    CONSTRAINT fk_employment_position
        FOREIGN KEY (position_id) REFERENCES hr.position (id) ON DELETE RESTRICT,
    CONSTRAINT ck_employment_type CHECK (employment_type IN ('Permanent','Contract','Outsource')),
    CONSTRAINT ck_employment_range CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

-- =============================================================================
-- hr.employee_compensation  (sensitive PII — amount encrypted at app layer)
-- =============================================================================
CREATE TABLE hr.employee_compensation (
    id               UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID           NOT NULL,
    employee_id      UUID           NOT NULL,
    component_code   VARCHAR(30)    NOT NULL,   -- basic/allowance/deduction
    amount           NUMERIC(20,4)  NOT NULL DEFAULT 0,   -- PII (sensitive)
    effective_from   DATE           NOT NULL,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by       UUID           NOT NULL,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_employee_compensation PRIMARY KEY (id),
    CONSTRAINT fk_employee_compensation_employee
        FOREIGN KEY (employee_id) REFERENCES hr.employee (id) ON DELETE CASCADE
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_employee_status      ON hr.employee (organization_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_position_orgref      ON hr.position (organization_ref_id);
CREATE INDEX idx_position_parent      ON hr.position (parent_position_id);
CREATE INDEX idx_employment_employee  ON hr.employment (employee_id);
CREATE INDEX idx_employment_position  ON hr.employment (position_id);
CREATE INDEX idx_emp_comp_employee    ON hr.employee_compensation (employee_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE hr.employee              ENABLE ROW LEVEL SECURITY;
ALTER TABLE hr.position              ENABLE ROW LEVEL SECURITY;
ALTER TABLE hr.employment            ENABLE ROW LEVEL SECURITY;
ALTER TABLE hr.employee_compensation ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON hr.employee              USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON hr.position              USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON hr.employment            USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON hr.employee_compensation USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON hr.employee              FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON hr.position              FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON hr.employment            FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON hr.employee_compensation FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014) — emits hr.employee.updated
-- =============================================================================
CREATE TABLE hr.outbox (
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
CREATE INDEX idx_outbox_unpublished ON hr.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON hr.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON hr.outbox (correlation_id);

CREATE TABLE hr.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
