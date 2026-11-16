-- crm_db — rollback 000001_init

DROP SCHEMA IF EXISTS crm CASCADE;
DROP FUNCTION IF EXISTS public.current_org_scope();
DROP FUNCTION IF EXISTS public.tg_touch_row();
