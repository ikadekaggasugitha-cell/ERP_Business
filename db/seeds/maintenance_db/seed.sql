-- Seed: maintenance_db (reference). Idempotent. Run: psql -d maintenance_db -f this
-- Optional: -v org_id=<uuid> -v actor=<uuid> to seed for a specific tenant.
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO mnt.maintenance_schedule (id, organization_id, asset_ref, schedule_type, interval_value, interval_unit, is_active, created_by)
SELECT
  CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN v.sys_id ELSE md5(:'org_id' || ':mnt:sched:' || v.asset_ref::text)::uuid END,
  :'org_id'::uuid,
  v.asset_ref,
  v.sched_type,
  v.int_val,
  v.int_unit,
  true,
  :'actor'::uuid
FROM (VALUES
  ('d0000000-0000-4000-8000-000000000001'::uuid, 'a55e0000-0000-4000-8000-000000000001'::uuid, 'TimeBased', 30, 'Day'),
  ('d0000000-0000-4000-8000-000000000002'::uuid, 'a55e0000-0000-4000-8000-000000000002'::uuid, 'TimeBased', 6,  'Month')
) AS v(sys_id, asset_ref, sched_type, int_val, int_unit)
WHERE NOT EXISTS (
  SELECT 1 FROM mnt.maintenance_schedule ms
  WHERE ms.organization_id = :'org_id'::uuid
    AND ms.asset_ref = v.asset_ref
    AND ms.schedule_type = v.sched_type
)
ON CONFLICT (id) DO NOTHING;
