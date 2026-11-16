-- budget_db — rollback 000002_partition_hardening

-- Restore bud.outbox to non-partitioned.
DROP TABLE bud.outbox;
CREATE TABLE bud.outbox (
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
CREATE INDEX idx_outbox_unpublished ON bud.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON bud.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON bud.outbox (correlation_id);

DROP FUNCTION IF EXISTS public.detach_old_partitions(regclass, int);
DROP FUNCTION IF EXISTS public.ensure_future_partitions(regclass, int);
DROP FUNCTION IF EXISTS public.create_monthly_partition(regclass, date);

