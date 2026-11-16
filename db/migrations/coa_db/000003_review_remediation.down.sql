-- coa_db — rollback 000003_review_remediation
-- N4: recreate dropped indexes
CREATE INDEX idx_fiscal_year_calendar    ON fin.fiscal_year (fiscal_calendar_id);
CREATE INDEX idx_fiscal_period_year      ON fin.fiscal_period (fiscal_year_id);
CREATE INDEX idx_ledger_finorg           ON fin.ledger (financial_organization_id);
CREATE INDEX idx_account_coa             ON coa.account (chart_of_accounts_id);
CREATE INDEX idx_account_segment_coa     ON coa.account_segment (chart_of_accounts_id);
CREATE INDEX idx_segment_value_segment   ON coa.segment_value (segment_id);
CREATE INDEX idx_account_combination_coa ON coa.account_combination (chart_of_accounts_id);

-- N2: revert renames
ALTER INDEX fin.idx_financial_organization_orgref              RENAME TO idx_finorg_orgref;
ALTER INDEX fin.idx_financial_organization_chart_of_accounts   RENAME TO idx_finorg_coa;
ALTER INDEX fin.idx_financial_organization_fiscal_calendar     RENAME TO idx_finorg_calendar;
ALTER TABLE fin.financial_organization RENAME CONSTRAINT fk_financial_organization_chart_of_accounts TO fk_finorg_coa;
ALTER TABLE fin.financial_organization RENAME CONSTRAINT fk_financial_organization_fiscal_calendar   TO fk_finorg_calendar;
