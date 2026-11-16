-- =============================================================================
-- gl_db — Step 3 partition hardening (maintenance fns, outbox partitioning, journal_line partitioning)
-- Ref: roadmap Step 3 | golang-migrate: 000004
-- =============================================================================

-- Reusable partition maintenance (idempotent). Safe to run in any partitioned DB.
CREATE OR REPLACE FUNCTION public.create_monthly_partition(parent regclass, month date)
RETURNS void LANGUAGE plpgsql AS $fn$
DECLARE nsp text; base text; part text; s date; e date; has_rls boolean;
BEGIN
  SELECT n.nspname, c.relname, c.relrowsecurity INTO nsp, base, has_rls
  FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE c.oid = parent;
  s := date_trunc('month', month)::date;
  e := (s + interval '1 month')::date;
  part := format('%s_y%sm%s', base, to_char(s,'YYYY'), to_char(s,'MM'));
  IF to_regclass(format('%I.%I', nsp, part)) IS NULL THEN
    EXECUTE format('CREATE TABLE %I.%I PARTITION OF %s FOR VALUES FROM (%L) TO (%L)', nsp, part, parent, s, e);
    IF has_rls THEN
      EXECUTE format('ALTER TABLE %I.%I ENABLE ROW LEVEL SECURITY', nsp, part);
      EXECUTE format('CREATE POLICY scope_isolation ON %I.%I USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()))', nsp, part);
    END IF;
  END IF;
END;
$fn$;

CREATE OR REPLACE FUNCTION public.ensure_future_partitions(parent regclass, months_ahead int DEFAULT 3)
RETURNS void LANGUAGE plpgsql AS $fn$
DECLARE m date;
BEGIN
  FOR m IN SELECT generate_series(date_trunc('month', CURRENT_DATE)::date,
                                  (date_trunc('month', CURRENT_DATE) + (months_ahead || ' months')::interval)::date,
                                  '1 month')::date LOOP
    PERFORM public.create_monthly_partition(parent, m);
  END LOOP;
END;
$fn$;

CREATE OR REPLACE FUNCTION public.detach_old_partitions(parent regclass, keep_months int)
RETURNS void LANGUAGE plpgsql AS $fn$
DECLARE r record; cutoff date; ym text[]; pstart date;
BEGIN
  cutoff := (date_trunc('month', CURRENT_DATE) - (keep_months || ' months')::interval)::date;
  FOR r IN SELECT c.oid::regclass AS part, c.relname
           FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid WHERE i.inhparent = parent LOOP
    ym := regexp_match(r.relname, '_y([0-9]{4})m([0-9]{2})$');
    IF ym IS NULL THEN CONTINUE; END IF;
    pstart := make_date(ym[1]::int, ym[2]::int, 1);
    IF pstart < cutoff THEN
      EXECUTE format('ALTER TABLE %s DETACH PARTITION %s', parent, r.part);
      RAISE NOTICE 'detached % (start % < cutoff %)', r.part, pstart, cutoff;
    END IF;
  END LOOP;
END;
$fn$;

-- Convert gl.journal_line to RANGE-partitioned by posting_date (empty pre-prod).
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
    CONSTRAINT pk_journal_line PRIMARY KEY (id, posting_date),
    CONSTRAINT uq_journal_line UNIQUE (journal_entry_id, line_no, posting_date),
    CONSTRAINT fk_journal_line_entry FOREIGN KEY (journal_entry_id, posting_date)
        REFERENCES gl.journal_entry (id, posting_date) ON DELETE CASCADE,
    CONSTRAINT ck_journal_line_amounts CHECK (debit_amount >= 0 AND credit_amount >= 0)
) PARTITION BY RANGE (posting_date);
CREATE INDEX idx_journal_line_account ON gl.journal_line (account_id);
ALTER TABLE gl.journal_line ENABLE ROW LEVEL SECURITY;
CREATE POLICY scope_isolation ON gl.journal_line USING (organization_id = ANY (public.current_org_scope())) WITH CHECK (organization_id = ANY (public.current_org_scope()));
CREATE TRIGGER touch_row BEFORE UPDATE ON gl.journal_line FOR EACH ROW EXECUTE FUNCTION public.tg_touch_row();
DO $$ DECLARE m date; BEGIN
  FOR m IN SELECT generate_series('2026-01-01'::date, '2027-01-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('gl.journal_line'::regclass, m);
  END LOOP;
END $$;

-- Convert gl.outbox to RANGE-partitioned by created_at (empty pre-prod).
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
    CONSTRAINT pk_outbox PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);
CREATE INDEX idx_outbox_unpublished ON gl.outbox (created_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate   ON gl.outbox (aggregate_id);
CREATE INDEX idx_outbox_correlation ON gl.outbox (correlation_id);
DO $$ DECLARE m date; BEGIN
  FOR m IN SELECT generate_series('2026-01-01'::date, '2027-01-01'::date, '1 month')::date LOOP
    PERFORM public.create_monthly_partition('gl.outbox'::regclass, m);
  END LOOP;
END $$;

