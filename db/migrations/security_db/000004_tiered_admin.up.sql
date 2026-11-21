-- =============================================================================
-- security_db — Tiered Admin Hierarchy (000004_tiered_admin.up.sql)
-- Adds admin_tier (0=SYSTEM_ADMIN, 1=HOLDING_ADMIN, 2=SUBSIDIARY_ADMIN,
-- 3=BRANCH_ADMIN, 99=NON_ADMIN) and default_scope_type to sec.role, and widens
-- sec.role_scope.scope_type to include 'holding' and 'subsidiary'.
-- =============================================================================

ALTER TABLE sec.role
    ADD COLUMN IF NOT EXISTS admin_tier         SMALLINT    NOT NULL DEFAULT 99,
    ADD COLUMN IF NOT EXISTS default_scope_type VARCHAR(30) NOT NULL DEFAULT 'organization';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'ck_role_admin_tier'
    ) THEN
        ALTER TABLE sec.role
            ADD CONSTRAINT ck_role_admin_tier CHECK (admin_tier IN (0, 1, 2, 3, 99));
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'ck_role_default_scope_type'
    ) THEN
        ALTER TABLE sec.role
            ADD CONSTRAINT ck_role_default_scope_type
            CHECK (default_scope_type IN ('all','holding','subsidiary','organization','branch','self'));
    END IF;
END $$;

ALTER TABLE sec.role_scope DROP CONSTRAINT IF EXISTS ck_role_scope_type;
ALTER TABLE sec.role_scope
    ADD CONSTRAINT ck_role_scope_type
    CHECK (scope_type IN ('all','holding','subsidiary','organization','branch','self'));

UPDATE sec.role SET admin_tier = 0, default_scope_type = 'all'        WHERE code = 'SYSTEM_ADMIN';
UPDATE sec.role SET admin_tier = 1, default_scope_type = 'holding'    WHERE code = 'HOLDING_ADMIN';
UPDATE sec.role SET admin_tier = 2, default_scope_type = 'subsidiary' WHERE code = 'SUBSIDIARY_ADMIN';
UPDATE sec.role SET admin_tier = 3, default_scope_type = 'branch'     WHERE code = 'BRANCH_ADMIN';
