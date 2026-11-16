-- dimension_db — rollback 000003_review_remediation
-- N4: recreate dropped indexes
CREATE INDEX idx_cca_cost_center   ON cc.cost_center_assignment (cost_center_id);
CREATE INDEX idx_pca_profit_center ON pc.profit_center_assignment (profit_center_id);

-- N2: revert renames
ALTER INDEX cc.idx_cost_center_assignment_orgref   RENAME TO idx_cca_orgref;
ALTER INDEX cc.idx_cost_allocation_rule_source     RENAME TO idx_car_source;
ALTER INDEX cc.idx_cost_allocation_rule_targets    RENAME TO idx_car_targets;
ALTER INDEX pc.idx_profit_center_assignment_orgref RENAME TO idx_pca_orgref;
ALTER TABLE cc.cost_center_assignment   RENAME CONSTRAINT fk_cost_center_assignment_cost_center       TO fk_cca_cost_center;
ALTER TABLE cc.cost_allocation_rule     RENAME CONSTRAINT fk_cost_allocation_rule_cost_center         TO fk_car_cost_center;
ALTER TABLE pc.profit_center_assignment RENAME CONSTRAINT fk_profit_center_assignment_profit_center   TO fk_pca_profit_center;
