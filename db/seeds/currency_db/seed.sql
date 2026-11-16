-- Seed: currency_db (reference). Idempotent. Run: psql -d currency_db -f this
-- Optional: -v org_id=<uuid> -v actor=<uuid> to seed for a specific tenant.
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO cur.currency (organization_id, code, numeric_code, name, precision, is_active, created_by) VALUES
  (:'org_id','IDR','360','Indonesian Rupiah',2,true,:'actor'),
  (:'org_id','USD','840','US Dollar',2,true,:'actor'),
  (:'org_id','EUR','978','Euro',2,true,:'actor'),
  (:'org_id','SGD','702','Singapore Dollar',2,true,:'actor'),
  (:'org_id','JPY','392','Japanese Yen',0,true,:'actor'),
  (:'org_id','CNY','156','Chinese Yuan',2,true,:'actor'),
  (:'org_id','MYR','458','Malaysian Ringgit',2,true,:'actor'),
  (:'org_id','AUD','036','Australian Dollar',2,true,:'actor'),
  (:'org_id','GBP','826','Pound Sterling',2,true,:'actor')
ON CONFLICT (organization_id, code) DO NOTHING;

INSERT INTO cur.exchange_rate_type (organization_id, code, name, created_by) VALUES
  (:'org_id','SPOT','Spot Rate',:'actor'),
  (:'org_id','CLOSING','Closing Rate',:'actor'),
  (:'org_id','AVERAGE','Average Rate',:'actor'),
  (:'org_id','HISTORICAL','Historical Rate',:'actor'),
  (:'org_id','BUDGET','Budget Rate',:'actor')
ON CONFLICT (organization_id, code) DO NOTHING;
