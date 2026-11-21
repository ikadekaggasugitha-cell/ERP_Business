-- Seed: asset_db (reference). Idempotent. Run: psql -d asset_db -f this
-- Optional: -v org_id=<uuid> -v actor=<uuid> to seed for a specific tenant.
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO ast.asset (id, organization_id, asset_code, name, category, specification, organization_ref, status, created_by) VALUES
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN 'a55e0000-0000-4000-8000-000000000001'::uuid ELSE md5(:'org_id' || ':ast:AST-TMPL-HVAC')::uuid END, :'org_id', 'AST-TMPL-HVAC', 'Reference HVAC Unit Template', 'HVAC', '{"template":true,"preventiveIntervalDays":30}'::jsonb, :'org_id', 'Idle', :'actor'),
  (CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN 'a55e0000-0000-4000-8000-000000000002'::uuid ELSE md5(:'org_id' || ':ast:AST-TMPL-GEN')::uuid END,  :'org_id', 'AST-TMPL-GEN',  'Reference Backup Generator Template', 'POWER', '{"template":true,"preventiveIntervalDays":180}'::jsonb, :'org_id', 'Idle', :'actor')
ON CONFLICT (organization_id, asset_code) DO NOTHING;
