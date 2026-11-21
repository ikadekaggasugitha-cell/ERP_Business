-- Seed: maintenance_db (reference). Idempotent. Run: psql -d maintenance_db -f this
-- Optional: -v org_id=<uuid> -v actor=<uuid> to seed for a specific tenant.
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO mnt.maintenance_schedule (id, organization_id, asset_ref, schedule_type, interval_value, interval_unit, is_active, created_by) VALUES
  ('d0000000-0000-4000-8000-000000000001', :'org_id', 'a55e0000-0000-4000-8000-000000000001', 'TimeBased', 30, 'Day', true, :'actor'),
  ('d0000000-0000-4000-8000-000000000002', :'org_id', 'a55e0000-0000-4000-8000-000000000002', 'TimeBased', 6, 'Month', true, :'actor')
ON CONFLICT (id) DO NOTHING;
