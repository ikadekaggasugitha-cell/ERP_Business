-- hr_db — rollback 000003_pii_and_partitions

DO $$
DECLARE r record;
BEGIN
  FOR r IN SELECT i.inhparent::regclass AS parent, c.oid::regclass AS part, c.relname
           FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
           WHERE i.inhparent IN ('hr.outbox'::regclass)
             AND (
               c.relname LIKE '%_default'
               OR (c.relname !~ '_y2026m(0[1-9]|1[0-2])$' AND c.relname !~ '_y2027m01$')
             ) LOOP
    EXECUTE format('ALTER TABLE %s DETACH PARTITION %s', r.parent, r.part);
    EXECUTE format('DROP TABLE IF EXISTS %s', r.part);
  END LOOP;
END $$;

DROP INDEX IF EXISTS hr.idx_employee_national_id_hash;
ALTER TABLE hr.employee
    DROP COLUMN IF EXISTS national_id_hash,
    DROP COLUMN IF EXISTS encrypted_email,
    DROP COLUMN IF EXISTS encrypted_dob,
    DROP COLUMN IF EXISTS encrypted_national_id,
    DROP COLUMN IF EXISTS encrypted_full_name,
    ALTER COLUMN email TYPE VARCHAR(255),
    ALTER COLUMN national_id TYPE VARCHAR(50),
    ALTER COLUMN full_name TYPE VARCHAR(200);
