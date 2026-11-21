-- CI fixture for db/tests/rls_smoke.sql (db/tests/rls_smoke_tenants.sql).
-- Inserts two synthetic organizations (one row each) into the reference tables
-- of seeded extended DBs so the smoke's checks B/C/D run behaviorally in CI
-- instead of vacuously. Applied by the db-regression workflow right after
-- seed.sql; guarded per DB via the :db psql variable, so running it against a
-- DB without a section below is a harmless no-op.
-- Not for local bring-up: tenant demo rows are out of scope for seeds
-- (see db/seeds/README.md).
\set ON_ERROR_STOP on
\if :{?db} \else \set db '' \endif

\if :db = 'fixedasset_db'
INSERT INTO fa.asset_category (id, organization_id, code, name, default_depreciation_method, default_useful_life_months, is_active, created_by) VALUES
  ('fa1c0000-0000-4000-8000-000000000101', '11111111-1111-1111-1111-111111111111', 'TENANT_A', 'Kategori Tenant A', 'StraightLine', 48, true, '00000000-0000-0000-0000-000000000001'),
  ('fa1c0000-0000-4000-8000-000000000102', '22222222-2222-2222-2222-222222222222', 'TENANT_B', 'Kategori Tenant B', 'StraightLine', 48, true, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif

\if :db = 'budget_db'
INSERT INTO bud.budget (id, organization_id, code, name, financial_organization_ref, fiscal_year_ref, version, status, is_active, created_by) VALUES
  ('b0d00000-0000-4000-8000-000000000101', '11111111-1111-1111-1111-111111111111', 'BUD_TENANT_A', 'Anggaran Tenant A', '11111111-1111-1111-1111-111111111111', '20260000-0000-4000-8000-000000000001', 1, 'Draft', false, '00000000-0000-0000-0000-000000000001'),
  ('b0d00000-0000-4000-8000-000000000102', '22222222-2222-2222-2222-222222222222', 'BUD_TENANT_B', 'Anggaran Tenant B', '22222222-2222-2222-2222-222222222222', '20260000-0000-4000-8000-000000000001', 1, 'Draft', false, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code, version) DO NOTHING;
\endif

\if :db = 'financialclose_db'
INSERT INTO fcl.closing_checklist_template (id, organization_id, code, name, financial_organization_ref, created_by) VALUES
  ('f0c10000-0000-4000-8000-000000000101', '11111111-1111-1111-1111-111111111111', 'TENANT_A', 'Checklist Tenant A', '11111111-1111-1111-1111-111111111111', '00000000-0000-0000-0000-000000000001'),
  ('f0c10000-0000-4000-8000-000000000102', '22222222-2222-2222-2222-222222222222', 'TENANT_B', 'Checklist Tenant B', '22222222-2222-2222-2222-222222222222', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif

\if :db = 'consolidation_db'
INSERT INTO cons.consolidation_group (id, organization_id, code, name, presentation_currency_code, created_by) VALUES
  ('c0500000-0000-4000-8000-000000000101', '11111111-1111-1111-1111-111111111111', 'GRP_TENANT_A', 'Grup Konsolidasi Tenant A', 'IDR', '00000000-0000-0000-0000-000000000001'),
  ('c0500000-0000-4000-8000-000000000102', '22222222-2222-2222-2222-222222222222', 'GRP_TENANT_B', 'Grup Konsolidasi Tenant B', 'IDR', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif

\if :db = 'project_db'
INSERT INTO prj.project (id, organization_id, code, name, project_type, organization_ref, currency_code, status, created_by) VALUES
  ('9a100000-0000-4000-8000-000000000101', '11111111-1111-1111-1111-111111111111', 'PRJ_TENANT_A', 'Proyek Tenant A', 'Internal', '11111111-1111-1111-1111-111111111111', 'IDR', 'Draft', '00000000-0000-0000-0000-000000000001'),
  ('9a100000-0000-4000-8000-000000000102', '22222222-2222-2222-2222-222222222222', 'PRJ_TENANT_B', 'Proyek Tenant B', 'Internal', '22222222-2222-2222-2222-222222222222', 'IDR', 'Draft', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif

\if :db = 'asset_db'
INSERT INTO ast.asset (id, organization_id, asset_code, name, category, organization_ref, status, created_by) VALUES
  ('a5700000-0000-4000-8000-000000000101', '11111111-1111-1111-1111-111111111111', 'AST_TENANT_A', 'Aset Tenant A', 'Equipment', '11111111-1111-1111-1111-111111111111', 'Active', '00000000-0000-0000-0000-000000000001'),
  ('a5700000-0000-4000-8000-000000000102', '22222222-2222-2222-2222-222222222222', 'AST_TENANT_B', 'Aset Tenant B', 'Equipment', '22222222-2222-2222-2222-222222222222', 'Active', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, asset_code) DO NOTHING;
\endif

\if :db = 'maintenance_db'
INSERT INTO mnt.maintenance_schedule (id, organization_id, asset_ref, schedule_type, interval_value, interval_unit, is_active, created_by) VALUES
  ('a4700000-0000-4000-8000-000000000101', '11111111-1111-1111-1111-111111111111', 'a5700000-0000-4000-8000-000000000101', 'TimeBased', 30, 'Day', true, '00000000-0000-0000-0000-000000000001'),
  ('a4700000-0000-4000-8000-000000000102', '22222222-2222-2222-2222-222222222222', 'a5700000-0000-4000-8000-000000000102', 'TimeBased', 30, 'Day', true, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (id) DO NOTHING;
\endif

\if :db = 'crm_db'
INSERT INTO crm.customer_segment (id, organization_id, code, name, criteria, created_by) VALUES
  ('c9000000-0000-4000-8000-000000000101', '11111111-1111-1111-1111-111111111111', 'TENANT_A', 'Segmen Tenant A', '{"tenant": "A"}'::jsonb, '00000000-0000-0000-0000-000000000001'),
  ('c9000000-0000-4000-8000-000000000102', '22222222-2222-2222-2222-222222222222', 'TENANT_B', 'Segmen Tenant B', '{"tenant": "B"}'::jsonb, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif

\if :db = 'coa_db'
INSERT INTO fin.fiscal_calendar (id, organization_id, code, name, period_type, created_by) VALUES
  ('ca1e0000-0000-4000-8000-000000000101', '11111111-1111-1111-1111-111111111111', 'CAL_A', 'Calendar A', 'Monthly', '00000000-0000-0000-0000-000000000001'),
  ('ca1e0000-0000-4000-8000-000000000102', '22222222-2222-2222-2222-222222222222', 'CAL_B', 'Calendar B', 'Monthly', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif

\if :db = 'currency_db'
INSERT INTO cur.exchange_rate_type (organization_id, code, name, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'TENANT_A', 'Rate Type A', '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'TENANT_B', 'Rate Type B', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif

\if :db = 'localization_db'
INSERT INTO loc.language (organization_id, code, name, is_active, is_fallback, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'xa', 'Tenant A Lang', true, false, '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'xb', 'Tenant B Lang', true, false, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif

\if :db = 'org_db'
INSERT INTO org.organization_type (organization_id, code, name, level_hint, is_active, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'TENANT_A', 'Type A', 1, true, '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'TENANT_B', 'Type B', 1, true, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif

\if :db = 'security_db'
INSERT INTO sec.role (organization_id, code, name, is_system, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'ROLE_TENANT_A', 'Role Tenant A', false, '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'ROLE_TENANT_B', 'Role Tenant B', false, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif

\if :db = 'tax_db'
INSERT INTO tax.tax_code (organization_id, code, rate, tax_type, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'TAX_A', 11.0000, 'VAT', '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'TAX_B', 11.0000, 'VAT', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif

\if :db = 'dimension_db'
INSERT INTO cc.cost_center (organization_id, code, name, effective_from, status, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'CC_A', 'Cost Center A', '2026-01-01', 'Active', '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'CC_B', 'Cost Center B', '2026-01-01', 'Active', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif
