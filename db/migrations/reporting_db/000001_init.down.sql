-- reporting_db — rollback 000001_init

DROP SCHEMA IF EXISTS rpt CASCADE;
DROP FUNCTION IF EXISTS public.current_org_scope();
