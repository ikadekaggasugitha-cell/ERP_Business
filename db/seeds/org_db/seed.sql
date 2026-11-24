-- Seed: org_db (organization hierarchy types + starter Holding/Subsidiary/Branch + 8 Enterprise Divisions).
-- Idempotent. Run: psql -d org_db -f this
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO org.organization_type (organization_id, code, name, level_hint, is_active, created_by) VALUES
  (:'org_id', 'HOLDING',       'Holding Company',       1, true, :'actor'),
  (:'org_id', 'SUB_HOLDING',   'Sub-Holding Group',     2, true, :'actor'),
  (:'org_id', 'COMPANY',       'Operating Company',     3, true, :'actor'),
  (:'org_id', 'BUSINESS_UNIT', 'Business Unit',         4, true, :'actor'),
  (:'org_id', 'DIVISION',      'Division',              4, true, :'actor'),
  (:'org_id', 'DEPARTMENT',    'Department',            5, true, :'actor'),
  (:'org_id', 'BRANCH',        'Branch Office',         5, true, :'actor'),
  (:'org_id', 'PROJECT_SITE',  'Project Site',          6, true, :'actor'),
  (:'org_id', 'PLACE',         'Site / Place / Outlet', 6, true, :'actor'),
  (:'org_id', 'WAREHOUSE',     'Warehouse / Facility',  7, true, :'actor')
ON CONFLICT (organization_id, code) DO NOTHING;

-- Starter Enterprise Org Tree (seeded once under Holding tenant 11111111-1111-1111-1111-111111111111
-- when invoked with default system org_id).
DO $$
DECLARE
  v_arg_org uuid := :'org_id'::uuid;
  v_actor   uuid := :'actor'::uuid;
  v_hld     uuid := '11111111-1111-1111-1111-111111111111';
  v_sub_pro uuid := '22222222-2222-2222-2222-222222222222';
  v_sub_log uuid := '33333333-3333-3333-3333-333333333333';
  v_brc_jkt uuid := '44444444-4444-4444-4444-444444444444';
  v_t_hld   uuid;
  v_t_co    uuid;
  v_t_brc   uuid;
  v_t_div   uuid;
BEGIN
  IF v_arg_org <> '00000000-0000-0000-0000-000000000000'::uuid THEN
    RETURN;
  END IF;

  SELECT id INTO v_t_hld FROM org.organization_type WHERE organization_id = v_arg_org AND code = 'HOLDING' LIMIT 1;
  SELECT id INTO v_t_co  FROM org.organization_type WHERE organization_id = v_arg_org AND code = 'COMPANY' LIMIT 1;
  SELECT id INTO v_t_brc FROM org.organization_type WHERE organization_id = v_arg_org AND code = 'BRANCH'  LIMIT 1;
  SELECT id INTO v_t_div FROM org.organization_type WHERE organization_id = v_arg_org AND code = 'DIVISION' LIMIT 1;

  -- Root Holding, 2 Subsidiaries, 1 Branch
  INSERT INTO org.organization (id, organization_id, code, name, organization_type_id, parent_id, path, status, effective_from, created_by) VALUES
    (v_hld,     v_hld,     'HLD-01',  'PT Sinergi Investasi Properti (Head Office / Holding)', v_t_hld, NULL,      '/HLD-01',                  'Active', '2026-01-01', v_actor),
    (v_sub_pro, v_sub_pro, 'SUB-PRO', 'PT Sinergi Properti Jakarta (Anak Perusahaan)',         v_t_co,  v_hld,     '/HLD-01/SUB-PRO',          'Active', '2026-01-01', v_actor),
    (v_sub_log, v_sub_log, 'SUB-LOG', 'PT Sinergi Logistik Nusantara (Anak Perusahaan)',       v_t_co,  v_hld,     '/HLD-01/SUB-LOG',          'Active', '2026-01-01', v_actor),
    (v_brc_jkt, v_brc_jkt, 'BRC-JKT', 'Cabang Jakarta Selatan (Branch)',                       v_t_brc, v_sub_pro, '/HLD-01/SUB-PRO/BRC-JKT',  'Active', '2026-01-01', v_actor)
  ON CONFLICT (id) DO NOTHING;

  -- 8 Enterprise Divisions
  INSERT INTO org.organization (id, organization_id, code, name, organization_type_id, parent_id, path, status, effective_from, created_by) VALUES
    ('11111111-0000-4000-8000-000000000001', v_hld,     'DIV-FIN', 'Divisi Finance & Accounting',           v_t_div, v_hld,     '/HLD-01/DIV-FIN',         'Active', '2026-01-01', v_actor),
    ('11111111-0000-4000-8000-000000000002', v_hld,     'DIV-TAX', 'Divisi Tax & Treasury',                 v_t_div, v_hld,     '/HLD-01/DIV-TAX',         'Active', '2026-01-01', v_actor),
    ('11111111-0000-4000-8000-000000000003', v_sub_pro, 'DIV-PRC', 'Divisi Procurement & Purchasing',       v_t_div, v_sub_pro, '/HLD-01/SUB-PRO/DIV-PRC', 'Active', '2026-01-01', v_actor),
    ('11111111-0000-4000-8000-000000000004', v_sub_log, 'DIV-LOG', 'Divisi Warehouse & Logistics',          v_t_div, v_sub_log, '/HLD-01/SUB-LOG/DIV-LOG', 'Active', '2026-01-01', v_actor),
    ('11111111-0000-4000-8000-000000000005', v_sub_pro, 'DIV-SLS', 'Divisi Sales & CRM',                    v_t_div, v_sub_pro, '/HLD-01/SUB-PRO/DIV-SLS', 'Active', '2026-01-01', v_actor),
    ('11111111-0000-4000-8000-000000000006', v_sub_pro, 'DIV-OPS', 'Divisi Operations, Project & Maintenance', v_t_div, v_sub_pro, '/HLD-01/SUB-PRO/DIV-OPS', 'Active', '2026-01-01', v_actor),
    ('11111111-0000-4000-8000-000000000007', v_hld,     'DIV-HR',  'Divisi HR & Payroll',                   v_t_div, v_hld,     '/HLD-01/DIV-HR',          'Active', '2026-01-01', v_actor),
    ('11111111-0000-4000-8000-000000000008', v_hld,     'DIV-AUD', 'Divisi IT & Internal Audit',            v_t_div, v_hld,     '/HLD-01/DIV-AUD',         'Active', '2026-01-01', v_actor)
  ON CONFLICT (id) DO NOTHING;

  -- Self closures (depth 0)
  INSERT INTO org.organization_closure (organization_id, ancestor_id, descendant_id, depth, created_by)
  SELECT o.organization_id, o.id, o.id, 0, v_actor
  FROM org.organization o
  WHERE o.id IN (
    v_hld, v_sub_pro, v_sub_log, v_brc_jkt,
    '11111111-0000-4000-8000-000000000001','11111111-0000-4000-8000-000000000002',
    '11111111-0000-4000-8000-000000000003','11111111-0000-4000-8000-000000000004',
    '11111111-0000-4000-8000-000000000005','11111111-0000-4000-8000-000000000006',
    '11111111-0000-4000-8000-000000000007','11111111-0000-4000-8000-000000000008'
  )
  ON CONFLICT (ancestor_id, descendant_id) DO NOTHING;

  -- Direct parent closures (depth 1)
  INSERT INTO org.organization_closure (organization_id, ancestor_id, descendant_id, depth, created_by)
  SELECT o.organization_id, o.parent_id, o.id, 1, v_actor
  FROM org.organization o
  WHERE o.parent_id IS NOT NULL
    AND o.id IN (
      v_sub_pro, v_sub_log, v_brc_jkt,
      '11111111-0000-4000-8000-000000000001','11111111-0000-4000-8000-000000000002',
      '11111111-0000-4000-8000-000000000003','11111111-0000-4000-8000-000000000004',
      '11111111-0000-4000-8000-000000000005','11111111-0000-4000-8000-000000000006',
      '11111111-0000-4000-8000-000000000007','11111111-0000-4000-8000-000000000008'
    )
  ON CONFLICT (ancestor_id, descendant_id) DO NOTHING;

  -- Grandparent closures (Holding -> Branch & Subsidiary Divisions, depth 2)
  INSERT INTO org.organization_closure (organization_id, ancestor_id, descendant_id, depth, created_by)
  SELECT o.organization_id, p.parent_id, o.id, 2, v_actor
  FROM org.organization o
  JOIN org.organization p ON p.id = o.parent_id
  WHERE p.parent_id IS NOT NULL
    AND o.id IN (
      v_brc_jkt,
      '11111111-0000-4000-8000-000000000003','11111111-0000-4000-8000-000000000004',
      '11111111-0000-4000-8000-000000000005','11111111-0000-4000-8000-000000000006'
    )
  ON CONFLICT (ancestor_id, descendant_id) DO NOTHING;
END $$;
