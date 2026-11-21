-- Seed: coa_db (fiscal calendar template + FY2026). Idempotent. Run: psql -d coa_db -f this
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif
-- deterministic ids so re-runs & child FKs are stable
\set cal_id  'ca1e0000-0000-0000-0000-000000000001'
\set year_id 'fea62026-0000-0000-0000-000000000001'

INSERT INTO fin.fiscal_calendar (id, organization_id, code, name, period_type, created_by)
VALUES (
  CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN :'cal_id'::uuid ELSE md5(:'org_id' || ':coa:cal:STD-MONTHLY')::uuid END,
  :'org_id', 'STD-MONTHLY', 'Standard Monthly Calendar', 'Monthly', :'actor'
)
ON CONFLICT (organization_id, code) DO NOTHING;

INSERT INTO fin.fiscal_year (id, organization_id, fiscal_calendar_id, year_no, name, start_date, end_date, status, created_by)
SELECT
  CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN :'year_id'::uuid ELSE md5(:'org_id' || ':coa:fy:2026')::uuid END,
  :'org_id'::uuid, c.id, 2026, 'FY2026', '2026-01-01'::date, '2026-12-31'::date, 'Open', :'actor'::uuid
FROM fin.fiscal_calendar c
WHERE c.organization_id = :'org_id'::uuid AND c.code = 'STD-MONTHLY'
ON CONFLICT (fiscal_calendar_id, year_no) DO NOTHING;

INSERT INTO fin.fiscal_period (organization_id, fiscal_year_id, period_no, start_date, end_date, status, created_by)
SELECT :'org_id'::uuid, y.id, m,
       make_date(2026, m, 1),
       (make_date(2026, m, 1) + INTERVAL '1 month - 1 day')::date,
       'Open', :'actor'::uuid
FROM fin.fiscal_year y
JOIN fin.fiscal_calendar c ON c.id = y.fiscal_calendar_id
CROSS JOIN generate_series(1,12) m
WHERE c.organization_id = :'org_id'::uuid AND c.code = 'STD-MONTHLY' AND y.year_no = 2026
ON CONFLICT (fiscal_year_id, period_no) DO NOTHING;
