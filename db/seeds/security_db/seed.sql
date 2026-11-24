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

-- System roles (15 roles: 4 tiered admins + 11 functional division roles)
INSERT INTO sec.role (id, organization_id, code, name, is_system, admin_tier, default_scope_type, created_by) VALUES
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000001'::uuid ELSE md5(:'org_id' || ':role:SYSTEM_ADMIN')::uuid END,         :'org_id','SYSTEM_ADMIN',         'System Administrator',                 true, 0,  'all',          :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000002'::uuid ELSE md5(:'org_id' || ':role:HOLDING_ADMIN')::uuid END,        :'org_id','HOLDING_ADMIN',        'Head Office / Holding Administrator',  true, 1,  'holding',      :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000003'::uuid ELSE md5(:'org_id' || ':role:SUBSIDIARY_ADMIN')::uuid END,     :'org_id','SUBSIDIARY_ADMIN',     'Subsidiary / Company Administrator',   true, 2,  'subsidiary',   :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000004'::uuid ELSE md5(:'org_id' || ':role:BRANCH_ADMIN')::uuid END,         :'org_id','BRANCH_ADMIN',         'Branch / Unit Administrator',          true, 3,  'branch',       :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000005'::uuid ELSE md5(:'org_id' || ':role:FINANCE_MANAGER')::uuid END,      :'org_id','FINANCE_MANAGER',      'Finance Manager',                      true, 99, 'organization', :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000006'::uuid ELSE md5(:'org_id' || ':role:ACCOUNTANT')::uuid END,           :'org_id','ACCOUNTANT',           'Accountant',                           true, 99, 'organization', :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000007'::uuid ELSE md5(:'org_id' || ':role:TAX_TREASURY_OFFICER')::uuid END, :'org_id','TAX_TREASURY_OFFICER', 'Tax & Treasury Officer',               true, 99, 'organization', :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000008'::uuid ELSE md5(:'org_id' || ':role:PROCUREMENT_MANAGER')::uuid END,  :'org_id','PROCUREMENT_MANAGER',  'Procurement Manager',                  true, 99, 'organization', :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000009'::uuid ELSE md5(:'org_id' || ':role:WAREHOUSE_MANAGER')::uuid END,    :'org_id','WAREHOUSE_MANAGER',    'Warehouse & Logistics Manager',        true, 99, 'organization', :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000010'::uuid ELSE md5(:'org_id' || ':role:SALES_MANAGER')::uuid END,        :'org_id','SALES_MANAGER',        'Sales & CRM Manager',                  true, 99, 'organization', :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000011'::uuid ELSE md5(:'org_id' || ':role:OPERATIONS_MANAGER')::uuid END,   :'org_id','OPERATIONS_MANAGER',   'Operations, Project & Maintenance Mgr',true, 99, 'organization', :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000012'::uuid ELSE md5(:'org_id' || ':role:HR_MANAGER')::uuid END,           :'org_id','HR_MANAGER',           'HR & Payroll Manager',                 true, 99, 'organization', :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000013'::uuid ELSE md5(:'org_id' || ':role:AUDITOR')::uuid END,              :'org_id','AUDITOR',              'Internal Auditor',                     true, 99, 'organization', :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000014'::uuid ELSE md5(:'org_id' || ':role:STAFF_OPERATOR')::uuid END,       :'org_id','STAFF_OPERATOR',       'Operational Staff',                    true, 99, 'organization', :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '5e000000-0000-4000-8000-000000000015'::uuid ELSE md5(:'org_id' || ':role:VIEWER')::uuid END,               :'org_id','VIEWER',               'Viewer',                               true, 99, 'organization', :'actor')
ON CONFLICT (organization_id, code) DO UPDATE
  SET admin_tier = EXCLUDED.admin_tier,
      default_scope_type = EXCLUDED.default_scope_type,
      name = EXCLUDED.name;

-- SYSTEM_ADMIN & HOLDING_ADMIN -> all permissions across all 30 modules
INSERT INTO sec.role_permission (organization_id, role_id, permission_id, created_by)
SELECT :'org_id', r.id, p.id, :'actor'
FROM sec.role r JOIN sec.permission p ON p.organization_id = r.organization_id
WHERE r.organization_id = :'org_id' AND r.code IN ('SYSTEM_ADMIN','HOLDING_ADMIN')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- SUBSIDIARY_ADMIN -> full subsidiary operations + local user/security/org management, excluding Head Office consolidation
INSERT INTO sec.role_permission (organization_id, role_id, permission_id, created_by)
SELECT :'org_id', r.id, p.id, :'actor'
FROM sec.role r JOIN sec.permission p ON p.organization_id = r.organization_id
WHERE r.organization_id = :'org_id'
  AND r.code = 'SUBSIDIARY_ADMIN'
  AND p.module <> 'consolidation'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- BRANCH_ADMIN -> branch-level operations + local user management, excluding consolidation & security role administration
INSERT INTO sec.role_permission (organization_id, role_id, permission_id, created_by)
SELECT :'org_id', r.id, p.id, :'actor'
FROM sec.role r JOIN sec.permission p ON p.organization_id = r.organization_id
WHERE r.organization_id = :'org_id'
  AND r.code = 'BRANCH_ADMIN'
  AND p.module NOT IN ('consolidation','security')
  AND NOT (p.module = 'org' AND p.action IN ('create','update','delete','write'))
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Default role_scope entries for tiered admin roles
INSERT INTO sec.role_scope (organization_id, role_id, scope_type, created_by)
SELECT :'org_id', r.id, r.default_scope_type, :'actor'
FROM sec.role r
WHERE r.organization_id = :'org_id'
  AND r.code IN ('SYSTEM_ADMIN','HOLDING_ADMIN','SUBSIDIARY_ADMIN','BRANCH_ADMIN')
  AND NOT EXISTS (
    SELECT 1 FROM sec.role_scope rs
    WHERE rs.role_id = r.id AND rs.scope_type = r.default_scope_type AND rs.deleted_at IS NULL
  );

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

-- TAX_TREASURY_OFFICER
INSERT INTO sec.role_permission (organization_id, role_id, permission_id, created_by)
SELECT :'org_id', r.id, p.id, :'actor'
FROM sec.role r JOIN sec.permission p ON p.organization_id = r.organization_id
  AND (
    (p.module IN ('tax','treasury','currency') AND p.action IN ('create','read','update','write','approve'))
    OR (p.module IN ('ap','ar','gl','coa','reporting','org','workflow') AND p.action = 'read')
  )
WHERE r.organization_id = :'org_id' AND r.code = 'TAX_TREASURY_OFFICER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- WAREHOUSE_MANAGER
INSERT INTO sec.role_permission (organization_id, role_id, permission_id, created_by)
SELECT :'org_id', r.id, p.id, :'actor'
FROM sec.role r JOIN sec.permission p ON p.organization_id = r.organization_id
  AND (
    (p.module IN ('inventory','asset') AND p.action IN ('create','read','update','write','approve'))
    OR (p.module IN ('procurement','sales','maintenance','masterdata','reporting','org','workflow') AND p.action = 'read')
  )
WHERE r.organization_id = :'org_id' AND r.code = 'WAREHOUSE_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- OPERATIONS_MANAGER
INSERT INTO sec.role_permission (organization_id, role_id, permission_id, created_by)
SELECT :'org_id', r.id, p.id, :'actor'
FROM sec.role r JOIN sec.permission p ON p.organization_id = r.organization_id
  AND (
    (p.module IN ('manufacturing','project','asset','maintenance','inventory') AND p.action IN ('create','read','update','write','approve'))
    OR (p.module IN ('procurement','dimension','masterdata','reporting','org','workflow') AND p.action = 'read')
  )
WHERE r.organization_id = :'org_id' AND r.code = 'OPERATIONS_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- STAFF_OPERATOR (create/read/update operational drafts without approval)
INSERT INTO sec.role_permission (organization_id, role_id, permission_id, created_by)
SELECT :'org_id', r.id, p.id, :'actor'
FROM sec.role r JOIN sec.permission p ON p.organization_id = r.organization_id
  AND (
    (p.module IN ('inventory','procurement','sales','manufacturing','project','asset','maintenance','crm') AND p.action IN ('create','read','update','write'))
    OR (p.module IN ('masterdata','org','workflow') AND p.action = 'read')
  )
WHERE r.organization_id = :'org_id' AND r.code = 'STAFF_OPERATOR'
ON CONFLICT (role_id, permission_id) DO NOTHING;
