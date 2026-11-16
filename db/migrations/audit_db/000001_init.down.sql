-- audit_db — rollback 000001_init

DROP SCHEMA IF EXISTS audit CASCADE;
DROP FUNCTION IF EXISTS public.tg_immutable();
DROP FUNCTION IF EXISTS public.current_org_scope();
