-- =============================================================================
-- Safe batched cleanup of expired public.idempotency_key rows (db/ops/prune_idempotency_keys.sql)
-- Deletes up to 5,000 expired rows per batch (created_at < NOW() - INTERVAL '1 hour')
-- to avoid long table locks or WAL spikes. Safe to run repeatedly on any ERP DB.
-- Usage: psql -v ON_ERROR_STOP=1 -d <db> -f db/ops/prune_idempotency_keys.sql
-- =============================================================================
\set ON_ERROR_STOP on

DO $$
DECLARE
  v_batch_limit CONSTANT integer := 5000;
  v_max_batches CONSTANT integer := 20;
  v_deleted     integer := 0;
  v_total       integer := 0;
  v_iter        integer := 0;
BEGIN
  IF to_regclass('public.idempotency_key') IS NULL THEN
    RAISE NOTICE 'prune_idempotency_keys: table public.idempotency_key not present, skipping';
    RETURN;
  END IF;

  LOOP
    v_iter := v_iter + 1;
    WITH expired AS (
      SELECT ctid
      FROM public.idempotency_key
      WHERE created_at < NOW() - INTERVAL '1 hour'
      LIMIT v_batch_limit
    )
    DELETE FROM public.idempotency_key t
    USING expired e
    WHERE t.ctid = e.ctid;

    GET DIAGNOSTICS v_deleted = ROW_COUNT;
    v_total := v_total + v_deleted;

    EXIT WHEN v_deleted < v_batch_limit OR v_iter >= v_max_batches;
  END LOOP;

  RAISE NOTICE 'prune_idempotency_keys: deleted % expired row(s) in % batch(es)', v_total, v_iter;
END $$;
