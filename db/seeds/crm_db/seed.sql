-- Seed: crm_db (reference). Idempotent. Run: psql -d crm_db -f this
-- Optional: -v org_id=<uuid> -v actor=<uuid> to seed for a specific tenant.
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO crm.customer_segment (id, organization_id, code, name, criteria, created_by) VALUES
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN 'c9000000-0000-4000-8000-000000000001'::uuid ELSE md5(:'org_id' || ':crm:SMB')::uuid END,         :'org_id','SMB','Usaha Kecil Menengah','{"employee_count_max": 250, "annual_revenue_max_idr": 50000000000}'::jsonb,:'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN 'c9000000-0000-4000-8000-000000000002'::uuid ELSE md5(:'org_id' || ':crm:ENTERPRISE')::uuid END,  :'org_id','ENTERPRISE','Enterprise','{"employee_count_min": 250}'::jsonb,:'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN 'c9000000-0000-4000-8000-000000000003'::uuid ELSE md5(:'org_id' || ':crm:KEY_ACCOUNT')::uuid END, :'org_id','KEY_ACCOUNT','Akun Strategis','{"priority": "high"}'::jsonb,:'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN 'c9000000-0000-4000-8000-000000000004'::uuid ELSE md5(:'org_id' || ':crm:GOVERNMENT')::uuid END,  :'org_id','GOVERNMENT','Instansi Pemerintah','{"sector": "government"}'::jsonb,:'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000005'::uuid THEN 'c9000000-0000-4000-8000-000000000005'::uuid ELSE md5(:'org_id' || ':crm:RETAIL')::uuid END,      :'org_id','RETAIL','Perorangan / Retail','{"sector": "retail"}'::jsonb,:'actor')
ON CONFLICT (organization_id, code) DO NOTHING;
