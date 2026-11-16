-- Seed: org_db (organization hierarchy level types). Idempotent. Run: psql -d org_db -f this
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO org.organization_type (organization_id, code, name, level_hint, is_active, created_by) VALUES
  (:'org_id', 'HOLDING',       'Holding Company',       1, true, :'actor'),
  (:'org_id', 'SUB_HOLDING',   'Sub-Holding Group',     2, true, :'actor'),
  (:'org_id', 'COMPANY',       'Operating Company',     3, true, :'actor'),
  (:'org_id', 'BUSINESS_UNIT', 'Business Unit',         4, true, :'actor'),
  (:'org_id', 'BRANCH',        'Branch Office',         5, true, :'actor'),
  (:'org_id', 'PROJECT_SITE',  'Project Site',          6, true, :'actor'),
  (:'org_id', 'WAREHOUSE',     'Warehouse / Facility',  7, true, :'actor')
ON CONFLICT (organization_id, code) DO NOTHING;

