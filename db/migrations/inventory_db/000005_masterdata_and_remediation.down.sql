-- inventory_db — rollback 000005_masterdata_and_remediation

DO $$
DECLARE r record;
BEGIN
  FOR r IN SELECT i.inhparent::regclass AS parent, c.oid::regclass AS part, c.relname
           FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
           WHERE i.inhparent IN ('inv.stock_movement'::regclass, 'inv.outbox'::regclass)
             AND (
               c.relname LIKE '%_default'
               OR (c.relname !~ '_y2026m(0[1-9]|1[0-2])$' AND c.relname !~ '_y2027m01$')
             ) LOOP
    EXECUTE format('ALTER TABLE %s DETACH PARTITION %s', r.parent, r.part);
    EXECUTE format('DROP TABLE IF EXISTS %s', r.part);
  END LOOP;
END $$;

ALTER TABLE inv.stock_balance DROP CONSTRAINT IF EXISTS ck_stock_balance_nonneg;

DROP TABLE IF EXISTS inv.masterdata_cache;
