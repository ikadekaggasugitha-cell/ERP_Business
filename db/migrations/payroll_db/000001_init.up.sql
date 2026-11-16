-- =============================================================================
-- payroll_db — Payroll Management (SVC-017)
-- DBDD BAB 7.3/7.4 | Standards DBS-001..016 | BRD 9.17 | INT-003 (GL), INT-007 (BPJS)
-- Payroll posting to GL is Exactly-Once (ADR-005) via correlation_id + outbox.
-- gross/net amounts are sensitive PII (encrypted at app layer per BAB 7.4).
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

CREATE SCHEMA IF NOT EXISTS pay;

-- =============================================================================
-- pay.payroll_run
-- =============================================================================
CREATE TABLE pay.payroll_run (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    organization_id  UUID         NOT NULL,
    run_no           VARCHAR(50)  NOT NULL,
    period_month     INTEGER      NOT NULL,
    period_year      INTEGER      NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'Draft',
    gl_journal_ref   VARCHAR(50),                -- logical ref gl_db (INT-003)
    correlation_id   UUID         NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by       UUID         NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    row_version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pk_payroll_run PRIMARY KEY (id),
    CONSTRAINT uq_payroll_run_no UNIQUE (organization_id, run_no),
    CONSTRAINT ck_payroll_run_status CHECK (status IN ('Draft','Calculated','Approved','Posted','Paid')),
    CONSTRAINT ck_payroll_run_month CHECK (period_month BETWEEN 1 AND 12)
);

-- =============================================================================
-- pay.payroll_result  — employee_id logical ref (hr_db); amounts are PII
-- =============================================================================
CREATE TABLE pay.payroll_result (
    id                 UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id    UUID           NOT NULL,
    payroll_run_id     UUID           NOT NULL,
    employee_id        UUID           NOT NULL,   -- logical ref hr_db
    gross_amount       NUMERIC(20,4)  NOT NULL DEFAULT 0,   -- PII (sensitive)
    deduction_amount   NUMERIC(20,4)  NOT NULL DEFAULT 0,
    tax_amount         NUMERIC(20,4)  NOT NULL DEFAULT 0,   -- PPh 21
    net_amount         NUMERIC(20,4)  NOT NULL DEFAULT 0,   -- PII (sensitive)
    created_at         TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by         UUID           NOT NULL,
    updated_at         TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by         UUID,
    deleted_at         TIMESTAMPTZ,
    row_version        INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_payroll_result PRIMARY KEY (id),
    CONSTRAINT uq_payroll_result UNIQUE (payroll_run_id, employee_id),
    CONSTRAINT fk_payroll_result_run
        FOREIGN KEY (payroll_run_id) REFERENCES pay.payroll_run (id) ON DELETE CASCADE
);

-- =============================================================================
-- pay.payroll_result_line  (component breakdown)
-- =============================================================================
CREATE TABLE pay.payroll_result_line (
    id                  UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id     UUID           NOT NULL,
    payroll_result_id   UUID           NOT NULL,
    component_code      VARCHAR(30)    NOT NULL,   -- basic/allowance/deduction/...
    amount              NUMERIC(20,4)  NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by          UUID           NOT NULL,
    updated_at          TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by          UUID,
    deleted_at          TIMESTAMPTZ,
    row_version         INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_payroll_result_line PRIMARY KEY (id),
    CONSTRAINT uq_payroll_result_line UNIQUE (payroll_result_id, component_code),
    CONSTRAINT fk_payroll_result_line_result
        FOREIGN KEY (payroll_result_id) REFERENCES pay.payroll_result (id) ON DELETE CASCADE
);

-- =============================================================================
-- pay.statutory_contribution  (BPJS TK / Kesehatan — INT-007)
-- =============================================================================
CREATE TABLE pay.statutory_contribution (
    id                  UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id     UUID           NOT NULL,
    payroll_result_id   UUID           NOT NULL,
    contribution_type   VARCHAR(30)    NOT NULL,   -- BPJS-TK / BPJS-Kesehatan / ...
    employee_amount     NUMERIC(20,4)  NOT NULL DEFAULT 0,
    employer_amount     NUMERIC(20,4)  NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by          UUID           NOT NULL,
    updated_at          TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by          UUID,
    deleted_at          TIMESTAMPTZ,
    row_version         INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_statutory_contribution PRIMARY KEY (id),
    CONSTRAINT uq_statutory_contribution UNIQUE (payroll_result_id, contribution_type),
    CONSTRAINT fk_statutory_contribution_result
        FOREIGN KEY (payroll_result_id) REFERENCES pay.payroll_result (id) ON DELETE CASCADE
);

-- --- Indexes ----------------------------------------------------------------
CREATE INDEX idx_payroll_run_period    ON pay.payroll_run (organization_id, period_year, period_month);
CREATE INDEX idx_payroll_run_status    ON pay.payroll_run (organization_id, status);
CREATE INDEX idx_payroll_run_corr      ON pay.payroll_run (correlation_id);
CREATE INDEX idx_payroll_result_run    ON pay.payroll_result (payroll_run_id);
CREATE INDEX idx_payroll_result_emp    ON pay.payroll_result (employee_id);
CREATE INDEX idx_payroll_result_line   ON pay.payroll_result_line (payroll_result_id);
CREATE INDEX idx_statutory_contrib     ON pay.statutory_contribution (payroll_result_id);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE pay.payroll_run           ENABLE ROW LEVEL SECURITY;
ALTER TABLE pay.payroll_result        ENABLE ROW LEVEL SECURITY;
ALTER TABLE pay.payroll_result_line   ENABLE ROW LEVEL SECURITY;
ALTER TABLE pay.statutory_contribution ENABLE ROW LEVEL SECURITY;

CREATE POLICY scope_isolation ON pay.payroll_run            USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON pay.payroll_result         USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON pay.payroll_result_line    USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE POLICY scope_isolation ON pay.statutory_contribution USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Triggers ---------------------------------------------------------------
CREATE TRIGGER touch_row BEFORE UPDATE ON pay.payroll_run            FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON pay.payroll_result         FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON pay.payroll_result_line    FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
CREATE TRIGGER touch_row BEFORE UPDATE ON pay.statutory_contribution FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

-- =============================================================================
-- Outbox & Idempotency (DBS-014) — emits payroll.run.completed
-- =============================================================================
CREATE TABLE pay.outbox (
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
CREATE INDEX idx_outbox_unpublished ON pay.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON pay.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON pay.outbox (correlation_id);

CREATE TABLE pay.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
