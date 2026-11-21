-- Seed: project_db (reference). Idempotent. Run: psql -d project_db -f this
-- Optional: -v org_id=<uuid> -v actor=<uuid> to seed for a specific tenant.
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO prj.project (id, organization_id, code, name, project_type, organization_ref, currency_code, status, created_by) VALUES
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN '90000000-0000-4000-8000-000000000001'::uuid ELSE md5(:'org_id' || ':prj:TMPL-CAPEX')::uuid END, :'org_id', 'TMPL-CAPEX', 'Standard CapEx Project Template', 'Capital', :'org_id', 'IDR', 'Draft', :'actor')
ON CONFLICT (organization_id, code) DO NOTHING;

INSERT INTO prj.wbs_element (id, organization_id, project_id, parent_id, code, name, level_no, created_by)
SELECT
  CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN v.id ELSE md5(:'org_id' || ':prj:wbs:' || v.code)::uuid END,
  :'org_id'::uuid, p.id, NULL, v.code, v.name, 1, :'actor'::uuid
FROM (VALUES
  ('90001000-0000-4000-8000-000000000001'::uuid, '01-INIT',  'Initiation & Planning'),
  ('90001000-0000-4000-8000-000000000002'::uuid, '02-EXEC',  'Execution & Construction'),
  ('90001000-0000-4000-8000-000000000003'::uuid, '03-CLOSE', 'Handover & Closeout')
) AS v(id, code, name)
JOIN prj.project p
  ON p.organization_id = :'org_id'::uuid AND p.code = 'TMPL-CAPEX'
ON CONFLICT (project_id, code) DO NOTHING;
