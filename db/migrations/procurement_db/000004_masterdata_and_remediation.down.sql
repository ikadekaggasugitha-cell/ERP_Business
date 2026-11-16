-- procurement_db — rollback 000004_masterdata_and_remediation

DO $$
DECLARE r record;
BEGIN
  FOR r IN SELECT i.inhparent::regclass AS parent, c.oid::regclass AS part, c.relname
           FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
           WHERE i.inhparent IN ('pr.outbox'::regclass)
             AND (
               c.relname LIKE '%_default'
               OR (c.relname !~ '_y2026m(0[1-9]|1[0-2])$' AND c.relname !~ '_y2027m01$')
             ) LOOP
    EXECUTE format('ALTER TABLE %s DETACH PARTITION %s', r.parent, r.part);
    EXECUTE format('DROP TABLE IF EXISTS %s', r.part);
  END LOOP;
END $$;

ALTER TABLE po.po_line DROP CONSTRAINT IF EXISTS ck_po_line_qty;

ALTER TABLE pr.pr_line DROP CONSTRAINT IF EXISTS ck_pr_line_qty;

ALTER TABLE gr.goods_receipt DROP CONSTRAINT IF EXISTS ck_goods_receipt_status;

DROP TABLE IF EXISTS pr.masterdata_cache;
