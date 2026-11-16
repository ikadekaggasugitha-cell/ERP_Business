-- coa_db — rollback 000004_partition_hardening

-- Restore coa.outbox to non-partitioned.
DROP TABLE coa.outbox;
CREATE TABLE coa.outbox (
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
CREATE INDEX idx_outbox_unpublished ON coa.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON coa.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON coa.outbox (correlation_id);

DROP FUNCTION IF EXISTS public.detach_old_partitions(regclass, int);
DROP FUNCTION IF EXISTS public.ensure_future_partitions(regclass, int);
DROP FUNCTION IF EXISTS public.create_monthly_partition(regclass, date);

