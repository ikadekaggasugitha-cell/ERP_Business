-- currency_db — rollback 000002_partition_hardening

-- Restore cur.outbox to non-partitioned.
DROP TABLE cur.outbox;
CREATE TABLE cur.outbox (
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
CREATE INDEX idx_outbox_unpublished ON cur.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON cur.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON cur.outbox (correlation_id);

DROP FUNCTION IF EXISTS public.detach_old_partitions(regclass, int);
DROP FUNCTION IF EXISTS public.ensure_future_partitions(regclass, int);
DROP FUNCTION IF EXISTS public.create_monthly_partition(regclass, date);

