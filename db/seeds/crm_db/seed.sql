-- Seed: crm_db (reference). Idempotent. Run: psql -d crm_db -f this
-- Optional: -v org_id=<uuid> -v actor=<uuid> to seed for a specific tenant.
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO crm.customer_segment (id, organization_id, code, name, criteria, created_by) VALUES
  ('c9000000-0000-4000-8000-000000000001', :'org_id','SMB','Usaha Kecil Menengah','{"employee_count_max": 250, "annual_revenue_max_idr": 50000000000}'::jsonb,:'actor'),
  ('c9000000-0000-4000-8000-000000000002', :'org_id','ENTERPRISE','Enterprise','{"employee_count_min": 250}'::jsonb,:'actor'),
  ('c9000000-0000-4000-8000-000000000003', :'org_id','KEY_ACCOUNT','Akun Strategis','{"priority": "high"}'::jsonb,:'actor'),
  ('c9000000-0000-4000-8000-000000000004', :'org_id','GOVERNMENT','Instansi Pemerintah','{"sector": "government"}'::jsonb,:'actor'),
  ('c9000000-0000-4000-8000-000000000005', :'org_id','RETAIL','Perorangan / Retail','{"sector": "retail"}'::jsonb,:'actor')
ON CONFLICT (organization_id, code) DO NOTHING;
