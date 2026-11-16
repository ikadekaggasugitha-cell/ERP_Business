-- Seed: fixedasset_db (reference). Idempotent. Run: psql -d fixedasset_db -f this
-- Optional: -v org_id=<uuid> -v actor=<uuid> to seed for a specific tenant.
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO fa.asset_category (id, organization_id, code, name, default_depreciation_method, default_useful_life_months, is_active, created_by) VALUES
  ('fa1c0000-0000-4000-8000-000000000001', :'org_id','BUILDING','Gedung dan Bangunan','StraightLine',240,true,:'actor'),
  ('fa1c0000-0000-4000-8000-000000000002', :'org_id','VEHICLE','Kendaraan','StraightLine',96,true,:'actor'),
  ('fa1c0000-0000-4000-8000-000000000003', :'org_id','MACHINE','Mesin dan Peralatan Produksi','StraightLine',96,true,:'actor'),
  ('fa1c0000-0000-4000-8000-000000000004', :'org_id','OFFICE_EQUIPMENT','Peralatan Kantor','StraightLine',48,true,:'actor'),
  ('fa1c0000-0000-4000-8000-000000000005', :'org_id','COMPUTER','Peralatan Komputer','StraightLine',48,true,:'actor'),
  ('fa1c0000-0000-4000-8000-000000000006', :'org_id','LAND','Tanah (tidak disusutkan)','StraightLine',NULL,true,:'actor')
ON CONFLICT (organization_id, code) DO NOTHING;
