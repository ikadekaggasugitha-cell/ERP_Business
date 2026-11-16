-- Seed: localization_db (languages + locale formats). Idempotent. Run: psql -d localization_db -f this
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO loc.language (organization_id, code, name, is_active, is_fallback, created_by) VALUES
  (:'org_id', 'en', 'English',           true, true,  :'actor'),
  (:'org_id', 'id', 'Bahasa Indonesia',  true, false, :'actor')
ON CONFLICT (organization_id, code) DO NOTHING;

INSERT INTO loc.locale_format (organization_id, locale_code, date_format, number_format, currency_format, created_by) VALUES
  (:'org_id', 'id-ID', 'dd/MM/yyyy', '#.##0,00', 'Rp #.##0,00', :'actor'),
  (:'org_id', 'en-US', 'yyyy-MM-dd', '#,##0.00', '$#,##0.00',    :'actor')
ON CONFLICT (organization_id, locale_code) DO NOTHING;

