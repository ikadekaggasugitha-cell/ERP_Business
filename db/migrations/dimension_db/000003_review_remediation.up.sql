-- =============================================================================
-- dimension_db — Review remediation (N2 naming, N4 redundant indexes)
-- Ref: db/REVIEW.md (N2, N4) | golang-migrate: 000003
-- =============================================================================

-- N2: normalize abbreviated names
ALTER TABLE cc.cost_center_assignment  RENAME CONSTRAINT fk_cca_cost_center   TO fk_cost_center_assignment_cost_center;
ALTER TABLE cc.cost_allocation_rule    RENAME CONSTRAINT fk_car_cost_center   TO fk_cost_allocation_rule_cost_center;
ALTER TABLE pc.profit_center_assignment RENAME CONSTRAINT fk_pca_profit_center TO fk_profit_center_assignment_profit_center;
ALTER INDEX cc.idx_cca_orgref  RENAME TO idx_cost_center_assignment_orgref;
ALTER INDEX cc.idx_car_source  RENAME TO idx_cost_allocation_rule_source;
ALTER INDEX cc.idx_car_targets RENAME TO idx_cost_allocation_rule_targets;
ALTER INDEX pc.idx_pca_orgref  RENAME TO idx_profit_center_assignment_orgref;

-- N4: drop redundant single-col indexes (covered by composite unique leading col)
DROP INDEX IF EXISTS cc.idx_cca_cost_center;
DROP INDEX IF EXISTS pc.idx_pca_profit_center;
