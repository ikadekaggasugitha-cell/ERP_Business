-- Seed: fixedasset_db (reference). Idempotent. Run: psql -d fixedasset_db -f this
-- Optional: -v org_id=<uuid> -v actor=<uuid> to seed for a specific tenant.
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO fa.asset_category (id, organization_id, code, name, default_depreciation_method, default_useful_life_months, is_active, created_by) VALUES
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN 'fa1c0000-0000-4000-8000-000000000001'::uuid ELSE md5(:'org_id' || ':fa:BUILDING')::uuid END,         :'org_id','BUILDING','Gedung dan Bangunan','StraightLine',240,true,:'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN 'fa1c0000-0000-4000-8000-000000000002'::uuid ELSE md5(:'org_id' || ':fa:VEHICLE')::uuid END,          :'org_id','VEHICLE','Kendaraan','StraightLine',96,true,:'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN 'fa1c0000-0000-4000-8000-000000000003'::uuid ELSE md5(:'org_id' || ':fa:MACHINE')::uuid END,          :'org_id','MACHINE','Mesin dan Peralatan Produksi','StraightLine',96,true,:'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN 'fa1c0000-0000-4000-8000-000000000004'::uuid ELSE md5(:'org_id' || ':fa:OFFICE_EQUIPMENT')::uuid END, :'org_id','OFFICE_EQUIPMENT','Peralatan Kantor','StraightLine',48,true,:'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN 'fa1c0000-0000-4000-8000-000000000005'::uuid ELSE md5(:'org_id' || ':fa:COMPUTER')::uuid END,         :'org_id','COMPUTER','Peralatan Komputer','StraightLine',48,true,:'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN 'fa1c0000-0000-4000-8000-000000000006'::uuid ELSE md5(:'org_id' || ':fa:LAND')::uuid END,             :'org_id','LAND','Tanah (tidak disusutkan)','StraightLine',NULL,true,:'actor')
ON CONFLICT (organization_id, code) DO NOTHING;
