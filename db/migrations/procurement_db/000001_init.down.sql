-- procurement_db — rollback 000001_init

DROP SCHEMA IF EXISTS gr CASCADE;
DROP SCHEMA IF EXISTS po CASCADE;
DROP SCHEMA IF EXISTS pr CASCADE;
DROP FUNCTION IF EXISTS public.current_org_scope();
DROP FUNCTION IF EXISTS public.tg_touch_row();
