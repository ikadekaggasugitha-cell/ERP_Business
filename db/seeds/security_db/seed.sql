-- Seed: security_db (system RBAC). Idempotent. Run: psql -d security_db -f this
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

-- Permissions: <module>.<entity>.<action> for the core modules.
INSERT INTO sec.permission (organization_id, code, module, action, created_by)
SELECT :'org_id', m || '.' || a, m, a, :'actor'
FROM (VALUES ('org'),('user'),('security'),('gl'),('ap'),('ar'),('tax'),('treasury'),
             ('procurement'),('inventory'),('sales'),('hr'),('payroll'),('reporting')) mods(m)
CROSS JOIN (VALUES ('create'),('read'),('update'),('delete'),('approve')) acts(a)
ON CONFLICT (organization_id, code) DO NOTHING;

-- System roles
INSERT INTO sec.role (organization_id, code, name, is_system, created_by) VALUES
  (:'org_id','SYSTEM_ADMIN','System Administrator',true,:'actor'),
  (:'org_id','FINANCE_MANAGER','Finance Manager',true,:'actor'),
  (:'org_id','ACCOUNTANT','Accountant',true,:'actor'),
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

-- ACCOUNTANT -> create/read/update on financial modules
INSERT INTO sec.role_permission (organization_id, role_id, permission_id, created_by)
SELECT :'org_id', r.id, p.id, :'actor'
FROM sec.role r JOIN sec.permission p ON p.organization_id = r.organization_id
  AND p.module IN ('gl','ap','ar','tax','treasury') AND p.action IN ('create','read','update')
WHERE r.organization_id = :'org_id' AND r.code = 'ACCOUNTANT'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- FINANCE_MANAGER -> full financial CRUD + approval, read across ops/payroll/org, approve procurement/payroll
INSERT INTO sec.role_permission (organization_id, role_id, permission_id, created_by)
SELECT :'org_id', r.id, p.id, :'actor'
FROM sec.role r JOIN sec.permission p ON p.organization_id = r.organization_id
  AND (
    (p.module IN ('gl','ap','ar','tax','treasury','reporting') AND p.action IN ('create','read','update','approve'))
    OR (p.module IN ('procurement','payroll') AND p.action IN ('read','approve'))
    OR (p.module IN ('org','inventory','sales','hr') AND p.action = 'read')
  )
WHERE r.organization_id = :'org_id' AND r.code = 'FINANCE_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;
