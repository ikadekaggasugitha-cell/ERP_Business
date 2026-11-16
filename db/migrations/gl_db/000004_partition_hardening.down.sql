-- gl_db — rollback 000004_partition_hardening

-- Restore gl.outbox to non-partitioned.
DROP TABLE gl.outbox;
CREATE TABLE gl.outbox (
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
CREATE INDEX idx_outbox_unpublished ON gl.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON gl.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON gl.outbox (correlation_id);

-- Restore gl.journal_line to non-partitioned (post-000003 state).
DROP TABLE gl.journal_line;
CREATE TABLE gl.journal_line (
    id                       UUID           NOT NULL DEFAULT gen_random_uuid(),
    organization_id          UUID           NOT NULL,
    journal_entry_id         UUID           NOT NULL,
    posting_date             DATE           NOT NULL,
    line_no                  INTEGER        NOT NULL,
    account_id               UUID           NOT NULL,
    account_combination_id   UUID,
    debit_amount             NUMERIC(20,4)  NOT NULL DEFAULT 0,
    credit_amount            NUMERIC(20,4)  NOT NULL DEFAULT 0,
    functional_amount        NUMERIC(20,4)  NOT NULL DEFAULT 0,
    cost_center_id           UUID,
    project_id               UUID,
    description              TEXT,
    created_at               TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_by               UUID           NOT NULL,
    updated_at               TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by               UUID,
    deleted_at               TIMESTAMPTZ,
    row_version              INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT pk_journal_line PRIMARY KEY (id),
    CONSTRAINT uq_journal_line UNIQUE (journal_entry_id, line_no),
    CONSTRAINT fk_journal_line_entry FOREIGN KEY (journal_entry_id, posting_date)
        REFERENCES gl.journal_entry (id, posting_date) ON DELETE CASCADE,
    CONSTRAINT ck_journal_line_amounts CHECK (debit_amount >= 0 AND credit_amount >= 0)
);
CREATE INDEX idx_journal_line_account ON gl.journal_line (account_id);
ALTER TABLE gl.journal_line ENABLE ROW LEVEL SECURITY;
CREATE POLICY scope_isolation ON gl.journal_line USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE TRIGGER touch_row BEFORE UPDATE ON gl.journal_line FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();

DROP FUNCTION IF EXISTS public.detach_old_partitions(regclass, int);
DROP FUNCTION IF EXISTS public.ensure_future_partitions(regclass, int);
DROP FUNCTION IF EXISTS public.create_monthly_partition(regclass, date);

