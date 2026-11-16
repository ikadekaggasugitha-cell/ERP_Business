-- Seed: tax_db (Indonesian PPN/PPh standard tax codes). Idempotent. Run: psql -d tax_db -f this
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO tax.tax_code (organization_id, code, rate, tax_type, created_by) VALUES
  (:'org_id', 'PPN-11',     11.0000, 'VAT', :'actor'),
  (:'org_id', 'PPN-12',     12.0000, 'VAT', :'actor'),
  (:'org_id', 'PPN-0',       0.0000, 'VAT', :'actor'),
  (:'org_id', 'PPH-21',      5.0000, 'WHT', :'actor'),
  (:'org_id', 'PPH-22',      1.5000, 'WHT', :'actor'),
  (:'org_id', 'PPH-23-SVC',  2.0000, 'WHT', :'actor'),
  (:'org_id', 'PPH-23-ROY', 15.0000, 'WHT', :'actor'),
  (:'org_id', 'PPH-4-2-CON', 2.6500, 'WHT', :'actor'),
  (:'org_id', 'PPH-4-2-RNT',10.0000, 'WHT', :'actor')
ON CONFLICT (organization_id, code) DO NOTHING;

