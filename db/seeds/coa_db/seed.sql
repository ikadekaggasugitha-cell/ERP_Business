-- Seed: coa_db (fiscal calendar template + FY2026). Idempotent. Run: psql -d coa_db -f this
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif
-- deterministic ids so re-runs & child FKs are stable
\set cal_id  'ca1e0000-0000-0000-0000-000000000001'
\set year_id 'fea62026-0000-0000-0000-000000000001'

INSERT INTO fin.fiscal_calendar (id, organization_id, code, name, period_type, created_by)
VALUES (:'cal_id', :'org_id','STD-MONTHLY','Standard Monthly Calendar','Monthly',:'actor')
ON CONFLICT (organization_id, code) DO NOTHING;

INSERT INTO fin.fiscal_year (id, organization_id, fiscal_calendar_id, year_no, name, start_date, end_date, status, created_by)
VALUES (:'year_id', :'org_id', :'cal_id', 2026, 'FY2026','2026-01-01','2026-12-31','Open',:'actor')
ON CONFLICT (fiscal_calendar_id, year_no) DO NOTHING;

INSERT INTO fin.fiscal_period (organization_id, fiscal_year_id, period_no, start_date, end_date, status, created_by)
SELECT :'org_id', :'year_id', m,
       make_date(2026, m, 1),
       (make_date(2026, m, 1) + INTERVAL '1 month - 1 day')::date,
       'Open', :'actor'
FROM generate_series(1,12) m
ON CONFLICT (fiscal_year_id, period_no) DO NOTHING;
