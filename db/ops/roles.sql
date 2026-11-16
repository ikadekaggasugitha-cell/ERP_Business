-- =============================================================================
-- Cluster roles (run ONCE per PostgreSQL cluster, as superuser).
-- Closes REVIEW O1: the application connects as a NON-OWNER role WITHOUT
-- BYPASSRLS so Row-Level Security is actually enforced. Migrations run as the
-- owner role (erp_migrator); the app never owns the tables.
--   psql -U postgres -f db/ops/roles.sql
-- =============================================================================

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='erp_migrator') THEN CREATE ROLE erp_migrator NOLOGIN; END IF;  -- owns objects; runs migrations
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='erp_app_rw')  THEN CREATE ROLE erp_app_rw  NOLOGIN; END IF;  -- app read/write (RLS applies)
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='erp_app_ro')  THEN CREATE ROLE erp_app_ro  NOLOGIN; END IF;  -- reporting/read-only
END $$;

-- Never superuser / never bypass RLS (defense-in-depth invariant).
ALTER ROLE erp_migrator NOSUPERUSER NOBYPASSRLS;
ALTER ROLE erp_app_rw   NOSUPERUSER NOBYPASSRLS;
ALTER ROLE erp_app_ro   NOSUPERUSER NOBYPASSRLS;

-- Production safety defaults (tune per workload).
ALTER ROLE erp_app_rw SET statement_timeout = '30s';
ALTER ROLE erp_app_rw SET idle_in_transaction_session_timeout = '60s';
ALTER ROLE erp_app_rw SET lock_timeout = '5s';
ALTER ROLE erp_app_ro SET statement_timeout = '120s';
ALTER ROLE erp_app_ro SET default_transaction_read_only = on;

-- Login roles are created per-environment and GRANTed membership, e.g.:
--   CREATE ROLE gl_app LOGIN PASSWORD :'pw' IN ROLE erp_app_rw;
--   CREATE ROLE reporting_app LOGIN PASSWORD :'pw' IN ROLE erp_app_ro;
-- (Membership keeps them non-owner, so RLS still applies.)
