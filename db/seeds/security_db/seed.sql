-- Seed: security_db (system RBAC). Idempotent. Run: psql -d security_db -f this
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

-- Permissions: both dot-separated (<module>.<action>) and colon-separated (<module>:<action>)
-- across all 30 microservice modules (matching services/api-gateway rolePermissions).
INSERT INTO sec.permission (organization_id, code, module, action, created_by)
SELECT :'org_id', m || '.' || a, m, a, :'actor'
FROM (VALUES
  ('org'),('user'),('security'),('masterdata'),('workflow'),('audit'),
  ('gl'),('coa'),('ap'),('ar'),('tax'),('treasury'),('currency'),('reporting'),
  ('procurement'),('inventory'),('manufacturing'),('sales'),('hr'),('payroll'),
  ('dimension'),('localization'),('fixedasset'),('budget'),('financialclose'),
  ('consolidation'),('project'),('asset'),('maintenance'),('crm')
) mods(m)
CROSS JOIN (VALUES ('create'),('read'),('update'),('delete'),('approve')) acts(a)
ON CONFLICT (organization_id, code) DO NOTHING;

INSERT INTO sec.permission (organization_id, code, module, action, created_by)
SELECT :'org_id', m || ':' || a, m, a, :'actor'
FROM (VALUES
  ('org'),('user'),('security'),('masterdata'),('workflow'),('audit'),
  ('gl'),('coa'),('ap'),('ar'),('tax'),('treasury'),('currency'),('reporting'),
  ('procurement'),('inventory'),('manufacturing'),('sales'),('hr'),('payroll'),
  ('dimension'),('localization'),('fixedasset'),('budget'),('financialclose'),
  ('consolidation'),('project'),('asset'),('maintenance'),('crm')
) mods(m)
CROSS JOIN (VALUES ('read'),('write'),('approve')) acts(a)
ON CONFLICT (organization_id, code) DO NOTHING;

-- System roles
INSERT INTO sec.role (organization_id, code, name, is_system, created_by) VALUES
  (:'org_id','SYSTEM_ADMIN','System Administrator',true,:'actor'),
  (:'org_id','FINANCE_MANAGER','Finance Manager',true,:'actor'),
  (:'org_id','ACCOUNTANT','Accountant',true,:'actor'),
  (:'org_id','PROCUREMENT_MANAGER','Procurement Manager',true,:'actor'),
  (:'org_id','SALES_MANAGER','Sales Manager',true,:'actor'),
  (:'org_id','HR_MANAGER','HR Manager',true,:'actor'),
  (:'org_id','AUDITOR','Auditor',true,:'actor'),
  (:'org_id','VIEWER','Viewer',true,:'actor')
ON CONFLICT (organization_id, code) DO NOTHING;

-- SYSTEM_ADMIN -> all permissions
INSERT INTO sec.role_permission (organization_id, role_id, permission_id, created_by)
SELECT :'org_id', r.id, p.id, :'actor'
FROM sec.role r JOIN sec.permission p ON p.organization_id = r.organization_id
WHERE r.organization_id = :'org_id' AND r.code = 'SYSTEM_ADMIN'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- VIEWER / AUDITOR -> all read permissions
INSERT INTO sec.role_permission (organization_id, role_id, permission_id, created_by)
SELECT :'org_id', r.id, p.id, :'actor'
FROM sec.role r JOIN sec.permission p ON p.organization_id = r.organization_id AND p.action = 'read'
WHERE r.organization_id = :'org_id' AND r.code IN ('VIEWER','AUDITOR')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ACCOUNTANT -> create/read/update/write on financial modules + read across operational/extended
INSERT INTO sec.role_permission (organization_id, role_id, permission_id, created_by)
SELECT :'org_id', r.id, p.id, :'actor'
FROM sec.role r JOIN sec.permission p ON p.organization_id = r.organization_id
  AND (
    (p.module IN ('gl','coa','ap','ar','tax','treasury','currency','fixedasset','financialclose') AND p.action IN ('create','read','update','write'))
    OR (p.module IN ('reporting','org','workflow','inventory','procurement','sales','manufacturing','masterdata','dimension','localization','budget','consolidation') AND p.action = 'read')
  )
WHERE r.organization_id = :'org_id' AND r.code = 'ACCOUNTANT'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- FINANCE_MANAGER -> full financial CRUD + approval across core & extended finance, read across ops/payroll/org
INSERT INTO sec.role_permission (organization_id, role_id, permission_id, created_by)
SELECT :'org_id', r.id, p.id, :'actor'
FROM sec.role r JOIN sec.permission p ON p.organization_id = r.organization_id
  AND (
    (p.module IN ('gl','coa','ap','ar','tax','treasury','currency','reporting','dimension','fixedasset','budget','financialclose','consolidation','workflow','masterdata','payroll') AND p.action IN ('create','read','update','write','approve'))
    OR (p.module IN ('procurement') AND p.action IN ('read','approve'))
    OR (p.module IN ('org','audit','inventory','sales','manufacturing','hr','localization','project','asset','maintenance','crm') AND p.action = 'read')
  )
WHERE r.organization_id = :'org_id' AND r.code = 'FINANCE_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;
