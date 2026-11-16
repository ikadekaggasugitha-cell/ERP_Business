-- sqlc catalog for audit-service (plain DDL for type inference ONLY).
-- Authoritative schema (RANGE partitioning by occurred_at, RLS via app.org_scope,
-- append-only immutability trigger) lives in db/migrations/audit_db. Keep this
-- catalog in sync (sqlc diff gate in CI).
--
-- audit-service is a wildcard CONSUMER: no outbox (it produces no events); the
-- audit_log is append-only and there is a processed_event idempotency ledger.
CREATE SCHEMA IF NOT EXISTS audit;

-- Principal table. Partitioned by occurred_at in the real schema, so the PK is
-- composite (id, occurred_at) — mirrored here (PARTITION BY stripped for sqlc).
CREATE TABLE audit.audit_log (
    id               uuid         NOT NULL DEFAULT gen_random_uuid(),
    occurred_at      timestamptz  NOT NULL,
    organization_id  uuid         NOT NULL,
    actor_id         uuid,
    action           varchar(50)  NOT NULL,
    entity_type      varchar(100) NOT NULL,
    entity_id        uuid,
    before_value     jsonb,
    after_value      jsonb,
    reason           text,
    correlation_id   uuid         NOT NULL,
    PRIMARY KEY (id, occurred_at)
);

CREATE TABLE audit.processed_event (
    event_id     uuid         NOT NULL,
    event_type   varchar(150) NOT NULL,
    processed_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id)
);
