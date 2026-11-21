-- =============================================================================
-- security_db — Rollback Tiered Admin Hierarchy (000004_tiered_admin.down.sql)
-- =============================================================================

DELETE FROM sec.role_scope WHERE scope_type IN ('holding', 'subsidiary');

ALTER TABLE sec.role_scope DROP CONSTRAINT IF EXISTS ck_role_scope_type;
ALTER TABLE sec.role_scope
    ADD CONSTRAINT ck_role_scope_type
    CHECK (scope_type IN ('organization','branch','self','all'));

ALTER TABLE sec.role DROP CONSTRAINT IF EXISTS ck_role_default_scope_type;
ALTER TABLE sec.role DROP CONSTRAINT IF EXISTS ck_role_admin_tier;
ALTER TABLE sec.role
    DROP COLUMN IF EXISTS default_scope_type,
    DROP COLUMN IF EXISTS admin_tier;
