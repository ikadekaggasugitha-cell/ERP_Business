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

\if :db = 'ap_db'
INSERT INTO ap.ap_invoice (organization_id, invoice_no, vendor_id, invoice_date, due_date, currency_code, total_amount, outstanding_amount, status, created_by)
SELECT v.org_id, v.inv_no, v.vendor_id, '2026-06-15'::date, '2026-07-15'::date, 'IDR', 1000.0000, 1000.0000, 'Draft', '00000000-0000-0000-0000-000000000001'::uuid
FROM (VALUES
  ('11111111-1111-1111-1111-111111111111'::uuid, 'AP-RLS-A', '10000000-0000-4000-8000-0000000000a1'::uuid),
  ('22222222-2222-2222-2222-222222222222'::uuid, 'AP-RLS-B', '10000000-0000-4000-8000-0000000000b1'::uuid)
) AS v(org_id, inv_no, vendor_id)
WHERE NOT EXISTS (
  SELECT 1 FROM ap.ap_invoice x WHERE x.organization_id = v.org_id AND x.invoice_no = v.inv_no AND x.invoice_date = '2026-06-15'::date
);
\endif

\if :db = 'ar_db'
INSERT INTO ar.ar_invoice (organization_id, invoice_no, customer_id, invoice_date, due_date, currency_code, total_amount, outstanding_amount, status, created_by)
SELECT v.org_id, v.inv_no, v.customer_id, '2026-06-15'::date, '2026-07-15'::date, 'IDR', 1000.0000, 1000.0000, 'Issued', '00000000-0000-0000-0000-000000000001'::uuid
FROM (VALUES
  ('11111111-1111-1111-1111-111111111111'::uuid, 'AR-RLS-A', '10000000-0000-4000-8000-0000000000a2'::uuid),
  ('22222222-2222-2222-2222-222222222222'::uuid, 'AR-RLS-B', '10000000-0000-4000-8000-0000000000b2'::uuid)
) AS v(org_id, inv_no, customer_id)
WHERE NOT EXISTS (
  SELECT 1 FROM ar.ar_invoice x WHERE x.organization_id = v.org_id AND x.invoice_no = v.inv_no AND x.invoice_date = '2026-06-15'::date
);
\endif

\if :db = 'audit_db'
INSERT INTO audit.audit_log (id, occurred_at, organization_id, actor_id, action, entity_type, entity_id, correlation_id) VALUES
  ('a0d10000-0000-4000-8000-000000000101', '2026-06-15T00:00:00Z', '11111111-1111-1111-1111-111111111111', '00000000-0000-0000-0000-000000000001', 'rls.smoke.a', 'SmokeEntity', '11111111-1111-1111-1111-111111111111', 'c0000000-0000-4000-8000-000000000101'),
  ('a0d10000-0000-4000-8000-000000000102', '2026-06-15T00:00:00Z', '22222222-2222-2222-2222-222222222222', '00000000-0000-0000-0000-000000000001', 'rls.smoke.b', 'SmokeEntity', '22222222-2222-2222-2222-222222222222', 'c0000000-0000-4000-8000-000000000102')
ON CONFLICT (id, occurred_at) DO NOTHING;
\endif

\if :db = 'gl_db'
INSERT INTO gl.journal_entry (organization_id, journal_no, financial_organization_id, ledger_id, fiscal_period_id, posting_date, journal_source, currency_code, correlation_id, created_by)
SELECT v.org_id, v.jno, v.org_id, '1ed00000-0000-4000-8000-000000000001'::uuid, 'f9000000-0000-4000-8000-000000000001'::uuid, '2026-06-15'::date, 'Manual', 'IDR', v.corr_id, '00000000-0000-0000-0000-000000000001'::uuid
FROM (VALUES
  ('11111111-1111-1111-1111-111111111111'::uuid, 'JE-RLS-A', 'c0000000-0000-4000-8000-000000000201'::uuid),
  ('22222222-2222-2222-2222-222222222222'::uuid, 'JE-RLS-B', 'c0000000-0000-4000-8000-000000000202'::uuid)
) AS v(org_id, jno, corr_id)
WHERE NOT EXISTS (
  SELECT 1 FROM gl.journal_entry x WHERE x.organization_id = v.org_id AND x.journal_no = v.jno AND x.posting_date = '2026-06-15'::date
);
\endif

\if :db = 'hr_db'
INSERT INTO hr.employee (organization_id, employee_no, full_name, hire_date, status, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'EMP-RLS-A', 'Employee Tenant A', '2026-01-01', 'Active', '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'EMP-RLS-B', 'Employee Tenant B', '2026-01-01', 'Active', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, employee_no) DO NOTHING;
\endif

\if :db = 'inventory_db'
INSERT INTO inv.warehouse (organization_id, code, name, organization_ref_id, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'WH-RLS-A', 'Warehouse Tenant A', '11111111-1111-1111-1111-111111111111', '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'WH-RLS-B', 'Warehouse Tenant B', '22222222-2222-2222-2222-222222222222', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif

\if :db = 'manufacturing_db'
INSERT INTO mfg.bill_of_material (organization_id, code, product_item_id, version, is_active, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'BOM-RLS-A', '10000000-0000-4000-8000-0000000000a3', 1, true, '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'BOM-RLS-B', '10000000-0000-4000-8000-0000000000b3', 1, true, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code, version) DO NOTHING;
\endif

\if :db = 'masterdata_db'
INSERT INTO md.party (organization_id, code, name, party_type, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'PTY-RLS-A', 'Party Tenant A', 'customer', '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'PTY-RLS-B', 'Party Tenant B', 'vendor', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif

\if :db = 'payroll_db'
INSERT INTO pay.payroll_run (organization_id, run_no, period_month, period_year, status, correlation_id, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'PRUN-RLS-A', 6, 2026, 'Draft', 'c0000000-0000-4000-8000-000000000301', '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'PRUN-RLS-B', 6, 2026, 'Draft', 'c0000000-0000-4000-8000-000000000302', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, run_no) DO NOTHING;
\endif

\if :db = 'procurement_db'
INSERT INTO pr.purchase_requisition (organization_id, pr_no, requester_id, status, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'PR-RLS-A', '00000000-0000-0000-0000-000000000001', 'Draft', '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'PR-RLS-B', '00000000-0000-0000-0000-000000000001', 'Draft', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, pr_no) DO NOTHING;
\endif

\if :db = 'reporting_db'
INSERT INTO rpt.gl_balance_snapshot (organization_id, fiscal_period_id, account_code, closing_balance) VALUES
  ('11111111-1111-1111-1111-111111111111', 'f9000000-0000-4000-8000-000000000001', '1100', 1000000.0000),
  ('22222222-2222-2222-2222-222222222222', 'f9000000-0000-4000-8000-000000000001', '1100', 2000000.0000)
ON CONFLICT (organization_id, fiscal_period_id, account_code) DO NOTHING;
\endif

\if :db = 'sales_db'
INSERT INTO so.sales_order (organization_id, so_no, customer_id, order_date, currency_code, total_amount, status, correlation_id, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'SO-RLS-A', '10000000-0000-4000-8000-0000000000a4', '2026-06-15', 'IDR', 1500.0000, 'Draft', 'c0000000-0000-4000-8000-000000000401', '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'SO-RLS-B', '10000000-0000-4000-8000-0000000000b4', '2026-06-15', 'IDR', 2500.0000, 'Draft', 'c0000000-0000-4000-8000-000000000402', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, so_no) DO NOTHING;
\endif

\if :db = 'treasury_db'
INSERT INTO trs.bank_account (organization_id, code, bank_name, account_no, currency_code, gl_account_id, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'BANK-RLS-A', 'BCA Tenant A', '111000111', 'IDR', '10000000-0000-4000-8000-0000000000a5', '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'BANK-RLS-B', 'BCA Tenant B', '222000222', 'IDR', '10000000-0000-4000-8000-0000000000b5', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code) DO NOTHING;
\endif

\if :db = 'user_db'
INSERT INTO usr.app_user (organization_id, username, email, full_name, status, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'rls_user_a', 'rls_user_a@erp.local', 'RLS User A', 'Active', '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'rls_user_b', 'rls_user_b@erp.local', 'RLS User B', 'Active', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (username) DO NOTHING;
\endif

\if :db = 'workflow_db'
INSERT INTO wf.workflow_definition (organization_id, code, document_type, version, is_active, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111', 'WF-RLS-A', 'PurchaseOrder', 1, true, '00000000-0000-0000-0000-000000000001'),
  ('22222222-2222-2222-2222-222222222222', 'WF-RLS-B', 'PurchaseOrder', 1, true, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (organization_id, code, version) DO NOTHING;
\endif
