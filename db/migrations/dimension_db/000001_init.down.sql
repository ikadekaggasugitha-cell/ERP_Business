-- dimension_db — rollback 000001_init

DROP SCHEMA IF EXISTS pc CASCADE;
DROP SCHEMA IF EXISTS cc CASCADE;
DROP FUNCTION IF EXISTS public.current_org_scope();
DROP FUNCTION IF EXISTS public.tg_touch_row();
