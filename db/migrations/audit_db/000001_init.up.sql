-- =============================================================================
-- audit_db — Audit Trail (SVC-018)
-- DBDD BAB 3.7 | Standards DBS-001..016 | BRD 9.20 (BR-826..835)
-- Append-only, immutable, RANGE-partitioned by occurred_at (DBS-013).
-- Consumer of all domain events -> processed_event only (no outbox).
-- golang-migrate: 000001_init.up.sql
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS btree_gin;

CREATE OR REPLACE FUNCTION public.current_org_scope()
RETURNS uuid[] LANGUAGE sql STABLE AS $$
  SELECT NULLIF(current_setting('app.org_scope', true), '')::uuid[];
$$;

-- Immutability guard (BR — audit is append-only; no UPDATE/DELETE of rows).
CREATE OR REPLACE FUNCTION public.tg_immutable()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'audit_log is append-only and immutable';
END;
$$;

CREATE SCHEMA IF NOT EXISTS audit;

-- =============================================================================
-- audit.audit_log  (PARTITION BY RANGE (occurred_at))
-- PK includes the partition key (PostgreSQL requirement).
-- =============================================================================
CREATE TABLE audit.audit_log (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    occurred_at      TIMESTAMPTZ  NOT NULL,          -- partition key
    organization_id  UUID         NOT NULL,
    actor_id         UUID,                            -- acting user (logical ref)
    action           VARCHAR(50)  NOT NULL,           -- create/update/delete/login/...
    entity_type      VARCHAR(100) NOT NULL,
    entity_id        UUID,
    before_value     JSONB,                           -- BR-828
    after_value      JSONB,
    reason           TEXT,
    correlation_id   UUID         NOT NULL,
    CONSTRAINT pk_audit_log PRIMARY KEY (id, occurred_at)
) PARTITION BY RANGE (occurred_at);

-- --- Indexes on partitioned parent (inherited by every partition) ----------
CREATE INDEX idx_audit_log_org         ON audit.audit_log (organization_id, occurred_at);
CREATE INDEX idx_audit_log_entity      ON audit.audit_log (entity_type, entity_id);
CREATE INDEX idx_audit_log_correlation ON audit.audit_log (correlation_id);
CREATE INDEX idx_audit_log_occurred_brin ON audit.audit_log USING brin (occurred_at);

-- --- RLS --------------------------------------------------------------------
ALTER TABLE audit.audit_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY scope_isolation ON audit.audit_log
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));

-- --- Immutability -----------------------------------------------------------
-- Row-level UPDATE/DELETE blocked; partition DETACH (archiving) still allowed.
CREATE TRIGGER immutable_guard BEFORE UPDATE OR DELETE ON audit.audit_log
    FOR EACH ROW EXECUTE FUNCTION public.tg_immutable();

-- =============================================================================
-- audit.processed_event (idempotency — audit consumes every domain event)
-- =============================================================================
CREATE TABLE audit.processed_event (
    event_id      UUID         NOT NULL,
    event_type    VARCHAR(150) NOT NULL,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT pk_processed_event PRIMARY KEY (event_id)
);
