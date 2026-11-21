-- Widen audit.audit_log.action from varchar(50) to varchar(150) so full domain
-- event names (e.g. localization.locale_format.updated, maintenance.workorder.parts_requested)
-- never truncate or fail inserts.
ALTER TABLE audit.audit_log ALTER COLUMN action TYPE varchar(150);
