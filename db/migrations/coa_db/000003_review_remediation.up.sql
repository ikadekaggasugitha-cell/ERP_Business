-- =============================================================================
-- coa_db — Review remediation (N2 naming, N4 redundant indexes)
-- Ref: db/REVIEW.md (N2, N4) | golang-migrate: 000003
-- =============================================================================

-- N2: normalize abbreviated names -> fk_<table>_<ref> / idx_<table>_<cols>
ALTER TABLE fin.financial_organization RENAME CONSTRAINT fk_finorg_coa      TO fk_financial_organization_chart_of_accounts;
ALTER TABLE fin.financial_organization RENAME CONSTRAINT fk_finorg_calendar TO fk_financial_organization_fiscal_calendar;
ALTER INDEX fin.idx_finorg_orgref   RENAME TO idx_financial_organization_orgref;
ALTER INDEX fin.idx_finorg_coa      RENAME TO idx_financial_organization_chart_of_accounts;
ALTER INDEX fin.idx_finorg_calendar RENAME TO idx_financial_organization_fiscal_calendar;

-- N4: drop redundant single-col indexes (covered by composite unique leading col)
DROP INDEX IF EXISTS fin.idx_fiscal_year_calendar;
DROP INDEX IF EXISTS fin.idx_fiscal_period_year;
DROP INDEX IF EXISTS fin.idx_ledger_finorg;
DROP INDEX IF EXISTS coa.idx_account_coa;
DROP INDEX IF EXISTS coa.idx_account_segment_coa;
DROP INDEX IF EXISTS coa.idx_segment_value_segment;
DROP INDEX IF EXISTS coa.idx_account_combination_coa;
